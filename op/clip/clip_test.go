package clip_test

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/gpu/headless"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/op/clip"
	"github.com/nanorele/gio/op/paint"
)

func TestPathOutline(t *testing.T) {
	t.Run("closed path", func(t *testing.T) {
		defer func() {
			if err := recover(); err != nil {
				t.Error("Outline of a closed path did panic")
			}
		}()
		var p clip.Path
		p.Begin(new(op.Ops))
		p.MoveTo(f32.Pt(300, 200))
		p.LineTo(f32.Pt(150, 200))
		p.MoveTo(f32.Pt(150, 200))
		p.ArcTo(f32.Pt(300, 200), f32.Pt(300, 200), 3*math.Pi/4)
		p.LineTo(f32.Pt(300, 200))
		p.Close()
		clip.Outline{Path: p.End()}.Op()
	})
}

func TestPathBegin(t *testing.T) {
	ops := new(op.Ops)
	var p clip.Path
	p.Begin(ops)
	p.LineTo(f32.Pt(10, 10))
	p.Close()
	stack := clip.Outline{Path: p.End()}.Op().Push(ops)
	paint.Fill(ops, color.NRGBA{A: 255})
	stack.Pop()
	w := newWindow(t, 100, 100)
	if w == nil {
		return
	}

	_ = w.Frame(ops)
}

func TestTransformChecks(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Error("cross-macro Pop didn't panic")
		}
	}()
	var ops op.Ops
	st := clip.Op{}.Push(&ops)
	op.Record(&ops)
	st.Pop()
}

func TestEmptyPath(t *testing.T) {
	var ops op.Ops
	p := clip.Path{}
	p.Begin(&ops)
	defer clip.Stroke{
		Path:  p.End(),
		Width: 3,
	}.Op().Push(&ops).Pop()
}

func newWindow(t testing.TB, width, height int) *headless.Window {
	w, err := headless.NewWindow(width, height)
	if err != nil {
		t.Skipf("failed to create headless window, skipping: %v", err)
	}
	return w
}

func TestPathMethods(t *testing.T) {
	ops := new(op.Ops)
	var p clip.Path
	p.Begin(ops)
	p.Move(f32.Pt(10, 10))
	p.Line(f32.Pt(20, 0))
	p.Quad(f32.Pt(10, 10), f32.Pt(20, 0))
	p.Cube(f32.Pt(5, 5), f32.Pt(15, 5), f32.Pt(20, 0))
	p.Arc(f32.Pt(10, 0), f32.Pt(0, 10), math.Pi/2)
	p.Close()
	spec := p.End()
	_ = spec
}

func TestShapes(t *testing.T) {
	ops := new(op.Ops)
	t.Run("Rect", func(t *testing.T) {
		r := clip.Rect{Max: image.Pt(100, 100)}
		r.Op().Push(ops).Pop()
		r.Push(ops).Pop()
	})
	t.Run("RRect", func(t *testing.T) {
		rr := clip.UniformRRect(image.Rect(0, 0, 100, 100), 10)
		rr.Op(ops).Push(ops).Pop()
		rr.Push(ops).Pop()

		zero := clip.RRect{Rect: image.Rect(0, 0, 100, 100)}
		zero.Op(ops).Push(ops).Pop()
	})
	t.Run("Ellipse", func(t *testing.T) {
		el := clip.Ellipse(image.Rect(0, 0, 100, 50))
		el.Op(ops).Push(ops).Pop()
		el.Push(ops).Pop()

		empty := clip.Ellipse{}
		empty.Op(ops).Push(ops).Pop()
	})
}

func TestStroke(t *testing.T) {
	ops := new(op.Ops)
	r := clip.Rect{Max: image.Pt(100, 100)}
	clip.Stroke{
		Path:  r.Path(),
		Width: 5,
	}.Op().Push(ops).Pop()
}

