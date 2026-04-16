package ops

import (
	"testing"
)

func TestOps(t *testing.T) {
	var o Ops
	Reset(&o)

	// Test Write
	data := Write(&o, 10)
	if len(data) != 10 {
		t.Errorf("expected 10 bytes, got %d", len(data))
	}
	if len(o.data) != 10 {
		t.Errorf("expected ops.data length 10, got %d", len(o.data))
	}

	// Test Write1
	data = Write1(&o, 5, "ref1")
	if len(data) != 5 {
		t.Errorf("expected 5 bytes, got %d", len(data))
	}
	if len(o.refs) != 1 || o.refs[0] != "ref1" {
		t.Errorf("unexpected refs: %v", o.refs)
	}

	// Test Write2
	data = Write2(&o, 5, "ref2", "ref3")
	if len(o.refs) != 3 || o.refs[1] != "ref2" || o.refs[2] != "ref3" {
		t.Errorf("unexpected refs: %v", o.refs)
	}

	// Test Write3
	data = Write3(&o, 5, "ref4", "ref5", "ref6")
	if len(o.refs) != 6 || o.refs[3] != "ref4" || o.refs[4] != "ref5" || o.refs[5] != "ref6" {
		t.Errorf("unexpected refs: %v", o.refs)
	}

	// Test Write1String
	data = Write1String(&o, 5, "string1")
	if len(o.stringRefs) != 1 || o.stringRefs[0] != "string1" {
		t.Errorf("unexpected stringRefs: %v", o.stringRefs)
	}

	// Test Write2String
	data = Write2String(&o, 5, "ref7", "string2")
	if len(o.stringRefs) != 2 || o.stringRefs[1] != "string2" {
		t.Errorf("unexpected stringRefs: %v", o.stringRefs)
	}

	// Test Multi ops
	BeginMulti(&o)
	WriteMulti(&o, 5)
	EndMulti(&o)

	// Test panics
	assertPanic(t, func() { WriteMulti(&o, 5) }, "cannot use multi ops in single ops")
	BeginMulti(&o)
	assertPanic(t, func() { Write(&o, 5) }, "cannot mix multi ops with single ones")
	assertPanic(t, func() { BeginMulti(&o) }, "cannot interleave multi ops")
	EndMulti(&o)
	assertPanic(t, func() { EndMulti(&o) }, "cannot end non multi ops")

	// Test Macros
	startPC := PCFor(&o)
	Write(&o, TypeMacroLen)
	sid := PushMacro(&o)
	FillMacro(&o, startPC)
	PopMacro(&o, sid)

	// Test Stack ops
	sid, mid := PushOp(&o, TransStack)
	PopOp(&o, TransStack, sid, mid)
	assertPanic(t, func() { PopOp(&o, TransStack, sid, mid+1) }, "stack push and pop must not cross macro boundary")

	// Test Save/Load
	state := Save(&o)
	state.Load()
}

func TestOpType(t *testing.T) {
	tests := []OpType{
		TypeMacro, TypeCall, TypeDefer, TypeTransform, TypePopTransform,
		TypePushOpacity, TypePopOpacity, TypeImage, TypePaint, TypeColor,
		TypeLinearGradient, TypePass, TypePopPass, TypeInput, TypeKeyInputHint,
		TypeSave, TypeLoad, TypeAux, TypeClip, TypePopClip, TypeCursor,
		TypePath, TypeStroke, TypeSemanticLabel,
	}

	for _, ty := range tests {
		s := ty.String()
		if s == "" {
			t.Errorf("empty string for OpType %d", ty)
		}
		size := ty.Size()
		if size == 0 {
			t.Errorf("zero size for OpType %d", ty)
		}
		_ = ty.NumRefs()
	}

	assertPanic(t, func() { _ = OpType(0).String() }, "unknown OpType")
}

func TestReader(t *testing.T) {
	var o Ops
	Reset(&o)

	// Add some ops
	data := Write(&o, TypeColorLen)
	data[0] = byte(TypeColor)
	
	// Add a call
	var o2 Ops
	Reset(&o2)
	pc2 := PCFor(&o2)
	data2 := Write(&o2, TypePaintLen)
	data2[0] = byte(TypePaint)
	data2 = Write(&o2, TypePaintLen)
	data2[0] = byte(TypePaint)
	endPC2 := PCFor(&o2)
	
	AddCall(&o, &o2, pc2, endPC2)

	// Add a defer
	data = Write(&o, TypeDeferLen)
	data[0] = byte(TypeDefer)
	AddCall(&o, &o2, pc2, endPC2)

	var r Reader
	r.Reset(&o)
	
	count := 0
	for {
		op, ok := r.Decode()
		if !ok {
			break
		}
		count++
		_ = op
	}
	// Expected ops: Color, Paint, Paint (from call), Paint, Paint (from deferred call)
	// Wait, o2 has two Paint ops.
	// So: 1 (Color) + 2 (Paint from call) + 2 (Paint from deferred call) = 5
	if count != 5 {
		t.Errorf("expected 5 ops, got %d", count)
	}
}

func TestDecoding(t *testing.T) {
	t.Run("ClipOp", func(t *testing.T) {
		var op ClipOp
		data := make([]byte, TypeClipLen)
		data[0] = byte(TypeClip)
		op.Decode(data)
		assertPanic(t, func() { op.Decode(data[:1]) }, "invalid op")
		data[0] = 0
		assertPanic(t, func() { op.Decode(data) }, "invalid op")
	})

	t.Run("Transform", func(t *testing.T) {
		data := make([]byte, TypeTransformLen)
		data[0] = byte(TypeTransform)
		DecodeTransform(data)
		data[0] = 0
		assertPanic(t, func() { DecodeTransform(data) }, "invalid op")
	})

	t.Run("Opacity", func(t *testing.T) {
		data := make([]byte, TypePushOpacityLen)
		data[0] = byte(TypePushOpacity)
		DecodeOpacity(data)
		data[0] = 0
		assertPanic(t, func() { DecodeOpacity(data) }, "invalid op")
	})

	t.Run("Save", func(t *testing.T) {
		data := make([]byte, TypeSaveLen)
		data[0] = byte(TypeSave)
		DecodeSave(data)
		data[0] = 0
		assertPanic(t, func() { DecodeSave(data) }, "invalid op")
	})

	t.Run("Load", func(t *testing.T) {
		data := make([]byte, TypeLoadLen)
		data[0] = byte(TypeLoad)
		DecodeLoad(data)
		data[0] = 0
		assertPanic(t, func() { DecodeLoad(data) }, "invalid op")
	})
}

func TestAux(t *testing.T) {
	var o Ops
	Reset(&o)
	
	// Add a call that contains an Aux op
	var o2 Ops
	Reset(&o2)
	pc2 := PCFor(&o2)
	data2 := Write(&o2, TypeAuxLen)
	data2[0] = byte(TypeAux)
	Write(&o2, 10) // Aux data
	endPC2 := PCFor(&o2)
	
	AddCall(&o, &o2, pc2, endPC2)
	
	var r Reader
	r.Reset(&o)
	op, ok := r.Decode()
	if !ok || OpType(op.Data[0]) != TypeAux {
		t.Errorf("expected Aux op")
	}
	if len(op.Data) != 1+10 {
		t.Errorf("expected Aux op size %d, got %d", 1+10, len(op.Data))
	}
}

func TestMacro(t *testing.T) {
	var o Ops
	Reset(&o)
	
	startPC := PCFor(&o)
	Write(&o, TypeMacroLen)
	Write(&o, TypeColorLen)[0] = byte(TypeColor)
	FillMacro(&o, startPC)
	
	var r Reader
	r.ResetAt(&o, startPC)
	op, ok := r.Decode()
	if ok {
		t.Errorf("expected Macro to be skipped, got %v", OpType(op.Data[0]))
	}
}

func TestCommand(t *testing.T) {
	data := make([]byte, 16)
	cmd := DecodeCommand(data)
	EncodeCommand(data, cmd)
}

func TestPC(t *testing.T) {
	pc := PC{}
	pc = pc.Add(TypeColor)
	if pc.data != TypeColorLen || pc.refs != 0 {
		t.Errorf("unexpected PC after Add: %+v", pc)
	}
}

func assertPanic(t *testing.T, f func(), msg string) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected panic: %s", msg)
		}
	}()
	f()
}
