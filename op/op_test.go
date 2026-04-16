package op

import (
	"image"
	"testing"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/internal/ops"
)

func TestTransformChecks(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Error("cross-macro Pop didn't panic")
		}
	}()
	var ops Ops
	trans := Offset(image.Point{}).Push(&ops)
	Record(&ops)
	trans.Pop()
}

func TestIncompleteMacroReader(t *testing.T) {
	var o Ops

	Record(&o)
	Offset(image.Point{}).Push(&o)

	var r ops.Reader

	r.Reset(&o.Internal)
	if _, more := r.Decode(); more {
		t.Error("decoded an operation from a semantically empty Ops")
	}
}

func TestOpsReset(t *testing.T) {
	var o Ops
	Offset(image.Point{10, 10}).Add(&o)
	o.Reset()
	var r ops.Reader
	r.Reset(&o.Internal)
	if _, more := r.Decode(); more {
		t.Error("Ops not empty after Reset")
	}
}

func TestDefer(t *testing.T) {
	var o Ops
	m := Record(&o)
	Offset(image.Point{10, 10}).Add(&o)
	c := m.Stop()
	Defer(&o, c)
	Defer(&o, CallOp{}) // Should return early
}

func TestCallOpAdd(t *testing.T) {
	var o Ops
	m := Record(&o)
	Offset(image.Point{10, 10}).Add(&o)
	c := m.Stop()
	c.Add(&o)
	CallOp{}.Add(&o) // Should return early
}

func TestTransform(t *testing.T) {
	var o Ops
	Affine(f32.AffineId()).Add(&o)
	stack := Affine(f32.AffineId()).Push(&o)
	stack.Pop()
}

func TestInvalidateCmd(t *testing.T) {
	var cmd InvalidateCmd
	cmd.ImplementsCommand()
}
