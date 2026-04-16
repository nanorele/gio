package event

import (
	"testing"

	"github.com/nanorele/gio/op"
)

func TestOp(t *testing.T) {
	var ops op.Ops
	tag := new(int)
	Op(&ops, tag)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Op(nil) should panic")
		}
	}()
	Op(&ops, nil)
}
