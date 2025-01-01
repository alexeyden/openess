package commands

import (
	"bytes"
	"openess/internal/protocol"
	"testing"
)

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
