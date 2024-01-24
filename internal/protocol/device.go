package protocol

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"openess/internal/log"
	"strings"
	"time"
)

type Device struct {
	socket         net.Conn
	defaultTimeout time.Duration
}

func Connect(deviceAddr string, localPort int) (*Device, error) {
	log.PrDebug("proto: starting up tcp server on %d port\n", localPort)

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", localPort))

	if err != nil {
		return nil, err
	}

	defer listener.Close()

	conn, err := net.Dial("udp", deviceAddr)

	if err != nil {
		return nil, err
	}

	defer conn.Close()

	localAddr := strings.Split(conn.LocalAddr().String(), ":")[0]

	log.PrDebug("proto: requesting datalogger connection to %s:%d\n", localAddr, localPort)

	conn.SetDeadline(time.Now().Add(time.Second * 5))

	_, err = conn.Write([]byte(fmt.Sprintf("set>server=%s:%d;", localAddr, localPort)))

	if err != nil {
		return nil, err
	}

	conn.SetDeadline(time.Now().Add(time.Second * 5))

	data, err := bufio.NewReader(conn).ReadString(';')

	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(data, "rsp>server=") {
		return nil, errors.New(fmt.Sprintf("unexpected answer: %s", data))
	}

	conn, err = listener.Accept()
	if err != nil {
		return nil, err
	}

	log.PrDebug("proto: datalogger conncted\n")

	return &Device{socket: conn, defaultTimeout: time.Second * 5}, nil
}

func (device *Device) Close() {
	device.socket.Close()
}

func WriteRequest[R any, T RequestBody[R]](conn Device, request Request[R, T]) error {
	body, err := request.Body.EncodeRequest()
	if err != nil {
		return err
	}

	request.Header.Size = (uint16)(len(body))

	// silly thing seems to be unable to handle the request if it is split into multiple packets (multiple Write()s)

	buffer := new(bytes.Buffer)

	err = request.Header.Write(buffer)
	if err != nil {
		return err
	}

	_, err = buffer.Write(body)
	if err != nil {
		return err
	}

	_, err = conn.socket.Write(buffer.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func readResponseBody[R any, T RequestBody[R]](conn Device, size uint16, request Request[R, T]) (*R, error) {
	body := make([]byte, size)

	_, err := io.ReadFull(conn.socket, body)

	if err != nil {
		return nil, err
	}

	log.PrDebug("proto: read body %s", hex.EncodeToString(body))

	resp, err := request.Body.DecodeResponse(body)

	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func ReadResponse[R any, T RequestBody[R]](conn Device, request Request[R, T]) (*Response[R], error) {
	timeout := conn.defaultTimeout

	if request.Timeout != 0 {
		timeout = request.Timeout
	}

	log.PrDebug("proto: setting timeout %d ms\n", timeout.Milliseconds())

	conn.socket.SetReadDeadline(time.Now().Add(timeout))

	header, err := ReadHeader(conn.socket)
	if err != nil {
		return nil, err
	}

	log.PrDebug("proto: read header %+v\n", header)

	body, err := readResponseBody(conn, header.Size, request)
	if err != nil {
		return nil, err
	}

	if header.FuncCode != request.Header.FuncCode {
		return nil, errors.New(fmt.Sprintf("unexpected fcode in response (expected %d got %d)", request.Header.FuncCode, header.FuncCode))
	}

	return &Response[R]{
		Header: header,
		Body:   *body,
	}, nil
}
