package op

import (
	"image"
	"testing"
	"time"

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

func TestOpsSizeStartsZero(t *testing.T) {
	var o Ops
	if got := o.Size(); got != 0 {
		t.Errorf("fresh Ops.Size() = %d, want 0", got)
	}
}

func TestOpsSizeGrowsOnWrite(t *testing.T) {
	var o Ops
	Offset(image.Point{X: 1, Y: 2}).Add(&o)
	if o.Size() != ops.TypeTransformLen {
		t.Errorf("after one Add: Size()=%d, want %d", o.Size(), ops.TypeTransformLen)
	}
	Offset(image.Point{X: 3, Y: 4}).Add(&o)
	if o.Size() != 2*ops.TypeTransformLen {
		t.Errorf("after two Adds: Size()=%d, want %d", o.Size(), 2*ops.TypeTransformLen)
	}
}

func TestOpsResetClearsSize(t *testing.T) {
	var o Ops
	Offset(image.Point{X: 1, Y: 2}).Add(&o)
	stack := Affine(f32.AffineId()).Push(&o)
	stack.Pop()
	if o.Size() == 0 {
		t.Fatal("Size() unexpectedly 0 before Reset")
	}
	o.Reset()
	if o.Size() != 0 {
		t.Errorf("after Reset: Size()=%d, want 0", o.Size())
	}
	// Reader must see no ops.
	var r ops.Reader
	r.Reset(&o.Internal)
	if _, more := r.Decode(); more {
		t.Error("ops residue after Reset")
	}
}

func TestOpsResetReusable(t *testing.T) {
	var o Ops
	for i := 0; i < 3; i++ {
		Offset(image.Point{X: i, Y: i}).Add(&o)
		o.Reset()
		if o.Size() != 0 {
			t.Fatalf("iter %d: Size after Reset = %d", i, o.Size())
		}
	}
	// After Reset we should still be able to record macros.
	m := Record(&o)
	Offset(image.Point{X: 5, Y: 5}).Add(&o)
	c := m.Stop()
	if c.ops == nil {
		t.Error("CallOp.ops nil after Reset/Record cycle")
	}
}

func TestRecordStopAddByteCounts(t *testing.T) {
	var o Ops
	before := o.Size()
	m := Record(&o)
	// Record reserves space for a Macro header up front.
	if grew := o.Size() - before; grew != ops.TypeMacroLen {
		t.Errorf("Record grew Ops by %d, want %d", grew, ops.TypeMacroLen)
	}
	Offset(image.Point{X: 1, Y: 1}).Add(&o)
	c := m.Stop() // Stop does not write any bytes itself.
	expected := ops.TypeMacroLen + ops.TypeTransformLen
	if o.Size() != expected {
		t.Errorf("after Stop: Size=%d want %d", o.Size(), expected)
	}
	// Adding a CallOp writes a Call op into the stream.
	beforeAdd := o.Size()
	c.Add(&o)
	if grew := o.Size() - beforeAdd; grew != ops.TypeCallLen {
		t.Errorf("CallOp.Add grew Ops by %d, want %d", grew, ops.TypeCallLen)
	}
}

func TestMacroOpReAddDoesNotDoubleEncodeBody(t *testing.T) {
	// Re-Adding a MacroOp should only emit additional Call ops, never
	// re-emit the macro body.
	var o Ops
	m := Record(&o)
	Offset(image.Point{X: 1, Y: 1}).Add(&o)
	Offset(image.Point{X: 2, Y: 2}).Add(&o)
	c := m.Stop()
	beforeFirst := o.Size()
	c.Add(&o)
	firstGrowth := o.Size() - beforeFirst
	beforeSecond := o.Size()
	c.Add(&o)
	secondGrowth := o.Size() - beforeSecond
	if firstGrowth != secondGrowth {
		t.Errorf("Re-Add growth differs: first=%d second=%d", firstGrowth, secondGrowth)
	}
	if firstGrowth != ops.TypeCallLen {
		t.Errorf("CallOp.Add growth=%d, want exactly TypeCallLen=%d", firstGrowth, ops.TypeCallLen)
	}
}

func TestDeferReturnsEarlyOnZeroCallOp(t *testing.T) {
	var o Ops
	before := o.Size()
	Defer(&o, CallOp{})
	if o.Size() != before {
		t.Errorf("Defer with empty CallOp wrote %d bytes", o.Size()-before)
	}
}

func TestDeferAppendsBytes(t *testing.T) {
	var o Ops
	m := Record(&o)
	Offset(image.Point{X: 1, Y: 1}).Add(&o)
	c := m.Stop()
	before := o.Size()
	Defer(&o, c)
	if o.Size() <= before {
		t.Errorf("Defer did not grow Ops: before=%d after=%d", before, o.Size())
	}
}

func TestTransformPushPopBalances(t *testing.T) {
	var o Ops
	stack := Affine(f32.AffineId()).Push(&o)
	beforePop := o.Size()
	stack.Pop()
	grew := o.Size() - beforePop
	if grew != ops.TypePopTransformLen {
		t.Errorf("Pop grew Ops by %d, want %d", grew, ops.TypePopTransformLen)
	}
}

func TestTransformPushNestedPopsOrderEnforced(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("popping outer before inner did not panic")
		}
	}()
	var o Ops
	outer := Affine(f32.AffineId()).Push(&o)
	_ = Affine(f32.AffineId()).Push(&o)
	outer.Pop() // unbalanced: must panic
}

func TestExtraPopPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("extra Pop did not panic")
		}
	}()
	var o Ops
	stack := Affine(f32.AffineId()).Push(&o)
	stack.Pop()
	stack.Pop() // already popped — must panic
}

func TestTransformOpAddSize(t *testing.T) {
	var o Ops
	Affine(f32.AffineId()).Add(&o)
	if o.Size() != ops.TypeTransformLen {
		t.Errorf("Affine().Add size=%d want %d", o.Size(), ops.TypeTransformLen)
	}
}

func TestInvalidateCmdValue(t *testing.T) {
	now := time.Now()
	cmd := InvalidateCmd{At: now}
	if !cmd.At.Equal(now) {
		t.Error("InvalidateCmd.At not preserved")
	}
}

func TestOffsetEqualsAffineOffset(t *testing.T) {
	var o1, o2 Ops
	Offset(image.Point{X: 5, Y: -7}).Add(&o1)
	Affine(f32.AffineId().Offset(f32.Pt(5, -7))).Add(&o2)
	if o1.Size() != o2.Size() {
		t.Fatalf("size mismatch: %d vs %d", o1.Size(), o2.Size())
	}
	// Decode each and compare resulting affines.
	var r1, r2 ops.Reader
	r1.Reset(&o1.Internal)
	r2.Reset(&o2.Internal)
	enc1, more1 := r1.Decode()
	enc2, more2 := r2.Decode()
	if !more1 || !more2 {
		t.Fatal("expected one op in each Ops")
	}
	t1, _ := ops.DecodeTransform(enc1.Data)
	t2, _ := ops.DecodeTransform(enc2.Data)
	a1 := t1.Transform(f32.Pt(0, 0))
	a2 := t2.Transform(f32.Pt(0, 0))
	if a1 != a2 {
		t.Errorf("Offset vs Affine.Offset differ: %v vs %v", a1, a2)
	}
}
