package commands

import (
	"openess/internal/protocol"
)

type Result interface{}

type ResultCast[T any] interface {
	CastResult(response Result) T
	Handle(conn protocol.Device, descr *protocol.Descriptor) (Result, error)
}

type Command interface {
	Handle(conn protocol.Device, descr *protocol.Descriptor) (Result, error)
}
