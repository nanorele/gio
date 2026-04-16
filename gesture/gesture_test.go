package gesture

import (
	"image"
	"testing"
	"time"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/io/event"
	"github.com/nanorele/gio/io/input"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/op/clip"
	"github.com/nanorele/gio/unit"
)

func TestHover(t *testing.T) {
	ops := new(op.Ops)
	var h Hover
	rect := image.Rect(20, 20, 40, 40)
	stack := clip.Rect(rect).Push(ops)
	h.Add(ops)
	stack.Pop()
	r := new(input.Router)
	h.Update(r.Source())
	r.Frame(ops)

	r.Queue(
		pointer.Event{Kind: pointer.Move, Position: f32.Pt(30, 30)},
	)
	if !h.Update(r.Source()) {
		t.Fatal("expected hovered")
	}

	r.Queue(
		pointer.Event{Kind: pointer.Move, Position: f32.Pt(50, 50)},
	)
	if h.Update(r.Source()) {
		t.Fatal("expected not hovered")
	}
}

func TestMouseClicks(t *testing.T) {
	for _, tc := range []struct {
		label  string
		events []event.Event
		clicks []int
	}{
		{
			label:  "single click",
			events: mouseClickEvents(200 * time.Millisecond),
			clicks: []int{1},
		},
		{
			label: "double click",
			events: mouseClickEvents(
				100*time.Millisecond,
				100*time.Millisecond+doubleClickDuration-1),
			clicks: []int{1, 2},
		},
		{
			label: "two single clicks",
			events: mouseClickEvents(
				100*time.Millisecond,
				100*time.Millisecond+doubleClickDuration+1),
			clicks: []int{1, 1},
		},
	} {
		t.Run(tc.label, func(t *testing.T) {
			var click Click
			var ops op.Ops
			click.Add(&ops)

			var r input.Router
			click.Update(r.Source())
			r.Frame(&ops)
			r.Queue(tc.events...)

			var clicks []ClickEvent
			for {
				ev, ok := click.Update(r.Source())
				if !ok {
					break
				}
				if ev.Kind == KindClick {
					clicks = append(clicks, ev)
				}
			}
			if got, want := len(clicks), len(tc.clicks); got != want {
				t.Fatalf("got %d mouse clicks, expected %d", got, want)
			}

			for i, click := range clicks {
				if got, want := click.NumClicks, tc.clicks[i]; got != want {
					t.Errorf("got %d combined mouse clicks, expected %d", got, want)
				}
			}
		})
	}
}

func mouseClickEvents(times ...time.Duration) []event.Event {
	press := pointer.Event{
		Kind:    pointer.Press,
		Source:  pointer.Mouse,
		Buttons: pointer.ButtonPrimary,
	}
	events := make([]event.Event, 0, 2*len(times))
	for _, t := range times {
		press := press
		press.Time = t
		release := press
		release.Kind = pointer.Release
		events = append(events, press, release)
	}
	return events
}

func TestStrings(t *testing.T) {
	if Horizontal.String() != "Horizontal" || Vertical.String() != "Vertical" {
		t.Errorf("Axis.String() failed")
	}
	if KindPress.String() != "KindPress" || KindClick.String() != "KindClick" || KindCancel.String() != "KindCancel" {
		t.Errorf("ClickKind.String() failed")
	}
	if StateIdle.String() != "StateIdle" || StateDragging.String() != "StateDragging" || StateFlinging.String() != "StateFlinging" {
		t.Errorf("ScrollState.String() failed")
	}
	ClickEvent{}.ImplementsEvent()
}

func TestClickProperties(t *testing.T) {
	var c Click
	if c.Pressed() || c.Hovered() {
		t.Errorf("Click defaults failed")
	}
}

func TestClickTouch(t *testing.T) {
	var c Click
	var ops op.Ops

	rect := image.Rect(20, 20, 40, 40)
	stack := clip.Rect(rect).Push(&ops)
	c.Add(&ops)
	stack.Pop()

	var r input.Router
	c.Update(r.Source())
	r.Frame(&ops)

	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(30, 30)},
		pointer.Event{Kind: pointer.Move, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(100, 100)},
		pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(100, 100)},
	)

	for {
		ev, ok := c.Update(r.Source())
		if !ok {
			break
		}
		if ev.Kind == KindCancel {
			return
		}
	}
	t.Errorf("expected KindCancel")
}

func TestDrag(t *testing.T) {
	var d Drag
	var ops op.Ops
	d.Add(&ops)

	var r input.Router
	cfg := unit.Metric{PxPerDp: 1, PxPerSp: 1}

	ev, ok := d.Update(cfg, r.Source(), Both)
	if ok {
		t.Errorf("unexpected event %v", ev)
	}

	if d.Dragging() || d.Pressed() {
		t.Errorf("Drag defaults failed")
	}

	r.Frame(&ops)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(10, 10)},
		pointer.Event{Kind: pointer.Move, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(15, 15)},
		pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(15, 15)},
	)

	for i := 0; i < 3; i++ {
		ev, ok = d.Update(cfg, r.Source(), Both)
		if !ok {
			t.Errorf("expected event %d", i)
			break
		}
		if ev.Kind == pointer.Press && !d.Pressed() {
			t.Errorf("expected pressed")
		}
		if ev.Kind == pointer.Drag && !d.Dragging() {
			t.Errorf("expected dragging")
		}
	}

	if d.Pressed() || d.Dragging() {
		t.Errorf("expected not pressed or dragging after release")
	}
}

func TestScroll(t *testing.T) {
	var s Scroll
	var ops op.Ops
	s.Add(&ops)

	var r input.Router
	cfg := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	tm := time.Now()

	dist := s.Update(cfg, r.Source(), tm, Vertical, pointer.ScrollRange{}, pointer.ScrollRange{Min: -100, Max: 100})
	if dist != 0 {
		t.Errorf("unexpected scroll dist %v", dist)
	}

	if s.State() != StateIdle {
		t.Errorf("Scroll defaults failed")
	}

	r.Frame(&ops)
	r.Queue(
		pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Scroll: f32.Pt(0, 10)},
		pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Scroll: f32.Pt(0, -5)},
	)

	dist = s.Update(cfg, r.Source(), tm, Vertical, pointer.ScrollRange{}, pointer.ScrollRange{Min: -100, Max: 100})
	if dist != 5 {
		t.Errorf("expected scroll dist 5, got %v", dist)
	}

	s.Stop()
}
