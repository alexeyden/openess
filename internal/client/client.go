package client

import (
	"errors"
	"fmt"
	"io"
	"net"
	"openess/internal/commands"
	"openess/internal/log"
	"openess/internal/protocol"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Config struct {
	LocalPort  int
	DeviceAddr string
	ProtoPath  string
}

type Response struct {
	Response commands.Result
	Error    error
}

type clientTask struct {
	config          Config
	conn            *protocol.Device
	rxCom           chan commands.Command
	txResp          chan Response
	descMtx         *sync.Mutex
	desc            **protocol.Descriptor
	isConnectedCond *sync.Cond
	isConnected     *bool
}

type Client struct {
	config          Config
	txCom           chan commands.Command
	rxResp          chan Response
	descMtx         *sync.Mutex
	desc            **protocol.Descriptor
	isConnectedCond *sync.Cond
	isConnected     *bool
}

func StartClient(config Config) *Client {
	com := make(chan commands.Command)
	resp := make(chan Response)

	var desc *protocol.Descriptor

	var descMtx = new(sync.Mutex)
	var isConnectedCond = sync.NewCond(new(sync.Mutex))
	var isConnected = new(bool)

	client := Client{
		config:          config,
		txCom:           com,
		rxResp:          resp,
		descMtx:         descMtx,
		desc:            &desc,
		isConnectedCond: isConnectedCond,
		isConnected:     isConnected,
	}

	task := clientTask{
		config:          config,
		conn:            nil,
		rxCom:           com,
		txResp:          resp,
		descMtx:         descMtx,
		desc:            &desc,
		isConnectedCond: isConnectedCond,
		isConnected:     isConnected,
	}

	go task.eventLoop()

	return &client
}

func (this *Client) GetDescriptor() *protocol.Descriptor {
	this.descMtx.Lock()
	defer this.descMtx.Unlock()
	return *this.desc
}

func (this *Client) IsConnected() bool {
    this.isConnectedCond.L.Lock()
	defer this.isConnectedCond.L.Unlock()
	return *this.isConnected
}

func (this *Client) WaitConnection() {
	this.isConnectedCond.L.Lock()

	for !*this.isConnected {
		this.isConnectedCond.Wait()
	}

	this.isConnectedCond.L.Unlock()
}

func SendCommand[O commands.Result, R commands.ResultCast[O]](client *Client, req R) (*O, error) {
	client.txCom <- req
	val := <-client.rxResp

	if val.Response == nil {
		return nil, val.Error
	}

	r := req.CastResult(val.Response)

	return &r, val.Error
}

func (task *clientTask) connect() error {
	log.PrInfo("client: connecting to device %s\n", task.config.DeviceAddr)

	conn, err := protocol.Connect(task.config.DeviceAddr, task.config.LocalPort)
	if err != nil {
		log.PrError("client: failed to connect: %s\n", err)
		return err
	}

	infoCmd := commands.NewDeviceInfo()
	res, err := infoCmd.Handle(*conn, nil)
	if err != nil {
		return err
	}

	infoRes := infoCmd.CastResult(res)

	log.PrInfo("client: connected to datalogger: manufacturer %s device type %s (protocol v%s props %s)\n", infoRes.Manufacturer, infoRes.DeviceType, infoRes.ProtocolVersion, infoRes.DeviceProps)
	log.PrDebug("client: datalogger info %+v\n", infoRes)

	protoName := strings.Split(infoRes.DeviceProps, ",")[0]

	if protoName != "0925" {
		log.PrInfo("client: WARNING protocols other than 0925 are untested\n")
	}

    protocolFilePath := fmt.Sprintf(
		"%s/%s.json", task.config.ProtoPath, protoName)

	descriptor, err := protocol.LoadProtocolDescriptor(protocolFilePath)
	if err != nil {
		log.PrError("failed to load protocol descriptor: %s\n", err)
	}

	log.PrInfo("client: loaded protocol descriptor: %s\n", protocolFilePath)

	task.conn = conn

	task.descMtx.Lock()
	*task.desc = descriptor
	task.descMtx.Unlock()

    task.isConnectedCond.L.Lock()
	*task.isConnected = true
	task.isConnectedCond.L.Unlock()
	task.isConnectedCond.Broadcast()

	return nil
}

func (client *clientTask) handleCmd(req commands.Command) Response {
	var resp Response

	result, err := req.Handle(*client.conn, *client.desc)

	resp.Response = result
	resp.Error = err

	return resp
}

func isEofError(err error) bool {
	if err == io.EOF || errors.Is(err, syscall.EPIPE) {
		return true
	}

	return false
}

func isTimeoutError(err error) bool {
	if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
	    return true
	}

	return false
}

func (client *clientTask) eventLoop() {
	for {
		var backoff time.Duration = time.Second * 2

		for {
			time.Sleep(backoff)

			err := client.connect()

			if err != nil {
				log.PrError("client: failed to connect: %s, will try again in %.d sec\n", err, int(backoff.Seconds()))
				if backoff.Minutes() < 5 {
					backoff *= 2
				}
				continue
			}
	
			break
		}

		var err error
	cmdLoop:
		for {
			select {
			case cmd := <-client.rxCom:
				resp := client.handleCmd(cmd)
				client.txResp <- resp

				if isEofError(resp.Error) || isTimeoutError(resp.Error) {
					err = resp.Error
					client.conn.Close()
					client.conn = nil
					break cmdLoop
				}

			case <-time.After(3 * time.Second):
				log.PrDebug("client: sending heartbeat\n")
				cmd := commands.NewPing()
				resp := client.handleCmd(cmd)

				if resp.Error != nil {
				    log.PrDebug("client: heartbeat error: %v\n", resp.Error)
				}

				if isEofError(resp.Error) {
					err = resp.Error
					client.conn.Close()
					client.conn = nil
					break cmdLoop
				}
			}
		}

		client.isConnectedCond.L.Lock()
		*client.isConnected = false
		client.isConnectedCond.L.Unlock()
		client.isConnectedCond.Broadcast()

		log.PrError("client: connection lost: %s\n", err)
	}
}
