package commands

import (
	"bytes"
	"openess/internal/protocol"
	"testing"
)

func TestBitFieldExtraction(t *testing.T) {
	// Simulate APPShows register value: 0x00E7
	// Binary: 11100111
	// Bits 0-1 (Battery):  11 = 3 (Battery OK)
	// Bits 2-3 (PV):       01 = 1 (PV discharging)
	// Bits 4-5 (Mains):    10 = 2 (Mains supply discharging)
	// Bits 6-7 (Load):     11 = 3
	input := []byte{0xE7, 0x00} // LE: low byte first

	batteryOK := "Battery Ok"
	pvDischarging := "PV discharging"
	mainsDischarging := "Mains supply discharging"

	batteryEnum := protocol.Enumeration{
		Variants: map[protocol.EnumVariant]*string{
			0: strPtr("Battery disconnected"),
			1: strPtr("Battery discharging"),
			2: strPtr("Battery charging"),
			3: &batteryOK,
		},
	}
	pvEnum := protocol.Enumeration{
		Variants: map[protocol.EnumVariant]*string{
			0: strPtr("PV undervoltage"),
			1: &pvDischarging,
			2: strPtr("PV overvoltage"),
			3: strPtr("PV OK"),
		},
	}
	mainsEnum := protocol.Enumeration{
		Variants: map[protocol.EnumVariant]*string{
			0: strPtr("Mains supply abnormal"),
			1: strPtr("Mains supply absorption"),
			2: &mainsDischarging,
			3: strPtr("Mains supply OK"),
		},
	}

	descr := protocol.Descriptor{
		Configuration: protocol.Configuration{},
	}

	tests := []struct {
		name     string
		shift    uint
		mask     uint32
		enum     *protocol.Enumeration
		expected string
	}{
		{"Battery", 0, 0x3, &batteryEnum, batteryOK},
		{"PV", 2, 0x3, &pvEnum, pvDischarging},
		{"Mains", 4, 0x3, &mainsEnum, mainsDischarging},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := protocol.Register{
				ByteSort:           protocol.ByteSortLittleEndian,
				EnumerationStrings: tt.enum,
				ValueType:          1,
				Scale:              1,
				BitShift:           tt.shift,
				BitMask:            tt.mask,
			}

			buf := bytes.NewBuffer(input)
			val := NewRegValueFromBytes(buf, &reg, &descr)

			if val.ValueEnum == nil {
				t.Fatalf("expected enum value, got nil")
			}
			if *val.ValueEnum != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, *val.ValueEnum)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func TestEndianess(t* testing.T) {
    var expected uint32
    var input = []byte { 0xde, 0xad, 0xbe, 0xef }

    length := 2
    reg := protocol.Register {
        ByteSort: protocol.ByteSortBigEndian,
        Length: &length,
    }

    descr := protocol.Descriptor {
        Root: []protocol.Register { reg },
        Configuration: protocol.Configuration{},
    }

    buf := bytes.NewBuffer(input)
    val := NewRegValueFromBytes(buf, &descr.Root[0], &descr)
    expected = 0xbeefdead
    if val.ValueRaw != expected {
        t.Fatalf("0x%x != 0x%x", expected, val.ValueRaw)
    }

    buf = bytes.NewBuffer(input)
    descr.Root[0].ByteSort = protocol.ByteSortLittleEndian
    val = NewRegValueFromBytes(buf, &descr.Root[0], &descr)
    expected = 0xefbeadde
    if val.ValueRaw != expected {
        t.Fatalf("0x%x != 0x%x", expected, val.ValueRaw)
    }
}
