package commands

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"openess/internal/log"
	"openess/internal/protocol"
	"strconv"
)

type RegType int

const (
	RegTypeInt   RegType = 0
	RegTypeEnum  RegType = 1
	RegTypeFloat RegType = 2
)

type RegValue struct {
	Type       RegType
	ValueRaw   uint32
	ValueInt   *int
	ValueEnum  *string
	ValueFloat *float32
	Units      *string
}

func nativeIsBigEndian() bool {
	return binary.NativeEndian.Uint16([]byte{0x0a, 0x0b}) == uint16(0x0a0b)
}

func NewRegValueFromBytes(buf io.Reader, reg *protocol.Register, desc *protocol.Descriptor) RegValue {
	var order binary.ByteOrder = binary.BigEndian
	if reg.ByteSort == protocol.ByteSortLittleEndian {
		order = binary.LittleEndian
	}

	var value RegValue
	value.Units = &reg.Units

	if reg.Length == nil || *reg.Length == 1 {
		var val uint16
		binary.Read(buf, order, &val)
		value.ValueRaw = uint32(val)
	} else if *reg.Length == 2 {
		var val uint32
		var word uint16

		binary.Read(buf, order, &word)
		val = uint32(word)
		binary.Read(buf, order, &word)
		val |= (uint32(word) << 16)

		if nativeIsBigEndian() {
			val = ((val&0xff)<<16 | (val >> 16))
		}

		value.ValueRaw = val
	} else {
		log.PrError("reg_read_descr: unexpected register length: %d\n", *reg.Length)
	}

	if reg.EnumerationStrings != nil {
		value.Type = RegTypeEnum

		if reg.EnumerationStrings.External != nil {
			enum, enumOk := desc.OtherCodes[*reg.EnumerationStrings.External]

			if !enumOk {
				log.PrError("reg_read_descr: failed to find external enum: reg = %d enum = %s\n", reg.Address, *reg.EnumerationStrings.External)
				s := strconv.Itoa(int(value.ValueRaw))
				value.ValueEnum = &s
				return value
			}

			enumStr, ok := enum.Variants[int(value.ValueRaw)]

			if !ok {
				log.PrError("reg_read_descr: failed to find external enum value: reg = %d enum = %s value = %d\n", reg.Address, *reg.EnumerationStrings.External, int(value.ValueRaw))
				s := strconv.Itoa(int(value.ValueRaw))
				value.ValueEnum = &s
				return value
			}

			value.ValueEnum = &enumStr
			return value
		}

		enumStr, ok := reg.EnumerationStrings.Variants[protocol.EnumVariant(value.ValueRaw)]

		if !ok {
			log.PrError("reg_read_descr: failed to find symbolic enum value: reg = %d value = %d\n", reg.Address, value.ValueRaw)
			s := strconv.Itoa(int(value.ValueRaw))
			value.ValueEnum = &s
			return value
		}

		value.ValueEnum = enumStr
	} else if math.Abs(float64(reg.Scale)-1.0) < 0.0001 {
		value.Type = RegTypeInt
		v := int(value.ValueRaw)
		value.ValueInt = &v
	} else {
		value.Type = RegTypeFloat
		v := float32(value.ValueRaw) * reg.Scale
		value.ValueFloat = &v
	}

	return value
}

func (v RegValue) ToStringRaw() string {
	switch v.Type {
	case RegTypeInt:
		return fmt.Sprintf("%d", *v.ValueInt)
	case RegTypeEnum:
		return fmt.Sprintf("%s", *v.ValueEnum)
	case RegTypeFloat:
		return fmt.Sprintf("%.3f", *v.ValueFloat)
	}

	return ""
}

func (v RegValue) ToString() string {
	units := ""
	if v.Units != nil {
		units = *v.Units
	}
	switch v.Type {
	case RegTypeInt:
		return fmt.Sprintf("%d%s", *v.ValueInt, units)
	case RegTypeEnum:
		return fmt.Sprintf("%s", *v.ValueEnum)
	case RegTypeFloat:
		return fmt.Sprintf("%.3f%s", *v.ValueFloat, units)
	}

	return ""
}

type RegReadDescrCommand struct {
	Segment  *protocol.Segment
	Register *protocol.Register
}

type RegReadDescrResult struct {
	Value RegValue
}

func NewRegReadDescr(seg *protocol.Segment, reg *protocol.Register) RegReadDescrCommand {
	return RegReadDescrCommand{Segment: seg, Register: reg}
}

func (RegReadDescrCommand) CastResult(resp Result) RegReadDescrResult {
	return resp.(RegReadDescrResult)
}

func (r RegReadDescrCommand) Handle(dev protocol.Device, descr *protocol.Descriptor) (Result, error) {
	var result RegReadDescrResult

	if descr == nil {
		return nil, errors.New("descriptor is not loaded")
	}

	if r.Register == nil || r.Segment == nil {
		return nil, errors.New("invalid arguments: reg or segment is null")
	}

	if r.Register.ValueType != 1 {
		return nil, errors.New(fmt.Sprintf("unsupported value type %d", r.Register.ValueType))
	}

	devAddr := byte(descr.Configuration.DevAddrs[0])
	funcNumber := r.Segment.FunNumber
	addr := r.Register.Address
	length := 1
	if r.Register.Length != nil {
		length = *r.Register.Length
	}

	raw_req := NewRegReadRaw(devAddr, funcNumber, addr, uint16(length))

	res, err := raw_req.Handle(dev, descr)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(raw_req.CastResult(res).Data)

	val := NewRegValueFromBytes(buf, r.Register, descr)
	result.Value = val

	return result, nil
}
