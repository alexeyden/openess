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

type Register struct {
	Address            uint16
	ByteSort           int
	Length             *int
	Title              map[string]string
	EnumerationStrings *Enumeration
	ValueType          int
	Units              string
	Scale              float32
}

type Descriptor struct {
	Root          []Register
	Configuration Configuration
	OtherCodes    map[string]ExternEnum
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

	return &result, nil
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

	var seg *Segment

	for _, g := range desc.Configuration.SystemInfoVC {
		for _, s := range g.Segments {
			if s.StartAddress >= reg.Address && (s.StartAddress+s.Length) >= reg.Address {
				seg = &s
				goto exit
			}
		}
	}

	for _, g := range desc.Configuration.SystemSettingVC {
		for _, s := range g.Segments {
			if s.StartAddress >= reg.Address && (s.StartAddress+s.Length) >= reg.Address {
				seg = &s
				goto exit
			}
		}
	}

exit:
	return seg, reg
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
