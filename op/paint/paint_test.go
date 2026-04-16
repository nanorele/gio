package paint

import (
	"image"
	"image/color"
	"testing"

	"github.com/nanorele/gio/f32"
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
