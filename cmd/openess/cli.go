package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"openess/internal/client"
	"openess/internal/commands"
	"openess/internal/log"
	"openess/internal/protocol"
	"os"
	"strconv"
	"strings"
)

func readLine(reader *bufio.Reader) ([]string, error) {
	text, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	var args []string
	var quote bool = false
	var sb strings.Builder

	for _, c := range []byte(text) {
		if c == '"' && !quote {
			quote = true
			continue
		}
		if c == '"' && quote {
			quote = false
			continue
		}
		if c == '\n' {
			continue
		}
		if c == ' ' && !quote {
			if sb.Len() == 0 {
				continue
			}
			args = append(args, sb.String())
			sb.Reset()
			continue
		}
		sb.WriteByte(c)
	}

	if sb.Len() != 0 {
		args = append(args, sb.String())
	}

	return args, nil
}

func InteractiveMain(args Args) {
	config, err := LoadConfig(args.ConfPath)
	if err != nil {
		log.PrError("openess: failed to read config %s: %s", args.ConfPath, err)
		os.Exit(1)
	}

	clientConfig := client.Config{
		DeviceAddr: config.DeviceAddr,
		LocalPort:  config.BindPort,
		ProtoPath:  config.ProtoPath,
		Protocol:   config.Protocol,
	}

	if args.DeviceAddr != nil {
		clientConfig.DeviceAddr = *args.DeviceAddr
	}

	log.Init(args.LogLevel)

	var cli = client.StartClient(clientConfig)
	cli.WaitConnection()

	reader := bufio.NewReader(os.Stdin)

	for true {
		fmt.Print("% ")

		args, err := readLine(reader)
		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Printf("error: %s\n", err)
			continue
		}

		if len(args) == 0 {
			fmt.Printf("error: invalid command\n")
			continue
		}

		switch args[0] {
		case "help":
			fmt.Printf("Commands:\n")
			fmt.Printf("exit                                          Exit from cli\n")
			fmt.Printf("info                                          Read datalogger info\n")
			fmt.Printf("set-param PARAM                               Set datalogger param (ssid, password, restart)\n")
			fmt.Printf("ping                                          Ping datalogger\n")
			fmt.Printf("read-named NAME                               Read single register value (looks up register in descriptor by NAME)\n")
			fmt.Printf("read-all [SEGMENT]                            Read all registers in SEGMENT. If segment is not specified, attempts to read all registers in descriptor\n")
			fmt.Printf("read-raw DEV_ADDR FUNCTION REG_ADDR LENGTH    Read register range as hex dump\n")
			fmt.Printf("write-raw DEV_ADDR FUNCTION REG_ADDR DATA     Write single register as hex string\n")
			fmt.Printf("write-named NAME VALUE                        Write single register as integer (looks up register in descriptor by NAME)\n")
		case "exit":
			os.Exit(0)
		case "info":
			info := commands.NewDeviceInfo()
			resp, err := client.SendCommand(cli, info)

			if err != nil {
				fmt.Printf("request failed: %s\n", err)
				continue
			}

			fmt.Printf("response: %+v\n", resp)
		case "set-param":
			var par byte
			name := args[1]
			value := args[2]
			switch name {
			case "ssid":
				par = commands.DEVICE_PARAM_SSID
			case "password":
				par = commands.DEVICE_PARAM_PASSWORD
			case "restart":
				par = commands.DEVICE_PARAM_RESTART
			default:
				fmt.Printf("invalid param: %s\n", args[1])
				continue
			}
			req := commands.NewDeviceParam(par, value)
			resp, err := client.SendCommand(cli, req)
			if err != nil {
				fmt.Printf("request failed: %s\n", err)
				continue
			}
			fmt.Printf("status: %d\n", resp.Status)
		case "ping":
			ping := commands.NewPing()
			resp, err := client.SendCommand(cli, ping)

			if err != nil {
				fmt.Printf("request failed: %s\n", err)
				continue
			}

			fmt.Printf("response: %+v\n", resp.Pn)
		case "read-named":
			name := args[1]
			desc := cli.GetDescriptor()
			seg, reg := desc.FindRegister(name)
			req := commands.NewRegReadDescr(seg, reg)
			resp, err := client.SendCommand(cli, req)
			if err != nil {
				fmt.Printf("request failed: %s\n", err)
				continue
			}
			fmt.Printf("%s\n", resp.Value.ToString())
		case "read-all":
			name := ""
			if len(args) == 2 {
				name = args[1]
			}
			desc := cli.GetDescriptor()
			segs := []protocol.Segment{}
			if name != "" {
				segs = desc.FindGroup(name)
			} else {
				for _, g := range desc.Configuration.SystemInfoVC {
					segs = append(segs, g.Segments...)
				}
				for _, g := range desc.Configuration.SystemSettingVC {
					segs = append(segs, g.Segments...)
				}
			}

			for _, seg := range segs {
				req := commands.NewRegReadSeg(&seg)
				resp, err := client.SendCommand(cli, req)
				if err != nil {
					fmt.Printf("segment %d request failed: %s\n", seg.StartAddress, err)
					continue
				}
				for addr, v := range resp.Values {
					name := ""
					reg := desc.FindRegisterByAddr(addr)
					if reg != nil {
						name = reg.Title["base"]
					}
					fmt.Printf("[%d] %s = %s\n", addr, name, v.ToString())
				}
			}
		case "read-raw":
			addr, _ := strconv.Atoi(args[1])
			fun, _ := strconv.Atoi(args[2])
			reg, _ := strconv.Atoi(args[3])
			length, _ := strconv.Atoi(args[4])

			req := commands.NewRegReadRaw(byte(addr), byte(fun), uint16(reg), uint16(length))
			resp, err := client.SendCommand(cli, req)

			if err != nil {
				fmt.Printf("request failed: %s\n", err)
				continue
			}

			fmt.Printf("%s\n", hex.EncodeToString(resp.Data))
		case "write-raw":
			addr, _ := strconv.Atoi(args[1])
			fun, _ := strconv.Atoi(args[2])
			reg, _ := strconv.Atoi(args[3])
			b, err := hex.DecodeString(args[4])
			if err != nil {
				fmt.Printf("invalid hex string: %s", err)
				continue
			}

			req := commands.NewRegWriteRaw(byte(addr), byte(fun), uint16(reg), b)
			resp, err := client.SendCommand(cli, req)

			if err != nil {
				fmt.Printf("request failed: %s\n", err)
				continue
			}

			fmt.Printf("%s\n", hex.EncodeToString(resp.Data))
		case "write-named":
			name := args[1]
			value, _ := strconv.ParseFloat(args[2], 32)
			desc := cli.GetDescriptor()
			seg, reg := desc.FindRegister(name)
			req := commands.NewRegWriteDescr(seg, reg, float32(value))
			resp, err := client.SendCommand(cli, req)
			if err != nil {
				fmt.Printf("request failed: %s\n", err)
				continue
			}
			fmt.Printf("%s\n", hex.EncodeToString(resp.Data))
		}
	}
}
