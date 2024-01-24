package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"openess/internal/log"
	"time"
)

type HeartBeatReq struct {
	Timestamp time.Time
	Interval  uint16
}

type HeartBeatRsp struct {
	Pn string
}

func NewHeartBeatReq() Request[HeartBeatRsp, HeartBeatReq] {
	header := Header{
		TID:      0xbeef,
		DevCode:  1,
		DevAddr:  0xff,
		FuncCode: 1,
	}

	body := HeartBeatReq{
		Timestamp: time.Now().UTC(),
		Interval:  300,
	}

	req := Request[HeartBeatRsp, HeartBeatReq]{
		Header:  header,
		Body:    body,
		Timeout: 500 * time.Millisecond,
	}

	return req
}

func (req HeartBeatReq) EncodeRequest() ([]byte, error) {
	year := (byte)((req.Timestamp.Year() - 2000) % 256)
	month := (byte)(req.Timestamp.Month())
	day := (byte)(req.Timestamp.Day())
	hour := (byte)(req.Timestamp.Hour())
	minute := (byte)(req.Timestamp.Minute())
	sec := (byte)(req.Timestamp.Second())

	log.PrDebug("msg_heartbeat: %02d.%02d.%02d %d:%d:%d\n", year, month, day, hour, minute, sec)

	buf := new(bytes.Buffer)

	buf.WriteByte(year)
	buf.WriteByte(month)
	buf.WriteByte(day)
	buf.WriteByte(hour)
	buf.WriteByte(minute)
	buf.WriteByte(sec)
	binary.Write(buf, binary.BigEndian, req.Interval)

	return buf.Bytes(), nil
}

func (req HeartBeatReq) DecodeResponse(data []byte) (HeartBeatRsp, error) {
	s := hex.EncodeToString(data)
	rsp := HeartBeatRsp{Pn: s}

	return rsp, nil
}
