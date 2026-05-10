package gesture

import (
	"image"
	"testing"
	"time"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/io/event"
	"github.com/nanorele/gio/io/input"
	"github.com/nanorele/gio/io/key"
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

func TestScroll_Complete(t *testing.T) {
	var s Scroll
	var ops op.Ops

	rect := image.Rect(0, 0, 100, 100)
	stack := clip.Rect(rect).Push(&ops)
	s.Add(&ops)
	stack.Pop()

	var r input.Router
	cfg := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	tm := time.Now()

	// 1. Mouse Scroll
	s.Update(cfg, r.Source(), tm, Vertical, pointer.ScrollRange{}, pointer.ScrollRange{Min: -100, Max: 100})
	r.Frame(&ops)
	r.Queue(pointer.Event{
		Kind:     pointer.Scroll,
		Source:   pointer.Mouse,
		Position: f32.Pt(50, 50),
		Scroll:   f32.Pt(0, 20),
	})
	dist := s.Update(cfg, r.Source(), tm, Vertical, pointer.ScrollRange{Min: -100, Max: 100}, pointer.ScrollRange{Min: -100, Max: 100})
	if dist != 20 {
		t.Errorf("Mouse scroll: expected 20, got %d", dist)
	}

	// 2. Touch Drag (Scroll)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(50, 50)},
	)
	// Process press first to start dragging
	_ = s.Update(cfg, r.Source(), tm, Vertical, pointer.ScrollRange{Min: -100, Max: 100}, pointer.ScrollRange{Min: -100, Max: 100})

	r.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(50, 30), Priority: pointer.Grabbed},
	)
	dist = s.Update(cfg, r.Source(), tm, Vertical, pointer.ScrollRange{Min: -100, Max: 100}, pointer.ScrollRange{Min: -100, Max: 100})

	// last=50, current=30, dist = 50-30 = 20
	if dist != 20 {
		t.Errorf("Touch drag: expected 20, got %d", dist)
	}
	if s.State() != StateDragging {
		t.Errorf("expected dragging state, got %v", s.State())
	}
}

func TestDrag_Complete(t *testing.T) {
	var d Drag
	var ops op.Ops

	rect := image.Rect(0, 0, 100, 100)
	stack := clip.Rect(rect).Push(&ops)
	d.Add(&ops)
	stack.Pop()

	var r input.Router
	cfg := unit.Metric{PxPerDp: 1, PxPerSp: 1}

	d.Update(cfg, r.Source(), Both)
	r.Frame(&ops)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(50, 50)},
		pointer.Event{Kind: pointer.Move, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(70, 80), Priority: pointer.Grabbed},
	)

	ev, ok := d.Update(cfg, r.Source(), Both)
	if !ok || ev.Kind != pointer.Press {
		t.Errorf("expected Press, got %v (ok=%v)", ev, ok)
	}
	ev, ok = d.Update(cfg, r.Source(), Both)
	if !ok || ev.Kind != pointer.Drag {
		t.Errorf("expected Drag, got %v (ok=%v)", ev, ok)
	}
	if ev.Position != f32.Pt(70, 80) {
		t.Errorf("expected pos (70, 80), got %v", ev.Position)
	}
}

func TestScroll_Both(t *testing.T) {
	var s Scroll
	var ops op.Ops
	rect := image.Rect(0, 0, 100, 100)
	stack := clip.Rect(rect).Push(&ops)
	s.Add(&ops)
	stack.Pop()

	var r input.Router
	cfg := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	tm := time.Now()

	s.Update(cfg, r.Source(), tm, Both, pointer.ScrollRange{Min: -100, Max: 100}, pointer.ScrollRange{Min: -100, Max: 100})
	r.Frame(&ops)
	r.Queue(pointer.Event{
		Kind:     pointer.Scroll,
		Source:   pointer.Mouse,
		Position: f32.Pt(50, 50),
		Scroll:   f32.Pt(10, 20),
	})
	dist := s.Update(cfg, r.Source(), tm, Both, pointer.ScrollRange{Min: -100, Max: 100}, pointer.ScrollRange{Min: -100, Max: 100})
	if dist != 30 { // 10 + 20
		t.Errorf("Both axis scroll: expected 30, got %d", dist)
	}
}

func TestStrings(t *testing.T) {
	if Horizontal.String() != "Horizontal" || Vertical.String() != "Vertical" || Both.String() != "Both" {
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

func TestDrag_Axis(t *testing.T) {
	var d Drag
	var ops op.Ops
	d.Add(&ops)

	var r input.Router
	cfg := unit.Metric{PxPerDp: 1, PxPerSp: 1}

	r.Frame(&ops)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(10, 10)},
		pointer.Event{Kind: pointer.Move, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(20, 20), Priority: pointer.Grabbed},
	)

	ev, _ := d.Update(cfg, r.Source(), Horizontal)
	if ev.Kind == pointer.Drag && ev.Position.Y != 10 {
		t.Errorf("expected Y to be locked to 10, got %v", ev.Position.Y)
	}

	d = Drag{}
	d.Add(&ops)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 2, Position: f32.Pt(10, 10)},
		pointer.Event{Kind: pointer.Move, Source: pointer.Touch, PointerID: 2, Position: f32.Pt(20, 20), Priority: pointer.Grabbed},
	)
	ev, _ = d.Update(cfg, r.Source(), Vertical)
	if ev.Kind == pointer.Drag && ev.Position.X != 10 {
		t.Errorf("expected X to be locked to 10, got %v", ev.Position.X)
	}
}

// drainClick consumes all queued ClickEvents from the gesture and returns them.
func drainClick(c *Click, r *input.Router) []ClickEvent {
	var out []ClickEvent
	for {
		ev, ok := c.Update(r.Source())
		if !ok {
			break
		}
		out = append(out, ev)
	}
	return out
}

func registerClick(c *Click, ops *op.Ops, rect image.Rectangle) {
	stack := clip.Rect(rect).Push(ops)
	c.Add(ops)
	stack.Pop()
}

// TestClick_PressLeaveRelease verifies Press inside, then move outside (Leave),
// then Release outside produces a KindCancel rather than a KindClick.
func TestClick_PressLeaveRelease(t *testing.T) {
	var c Click
	var ops op.Ops
	registerClick(&c, &ops, image.Rect(20, 20, 40, 40))

	var r input.Router
	c.Update(r.Source())
	r.Frame(&ops)

	// Move into area (router synthesizes Enter), Press, Move outside
	// (router synthesizes Leave), Release outside.
	r.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(30, 30)},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(30, 30)},
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(100, 100)},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(100, 100)},
	)

	gotCancel := false
	gotClick := false
	for _, ev := range drainClick(&c, &r) {
		switch ev.Kind {
		case KindCancel:
			gotCancel = true
		case KindClick:
			gotClick = true
		}
	}
	if gotClick {
		t.Errorf("did not expect KindClick when releasing outside area")
	}
	if !gotCancel {
		t.Errorf("expected KindCancel after press+leave+release outside")
	}
	if c.Pressed() {
		t.Errorf("expected not pressed after release")
	}
}

// TestClick_DoubleClickBoundary verifies the double-click threshold is strictly
// less-than: at duration-1 -> double-click; at duration -> single click.
func TestClick_DoubleClickBoundary(t *testing.T) {
	cases := []struct {
		label string
		gap   time.Duration
		want  int // expected NumClicks on the second press
	}{
		{"just under threshold", doubleClickDuration - 1, 2},
		{"exactly at threshold", doubleClickDuration, 1},
		{"just over threshold", doubleClickDuration + 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			var c Click
			var ops op.Ops
			c.Add(&ops)
			var r input.Router
			c.Update(r.Source())
			r.Frame(&ops)

			r.Queue(mouseClickEvents(0, tc.gap)...)

			var lastClick ClickEvent
			seen := 0
			for _, ev := range drainClick(&c, &r) {
				if ev.Kind == KindClick {
					lastClick = ev
					seen++
				}
			}
			if seen != 2 {
				t.Fatalf("expected 2 KindClick events, got %d", seen)
			}
			if lastClick.NumClicks != tc.want {
				t.Errorf("gap=%v: NumClicks=%d, want %d", tc.gap, lastClick.NumClicks, tc.want)
			}
		})
	}
}

// TestClick_MultiplePointers verifies that with two simultaneous pointers,
// only the originally-pressed pointer drives the click; Release of another
// pointer must not produce a spurious KindClick.
func TestClick_MultiplePointers(t *testing.T) {
	var c Click
	var ops op.Ops
	registerClick(&c, &ops, image.Rect(0, 0, 100, 100))

	var r input.Router
	c.Update(r.Source())
	r.Frame(&ops)

	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(10, 10)},
		// Second pointer press; should be ignored while first is held.
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 2, Position: f32.Pt(20, 20)},
		// Release the SECOND pointer first; should NOT yield a click.
		pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 2, Position: f32.Pt(20, 20)},
		// Now release the first pointer; should yield a click.
		pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(10, 10)},
	)

	clicks := 0
	for _, ev := range drainClick(&c, &r) {
		if ev.Kind == KindClick {
			clicks++
		}
	}
	if clicks != 1 {
		t.Errorf("expected exactly 1 KindClick, got %d", clicks)
	}
}

// TestClick_ModifiersCarried checks that modifier keys are carried into both
// KindPress and KindClick events.
func TestClick_ModifiersCarried(t *testing.T) {
	var c Click
	var ops op.Ops
	c.Add(&ops)
	var r input.Router
	c.Update(r.Source())
	r.Frame(&ops)

	mods := key.ModCtrl | key.ModShift
	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Modifiers: mods},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Modifiers: mods},
	)

	sawPress := false
	sawClick := false
	for _, ev := range drainClick(&c, &r) {
		switch ev.Kind {
		case KindPress:
			if ev.Modifiers != mods {
				t.Errorf("KindPress modifiers=%v, want %v", ev.Modifiers, mods)
			}
			sawPress = true
		case KindClick:
			if ev.Modifiers != mods {
				t.Errorf("KindClick modifiers=%v, want %v", ev.Modifiers, mods)
			}
			sawClick = true
		}
	}
	if !sawPress || !sawClick {
		t.Errorf("expected both KindPress and KindClick (press=%v click=%v)", sawPress, sawClick)
	}
}

// TestClick_NonPrimaryMouseIgnored verifies that secondary/tertiary mouse
// buttons do NOT generate a click via the Click gesture.
func TestClick_NonPrimaryMouseIgnored(t *testing.T) {
	var c Click
	var ops op.Ops
	c.Add(&ops)
	var r input.Router
	c.Update(r.Source())
	r.Frame(&ops)

	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonSecondary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Buttons: pointer.ButtonSecondary},
	)

	for _, ev := range drainClick(&c, &r) {
		if ev.Kind == KindClick || ev.Kind == KindPress {
			t.Errorf("secondary mouse button should be ignored, got %v", ev.Kind)
		}
	}
	if c.Pressed() {
		t.Errorf("Click should not be pressed after secondary-button events")
	}
}

// TestClick_CancelClearsState verifies that a pointer.Cancel resets the
// pressed state and emits KindCancel only when actually pressed.
func TestClick_CancelClearsState(t *testing.T) {
	var c Click
	var ops op.Ops
	c.Add(&ops)
	var r input.Router
	c.Update(r.Source())
	r.Frame(&ops)

	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Cancel},
	)

	cancels := 0
	clicks := 0
	for _, ev := range drainClick(&c, &r) {
		switch ev.Kind {
		case KindCancel:
			cancels++
		case KindClick:
			clicks++
		}
	}
	if cancels != 1 || clicks != 0 {
		t.Errorf("expected 1 KindCancel and 0 KindClick after Cancel, got cancels=%d clicks=%d", cancels, clicks)
	}
	if c.Pressed() {
		t.Errorf("expected not pressed after Cancel")
	}
}

// --- Drag tests ---

// TestDrag_HorizontalLocksY verifies horizontal axis constraint.
func TestDrag_HorizontalLocksY(t *testing.T) {
	axisLockTest(t, Horizontal, f32.Pt(10, 20), f32.Pt(40, 80), 20, 0)
}

// TestDrag_VerticalLocksX verifies vertical axis constraint.
func TestDrag_VerticalLocksX(t *testing.T) {
	axisLockTest(t, Vertical, f32.Pt(10, 20), f32.Pt(40, 80), 0, 80)
}

// TestDrag_BothFree verifies Both axis allows movement on both axes.
func TestDrag_BothFree(t *testing.T) {
	axisLockTest(t, Both, f32.Pt(10, 20), f32.Pt(40, 80), 40, 80)
}

func axisLockTest(t *testing.T, axis Axis, start, move f32.Point, wantX, wantY float32) {
	t.Helper()
	_, _ = wantX, wantY
	var d Drag
	var ops op.Ops
	d.Add(&ops)
	var r input.Router
	cfg := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	d.Update(cfg, r.Source(), axis)
	r.Frame(&ops)

	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: start},
		pointer.Event{Kind: pointer.Move, Source: pointer.Touch, PointerID: 1, Position: move, Priority: pointer.Grabbed},
	)

	var dragEv pointer.Event
	saw := false
	for {
		ev, ok := d.Update(cfg, r.Source(), axis)
		if !ok {
			break
		}
		if ev.Kind == pointer.Drag {
			dragEv = ev
			saw = true
		}
	}
	if !saw {
		t.Fatalf("axis=%v: no Drag event observed", axis)
	}
	switch axis {
	case Horizontal:
		if dragEv.Position.Y != start.Y {
			t.Errorf("Horizontal: Y should be locked to %v, got %v", start.Y, dragEv.Position.Y)
		}
		if dragEv.Position.X != move.X {
			t.Errorf("Horizontal: X should be %v, got %v", move.X, dragEv.Position.X)
		}
	case Vertical:
		if dragEv.Position.X != start.X {
			t.Errorf("Vertical: X should be locked to %v, got %v", start.X, dragEv.Position.X)
		}
		if dragEv.Position.Y != move.Y {
			t.Errorf("Vertical: Y should be %v, got %v", move.Y, dragEv.Position.Y)
		}
	case Both:
		if dragEv.Position.X != move.X || dragEv.Position.Y != move.Y {
			t.Errorf("Both: position want %v, got %v", move, dragEv.Position)
		}
	}
}

// TestDrag_MultiTouchOnlyFirst verifies that only the first pointer drives the
// drag and subsequent pointers do not retarget the gesture.
func TestDrag_MultiTouchOnlyFirst(t *testing.T) {
	var d Drag
	var ops op.Ops
	d.Add(&ops)
	var r input.Router
	cfg := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	d.Update(cfg, r.Source(), Both)
	r.Frame(&ops)

	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(10, 10)},
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 2, Position: f32.Pt(50, 50)},
		// Drag from second pointer should be ignored.
		pointer.Event{Kind: pointer.Move, Source: pointer.Touch, PointerID: 2, Position: f32.Pt(60, 60), Priority: pointer.Grabbed},
		// Drag from first pointer should be honored.
		pointer.Event{Kind: pointer.Move, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(20, 20), Priority: pointer.Grabbed},
	)

	var drags []pointer.Event
	for {
		ev, ok := d.Update(cfg, r.Source(), Both)
		if !ok {
			break
		}
		if ev.Kind == pointer.Drag {
			drags = append(drags, ev)
		}
	}
	for _, dv := range drags {
		if dv.PointerID != 1 {
			t.Errorf("Drag honored pointer %d, expected only pointer 1", dv.PointerID)
		}
	}
}

// TestDrag_MultiTouchSecondReleaseDoesNotClearPressed verifies that releasing
// a second non-owning pointer does NOT clear Pressed/Dragging on the gesture.
// This guards against a bug where Drag.Release unconditionally clears
// d.pressed before checking pid.
func TestDrag_MultiTouchSecondReleaseDoesNotClearPressed(t *testing.T) {
	var d Drag
	var ops op.Ops
	d.Add(&ops)
	var r input.Router
	cfg := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	d.Update(cfg, r.Source(), Both)
	r.Frame(&ops)

	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(10, 10)},
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 2, Position: f32.Pt(50, 50)},
		pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 2, Position: f32.Pt(50, 50)},
	)

	// Drain.
	for {
		_, ok := d.Update(cfg, r.Source(), Both)
		if !ok {
			break
		}
	}
	if !d.Pressed() {
		t.Errorf("Pressed should remain true while owning pointer 1 is still down")
	}
	if !d.Dragging() {
		t.Errorf("Dragging should remain true while owning pointer 1 is still down")
	}
}

// TestDrag_ZeroDistance verifies a Press immediately followed by a Release at
// the same position does not trigger a drag (no Move/Grab).
func TestDrag_ZeroDistance(t *testing.T) {
	var d Drag
	var ops op.Ops
	d.Add(&ops)
	var r input.Router
	cfg := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	d.Update(cfg, r.Source(), Both)
	r.Frame(&ops)

	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(10, 10)},
		pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(10, 10)},
	)
	kinds := []pointer.Kind{}
	for {
		ev, ok := d.Update(cfg, r.Source(), Both)
		if !ok {
			break
		}
		kinds = append(kinds, ev.Kind)
	}
	for _, k := range kinds {
		if k == pointer.Drag {
			t.Errorf("unexpected Drag event for zero-distance gesture")
		}
	}
	if d.Dragging() || d.Pressed() {
		t.Errorf("expected not dragging or pressed after release, got dragging=%v pressed=%v", d.Dragging(), d.Pressed())
	}
}

// TestDrag_NonPrimaryMouseIgnored verifies a non-primary mouse press does not
// start a drag.
func TestDrag_NonPrimaryMouseIgnored(t *testing.T) {
	var d Drag
	var ops op.Ops
	d.Add(&ops)
	var r input.Router
	cfg := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	d.Update(cfg, r.Source(), Both)
	r.Frame(&ops)

	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonSecondary, Position: f32.Pt(10, 10)},
	)
	for {
		_, ok := d.Update(cfg, r.Source(), Both)
		if !ok {
			break
		}
	}
	if d.Dragging() || d.Pressed() {
		t.Errorf("secondary mouse should not start drag")
	}
}

// TestDrag_CancelEndsDrag verifies pointer.Cancel ends an in-progress drag.
func TestDrag_CancelEndsDrag(t *testing.T) {
	var d Drag
	var ops op.Ops
	d.Add(&ops)
	var r input.Router
	cfg := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	d.Update(cfg, r.Source(), Both)
	r.Frame(&ops)

	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(10, 10)},
		pointer.Event{Kind: pointer.Cancel, Source: pointer.Touch, PointerID: 1},
	)
	for {
		_, ok := d.Update(cfg, r.Source(), Both)
		if !ok {
			break
		}
	}
	if d.Dragging() || d.Pressed() {
		t.Errorf("drag should be cleared after Cancel: dragging=%v pressed=%v", d.Dragging(), d.Pressed())
	}
}

// --- Scroll tests ---

// TestScroll_AccumulatorFractional verifies that small fractional wheel events
// combine into integer steps and the fractional remainder is retained.
func TestScroll_AccumulatorFractional(t *testing.T) {
	var s Scroll
	var ops op.Ops
	s.Add(&ops)
	var r input.Router
	cfg := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	tm := time.Now()
	rng := pointer.ScrollRange{Min: -100, Max: 100}

	s.Update(cfg, r.Source(), tm, Vertical, pointer.ScrollRange{}, rng)
	r.Frame(&ops)

	// Three 0.4 wheel events: 0.4, 0.8 (->0 keep .8), 1.2 (->1 keep .2).
	r.Queue(
		pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Scroll: f32.Pt(0, 0.4)},
		pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Scroll: f32.Pt(0, 0.4)},
		pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Scroll: f32.Pt(0, 0.4)},
	)
	dist := s.Update(cfg, r.Source(), tm, Vertical, pointer.ScrollRange{}, rng)
	if dist != 1 {
		t.Errorf("expected accumulated dist 1, got %d", dist)
	}

	// Another 0.4 should carry remainder 0.2 + 0.4 = 0.6 -> still 0.
	r.Frame(&ops)
	r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Scroll: f32.Pt(0, 0.4)})
	dist = s.Update(cfg, r.Source(), tm, Vertical, pointer.ScrollRange{}, rng)
	if dist != 0 {
		t.Errorf("expected dist 0 (accumulator carry), got %d", dist)
	}

	// 0.5 more pushes accumulator over 1.
	r.Frame(&ops)
	r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Scroll: f32.Pt(0, 0.5)})
	dist = s.Update(cfg, r.Source(), tm, Vertical, pointer.ScrollRange{}, rng)
	if dist != 1 {
		t.Errorf("expected dist 1 after carry, got %d", dist)
	}
}

// TestScroll_NegativeScroll verifies negative scroll deltas accumulate and
// emit negative integer steps.
func TestScroll_NegativeScroll(t *testing.T) {
	var s Scroll
	var ops op.Ops
	s.Add(&ops)
	var r input.Router
	cfg := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	tm := time.Now()
	rng := pointer.ScrollRange{Min: -100, Max: 100}

	s.Update(cfg, r.Source(), tm, Vertical, pointer.ScrollRange{}, rng)
	r.Frame(&ops)
	r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Scroll: f32.Pt(0, -7)})
	dist := s.Update(cfg, r.Source(), tm, Vertical, pointer.ScrollRange{}, rng)
	if dist != -7 {
		t.Errorf("expected dist -7, got %d", dist)
	}
}

// TestScroll_HorizontalAxis verifies axis constraint to Horizontal ignores
// vertical wheel.
func TestScroll_HorizontalAxis(t *testing.T) {
	var s Scroll
	var ops op.Ops
	s.Add(&ops)
	var r input.Router
	cfg := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	tm := time.Now()
	rng := pointer.ScrollRange{Min: -100, Max: 100}

	s.Update(cfg, r.Source(), tm, Horizontal, rng, pointer.ScrollRange{})
	r.Frame(&ops)
	r.Queue(pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Scroll: f32.Pt(8, 4)})
	dist := s.Update(cfg, r.Source(), tm, Horizontal, rng, pointer.ScrollRange{})
	if dist != 8 {
		t.Errorf("expected dist 8 (horizontal only), got %d", dist)
	}
}

// TestScroll_StateAfterRelease verifies state returns to Idle after the touch
// drag completes.
func TestScroll_StateAfterRelease(t *testing.T) {
	var s Scroll
	var ops op.Ops
	rect := image.Rect(0, 0, 100, 100)
	stack := clip.Rect(rect).Push(&ops)
	s.Add(&ops)
	stack.Pop()

	var r input.Router
	cfg := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	tm := time.Now()
	rng := pointer.ScrollRange{Min: -100, Max: 100}

	s.Update(cfg, r.Source(), tm, Vertical, rng, rng)
	r.Frame(&ops)

	r.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(50, 50)})
	s.Update(cfg, r.Source(), tm, Vertical, rng, rng)
	if s.State() != StateDragging {
		t.Fatalf("expected dragging after press, got %v", s.State())
	}

	// Release with a tiny displacement should not start a fling.
	r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(50, 50)})
	s.Update(cfg, r.Source(), tm.Add(10*time.Millisecond), Vertical, rng, rng)
	if s.State() != StateIdle {
		t.Errorf("expected idle after release, got %v", s.State())
	}
}

// TestScroll_StopWhileFlinging is a smoke test: Stop() should always reset.
func TestScroll_StopIdempotent(t *testing.T) {
	var s Scroll
	s.Stop()
	s.Stop()
	if s.State() != StateIdle {
		t.Errorf("expected Idle, got %v", s.State())
	}
}

// TestScroll_CancelEndsDrag verifies pointer.Cancel during touch-drag returns
// state to idle.
func TestScroll_CancelEndsDrag(t *testing.T) {
	var s Scroll
	var ops op.Ops
	rect := image.Rect(0, 0, 100, 100)
	stack := clip.Rect(rect).Push(&ops)
	s.Add(&ops)
	stack.Pop()

	var r input.Router
	cfg := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	tm := time.Now()
	rng := pointer.ScrollRange{Min: -100, Max: 100}

	s.Update(cfg, r.Source(), tm, Vertical, rng, rng)
	r.Frame(&ops)
	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(50, 50)},
	)
	s.Update(cfg, r.Source(), tm, Vertical, rng, rng)
	r.Queue(
		pointer.Event{Kind: pointer.Cancel, Source: pointer.Touch, PointerID: 1},
	)
	s.Update(cfg, r.Source(), tm, Vertical, rng, rng)
	if s.State() != StateIdle {
		t.Errorf("expected idle after Cancel, got %v", s.State())
	}
}

// --- Hover tests ---

// TestHover_FirstFrameNoPointer verifies a brand-new Hover reports false and
// stays false after a Frame with no pointer events.
func TestHover_FirstFrameNoPointer(t *testing.T) {
	var h Hover
	var ops op.Ops
	rect := image.Rect(0, 0, 50, 50)
	stack := clip.Rect(rect).Push(&ops)
	h.Add(&ops)
	stack.Pop()

	var r input.Router
	if h.Update(r.Source()) {
		t.Errorf("Hover should be false initially")
	}
	r.Frame(&ops)
	if h.Update(r.Source()) {
		t.Errorf("Hover should be false on first frame with no events")
	}
}

// TestHover_EnterLeaveCycle exercises a back-and-forth.
func TestHover_EnterLeaveCycle(t *testing.T) {
	var h Hover
	var ops op.Ops
	rect := image.Rect(0, 0, 50, 50)
	stack := clip.Rect(rect).Push(&ops)
	h.Add(&ops)
	stack.Pop()

	var r input.Router
	h.Update(r.Source())
	r.Frame(&ops)

	for i := 0; i < 3; i++ {
		r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(10, 10)})
		if !h.Update(r.Source()) {
			t.Fatalf("iter %d: expected hovered after entering", i)
		}
		r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(100, 100)})
		if h.Update(r.Source()) {
			t.Fatalf("iter %d: expected NOT hovered after leaving", i)
		}
	}
}

// TestHover_TwoPointersFirstLeaves verifies that when two pointers are both
// hovering and the first one leaves, Hover remains true while the second is
// still inside. This is a regression test for a bug where a Leave event from
// the originally-tracked pointer cleared `entered` even though another
// pointer was still hovering.
//
// Two touch pointers are pressed inside the area (each Press synthesizes an
// Enter for the gesture). Then each is released outside (synthesizing a
// Leave) one at a time.
func TestHover_TwoPointersFirstLeaves(t *testing.T) {
	var h Hover
	var ops op.Ops
	rect := image.Rect(0, 0, 100, 100)
	stack := clip.Rect(rect).Push(&ops)
	h.Add(&ops)
	stack.Pop()

	var r input.Router
	h.Update(r.Source())
	r.Frame(&ops)

	// Both pointers press inside the area -> two Enter events for h.
	r.Queue(
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(10, 10)},
		pointer.Event{Kind: pointer.Press, Source: pointer.Touch, PointerID: 2, Position: f32.Pt(20, 20)},
	)
	if !h.Update(r.Source()) {
		t.Fatalf("expected hovered after two pointers pressed inside")
	}

	// Pointer 1 releases outside the area -> Leave for pointer 1.
	// Pointer 2 is still inside, so Hover MUST stay true. This is the
	// regression: the buggy code cleared `entered` because the leaving
	// pointer's ID matched h.pid.
	r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 1, Position: f32.Pt(500, 500)})
	if !h.Update(r.Source()) {
		t.Errorf("expected still hovered: pointer 2 is still inside")
	}

	// Pointer 2 also releases outside -> no longer hovered.
	r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Touch, PointerID: 2, Position: f32.Pt(500, 500)})
	if h.Update(r.Source()) {
		t.Errorf("expected NOT hovered after both pointers left")
	}
}

// TestHover_CancelExits verifies pointer.Cancel clears the hovered flag.
func TestHover_CancelExits(t *testing.T) {
	var h Hover
	var ops op.Ops
	rect := image.Rect(0, 0, 50, 50)
	stack := clip.Rect(rect).Push(&ops)
	h.Add(&ops)
	stack.Pop()

	var r input.Router
	h.Update(r.Source())
	r.Frame(&ops)

	r.Queue(pointer.Event{Kind: pointer.Move, Position: f32.Pt(10, 10)})
	if !h.Update(r.Source()) {
		t.Fatalf("expected hovered after enter")
	}
	// Synthesize a Cancel for the same pointer ID (default 0).
	r.Queue(pointer.Event{Kind: pointer.Cancel, PointerID: 0})
	if h.Update(r.Source()) {
		t.Errorf("expected not hovered after Cancel")
	}
}
