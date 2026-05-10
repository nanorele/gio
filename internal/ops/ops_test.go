package ops

import (
	"encoding/binary"
	"image"
	"math"
	"testing"

	"github.com/nanorele/gio/internal/scene"
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
	_ = Write2(&o, 5, "ref2", "ref3")
	if len(o.refs) != 3 || o.refs[1] != "ref2" || o.refs[2] != "ref3" {
		t.Errorf("unexpected refs: %v", o.refs)
	}

	// Test Write3
	_ = Write3(&o, 5, "ref4", "ref5", "ref6")
	if len(o.refs) != 6 || o.refs[3] != "ref4" || o.refs[4] != "ref5" || o.refs[5] != "ref6" {
		t.Errorf("unexpected refs: %v", o.refs)
	}

	// Test Write1String
	_ = Write1String(&o, 5, "string1")
	if len(o.stringRefs) != 1 || o.stringRefs[0] != "string1" {
		t.Errorf("unexpected stringRefs: %v", o.stringRefs)
	}

	// Test Write2String
	_ = Write2String(&o, 5, "ref7", "string2")
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

func TestDataLen(t *testing.T) {
	var o Ops
	if got := DataLen(&o); got != 0 {
		t.Errorf("DataLen on fresh Ops = %d, want 0", got)
	}
	Write(&o, 7)
	if got := DataLen(&o); got != 7 {
		t.Errorf("DataLen after Write(7) = %d, want 7", got)
	}
	Write(&o, 3)
	if got := DataLen(&o); got != 10 {
		t.Errorf("DataLen after second write = %d, want 10", got)
	}
	Reset(&o)
	if got := DataLen(&o); got != 0 {
		t.Errorf("DataLen after Reset = %d, want 0", got)
	}
}

func TestResetClearsRefs(t *testing.T) {
	var o Ops
	r1 := &Ops{}
	Write1(&o, 4, r1)
	Write1String(&o, 4, "hello")
	if len(o.refs) != 2 {
		t.Fatalf("refs len = %d", len(o.refs))
	}
	if len(o.stringRefs) != 1 {
		t.Fatalf("stringRefs len = %d", len(o.stringRefs))
	}
	Reset(&o)
	if len(o.refs) != 0 || len(o.stringRefs) != 0 || len(o.data) != 0 {
		t.Errorf("Reset didn't clear lengths: refs=%d stringRefs=%d data=%d",
			len(o.refs), len(o.stringRefs), len(o.data))
	}
	if o.nextStateID != 0 {
		t.Errorf("nextStateID = %d, want 0", o.nextStateID)
	}
}

func TestResetIncrementsVersion(t *testing.T) {
	var o Ops
	v := o.version
	Reset(&o)
	if o.version != v+1 {
		t.Errorf("version = %d, want %d", o.version, v+1)
	}
	Reset(&o)
	if o.version != v+2 {
		t.Errorf("version = %d, want %d", o.version, v+2)
	}
}

func TestWriteCapReuse(t *testing.T) {
	var o Ops
	Write(&o, 100)
	cap1 := cap(o.data)
	Reset(&o)
	if cap(o.data) != cap1 {
		t.Errorf("Reset shouldn't shrink cap: was %d, now %d", cap1, cap(o.data))
	}
	d := Write(&o, 50)
	if cap(o.data) != cap1 {
		t.Errorf("Write within cap shouldn't realloc")
	}
	for i := range d {
		if d[i] != 0 {
			t.Errorf("Write didn't zero out byte %d: %v", i, d[i])
		}
	}
}

func TestWriteZeroes(t *testing.T) {
	var o Ops
	d := Write(&o, 5)
	for i := range d {
		d[i] = 0xff
	}
	Reset(&o)
	d = Write(&o, 5)
	for i := range d {
		if d[i] != 0 {
			t.Errorf("Write didn't zero reused slot %d: %v", i, d[i])
		}
	}
}

func TestWrite1ZeroesAndAppendsRef(t *testing.T) {
	var o Ops
	d := Write1(&o, 3, "ref")
	for i := range d {
		if d[i] != 0 {
			t.Errorf("Write1 didn't zero byte %d", i)
		}
	}
	if len(o.refs) != 1 || o.refs[0] != "ref" {
		t.Errorf("refs not appended correctly: %v", o.refs)
	}
}

func TestWriteMultiZeroes(t *testing.T) {
	var o Ops
	BeginMulti(&o)
	d := WriteMulti(&o, 4)
	for i := range d {
		if d[i] != 0 {
			t.Errorf("WriteMulti didn't zero byte %d", i)
		}
	}
	EndMulti(&o)
}

func TestWriteMultiCapReuse(t *testing.T) {
	var o Ops
	Write(&o, 200)
	Reset(&o)
	BeginMulti(&o)
	d := WriteMulti(&o, 50)
	for i := range d {
		d[i] = 0xff
	}
	EndMulti(&o)
	Reset(&o)
	BeginMulti(&o)
	d = WriteMulti(&o, 50)
	for i := range d {
		if d[i] != 0 {
			t.Errorf("WriteMulti didn't zero reused byte %d", i)
		}
	}
	EndMulti(&o)
}

func TestPCFor(t *testing.T) {
	var o Ops
	pc := PCFor(&o)
	if pc.data != 0 || pc.refs != 0 {
		t.Errorf("PCFor empty Ops = %+v", pc)
	}
	Write(&o, 7)
	Write1(&o, 3, "a")
	pc = PCFor(&o)
	if pc.data != 10 || pc.refs != 1 {
		t.Errorf("PCFor = %+v, want {10, 1}", pc)
	}
}

func TestPCAdd(t *testing.T) {
	pc := PC{data: 5, refs: 2}
	pc2 := pc.Add(TypeCall) // size 17, refs 1
	if pc2.data != 5+TypeCallLen || pc2.refs != 3 {
		t.Errorf("PC.Add(TypeCall) = %+v", pc2)
	}
	pc3 := PC{}.Add(TypeImage) // size 2, refs 2
	if pc3.data != TypeImageLen || pc3.refs != 2 {
		t.Errorf("PC.Add(TypeImage) = %+v", pc3)
	}
}

func TestStackPushPop(t *testing.T) {
	var s stack
	a := s.push()
	if a.id != 1 || a.prev != 0 || s.currentID != 1 {
		t.Errorf("first push = %+v, current=%d", a, s.currentID)
	}
	b := s.push()
	if b.id != 2 || b.prev != 1 || s.currentID != 2 {
		t.Errorf("second push = %+v, current=%d", b, s.currentID)
	}
	s.pop(b)
	if s.currentID != 1 {
		t.Errorf("after pop b, currentID=%d", s.currentID)
	}
	s.pop(a)
	if s.currentID != 0 {
		t.Errorf("after pop a, currentID=%d", s.currentID)
	}
}

func TestStackUnbalancedPanic(t *testing.T) {
	var s stack
	a := s.push()
	b := s.push()
	// pop in wrong order: pop a while b is current
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for unbalanced pop")
		}
	}()
	s.pop(a)
	_ = b
}

func TestStackNextIDMonotonic(t *testing.T) {
	var s stack
	a := s.push()
	s.pop(a)
	b := s.push()
	if b.id != 2 {
		t.Errorf("nextID should be monotonic, got id=%d", b.id)
	}
	if b.prev != 0 {
		t.Errorf("prev after pop should be 0, got %d", b.prev)
	}
}

func TestPushPopOpAcrossKinds(t *testing.T) {
	var o Ops
	sid1, mid1 := PushOp(&o, ClipStack)
	sid2, mid2 := PushOp(&o, TransStack)
	if mid1 != 0 || mid2 != 0 {
		t.Errorf("macroID should be 0, got %d, %d", mid1, mid2)
	}
	PopOp(&o, TransStack, sid2, mid2)
	PopOp(&o, ClipStack, sid1, mid1)
}

func TestPushOpStacksAreSeparate(t *testing.T) {
	var o Ops
	c1, _ := PushOp(&o, ClipStack)
	t1, _ := PushOp(&o, TransStack)
	if c1.id != 1 || t1.id != 1 {
		t.Errorf("each stack kind should have its own counter: clip=%d trans=%d", c1.id, t1.id)
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	var o Ops
	s1 := Save(&o)
	if s1.id != 1 || s1.ops != &o {
		t.Errorf("Save id=%d ops=%v", s1.id, s1.ops)
	}
	s2 := Save(&o)
	if s2.id != 2 {
		t.Errorf("second Save id=%d", s2.id)
	}
	// Verify Save serialized id correctly
	bo := binary.LittleEndian
	off := 0
	if OpType(o.data[off]) != TypeSave {
		t.Errorf("first byte = %d, want TypeSave", o.data[off])
	}
	if got := bo.Uint32(o.data[off+1:]); got != 1 {
		t.Errorf("Save id in data = %d, want 1", got)
	}
	off += TypeSaveLen
	if got := bo.Uint32(o.data[off+1:]); got != 2 {
		t.Errorf("second Save id in data = %d, want 2", got)
	}
	s1.Load()
	off += TypeSaveLen
	if OpType(o.data[off]) != TypeLoad {
		t.Errorf("Load type byte = %d, want TypeLoad", o.data[off])
	}
	if got := bo.Uint32(o.data[off+1:]); got != 1 {
		t.Errorf("Load serialized id = %d, want 1", got)
	}
}

func TestDecodeSaveValue(t *testing.T) {
	var o Ops
	s := Save(&o)
	if s.id != 1 {
		t.Fatalf("unexpected id")
	}
	got := DecodeSave(o.data)
	if got != 1 {
		t.Errorf("DecodeSave = %d, want 1", got)
	}
}

func TestDecodeLoadValue(t *testing.T) {
	var o Ops
	s := Save(&o)
	s.Load()
	got := DecodeLoad(o.data[TypeSaveLen:])
	if got != 1 {
		t.Errorf("DecodeLoad = %d, want 1", got)
	}
}

func TestDecodeOpacityValue(t *testing.T) {
	data := make([]byte, TypePushOpacityLen)
	data[0] = byte(TypePushOpacity)
	bo := binary.LittleEndian
	bo.PutUint32(data[1:], math.Float32bits(0.75))
	if got := DecodeOpacity(data); got != 0.75 {
		t.Errorf("DecodeOpacity = %v, want 0.75", got)
	}
}

func TestDecodeTransformValues(t *testing.T) {
	data := make([]byte, TypeTransformLen)
	data[0] = byte(TypeTransform)
	data[1] = 1 // push
	bo := binary.LittleEndian
	want := []float32{1, 2, 3, 4, 5, 6}
	for i, v := range want {
		bo.PutUint32(data[2+4*i:], math.Float32bits(v))
	}
	aff, push := DecodeTransform(data)
	if !push {
		t.Errorf("push = false, want true")
	}
	a, b, c, d, e, f := aff.Elems()
	got := []float32{a, b, c, d, e, f}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("affine[%d] = %v, want %v", i, got[i], want[i])
		}
	}
	// push=false path
	data[1] = 0
	_, push = DecodeTransform(data)
	if push {
		t.Errorf("push = true, want false")
	}
}

func TestDecodeClipValues(t *testing.T) {
	data := make([]byte, TypeClipLen)
	data[0] = byte(TypeClip)
	bo := binary.LittleEndian
	x1, y1 := int32(-5), int32(-10)
	x2, y2 := int32(20), int32(30)
	bo.PutUint32(data[1:], uint32(x1))
	bo.PutUint32(data[5:], uint32(y1))
	bo.PutUint32(data[9:], uint32(x2))
	bo.PutUint32(data[13:], uint32(y2))
	data[17] = 1 // outline
	data[18] = byte(Ellipse)

	var op ClipOp
	op.Decode(data)
	want := image.Rect(-5, -10, 20, 30)
	if op.Bounds != want {
		t.Errorf("Bounds = %v, want %v", op.Bounds, want)
	}
	if !op.Outline {
		t.Errorf("Outline = false, want true")
	}
	if op.Shape != Ellipse {
		t.Errorf("Shape = %d, want %d", op.Shape, Ellipse)
	}

	data[17] = 0
	op.Decode(data)
	if op.Outline {
		t.Errorf("Outline = true, want false")
	}
}

func TestOpTypeString(t *testing.T) {
	cases := map[OpType]string{
		TypeMacro:            "Macro",
		TypeCall:             "Call",
		TypeDefer:            "Defer",
		TypeTransform:        "Transform",
		TypePopTransform:     "PopTransform",
		TypePushOpacity:      "PushOpacity",
		TypePopOpacity:       "PopOpacity",
		TypeImage:            "Image",
		TypePaint:            "Paint",
		TypeColor:            "Color",
		TypeLinearGradient:   "LinearGradient",
		TypePass:             "Pass",
		TypePopPass:          "PopPass",
		TypeInput:            "Input",
		TypeKeyInputHint:     "KeyInputHint",
		TypeSave:             "Save",
		TypeLoad:             "Load",
		TypeAux:              "Aux",
		TypeClip:             "Clip",
		TypePopClip:          "PopClip",
		TypeCursor:           "Cursor",
		TypePath:             "Path",
		TypeStroke:           "Stroke",
		TypeSemanticLabel:    "SemanticLabel",
		TypeSemanticDesc:     "SemanticDescription",
		TypeSemanticClass:    "SemanticClass",
		TypeSemanticSelected: "SemanticSelected",
		TypeSemanticEnabled:  "SemanticEnabled",
		TypeActionInput:      "ActionInput",
	}
	for ty, want := range cases {
		got := ty.String()
		if got != want {
			t.Errorf("OpType(%d).String() = %q, want %q", ty, got, want)
		}
	}
}

func TestOpTypeSizeAndNumRefs(t *testing.T) {
	cases := []struct {
		t       OpType
		size    uint32
		numRefs uint32
	}{
		{TypeMacro, TypeMacroLen, 0},
		{TypeCall, TypeCallLen, 1},
		{TypeImage, TypeImageLen, 2},
		{TypeInput, TypeInputLen, 1},
		{TypeKeyInputHint, TypeKeyInputHintLen, 1},
		{TypeSemanticLabel, TypeSemanticLabelLen, 1},
		{TypeSemanticDesc, TypeSemanticDescLen, 1},
		{TypeColor, TypeColorLen, 0},
	}
	for _, c := range cases {
		if got := c.t.Size(); got != c.size {
			t.Errorf("%v.Size() = %d, want %d", c.t, got, c.size)
		}
		if got := c.t.NumRefs(); got != c.numRefs {
			t.Errorf("%v.NumRefs() = %d, want %d", c.t, got, c.numRefs)
		}
	}
}

func TestCommandRoundTrip(t *testing.T) {
	var src scene.Command
	for i := range src {
		src[i] = uint32(0xdeadbe00 + i)
	}
	out := make([]byte, 36)
	EncodeCommand(out, src)
	got := DecodeCommand(out)
	if got != src {
		t.Errorf("roundtrip mismatch: got %v, want %v", got, src)
	}
}

func TestEmptyReader(t *testing.T) {
	var r Reader
	op, ok := r.Decode()
	if ok {
		t.Errorf("empty reader returned ok=true, op=%v", op)
	}
}

func TestEmptyOpsReader(t *testing.T) {
	var o Ops
	var r Reader
	r.Reset(&o)
	op, ok := r.Decode()
	if ok {
		t.Errorf("empty ops reader returned ok=true, op=%v", op)
	}
}

func TestReaderSingleOp(t *testing.T) {
	var o Ops
	d := Write(&o, TypeColorLen)
	d[0] = byte(TypeColor)
	var r Reader
	r.Reset(&o)
	op, ok := r.Decode()
	if !ok {
		t.Fatalf("expected one op")
	}
	if OpType(op.Data[0]) != TypeColor {
		t.Errorf("got type %d, want Color", op.Data[0])
	}
	if len(op.Data) != TypeColorLen {
		t.Errorf("data len = %d, want %d", len(op.Data), TypeColorLen)
	}
	_, ok = r.Decode()
	if ok {
		t.Errorf("expected end of ops")
	}
}

func TestReaderResetAt(t *testing.T) {
	var o Ops
	d := Write(&o, TypeColorLen)
	d[0] = byte(TypeColor)
	skipPC := PCFor(&o)
	d = Write(&o, TypePaintLen)
	d[0] = byte(TypePaint)

	var r Reader
	r.ResetAt(&o, skipPC)
	op, ok := r.Decode()
	if !ok || OpType(op.Data[0]) != TypePaint {
		t.Errorf("expected Paint op, got ok=%v", ok)
	}
}

func TestReaderKeyVersion(t *testing.T) {
	var o Ops
	d := Write(&o, TypeColorLen)
	d[0] = byte(TypeColor)
	v := o.version

	var r Reader
	r.Reset(&o)
	op, ok := r.Decode()
	if !ok {
		t.Fatalf("expected op")
	}
	if op.Key.version != v {
		t.Errorf("key version = %d, want %d", op.Key.version, v)
	}
	if op.Key.ops != &o {
		t.Errorf("key.ops mismatch")
	}
	if op.Key.pc != 0 {
		t.Errorf("key.pc = %d, want 0", op.Key.pc)
	}
}

func TestOpMacroDefDecodePanics(t *testing.T) {
	var op opMacroDef
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on short data")
		}
	}()
	op.decode([]byte{byte(TypeMacro)})
}

func TestOpMacroDefDecodeWrongType(t *testing.T) {
	var op opMacroDef
	data := make([]byte, TypeMacroLen)
	data[0] = byte(TypeColor)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on wrong type")
		}
	}()
	op.decode(data)
}

func TestMacroOpDecodePanics(t *testing.T) {
	var op macroOp
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on short data")
		}
	}()
	op.decode([]byte{byte(TypeCall)}, []any{&Ops{}})
}

func TestMacroOpDecodeNoRefs(t *testing.T) {
	var op macroOp
	data := make([]byte, TypeCallLen)
	data[0] = byte(TypeCall)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on missing refs")
		}
	}()
	op.decode(data, nil)
}

func TestMacroOpDecodeRoundTrip(t *testing.T) {
	var o Ops
	var inner Ops
	pc := PC{data: 4, refs: 0}
	end := PC{data: 8, refs: 0}
	AddCall(&o, &inner, pc, end)

	var op macroOp
	op.decode(o.data, o.refs)
	if op.ops != &inner {
		t.Errorf("ops ref mismatch")
	}
	if op.start != pc {
		t.Errorf("start = %+v, want %+v", op.start, pc)
	}
	if op.end != end {
		t.Errorf("end = %+v, want %+v", op.end, end)
	}
}

func TestReaderDeferOnly(t *testing.T) {
	var o Ops
	d := Write(&o, TypeDeferLen)
	d[0] = byte(TypeDefer)

	var r Reader
	r.Reset(&o)
	_, ok := r.Decode()
	if ok {
		t.Errorf("Defer alone (not followed by Call) shouldn't yield ops")
	}
}

func TestReaderMacroDefSkipsBody(t *testing.T) {
	var o Ops
	startPC := PCFor(&o)
	Write(&o, TypeMacroLen)
	d := Write(&o, TypeColorLen)
	d[0] = byte(TypeColor)
	FillMacro(&o, startPC)

	// Append one more op AFTER the macro definition; it should be reached.
	d = Write(&o, TypePaintLen)
	d[0] = byte(TypePaint)

	var r Reader
	r.Reset(&o)
	op, ok := r.Decode()
	if !ok || OpType(op.Data[0]) != TypePaint {
		t.Errorf("expected Paint after macro body skip, got ok=%v", ok)
	}
}

func TestPushOpacityEncodeDecode(t *testing.T) {
	var o Ops
	d := Write(&o, TypePushOpacityLen)
	d[0] = byte(TypePushOpacity)
	bo := binary.LittleEndian
	bo.PutUint32(d[1:], math.Float32bits(0.5))
	if got := DecodeOpacity(d); got != 0.5 {
		t.Errorf("DecodeOpacity = %v, want 0.5", got)
	}
}

// TestOpProps verifies opProps[t] is consistent with t.Size()/NumRefs().
func TestOpPropsConsistency(t *testing.T) {
	all := []OpType{
		TypeMacro, TypeCall, TypeDefer, TypeTransform, TypePopTransform,
		TypePushOpacity, TypePopOpacity, TypeImage, TypePaint, TypeColor,
		TypeLinearGradient, TypePass, TypePopPass, TypeInput, TypeKeyInputHint,
		TypeSave, TypeLoad, TypeAux, TypeClip, TypePopClip, TypeCursor,
		TypePath, TypeStroke, TypeSemanticLabel, TypeSemanticDesc,
		TypeSemanticClass, TypeSemanticSelected, TypeSemanticEnabled,
		TypeActionInput,
	}
	for _, ty := range all {
		size, refs := ty.props()
		if size != ty.Size() {
			t.Errorf("%v: props.Size %d != Size() %d", ty, size, ty.Size())
		}
		if refs != ty.NumRefs() {
			t.Errorf("%v: props.NumRefs %d != NumRefs() %d", ty, refs, ty.NumRefs())
		}
		if size == 0 {
			t.Errorf("%v has Size 0", ty)
		}
	}
}

func TestUnknownOpTypeProps(t *testing.T) {
	// Unmapped op type should yield zero props (default value).
	var ty OpType = 0
	size, refs := ty.props()
	if size != 0 || refs != 0 {
		t.Errorf("unknown OpType props = (%d, %d), want (0, 0)", size, refs)
	}
}

func TestWrite2StringPointersToString(t *testing.T) {
	var o Ops
	Write2String(&o, 4, "ref1", "str1")
	if len(o.refs) != 2 {
		t.Fatalf("refs len = %d, want 2", len(o.refs))
	}
	if o.refs[0] != "ref1" {
		t.Errorf("refs[0] = %v, want ref1", o.refs[0])
	}
	sp, ok := o.refs[1].(*string)
	if !ok {
		t.Fatalf("refs[1] is not *string: %T", o.refs[1])
	}
	if *sp != "str1" {
		t.Errorf("*refs[1] = %q, want str1", *sp)
	}
}

func TestStringRefsIndependent(t *testing.T) {
	var o Ops
	Write1String(&o, 1, "first")
	Write1String(&o, 1, "second")
	if len(o.stringRefs) != 2 {
		t.Fatalf("got %d stringRefs", len(o.stringRefs))
	}
	if o.stringRefs[0] != "first" || o.stringRefs[1] != "second" {
		t.Errorf("stringRefs = %v", o.stringRefs)
	}
}

func TestBeginMultiAfterEnd(t *testing.T) {
	var o Ops
	BeginMulti(&o)
	EndMulti(&o)
	// Should be allowed again.
	BeginMulti(&o)
	EndMulti(&o)
}

func TestPopMacroPanicWrongID(t *testing.T) {
	var o Ops
	sid := PushMacro(&o)
	bad := StackID{id: sid.id + 99, prev: sid.prev}
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on wrong macro id")
		}
	}()
	PopMacro(&o, bad)
}

func TestFillMacroWritesPC(t *testing.T) {
	var o Ops
	startPC := PCFor(&o)
	Write(&o, TypeMacroLen)
	Write(&o, TypeColorLen)[0] = byte(TypeColor)
	endPC := PCFor(&o)
	FillMacro(&o, startPC)
	if OpType(o.data[0]) != TypeMacro {
		t.Errorf("FillMacro didn't set type byte")
	}
	bo := binary.LittleEndian
	if got := bo.Uint32(o.data[1:]); got != endPC.data {
		t.Errorf("FillMacro data PC = %d, want %d", got, endPC.data)
	}
	if got := bo.Uint32(o.data[5:]); got != endPC.refs {
		t.Errorf("FillMacro refs PC = %d, want %d", got, endPC.refs)
	}
}
