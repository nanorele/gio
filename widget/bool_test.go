package widget_test

import (
	"image"
	"testing"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/io/input"
	"github.com/nanorele/gio/io/key"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/widget"
)

func TestBoolUpdateNoEvents(t *testing.T) {
	var (
		r input.Router
		b widget.Bool
	)
	gtx := layout.Context{Ops: new(op.Ops), Source: r.Source()}
	if b.Update(gtx) {
		t.Error("Update returned true with no events")
	}
	if b.Value {
		t.Error("Value should default to false")
	}
}

func TestBoolUpdateMultipleClicks(t *testing.T) {
	var (
		r input.Router
		b widget.Bool
	)
	gtx := layout.Context{Ops: new(op.Ops), Source: r.Source()}

	doLayout := func() {
		b.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(100, 100)}
		})
	}
	frame := func() {
		gtx.Reset()
		doLayout()
		r.Frame(gtx.Ops)
	}

	frame()
	wantValues := []bool{true, false, true}
	for i, want := range wantValues {
		r.Queue(
			pointer.Event{Source: pointer.Touch, Kind: pointer.Press, Position: f32.Pt(50, 50)},
			pointer.Event{Source: pointer.Touch, Kind: pointer.Release, Position: f32.Pt(50, 50)},
		)
		gtx.Reset()
		doLayout()
		if got := b.Value; got != want {
			t.Errorf("click %d: got Value=%v want %v", i, got, want)
		}
		r.Frame(gtx.Ops)
	}
}

func TestBoolUpdateChangedReturn(t *testing.T) {
	var (
		r input.Router
		b widget.Bool
	)
	gtx := layout.Context{Ops: new(op.Ops), Source: r.Source()}
	doLayout := func() {
		b.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(50, 50)}
		})
	}

	gtx.Execute(key.FocusCmd{Tag: &b})
	gtx.Reset()
	doLayout()
	r.Frame(gtx.Ops)

	r.Queue(
		key.Event{Name: key.NameReturn, State: key.Press},
		key.Event{Name: key.NameReturn, State: key.Release},
	)
	gtx.Reset()
	if !b.Update(gtx) {
		t.Error("Update should report change after return key")
	}
	if !b.Value {
		t.Error("Value should toggle to true")
	}

	if b.Update(gtx) {
		t.Error("Update should not report change a second time without a new event")
	}
}

func TestBoolHoveredPressedInitial(t *testing.T) {
	var b widget.Bool
	if b.Hovered() {
		t.Error("zero Bool should not be Hovered")
	}
	if b.Pressed() {
		t.Error("zero Bool should not be Pressed")
	}
	if got := b.History(); len(got) != 0 {
		t.Errorf("zero Bool History should be empty, got %v", got)
	}
}
