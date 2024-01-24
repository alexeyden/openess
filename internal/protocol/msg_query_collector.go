package protocol

type QueryCollectorReq struct {
	Pars []byte
}

type QueryCollectorRsp struct {
	Code byte
	Par  byte
	Data string
}

func NewQueryCollectorReq(pars []byte) Request[QueryCollectorRsp, QueryCollectorReq] {
    header := Header {
        TID: 1,
        DevCode: 1,
        DevAddr: 0xff,
        FuncCode: 2,
    }

    body := QueryCollectorReq { Pars: pars }

    req := Request[QueryCollectorRsp, QueryCollectorReq] {
        Header: header,
        Body: body,
    }

    return req
}

func (req QueryCollectorReq) EncodeRequest() ([]byte, error) {
	return req.Pars, nil
}

func (req QueryCollectorReq) DecodeResponse(data []byte) (QueryCollectorRsp, error) {
	code := data[0]
	par := data[1]
	dat := string(data[2:])

	rsp := QueryCollectorRsp{Code: code, Par: par, Data: dat}

	return rsp, nil
}
