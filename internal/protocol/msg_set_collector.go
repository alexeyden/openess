package protocol

type SetCollectorReq struct {
	Par   byte
	Value string
}

type SetCollectorRsp struct {
	Status byte
	Par    byte
}

func NewSetCollectorReq(par byte, val string) Request[SetCollectorRsp, SetCollectorReq] {
    header := Header {
        TID: 1,
        DevCode: 1,
        DevAddr: 0xff,
        FuncCode: 3,
    }

    body := SetCollectorReq { Par: par, Value: val }

    req := Request[SetCollectorRsp, SetCollectorReq] {
        Header: header,
        Body: body,
    }

    return req
}

func (req SetCollectorReq) EncodeRequest() ([]byte, error) {
	data := []byte{req.Par}
	return append(data, ([]byte)(req.Value)...), nil
}

func (req SetCollectorReq) DecodeResponse(data []byte) (SetCollectorRsp, error) {
	rsp := SetCollectorRsp{Status: data[0], Par: data[1]}

	return rsp, nil
}
