package commands

import (
	"openess/internal/protocol"
)

type RegReadRawCommand struct {
	DevAddr    byte
	FuncNumber byte
	Start      uint16
	Length     uint16
}

type RegReadRawResult struct {
	Data []byte
}

func NewRegReadRaw(addr byte, fun byte, start uint16, length uint16) RegReadRawCommand {
	return RegReadRawCommand{DevAddr: addr, FuncNumber: fun, Start: start, Length: length}
}

func (RegReadRawCommand) CastResult(resp Result) RegReadRawResult {
	return resp.(RegReadRawResult)
}

func (r RegReadRawCommand) Handle(conn protocol.Device, descr *protocol.Descriptor) (Result, error) {
	req := protocol.NewReadForwardReq(r.DevAddr, r.FuncNumber, r.Start, r.Length)

	err := protocol.WriteRequest(conn, req)
	if err != nil {
		return nil, err
	}

	var res RegReadRawResult

	resp, err := protocol.ReadResponse(conn, req)
	if err != nil {
		return nil, err
	}

	res.Data = resp.Body.Data

	return res, nil
}
