package widget_test

import (
	"image"
	"testing"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/io/input"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/io/system"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/widget"
)

func TestDecorationsClickableSameInstance(t *testing.T) {
	var d widget.Decorations
	c1 := d.Clickable(system.ActionMinimize)
	c2 := d.Clickable(system.ActionMinimize)
	if c1 != c2 {
		t.Error("Clickable should return the same instance for the same action")
	}
	c3 := d.Clickable(system.ActionMaximize)
	if c1 == c3 {
		t.Error("Clickable should return different instances for different actions")
	}
}

func TestDecorationsClickablePanicsOnComposite(t *testing.T) {
	var d widget.Decorations
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on composite action")
		}
	}()
	d.Clickable(system.ActionMinimize | system.ActionMaximize)
}

func TestDecorationsUpdateNoClicks(t *testing.T) {
	var (
		r input.Router
		d widget.Decorations
	)
	gtx := layout.Context{Ops: new(op.Ops), Source: r.Source()}
	if got := d.Update(gtx); got != 0 {
		t.Errorf("Update with no clicks = %v want 0", got)
	}
}

func TestDecorationsClickProducesAction(t *testing.T) {
	var (
		r input.Router
		d widget.Decorations
	)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(100, 100)),
		Source:      r.Source(),
	}
	clk := d.Clickable(system.ActionClose)
	doLayout := func() {
		clk.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(100, 100)}
		})
	}

	gtx.Reset()
	doLayout()
	r.Frame(gtx.Ops)

	r.Queue(
		pointer.Event{Source: pointer.Touch, Kind: pointer.Press, Position: f32.Pt(50, 50)},
		pointer.Event{Source: pointer.Touch, Kind: pointer.Release, Position: f32.Pt(50, 50)},
	)
	gtx.Reset()
	got := d.Update(gtx)
	if got&system.ActionClose == 0 {
		t.Errorf("expected ActionClose, got %v", got)
	}
	doLayout()
}

func TestDecorationsMaximizeToggle(t *testing.T) {
	var (
		r input.Router
		d widget.Decorations
	)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(100, 100)),
		Source:      r.Source(),
	}
	clk := d.Clickable(system.ActionMaximize)
	doLayout := func() {
		clk.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(100, 100)}
		})
	}
	doLayout()
	r.Frame(gtx.Ops)

	// Click while not maximized => ActionMaximize.
	r.Queue(
		pointer.Event{Source: pointer.Touch, Kind: pointer.Press, Position: f32.Pt(20, 20)},
		pointer.Event{Source: pointer.Touch, Kind: pointer.Release, Position: f32.Pt(20, 20)},
	)
	gtx.Reset()
	got := d.Update(gtx)
	if got&system.ActionMaximize == 0 || got&system.ActionUnmaximize != 0 {
		t.Errorf("expected ActionMaximize, got %v", got)
	}
	doLayout()

	// Now mark maximized; same Maximize-bound clickable should report Unmaximize.
	d.Maximized = true
	r.Frame(gtx.Ops)
	r.Queue(
		pointer.Event{Source: pointer.Touch, Kind: pointer.Press, Position: f32.Pt(20, 20)},
		pointer.Event{Source: pointer.Touch, Kind: pointer.Release, Position: f32.Pt(20, 20)},
	)
	gtx.Reset()
	got = d.Update(gtx)
	if got&system.ActionUnmaximize == 0 {
		t.Errorf("expected ActionUnmaximize, got %v", got)
	}
	doLayout()
}
