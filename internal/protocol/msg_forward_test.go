package protocol

import (
	"testing"
)

func TestHandlingExtraBytes(t* testing.T) {
    readReq := ForwardReadReq {
        DevAddr: 1,
        FuncNumber: 3,
        Address: 1046,
        Length: 2,
    }

    readRespBytes := []byte("\x01\x03\x04\x16\x1e\x00\x00\x9e\x7d\x11\x11")

    _, err := readReq.DecodeResponse(readRespBytes)

    if err != nil {
        t.Fatalf("failed to decode read response: %s", err)
    }
}

