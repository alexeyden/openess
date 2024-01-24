package protocol

import (
	"encoding/binary"
	"io"
	"time"
)

type Header struct {
	TID      uint16
	DevCode  uint16
	Size     uint16
	DevAddr  byte
	FuncCode byte
}

type RequestBody[R any] interface {
	EncodeRequest() ([]byte, error)
	DecodeResponse(body []byte) (R, error)
}

type Request[R any, T RequestBody[R]] struct {
	Header     Header
	Body       T
	Timeout    time.Duration
}

type Response[T any] struct {
	Header Header
	Body   T
}

func (header Header) Write(writer io.Writer) error {
	var err error

	body_len := header.Size + 2

	err = binary.Write(writer, binary.BigEndian, header.TID)
	if err != nil {
		return err
	}

	err = binary.Write(writer, binary.BigEndian, header.DevCode)
	if err != nil {
		return err
	}

	err = binary.Write(writer, binary.BigEndian, body_len)
	if err != nil {
		return err
	}

	err = binary.Write(writer, binary.BigEndian, header.DevAddr)
	if err != nil {
		return err
	}

	err = binary.Write(writer, binary.BigEndian, header.FuncCode)
	if err != nil {
		return err
	}

	return nil
}

func ReadHeader(reader io.Reader) (Header, error) {
	var err error

	header := Header{
		TID:      0,
		DevCode:  0,
		Size:     0,
		DevAddr:  0,
		FuncCode: 0,
	}

	err = binary.Read(reader, binary.BigEndian, &header.TID)
	if err != nil {
		return header, err
	}

	err = binary.Read(reader, binary.BigEndian, &header.DevCode)
	if err != nil {
		return header, err
	}

	err = binary.Read(reader, binary.BigEndian, &header.Size)
	if err != nil {
		return header, err
	}

	err = binary.Read(reader, binary.BigEndian, &header.DevAddr)
	if err != nil {
		return header, err
	}

	err = binary.Read(reader, binary.BigEndian, &header.FuncCode)
	if err != nil {
		return header, err
	}

	header.Size -= 2

	return header, nil
}
