package app

import (
	"testing"

	"github.com/nanorele/gio/f32"
)

// DragFilesEvent and DropFilesEvent must be delivered to the application through
// the coalesced event queue, NOT routed into the input.Router (which only knows
// pointer/key events and panics with "unknown event type" on anything else).
// This reproduces the crash seen when a file was dragged over the window.

func TestDragFilesEventDeliveredViaCoalesced(t *testing.T) {
	var w Window
	w.processEvent(DragFilesEvent{Position: f32.Point{X: 3, Y: 4}, Active: true})

	if len(w.coalesced.dragFiles) != 1 {
		t.Fatalf("DragFilesEvent must be coalesced for the app, got %d queued", len(w.coalesced.dragFiles))
	}
	e, ok := w.nextEvent()
	if !ok {
		t.Fatal("nextEvent must return the queued drag event")
	}
	de, ok := e.(DragFilesEvent)
	if !ok {
		t.Fatalf("nextEvent returned %T, want DragFilesEvent", e)
	}
	if !de.Active || de.Position != (f32.Point{X: 3, Y: 4}) {
		t.Errorf("drag event = %+v, want active at (3,4)", de)
	}
	x, y, active := de.DraggedFiles()
	if x != 3 || y != 4 || !active {
		t.Errorf("DraggedFiles() = (%v,%v,%v), want (3,4,true)", x, y, active)
	}
}

func TestDropFilesEventDeliveredViaCoalesced(t *testing.T) {
	var w Window
	w.processEvent(DropFilesEvent{Paths: []string{"a.har"}, Position: f32.Point{X: 7, Y: 9}})

	e, ok := w.nextEvent()
	if !ok {
		t.Fatal("nextEvent must return the queued drop event")
	}
	de, ok := e.(DropFilesEvent)
	if !ok {
		t.Fatalf("nextEvent returned %T, want DropFilesEvent", e)
	}
	if len(de.Paths) != 1 || de.Paths[0] != "a.har" {
		t.Errorf("drop paths = %v", de.Paths)
	}
	if x, y := de.DroppedPosition(); x != 7 || y != 9 {
		t.Errorf("DroppedPosition() = (%v,%v), want (7,9)", x, y)
	}
}
