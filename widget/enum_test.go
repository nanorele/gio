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

func TestEnumZeroValue(t *testing.T) {
	var (
		r input.Router
		e widget.Enum
	)
	gtx := layout.Context{Ops: new(op.Ops), Source: r.Source()}
	if e.Update(gtx) {
		t.Error("zero Enum should not report change")
	}
	if e.Value != "" {
		t.Errorf("zero Value want empty, got %q", e.Value)
	}
	if k, ok := e.Hovered(); ok {
		t.Errorf("zero Enum should not be hovered, got %q,%v", k, ok)
	}
	if k, ok := e.Focused(); ok {
		t.Errorf("zero Enum should not be focused, got %q,%v", k, ok)
	}
}

func TestEnumClickSelectsKey(t *testing.T) {
	var (
		r input.Router
		e widget.Enum
	)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(100, 100)),
		Source:      r.Source(),
	}
	doLayout := func() {
		e.Layout(gtx, "a", func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(100, 100)}
		})
	}
	frame := func() {
		gtx.Reset()
		doLayout()
		r.Frame(gtx.Ops)
	}

	// Two frames are required: the first registers the key, the second
	// gives state.click a chance to install its pointer filter.
	frame()
	frame()
	r.Queue(
		pointer.Event{Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Kind: pointer.Press, Position: f32.Pt(50, 50)},
		pointer.Event{Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Kind: pointer.Release, Position: f32.Pt(50, 50)},
	)
	gtx.Reset()
	doLayout()
	if e.Value != "a" {
		t.Errorf("Value should be %q, got %q", "a", e.Value)
	}
	r.Frame(gtx.Ops)

	// Re-clicking the same key should not change Value.
	r.Queue(
		pointer.Event{Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Kind: pointer.Press, Position: f32.Pt(50, 50)},
		pointer.Event{Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Kind: pointer.Release, Position: f32.Pt(50, 50)},
	)
	gtx.Reset()
	doLayout()
	if e.Value != "a" {
		t.Errorf("Value should remain %q, got %q", "a", e.Value)
	}
}

func TestEnumSingleSelection(t *testing.T) {
	var e widget.Enum
	e.Value = "a"
	// Calling Update with no event source / events: should remain "a".
	gtx := layout.Context{Ops: new(op.Ops)}
	if e.Update(gtx) {
		t.Error("Update without events should not report change")
	}
	if e.Value != "a" {
		t.Errorf("Value mutated unexpectedly: %q", e.Value)
	}
}

func TestEnumDisabledClearsFocus(t *testing.T) {
	var (
		r input.Router
		e widget.Enum
	)
	gtx := layout.Context{Ops: new(op.Ops), Source: r.Source()}
	// Drive Update with Disabled context: focus must be reset.
	dis := gtx.Disabled()
	if e.Update(dis) {
		t.Error("Update on disabled with no events should not report change")
	}
	if _, ok := e.Focused(); ok {
		t.Error("disabled enum should not report focused")
	}
}

func TestEnumLayoutRegistersKey(t *testing.T) {
	var (
		r input.Router
		e widget.Enum
	)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(50, 50)),
		Source:      r.Source(),
	}
	for _, k := range []string{"x", "y", "z"} {
		k := k
		e.Layout(gtx, k, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(50, 50)}
		})
	}
	// Re-layout same keys – should not duplicate internal state.
	for _, k := range []string{"x", "y", "z"} {
		k := k
		e.Layout(gtx, k, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(50, 50)}
		})
	}
	// Sanity: Update should be a no-op without queued events.
	if e.Update(gtx) {
		t.Error("Update without events should not report change")
	}
}
