package commands

import (
	"openess/internal/protocol"
)

type RegWriteRawCommand struct {
	DevAddr    byte
	FuncNumber byte
	Addr       uint16
	Data       []byte
}

type RegWriteRawResult struct {
	Data []byte
}

func NewRegWriteRaw(devaddr byte, fun byte, addr uint16, data []byte) RegWriteRawCommand {
	return RegWriteRawCommand{DevAddr: devaddr, FuncNumber: fun, Addr: addr, Data: data}
}

func (RegWriteRawCommand) CastResult(resp Result) RegWriteRawResult {
	return resp.(RegWriteRawResult)
}

func (r RegWriteRawCommand) Handle(conn protocol.Device, descr *protocol.Descriptor) (Result, error) {
	req := protocol.NewWriteForwardReq(r.DevAddr, r.FuncNumber, r.Addr, r.Data)

	err := protocol.WriteRequest(conn, req)
	if err != nil {
		return nil, err
	}

	var res RegWriteRawResult

	resp, err := protocol.ReadResponse(conn, req)
	if err != nil {
		return nil, err
	}

	res.Data = resp.Body.Data

	return res, nil
}
