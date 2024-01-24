package commands

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"openess/internal/protocol"
)

type RegWriteDescrCommand struct {
	Segment  *protocol.Segment
	Register *protocol.Register
	Value    float32
}

type RegWriteDescrResult struct {
	Data []byte
}

func NewRegWriteDescr(seg *protocol.Segment, reg *protocol.Register, value float32) RegWriteDescrCommand {
	return RegWriteDescrCommand{Segment: seg, Register: reg, Value: value}
}

func (RegWriteDescrCommand) CastResult(resp Result) RegWriteDescrResult {
	return resp.(RegWriteDescrResult)
}

func (r RegWriteDescrCommand) Handle(dev protocol.Device, descr *protocol.Descriptor) (Result, error) {
	var result RegWriteDescrResult

	if descr == nil {
		return nil, errors.New("descriptor is not loaded")
	}

	if r.Register == nil || r.Segment == nil {
		return nil, errors.New("invalid arguments: reg or segment is null")
	}

	if r.Register.ValueType != 1 {
		return nil, errors.New(fmt.Sprintf("unsupported value type %d", r.Register.ValueType))
	}

	var order binary.ByteOrder = binary.BigEndian
	if r.Register.ByteSort == protocol.ByteSortLittleEndian {
		order = binary.LittleEndian
	}

	devAddr := byte(descr.Configuration.DevAddrs[0])
	funcNumber := descr.Configuration.WriteOneFunCode
	addr := r.Register.Address
	length := 1
	if r.Register.Length != nil {
		length = *r.Register.Length
	}

	if length != 1 {
		return nil, errors.New(fmt.Sprintf("unsupported register length: %d\n", length))
	}

	buf := new(bytes.Buffer)

	rawValue := 0

	if (float64(r.Register.Scale)-1.0) < 0.0001 || r.Register.EnumerationStrings != nil {
		rawValue = int(r.Value)
	} else {
		rawValue = int(r.Value / r.Register.Scale)
	}

	binary.Write(buf, order, uint16(rawValue))

	raw_req := NewRegWriteRaw(devAddr, funcNumber, addr, buf.Bytes())

	res, err := raw_req.Handle(dev, descr)
	if err != nil {
		return nil, err
	}

	result.Data = raw_req.CastResult(res).Data

	return result, nil
}
