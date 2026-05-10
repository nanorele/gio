package widget_test

import (
	"image"
	"testing"

	"github.com/nanorele/gio/io/input"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/widget"
)

func TestScrollbarZero(t *testing.T) {
	var s widget.Scrollbar
	if s.Dragging() {
		t.Error("zero Scrollbar should not be Dragging")
	}
	if s.IndicatorHovered() || s.TrackHovered() {
		t.Error("zero Scrollbar hovers should be false")
	}
	if got := s.ScrollDistance(); got != 0 {
		t.Errorf("zero ScrollDistance = %v want 0", got)
	}
}

func TestScrollbarUpdateNoEventsResetsDelta(t *testing.T) {
	var (
		r input.Router
		s widget.Scrollbar
	)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(100, 100)),
		Source:      r.Source(),
	}
	s.Update(gtx, layout.Vertical, 0.0, 0.5)
	if got := s.ScrollDistance(); got != 0 {
		t.Errorf("ScrollDistance after empty Update = %v want 0", got)
	}
}

func TestScrollbarAddOps(t *testing.T) {
	// Smoke-test that Add* methods do not panic with a fresh Ops.
	var s widget.Scrollbar
	ops := new(op.Ops)
	s.AddTrack(ops)
	s.AddIndicator(ops)
	s.AddDrag(ops)
}

func TestListEmbedsScrollbarAndLayoutList(t *testing.T) {
	var l widget.List
	// Confirm pure field semantics: scrollbar half is not dragging, and the
	// embedded layout.List has zero Position by default.
	if l.Scrollbar.Dragging() {
		t.Error("zero Scrollbar should not be Dragging")
	}
	if l.Position != (layout.Position{}) {
		t.Errorf("zero List.Position should be zero, got %+v", l.Position)
	}
}

