package widget_test

import (
	"image"
	"testing"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/io/input"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/widget"
)

func TestFloatZeroValue(t *testing.T) {
	var f widget.Float
	if f.Value != 0 {
		t.Errorf("zero Value want 0, got %v", f.Value)
	}
	if f.Dragging() {
		t.Error("zero Float should not be Dragging")
	}
}

func TestFloatUpdateNoLayout(t *testing.T) {
	var (
		r input.Router
		f widget.Float
	)
	gtx := layout.Context{Ops: new(op.Ops), Source: r.Source()}
	// With no Layout call, length is zero and any pending event must be ignored.
	if f.Update(gtx) {
		t.Error("Update without layout/events should not report change")
	}
}

func TestFloatPressSetsValueHorizontal(t *testing.T) {
	var (
		r input.Router
		f widget.Float
	)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(100, 20)),
		Source:      r.Source(),
	}
	doLayout := func() {
		f.Layout(gtx, layout.Horizontal, 0)
	}

	gtx.Reset()
	doLayout()
	r.Frame(gtx.Ops)

	r.Queue(
		pointer.Event{Source: pointer.Touch, Kind: pointer.Press, Position: f32.Pt(25, 10)},
	)
	gtx.Reset()
	doLayout()
	if got, want := f.Value, float32(0.25); got != want {
		t.Errorf("Value = %v want %v", got, want)
	}
}

func TestFloatClampsValue(t *testing.T) {
	var (
		r input.Router
		f widget.Float
	)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(100, 20)),
		Source:      r.Source(),
	}
	doLayout := func() { f.Layout(gtx, layout.Horizontal, 0) }

	doLayout()
	r.Frame(gtx.Ops)

	// Press with primary mouse button, then a Move (Router translates to Drag).
	r.Queue(
		pointer.Event{Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Kind: pointer.Press, Position: f32.Pt(50, 10)},
		pointer.Event{Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Kind: pointer.Move, Position: f32.Pt(500, 10)},
	)
	gtx.Reset()
	doLayout()
	if f.Value != 1 {
		t.Errorf("Value should clamp to 1, got %v", f.Value)
	}

	r.Frame(gtx.Ops)
	r.Queue(
		pointer.Event{Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Kind: pointer.Move, Position: f32.Pt(-500, 10)},
	)
	gtx.Reset()
	doLayout()
	if f.Value != 0 {
		t.Errorf("Value should clamp to 0, got %v", f.Value)
	}
}

func TestFloatVerticalAxisInverted(t *testing.T) {
	// On vertical axis, Position.Y near 0 means top => Value near 1.
	var (
		r input.Router
		f widget.Float
	)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(20, 100)),
		Source:      r.Source(),
	}
	doLayout := func() { f.Layout(gtx, layout.Vertical, 0) }

	doLayout()
	r.Frame(gtx.Ops)

	r.Queue(
		pointer.Event{Source: pointer.Touch, Kind: pointer.Press, Position: f32.Pt(10, 25)},
	)
	gtx.Reset()
	doLayout()
	if got, want := f.Value, float32(0.75); got != want {
		t.Errorf("Vertical Value = %v want %v", got, want)
	}
}

func TestFloatUpdateReturnsChanged(t *testing.T) {
	var (
		r input.Router
		f widget.Float
	)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(100, 20)),
		Source:      r.Source(),
	}
	f.Layout(gtx, layout.Horizontal, 0)
	r.Frame(gtx.Ops)

	r.Queue(
		pointer.Event{Source: pointer.Touch, Kind: pointer.Press, Position: f32.Pt(40, 10)},
	)
	gtx.Reset()
	f.Layout(gtx, layout.Horizontal, 0)
	// Layout already calls Update once; another Update should yield no change.
	if f.Update(gtx) {
		t.Error("subsequent Update without new events should not report change")
	}
}
