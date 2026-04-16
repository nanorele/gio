package event

import (
	"github.com/nanorele/gio/internal/ops"
	"github.com/nanorele/gio/op"
)

type Tag any

type Event interface {
	ImplementsEvent()
}

type Filter interface {
	ImplementsFilter()
}

func Op(o *op.Ops, tag Tag) {
	if tag == nil {
		panic("Tag must be non-nil")
	}
	data := ops.Write1(&o.Internal, ops.TypeInputLen, tag)
	data[0] = byte(ops.TypeInput)
}
