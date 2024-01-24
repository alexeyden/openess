package commands

import (
	"errors"
	"fmt"
	"openess/internal/protocol"
)

const (
	DEVICE_PARAM_SSID     byte = 41
	DEVICE_PARAM_PASSWORD      = 43
	DEVICE_PARAM_RESTART       = 29
)

type DeviceParamCommand struct {
	Par   byte
	Value string
}

type DeviceParamResult struct {
	Status byte
}

func NewDeviceParam(par byte, value string) DeviceParamCommand {
	return DeviceParamCommand{
		Par:   par,
		Value: value,
	}
}

func (DeviceParamCommand) CastResult(resp Result) DeviceParamResult {
	return resp.(DeviceParamResult)
}

func (cmd DeviceParamCommand) Handle(conn protocol.Device, descr *protocol.Descriptor) (Result, error) {
	req := protocol.NewSetCollectorReq(cmd.Par, cmd.Value)
	err := protocol.WriteRequest(conn, req)
	if err != nil {
		return nil, err
	}

	var res DeviceParamResult
	resp, err := protocol.ReadResponse(conn, req)
	if err != nil {
		herr := errors.New(fmt.Sprintf("failed to set param %d", cmd.Par))
		return nil, errors.Join(herr, err)
	}

	res.Status = resp.Body.Status

	return res, nil
}
