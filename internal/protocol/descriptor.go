package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Segment struct {
	CanEdit      bool
	Length       uint16
	FunNumber    byte `json:",string"`
	StartAddress uint16
}

type ConfigurationGroup struct {
	Title    map[string]string
	Segments []Segment
}

type AddressOffset struct {
	OffsetType    int
	OffsetAddress int
	OffsetBase    int
}

type ExternEnum struct {
	Variants map[int]string
}

func (this *ExternEnum) UnmarshalJSON(data []byte) error {
	type baseVariants struct {
		Base map[string]string
	}

	variants := make(map[int]string)

	if data[0] == '{' {
		var inner baseVariants
		err := json.Unmarshal(data, &inner)
		if err != nil {
			return err
		}
		for k, v := range inner.Base {
			num, err := strconv.Atoi(k)
			if err != nil {
				return fmt.Errorf("failed to parse enum key as int: %v", err)
			}

			variants[num] = v
		}
	}

	this.Variants = variants

	return nil
}

type DevAddr byte

func (addr *DevAddr) UnmarshalJSON(data []byte) error {
	s := string(data)
	val, err := strconv.Atoi(strings.Trim(s, "\""))
	if err != nil {
		return err
	}
	*addr = DevAddr(val)
	return nil
}

type EnumVariant int

func (v *EnumVariant) UnmarshalJSON(data []byte) error {
	s := string(data)
	val, err := strconv.Atoi(strings.Trim(s, "\""))
	if err != nil {
		return err
	}
	*v = EnumVariant(val)
	return nil
}

type Configuration struct {
	DevAddrs         []DevAddr
	SystemSettingVC  []ConfigurationGroup
	SystemInfoVC     []ConfigurationGroup
	WriteMoreFunCode byte
	WriteOneFunCode  byte
	AddressOffset    AddressOffset
}

const (
	ByteSortLittleEndian = 1
	ByteSortBigEndian    = 0
)

type Enumeration struct {
	Variants map[EnumVariant]*string
	External *string
}

func (v *Enumeration) UnmarshalJSON(data []byte) error {
	type baseVariants struct {
		Base map[EnumVariant]*string
	}

	var variants baseVariants

	err := json.Unmarshal(data, &variants)
	if err == nil {
		v.Variants = variants.Base
		return nil
	}

	var defaultVariant string
	err = json.Unmarshal(data, &defaultVariant)
	if err != nil {
		return err
	}

	v.External = &defaultVariant

	return nil
}

type BitField struct {
	Address            int
	Length             int
	Title              map[string]string
	EnumerationStrings *Enumeration
}

type Register struct {
	Address            uint16
	ByteSort           int
	Length             *int
	Title              map[string]string
	EnumerationStrings *Enumeration
	SubModels          *string
	ValueType          int
	Units              string
	Scale              float32
	// Set programmatically for synthesized sub-field registers
	BitShift uint   `json:"-"`
	BitMask  uint32 `json:"-"`
}

type Descriptor struct {
	Root          []Register
	Configuration Configuration
	OtherCodes    map[string]ExternEnum
	SubModelDefs  map[string][]BitField `json:"-"`
}

func LoadProtocolDescriptor(path string) (*Descriptor, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var result Descriptor

	err = json.Unmarshal([]byte(data), &result)
	if err != nil {
		return nil, err
	}

	if result.Configuration.AddressOffset.OffsetType != 0 {
		return nil, errors.New("unsupported addressing mode (offsetType != 0)")
	}

	result.SubModelDefs = parseSubModelDefs(data)

	return &result, nil
}

func parseSubModelDefs(data []byte) map[string][]BitField {
	var raw struct {
		OtherCodes map[string]json.RawMessage
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	result := make(map[string][]BitField)

	for name, rawVal := range raw.OtherCodes {
		if len(rawVal) == 0 || rawVal[0] != '[' {
			continue
		}

		var fields []BitField
		if err := json.Unmarshal(rawVal, &fields); err != nil {
			continue
		}

		if len(fields) > 0 {
			result[name] = fields
		}
	}

	return result
}

func (desc Descriptor) FindRegisterByAddr(addr uint16) *Register {
	var reg *Register

	for _, r := range desc.Root {
		if r.Address == addr {
			return &r
		}
	}

	return reg
}

func (desc Descriptor) FindRegister(name string) (*Segment, *Register) {
	if dotIdx := strings.Index(name, "."); dotIdx >= 0 {
		parentName := name[:dotIdx]
		childName := name[dotIdx+1:]
		return desc.findSubRegister(parentName, childName)
	}

	var reg *Register

	for _, r := range desc.Root {
		regName, ok := r.Title["base"]

		if ok && regName == name {
			reg = &r
			break
		}
	}

	if reg == nil {
		return nil, nil
	}

	seg := desc.findSegment(reg.Address)
	return seg, reg
}

func (desc Descriptor) findSegment(addr uint16) *Segment {
	for _, g := range desc.Configuration.SystemInfoVC {
		for _, s := range g.Segments {
			if s.StartAddress >= addr && (s.StartAddress+s.Length) >= addr {
				return &s
			}
		}
	}

	for _, g := range desc.Configuration.SystemSettingVC {
		for _, s := range g.Segments {
			if s.StartAddress >= addr && (s.StartAddress+s.Length) >= addr {
				return &s
			}
		}
	}

	return nil
}

func (desc Descriptor) findSubRegister(parentName, childName string) (*Segment, *Register) {
	var parentReg *Register

	for _, r := range desc.Root {
		regName, ok := r.Title["base"]
		if ok && regName == parentName {
			parentReg = &r
			break
		}
	}

	if parentReg == nil || parentReg.SubModels == nil {
		return nil, nil
	}

	fields, ok := desc.SubModelDefs[*parentReg.SubModels]
	if !ok {
		return nil, nil
	}

	for _, field := range fields {
		fieldName, ok := field.Title["base"]
		if !ok || fieldName != childName {
			continue
		}

		bitLen := field.Length
		if bitLen == 0 {
			bitLen = 1
		}

		synth := Register{
			Address:            parentReg.Address,
			ByteSort:           parentReg.ByteSort,
			Length:             parentReg.Length,
			Title:              field.Title,
			EnumerationStrings: field.EnumerationStrings,
			ValueType:          parentReg.ValueType,
			Units:              parentReg.Units,
			Scale:              1,
			BitShift:           uint(field.Address),
			BitMask:            (1 << uint(bitLen)) - 1,
		}

		seg := desc.findSegment(parentReg.Address)
		return seg, &synth
	}

	return nil, nil
}

func (desc Descriptor) FindGroup(name string) []Segment {
	for _, g := range desc.Configuration.SystemInfoVC {
		if g.Title["base"] == name {
			return g.Segments
		}
	}

	for _, g := range desc.Configuration.SystemSettingVC {
		if g.Title["base"] == name {
			return g.Segments
		}
	}

	return []Segment{}
}
