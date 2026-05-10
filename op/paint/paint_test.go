package paint

import (
	"encoding/binary"
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/internal/ops"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/op/clip"
)

func TestImageOp(t *testing.T) {
	var ops op.Ops

	// image.Uniform
	img1 := image.NewUniform(color.NRGBA{R: 255, A: 255})
	io1 := NewImageOp(img1)
	io1.Add(&ops)
	if io1.Size() != (image.Point{}) {
		t.Errorf("expected zero size for uniform image, got %v", io1.Size())
	}

	// *image.RGBA
	img2 := image.NewRGBA(image.Rect(0, 0, 10, 10))
	io2 := NewImageOp(img2)
	io2.Add(&ops)
	if io2.Size() != image.Pt(10, 10) {
		t.Errorf("expected 10x10 size, got %v", io2.Size())
	}

	// other image type (e.g. *image.Gray)
	img3 := image.NewGray(image.Rect(0, 0, 5, 5))
	io3 := NewImageOp(img3)
	io3.Add(&ops)
	if io3.Size() != image.Pt(5, 5) {
		t.Errorf("expected 5x5 size, got %v", io3.Size())
	}

	// empty ImageOp
	var io4 ImageOp
	io4.Add(&ops)
	if io4.Size() != (image.Point{}) {
		t.Errorf("expected zero size for empty image, got %v", io4.Size())
	}
}

func TestColorOp(t *testing.T) {
	var ops op.Ops
	co := ColorOp{Color: color.NRGBA{R: 255, G: 128, B: 64, A: 32}}
	co.Add(&ops)
}

func TestLinearGradientOp(t *testing.T) {
	var ops op.Ops
	lg := LinearGradientOp{
		Stop1:  f32.Pt(0, 0),
		Color1: color.NRGBA{R: 255, A: 255},
		Stop2:  f32.Pt(100, 100),
		Color2: color.NRGBA{B: 255, A: 255},
	}
	lg.Add(&ops)
}

func TestPaintOp(t *testing.T) {
	var ops op.Ops
	PaintOp{}.Add(&ops)
}

func TestFill(t *testing.T) {
	var ops op.Ops
	Fill(&ops, color.NRGBA{R: 255, A: 255})
}

func TestFillShape(t *testing.T) {
	var ops op.Ops
	FillShape(&ops, color.NRGBA{R: 255, A: 255}, clip.Rect(image.Rect(0, 0, 10, 10)).Op())
}

func TestPushOpacity(t *testing.T) {
	var ops op.Ops
	stack := PushOpacity(&ops, 0.5)
	stack.Pop()

	// Test clamping
	PushOpacity(&ops, 1.5).Pop()
	PushOpacity(&ops, -0.5).Pop()
}

func countOps(o *op.Ops) map[ops.OpType]int {
	counts := make(map[ops.OpType]int)
	var r ops.Reader
	r.Reset(&o.Internal)
	for {
		enc, ok := r.Decode()
		if !ok {
			break
		}
		counts[ops.OpType(enc.Data[0])]++
	}
	return counts
}

func findFirst(o *op.Ops, t ops.OpType) []byte {
	var r ops.Reader
	r.Reset(&o.Internal)
	for {
		enc, ok := r.Decode()
		if !ok {
			return nil
		}
		if ops.OpType(enc.Data[0]) == t {
			out := make([]byte, len(enc.Data))
			copy(out, enc.Data)
			return out
		}
	}
}

func TestColorOpEncoding(t *testing.T) {
	var o op.Ops
	c := color.NRGBA{R: 10, G: 20, B: 30, A: 40}
	ColorOp{Color: c}.Add(&o)
	data := findFirst(&o, ops.TypeColor)
	if data == nil {
		t.Fatal("no color op emitted")
	}
	if data[1] != 10 || data[2] != 20 || data[3] != 30 || data[4] != 40 {
		t.Errorf("color bytes = %v %v %v %v", data[1], data[2], data[3], data[4])
	}
}

func TestColorOpZero(t *testing.T) {
	var o op.Ops
	ColorOp{}.Add(&o)
	data := findFirst(&o, ops.TypeColor)
	if data == nil {
		t.Fatal("zero color op not emitted")
	}
	if data[1] != 0 || data[2] != 0 || data[3] != 0 || data[4] != 0 {
		t.Errorf("zero color bytes nonzero: %v", data[1:5])
	}
}

func TestColorOpAlphaZero(t *testing.T) {
	var o op.Ops
	ColorOp{Color: color.NRGBA{R: 200, A: 0}}.Add(&o)
	data := findFirst(&o, ops.TypeColor)
	if data == nil {
		t.Fatal("color op missing")
	}
	if data[4] != 0 {
		t.Errorf("alpha = %d, want 0", data[4])
	}
}

func TestImageOpUniformAddsColor(t *testing.T) {
	var o op.Ops
	src := image.NewUniform(color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	io := NewImageOp(src)
	if !io.uniform {
		t.Fatal("uniform image not detected")
	}
	io.Add(&o)
	counts := countOps(&o)
	if counts[ops.TypeColor] != 1 {
		t.Errorf("uniform image should add color op, got counts %v", counts)
	}
	if counts[ops.TypeImage] != 0 {
		t.Errorf("uniform image must not add image op, got %d", counts[ops.TypeImage])
	}
}

func TestImageOpRGBASize(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 7))
	io := NewImageOp(src)
	if io.uniform {
		t.Errorf("RGBA must not be marked uniform")
	}
	if io.src != src {
		t.Errorf("RGBA src not stored directly")
	}
	if io.handle == nil {
		t.Errorf("RGBA handle missing")
	}
	if io.Size() != image.Pt(4, 7) {
		t.Errorf("Size = %v", io.Size())
	}
}

func TestImageOpConvertsNonRGBA(t *testing.T) {
	src := image.NewGray(image.Rect(0, 0, 3, 5))
	io := NewImageOp(src)
	if io.src == nil {
		t.Fatal("converted src is nil")
	}
	if io.Size() != image.Pt(3, 5) {
		t.Errorf("Size = %v, want (3,5)", io.Size())
	}
}

func TestImageOpEmptyAdd(t *testing.T) {
	var o op.Ops
	src := image.NewRGBA(image.Rect(0, 0, 0, 0))
	io := NewImageOp(src)
	io.Add(&o)
	counts := countOps(&o)
	if counts[ops.TypeImage] != 0 {
		t.Errorf("empty image must not emit image op, got %d", counts[ops.TypeImage])
	}
}

func TestImageOpAddEncodesFilter(t *testing.T) {
	var o op.Ops
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	io := NewImageOp(src)
	io.Filter = FilterNearest
	io.Add(&o)
	data := findFirst(&o, ops.TypeImage)
	if data == nil {
		t.Fatal("image op not emitted")
	}
	if data[1] != byte(FilterNearest) {
		t.Errorf("filter byte = %d, want %d", data[1], FilterNearest)
	}
}

func TestImageOpDefaultFilter(t *testing.T) {
	var o op.Ops
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	io := NewImageOp(src)
	io.Add(&o)
	data := findFirst(&o, ops.TypeImage)
	if data == nil {
		t.Fatal("image op not emitted")
	}
	if data[1] != byte(FilterLinear) {
		t.Errorf("default filter = %d, want FilterLinear (0)", data[1])
	}
}

func TestZeroImageOpSize(t *testing.T) {
	var io ImageOp
	if io.Size() != (image.Point{}) {
		t.Errorf("zero ImageOp size = %v", io.Size())
	}
}

func TestLinearGradientEncoding(t *testing.T) {
	var o op.Ops
	lg := LinearGradientOp{
		Stop1:  f32.Pt(1.5, 2.5),
		Color1: color.NRGBA{R: 1, G: 2, B: 3, A: 4},
		Stop2:  f32.Pt(-3.25, 7.0),
		Color2: color.NRGBA{R: 5, G: 6, B: 7, A: 8},
	}
	lg.Add(&o)
	data := findFirst(&o, ops.TypeLinearGradient)
	if data == nil {
		t.Fatal("linear gradient op missing")
	}
	bo := binary.LittleEndian
	if math.Float32frombits(bo.Uint32(data[1:])) != 1.5 {
		t.Errorf("stop1.X mismatch")
	}
	if math.Float32frombits(bo.Uint32(data[5:])) != 2.5 {
		t.Errorf("stop1.Y mismatch")
	}
	if math.Float32frombits(bo.Uint32(data[9:])) != -3.25 {
		t.Errorf("stop2.X mismatch")
	}
	if math.Float32frombits(bo.Uint32(data[13:])) != 7.0 {
		t.Errorf("stop2.Y mismatch")
	}
	if data[17] != 1 || data[18] != 2 || data[19] != 3 || data[20] != 4 {
		t.Errorf("color1 bytes wrong: %v", data[17:21])
	}
	if data[21] != 5 || data[22] != 6 || data[23] != 7 || data[24] != 8 {
		t.Errorf("color2 bytes wrong: %v", data[21:25])
	}
}

func TestLinearGradientSameStops(t *testing.T) {
	var o op.Ops
	lg := LinearGradientOp{
		Stop1:  f32.Pt(5, 5),
		Color1: color.NRGBA{R: 255, A: 255},
		Stop2:  f32.Pt(5, 5),
		Color2: color.NRGBA{B: 255, A: 255},
	}
	// Should not panic even with degenerate stops.
	lg.Add(&o)
	if findFirst(&o, ops.TypeLinearGradient) == nil {
		t.Fatal("linear gradient op missing")
	}
}

func TestPaintOpAddsTag(t *testing.T) {
	var o op.Ops
	PaintOp{}.Add(&o)
	data := findFirst(&o, ops.TypePaint)
	if data == nil {
		t.Fatal("paint op missing")
	}
	if data[0] != byte(ops.TypePaint) {
		t.Errorf("paint tag = %d, want %d", data[0], ops.TypePaint)
	}
}

func TestFillEmitsColorAndPaint(t *testing.T) {
	var o op.Ops
	Fill(&o, color.NRGBA{R: 10, A: 255})
	counts := countOps(&o)
	if counts[ops.TypeColor] != 1 {
		t.Errorf("Fill should emit 1 color op, got %d", counts[ops.TypeColor])
	}
	if counts[ops.TypePaint] != 1 {
		t.Errorf("Fill should emit 1 paint op, got %d", counts[ops.TypePaint])
	}
}

func TestFillShapeEndToEnd(t *testing.T) {
	var o op.Ops
	FillShape(&o, color.NRGBA{R: 1, G: 2, B: 3, A: 4}, clip.Rect(image.Rect(0, 0, 10, 10)).Op())
	counts := countOps(&o)
	if counts[ops.TypeClip] != 1 {
		t.Errorf("FillShape should push 1 clip, got %d", counts[ops.TypeClip])
	}
	if counts[ops.TypePopClip] != 1 {
		t.Errorf("FillShape should pop 1 clip, got %d", counts[ops.TypePopClip])
	}
	if counts[ops.TypeColor] != 1 || counts[ops.TypePaint] != 1 {
		t.Errorf("FillShape missing color or paint op: %v", counts)
	}
}

func TestFillShapeZeroAreaShape(t *testing.T) {
	var o op.Ops
	// Empty rectangle shape should still encode push/pop balanced.
	FillShape(&o, color.NRGBA{A: 255}, clip.Rect(image.Rect(5, 5, 5, 5)).Op())
	counts := countOps(&o)
	if counts[ops.TypeClip] != counts[ops.TypePopClip] {
		t.Errorf("clip push/pop unbalanced: %d vs %d", counts[ops.TypeClip], counts[ops.TypePopClip])
	}
}

func TestFillShapeAlphaZero(t *testing.T) {
	var o op.Ops
	FillShape(&o, color.NRGBA{R: 255, A: 0}, clip.Rect(image.Rect(0, 0, 10, 10)).Op())
	data := findFirst(&o, ops.TypeColor)
	if data == nil {
		t.Fatal("color op missing")
	}
	if data[4] != 0 {
		t.Errorf("alpha = %d, want 0", data[4])
	}
}

func TestPushOpacityEncodes(t *testing.T) {
	var o op.Ops
	st := PushOpacity(&o, 0.25)
	st.Pop()
	data := findFirst(&o, ops.TypePushOpacity)
	if data == nil {
		t.Fatal("push opacity missing")
	}
	got := math.Float32frombits(binary.LittleEndian.Uint32(data[1:]))
	if got != 0.25 {
		t.Errorf("opacity = %v, want 0.25", got)
	}
	counts := countOps(&o)
	if counts[ops.TypePopOpacity] != 1 {
		t.Errorf("missing pop opacity, got %d", counts[ops.TypePopOpacity])
	}
}

func TestPushOpacityClampHigh(t *testing.T) {
	var o op.Ops
	PushOpacity(&o, 5).Pop()
	data := findFirst(&o, ops.TypePushOpacity)
	got := math.Float32frombits(binary.LittleEndian.Uint32(data[1:]))
	if got != 1 {
		t.Errorf("opacity clamp high = %v, want 1", got)
	}
}

func TestPushOpacityClampLow(t *testing.T) {
	var o op.Ops
	PushOpacity(&o, -5).Pop()
	data := findFirst(&o, ops.TypePushOpacity)
	got := math.Float32frombits(binary.LittleEndian.Uint32(data[1:]))
	if got != 0 {
		t.Errorf("opacity clamp low = %v, want 0", got)
	}
}

func TestPushOpacityCrossMacroPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("popping opacity across macro should panic")
		}
	}()
	var o op.Ops
	st := PushOpacity(&o, 0.5)
	op.Record(&o)
	st.Pop()
}

func TestImageOpReuseOnSameRGBA(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	a := NewImageOp(src)
	b := NewImageOp(src)
	if a.src != b.src {
		t.Errorf("RGBA src should be reused, got different pointers")
	}
}

func TestImageOpHandleStableAcrossAdds(t *testing.T) {
	var o op.Ops
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	io := NewImageOp(src)
	h1 := io.handle
	io.Add(&o)
	io.Add(&o)
	if io.handle != h1 {
		t.Errorf("handle should not change on Add")
	}
}

func TestFillShapeMultipleStacked(t *testing.T) {
	var o op.Ops
	FillShape(&o, color.NRGBA{R: 1, A: 255}, clip.Rect(image.Rect(0, 0, 10, 10)).Op())
	FillShape(&o, color.NRGBA{G: 1, A: 255}, clip.Rect(image.Rect(0, 0, 5, 5)).Op())
	counts := countOps(&o)
	if counts[ops.TypeClip] != 2 || counts[ops.TypePopClip] != 2 {
		t.Errorf("expected 2 clip pushes and pops, got %v", counts)
	}
	if counts[ops.TypeColor] != 2 || counts[ops.TypePaint] != 2 {
		t.Errorf("expected 2 color and 2 paint, got %v", counts)
	}
}
