package commands

import (
	"openess/internal/protocol"
)

type PingCommand struct{}

type PingResult struct {
	Pn string
}

func NewPing() PingCommand {
	return PingCommand{}
}

func (PingCommand) CastResult(resp Result) PingResult {
	return resp.(PingResult)
}

func (PingCommand) Handle(conn protocol.Device, descr *protocol.Descriptor) (Result, error) {
	req := protocol.NewHeartBeatReq()

	err := protocol.WriteRequest(conn, req)
	if err != nil {
		return nil, err
	}

	var res PingResult

	resp, err := protocol.ReadResponse(conn, req)
	if err != nil {
		return nil, err
	}

	res.Pn = resp.Body.Pn

	return res, nil
}
