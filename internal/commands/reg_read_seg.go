package commands

import (
	"bytes"
	"errors"
	"openess/internal/log"
	"openess/internal/protocol"
)

type RegReadSegCommand struct {
	Segment *protocol.Segment
}

type RegReadSegResult struct {
	Values map[uint16]RegValue
}

func NewRegReadSeg(seg *protocol.Segment) RegReadSegCommand {
	return RegReadSegCommand{Segment: seg}
}

func (RegReadSegCommand) CastResult(resp Result) RegReadSegResult {
	return resp.(RegReadSegResult)
}

func (r RegReadSegCommand) Handle(dev protocol.Device, descr *protocol.Descriptor) (Result, error) {
	if descr == nil {
		return nil, errors.New("descriptor is not loaded")
	}

	if r.Segment == nil {
		return nil, errors.New("invalid arguments: reg or segment is null")
	}

	if r.Segment.CanEdit {
		return r.HandleSparse(dev, descr)
	} else {
		return r.HandleContinuous(dev, descr)
	}
}

func (r RegReadSegCommand) HandleContinuous(dev protocol.Device, descr *protocol.Descriptor) (Result, error) {
	var result RegReadSegResult
	result.Values = make(map[uint16]RegValue)

	devAddr := byte(descr.Configuration.DevAddrs[0])
	funcNumber := r.Segment.FunNumber
	addr := r.Segment.StartAddress
	length := r.Segment.Length

	raw_req := NewRegReadRaw(devAddr, funcNumber, addr, length)

	res, err := raw_req.Handle(dev, descr)
	if err != nil {
		return nil, err
	}

	raw_result := raw_req.CastResult(res)
	buf := bytes.NewBuffer(raw_result.Data)

	for {
		reg := descr.FindRegisterByAddr(addr)
		if reg == nil {
			log.PrError("commands:read_segment: failed to find register (invalid descrptor): %d", addr)
			continue
		}

		val := NewRegValueFromBytes(buf, reg, descr)
		result.Values[addr] = val

		if reg.Length == nil {
			addr += 1
		} else {
			addr += uint16(*reg.Length)
		}

		if buf.Len() == 0 {
			break
		}
	}

	return result, nil
}

func (r RegReadSegCommand) HandleSparse(dev protocol.Device, descr *protocol.Descriptor) (Result, error) {
	var result RegReadSegResult
	result.Values = make(map[uint16]RegValue)

	devAddr := byte(descr.Configuration.DevAddrs[0])
	funcNumber := r.Segment.FunNumber
	addr := r.Segment.StartAddress

	for {
		reg := descr.FindRegisterByAddr(addr)
		if reg == nil {
			log.PrError("commands:read_segment: failed to find register (invalid descrptor): %d", addr)
			continue
		}

		length := 1
		if reg.Length != nil {
			length = *reg.Length
		}
		raw_req := NewRegReadRaw(devAddr, funcNumber, addr, uint16(length))

		res, err := raw_req.Handle(dev, descr)
		if err != nil {
			log.PrError("commands:read_segment: failed to read register %d: %s", addr, err)
			continue
		}

		raw_result := raw_req.CastResult(res)
		buf := bytes.NewBuffer(raw_result.Data)
		val := NewRegValueFromBytes(buf, reg, descr)
		result.Values[addr] = val

		addr += uint16(length)

		if addr >= r.Segment.StartAddress+r.Segment.Length {
			break
		}
	}

	return result, nil
}
