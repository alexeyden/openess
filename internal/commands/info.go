package commands

import (
	"errors"
	"fmt"
	"openess/internal/log"
	"openess/internal/protocol"
)

type DeviceInfoCommand struct{}

type DeviceInfoResult struct {
	DeviceType       string
	SerialNumber     string
	FirmwareVersion  string
	HardwareVersion  string
	FactoryTime      string
	DevicesOnline    string
	MonitoredDevices string
	ConnectionStatus string
	Manufacturer     string
	ProtocolVersion  string
	DeviceProps      string
	SerialBaudrate   string
	SSID             string
}

func NewDeviceInfo() DeviceInfoCommand {
	return DeviceInfoCommand{}
}

func (DeviceInfoCommand) CastResult(resp Result) DeviceInfoResult {
	return resp.(DeviceInfoResult)
}

func (DeviceInfoCommand) Handle(conn protocol.Device, descr *protocol.Descriptor) (Result, error) {
	req := protocol.NewQueryCollectorReq([]byte{
		1, 2, 5, 6, 7, 11, 12, 48, 3, 4, 14, 34, 41,
	})

	err := protocol.WriteRequest(conn, req)
	if err != nil {
		return nil, err
	}

	var res DeviceInfoResult

	for i := 0; i < len(req.Body.Pars); i++ {
		resp, err := protocol.ReadResponse(conn, req)
		if err != nil {
			herr := errors.New(fmt.Sprintf("failed to read param %d", i))
			return nil, errors.Join(herr, err)
		}

		log.PrDebug("commands:info: got response: par = %d data = %s\n", resp.Body.Par, resp.Body.Data)

		switch resp.Body.Par {
		case 1:
			res.DeviceType = resp.Body.Data
		case 2:
			res.SerialNumber = resp.Body.Data
		case 5:
			res.FirmwareVersion = resp.Body.Data
		case 6:
			res.HardwareVersion = resp.Body.Data
		case 7:
			res.FactoryTime = resp.Body.Data
		case 11:
			res.DevicesOnline = resp.Body.Data
		case 12:
			res.MonitoredDevices = resp.Body.Data
		case 48:
			res.ConnectionStatus = resp.Body.Data
		case 3:
			res.Manufacturer = resp.Body.Data
		case 4:
			res.ProtocolVersion = resp.Body.Data
		case 14:
			res.DeviceProps = resp.Body.Data
		case 34:
			res.SerialBaudrate = resp.Body.Data
		case 41:
			res.SSID = resp.Body.Data
		default:
			return nil, errors.New(fmt.Sprintf("unknown par %d = %s", resp.Body.Par, resp.Body.Data))
		}
	}

	return res, nil
}
