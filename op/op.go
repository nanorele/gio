package op

import (
	"encoding/binary"
	"image"
	"math"
	"time"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/internal/ops"
)

type Ops struct {
	Internal ops.Ops
}

type MacroOp struct {
	ops *ops.Ops
	id  ops.StackID
	pc  ops.PC
}

type CallOp struct {
	ops   *ops.Ops
	start ops.PC
	end   ops.PC
}

type InvalidateCmd struct {
	At time.Time
}

type TransformOp struct {
	t f32.Affine2D
}

type TransformStack struct {
	id      ops.StackID
	macroID uint32
	ops     *ops.Ops
}

func Defer(o *Ops, c CallOp) {
	if c.ops == nil {
		return
	}
	state := ops.Save(&o.Internal)

	m := Record(o)
	state.Load()
	c.Add(o)
	c = m.Stop()

	data := ops.Write(&o.Internal, ops.TypeDeferLen)
	data[0] = byte(ops.TypeDefer)
	c.Add(o)
}

func (o *Ops) Reset() {
	ops.Reset(&o.Internal)
}

// Size returns the byte length of the op buffer's encoded operations.
// Intended for monitoring growth of long-lived buffers (e.g. a
// shaper's persistent path buffer) without exposing internal state.
func (o *Ops) Size() int {
	return ops.DataLen(&o.Internal)
}

func Record(o *Ops) MacroOp {
	m := MacroOp{
		ops: &o.Internal,
		id:  ops.PushMacro(&o.Internal),
		pc:  ops.PCFor(&o.Internal),
	}

	data := ops.Write(m.ops, ops.TypeMacroLen)
	data[0] = byte(ops.TypeMacro)
	return m
}

func (m MacroOp) Stop() CallOp {
	ops.PopMacro(m.ops, m.id)
	ops.FillMacro(m.ops, m.pc)
	return CallOp{
		ops: m.ops,

		start: m.pc.Add(ops.TypeMacro),
		end:   ops.PCFor(m.ops),
	}
}

func (c CallOp) Add(o *Ops) {
	if c.ops == nil {
		return
	}
	ops.AddCall(&o.Internal, c.ops, c.start, c.end)
}

func Offset(off image.Point) TransformOp {
	offf := f32.Pt(float32(off.X), float32(off.Y))
	return Affine(f32.AffineId().Offset(offf))
}

func Affine(a f32.Affine2D) TransformOp {
	return TransformOp{t: a}
}

func (t TransformOp) Push(o *Ops) TransformStack {
	id, macroID := ops.PushOp(&o.Internal, ops.TransStack)
	t.add(o, true)
	return TransformStack{ops: &o.Internal, id: id, macroID: macroID}
}

func (t TransformOp) Add(o *Ops) {
	t.add(o, false)
}

func (t TransformOp) add(o *Ops, push bool) {
	data := ops.Write(&o.Internal, ops.TypeTransformLen)
	data[0] = byte(ops.TypeTransform)
	if push {
		data[1] = 1
	}
	bo := binary.LittleEndian
	a, b, c, d, e, f := t.t.Elems()
	bo.PutUint32(data[2:], math.Float32bits(a))
	bo.PutUint32(data[2+4*1:], math.Float32bits(b))
	bo.PutUint32(data[2+4*2:], math.Float32bits(c))
	bo.PutUint32(data[2+4*3:], math.Float32bits(d))
	bo.PutUint32(data[2+4*4:], math.Float32bits(e))
	bo.PutUint32(data[2+4*5:], math.Float32bits(f))
}

func (t TransformStack) Pop() {
	ops.PopOp(t.ops, ops.TransStack, t.id, t.macroID)
	data := ops.Write(t.ops, ops.TypePopTransformLen)
	data[0] = byte(ops.TypePopTransform)
}

func (InvalidateCmd) ImplementsCommand() {}
