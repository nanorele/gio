package input

import (
	"image"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/io/clipboard"
	"github.com/nanorele/gio/io/event"
	"github.com/nanorele/gio/io/key"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/io/transfer"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/op/clip"
)

func TestNoFilterAllocs(t *testing.T) {
	b := testing.Benchmark(func(b *testing.B) {
		var r Router
		s := r.Source()
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			s.Event(pointer.Filter{})
		}
	})
	if allocs := b.AllocsPerOp(); allocs != 0 {
		t.Fatalf("expected 0 AllocsPerOp, got %d", allocs)
	}
}

func TestRouterWakeup(t *testing.T) {
	r := new(Router)
	r.Source().Execute(op.InvalidateCmd{})
	r.Frame(new(op.Ops))
	if _, wake := r.WakeupTime(); !wake {
		t.Errorf("InvalidateCmd did not trigger a redraw")
	}
}

func TestSourceEventNoEvents(t *testing.T) {
	r := new(Router)
	if e, ok := r.Source().Event(); ok {
		t.Errorf("expected no event, got %v", e)
	}
	h := new(int)
	if e, ok := r.Source().Event(pointer.Filter{Target: h, Kinds: pointer.Press}); ok {
		t.Errorf("unexpected event %v on first call (after reset already consumed)", e)
	}
}

func TestSourceEventDisabled(t *testing.T) {
	r := new(Router)
	src := r.Source().Disabled()
	if src.Enabled() {
		t.Fatal("disabled source still reports enabled")
	}
	if e, ok := src.Event(pointer.Filter{Target: new(int)}); ok {
		t.Errorf("disabled source returned event %v", e)
	}
	src.Execute(key.SoftKeyboardCmd{Show: true})
	if r.state().state == TextInputOpen {
		t.Errorf("disabled source executed command")
	}
}

func TestSourceEventNilSource(t *testing.T) {
	var s Source
	if s.Enabled() {
		t.Fatal("zero Source should not be enabled")
	}
	if _, ok := s.Event(pointer.Filter{Target: new(int)}); ok {
		t.Fatal("zero Source returned event")
	}
	if s.Focused(new(int)) {
		t.Fatal("zero Source reports focused")
	}

	s.Execute(key.SoftKeyboardCmd{Show: true})
}

func TestSourceEventFilterMismatch(t *testing.T) {
	r := new(Router)
	h := new(int)

	events(r, -1, pointer.Filter{Target: h, Kinds: pointer.Press})
	r.Frame(new(op.Ops))

	r.Queue(pointer.Event{Kind: pointer.Press, Position: f32.Pt(5, 5)})

	if e, ok := r.Source().Event(pointer.Filter{Target: h, Kinds: pointer.Release}); ok {
		t.Errorf("filter mismatch returned event %v", e)
	}
	if e, ok := r.Source().Event(pointer.Filter{Target: new(int), Kinds: pointer.Press}); ok {
		t.Errorf("different tag returned event %v", e)
	}
}

func TestRouterFrameRegistersAndCleansUp(t *testing.T) {
	r := new(Router)
	h := new(int)
	ops := new(op.Ops)

	cl := clip.Rect(image.Rect(0, 0, 100, 100)).Push(ops)
	event.Op(ops, h)
	cl.Pop()

	events(r, -1, pointer.Filter{Target: h, Kinds: pointer.Press})
	r.Frame(ops)

	if _, ok := r.handlers[h]; !ok {
		t.Fatalf("handler not registered after Frame")
	}

	ops.Reset()
	r.Frame(ops)
	r.Frame(ops)
	if _, ok := r.handlers[h]; ok {
		t.Errorf("handler still registered after frames without registration")
	}
}

func TestRouterFrameWithoutEvents(t *testing.T) {
	r := new(Router)

	for range 3 {
		r.Frame(new(op.Ops))
	}
	if _, wake := r.WakeupTime(); wake {
		t.Errorf("idle frames triggered a wakeup")
	}
}

func TestRouterFrameNilOps(t *testing.T) {
	r := new(Router)
	r.Frame(nil)
	r.Frame(nil)
}

func TestSourceFocused(t *testing.T) {
	r := new(Router)
	h1 := new(int)
	h2 := new(int)
	ops := new(op.Ops)

	cl1 := clip.Rect(image.Rect(0, 0, 50, 50)).Push(ops)
	event.Op(ops, h1)
	cl1.Pop()
	cl2 := clip.Rect(image.Rect(50, 0, 100, 50)).Push(ops)
	event.Op(ops, h2)
	cl2.Pop()

	filters := []event.Filter{
		key.FocusFilter{Target: h1},
		key.FocusFilter{Target: h2},
	}
	events(r, -1, filters...)
	r.Frame(ops)

	if r.Source().Focused(h1) || r.Source().Focused(h2) {
		t.Errorf("nothing should be focused initially")
	}

	r.Source().Execute(key.FocusCmd{Tag: h1})
	events(r, -1, filters...)
	r.Frame(ops)
	events(r, -1, filters...)
	if !r.Source().Focused(h1) {
		t.Errorf("h1 should be focused after FocusCmd")
	}
	if r.Source().Focused(h2) {
		t.Errorf("h2 unexpectedly focused")
	}

	r.Source().Execute(key.FocusCmd{Tag: h2})
	events(r, -1, filters...)
	r.Frame(ops)
	events(r, -1, filters...)
	if r.Source().Focused(h1) {
		t.Errorf("h1 still focused after focus moved")
	}
	if !r.Source().Focused(h2) {
		t.Errorf("h2 should be focused after focus transfer")
	}

	r.Source().Execute(key.FocusCmd{Tag: nil})
	events(r, -1, filters...)
	r.Frame(ops)
	events(r, -1, filters...)
	if r.Source().Focused(h1) || r.Source().Focused(h2) {
		t.Errorf("focus should be cleared")
	}
}

func TestFocusedDisabled(t *testing.T) {
	r := new(Router)
	h := new(int)
	ops := new(op.Ops)
	cl := clip.Rect(image.Rect(0, 0, 10, 10)).Push(ops)
	event.Op(ops, h)
	cl.Pop()

	events(r, -1, key.FocusFilter{Target: h})
	r.Frame(ops)
	r.Source().Execute(key.FocusCmd{Tag: h})
	events(r, -1, key.FocusFilter{Target: h})
	r.Frame(ops)
	events(r, -1, key.FocusFilter{Target: h})
	if !r.Source().Focused(h) {
		t.Fatalf("h should be focused")
	}
	if r.Source().Disabled().Focused(h) {
		t.Errorf("disabled source should not report focus")
	}
}

func TestMultipleFiltersSameTag(t *testing.T) {
	r := new(Router)
	h := new(int)
	ops := new(op.Ops)

	cl := clip.Rect(image.Rect(0, 0, 100, 100)).Push(ops)
	event.Op(ops, h)
	cl.Pop()

	events(r, -1,
		pointer.Filter{Target: h, Kinds: pointer.Press},
		pointer.Filter{Target: h, Kinds: pointer.Release},
		key.FocusFilter{Target: h},
	)
	r.Frame(ops)

	r.Queue(
		pointer.Event{Kind: pointer.Press, Position: f32.Pt(5, 5)},
		pointer.Event{Kind: pointer.Release, Position: f32.Pt(5, 5)},
	)

	got := events(r, -1,
		pointer.Filter{Target: h, Kinds: pointer.Press},
		pointer.Filter{Target: h, Kinds: pointer.Release},
		key.FocusFilter{Target: h},
	)

	var pressSeen, releaseSeen bool
	for _, e := range got {
		if pe, ok := e.(pointer.Event); ok {
			switch pe.Kind {
			case pointer.Press:
				pressSeen = true
			case pointer.Release:
				releaseSeen = true
			}
		}
	}
	if !pressSeen || !releaseSeen {
		t.Errorf("merged filters did not deliver both Press and Release; got %v", got)
	}
}

func TestFiltersAcrossMultipleTags(t *testing.T) {
	r := new(Router)
	h1 := new(int)
	h2 := new(int)
	ops := new(op.Ops)

	cl1 := clip.Rect(image.Rect(0, 0, 50, 50)).Push(ops)
	event.Op(ops, h1)
	cl1.Pop()

	cl2 := clip.Rect(image.Rect(50, 0, 100, 50)).Push(ops)
	event.Op(ops, h2)
	cl2.Pop()

	events(r, -1,
		pointer.Filter{Target: h1, Kinds: pointer.Press},
		pointer.Filter{Target: h2, Kinds: pointer.Press},
	)
	r.Frame(ops)

	r.Queue(pointer.Event{Kind: pointer.Press, PointerID: 1, Position: f32.Pt(25, 25)})
	r.Queue(pointer.Event{Kind: pointer.Press, PointerID: 2, Position: f32.Pt(75, 25)})

	got1 := events(r, -1, pointer.Filter{Target: h1, Kinds: pointer.Press})
	got2 := events(r, -1, pointer.Filter{Target: h2, Kinds: pointer.Press})

	if !hasPress(got1) {
		t.Errorf("h1 missed Press: %v", got1)
	}
	if !hasPress(got2) {
		t.Errorf("h2 missed Press: %v", got2)
	}
}

func hasPress(evts []event.Event) bool {
	for _, e := range evts {
		if pe, ok := e.(pointer.Event); ok && pe.Kind == pointer.Press {
			return true
		}
	}
	return false
}

func TestSoftKeyboardCmd(t *testing.T) {
	r := new(Router)

	if got := r.state().state; got != TextInputKeep {
		t.Errorf("initial state %v; want Keep", got)
	}

	r.Source().Execute(key.SoftKeyboardCmd{Show: true})
	if got := r.state().state; got != TextInputOpen {
		t.Errorf("after Show=true got %v; want Open", got)
	}

	r.Source().Execute(key.SoftKeyboardCmd{Show: false})
	if got := r.state().state; got != TextInputClose {
		t.Errorf("after Show=false got %v; want Close", got)
	}
}

func TestClipboardWriteCmd(t *testing.T) {
	r := new(Router)
	const mime = "text/plain"
	r.Source().Execute(clipboard.WriteCmd{Type: mime, Data: io.NopCloser(strings.NewReader("hello"))})

	gotMime, content, ok := r.WriteClipboard()
	if !ok {
		t.Fatal("WriteClipboard returned ok=false")
	}
	if gotMime != mime {
		t.Errorf("mime: got %q want %q", gotMime, mime)
	}
	if string(content) != "hello" {
		t.Errorf("content: got %q want %q", content, "hello")
	}

	if _, _, ok := r.WriteClipboard(); ok {
		t.Errorf("WriteClipboard should be idempotent")
	}
}

func TestClipboardReadCmd(t *testing.T) {
	r := new(Router)
	h := new(int)
	r.Source().Execute(clipboard.ReadCmd{Tag: h})

	if !r.ClipboardRequested() {
		t.Fatal("ClipboardRequested should be true after ReadCmd")
	}
	if r.ClipboardRequested() {
		t.Error("ClipboardRequested should be cleared after observation")
	}
}

func TestWakeupAtStoresEarliestTime(t *testing.T) {
	r := new(Router)
	now := time.Now()
	later := now.Add(time.Second)

	r.Source().Execute(op.InvalidateCmd{At: later})
	r.Frame(new(op.Ops))
	gotT, wake := r.WakeupTime()
	if !wake {
		t.Fatal("wake not set after InvalidateCmd{At: later}")
	}
	if !gotT.Equal(later) {
		t.Errorf("wakeupTime = %v, want %v", gotT, later)
	}

	r.Source().Execute(op.InvalidateCmd{At: now})
	r.Source().Execute(op.InvalidateCmd{At: later.Add(time.Hour)})
	r.Frame(new(op.Ops))
	gotT, wake = r.WakeupTime()
	if !wake {
		t.Fatal("wake not set after multiple InvalidateCmd")
	}
	if !gotT.Equal(now) {
		t.Errorf("wakeupTime = %v, want earliest %v", gotT, now)
	}
}

func TestWakeupClearedAfterRead(t *testing.T) {
	r := new(Router)
	r.Source().Execute(op.InvalidateCmd{At: time.Now().Add(time.Hour)})
	r.Frame(new(op.Ops))
	if _, wake := r.WakeupTime(); !wake {
		t.Fatal("wake should be true once")
	}
	if _, wake := r.WakeupTime(); wake {
		t.Errorf("wake should be cleared after WakeupTime()")
	}
}

func TestQueueMatchingKeyEvent(t *testing.T) {
	r := new(Router)
	r.Event(key.Filter{Name: "Q"})
	r.Frame(new(op.Ops))

	ke := key.Event{Name: "Q"}
	r.Queue(ke)

	got := events(r, -1, key.Filter{Name: "Q"})
	if len(got) != 1 || got[0] != ke {
		t.Errorf("expected 1 event %v; got %v", ke, got)
	}
}

func TestQueueNonMatchingKeyEvent(t *testing.T) {
	r := new(Router)
	r.Event(key.Filter{Name: "A"})
	r.Frame(new(op.Ops))

	r.Queue(key.Event{Name: "B"})

	got := events(r, -1, key.Filter{Name: "A"})
	if len(got) != 0 {
		t.Errorf("filter mismatch: got %v", got)
	}
}

func TestRouterMultipleFramesNoLeak(t *testing.T) {
	r := new(Router)
	h := new(int)
	ops := new(op.Ops)

	cl := clip.Rect(image.Rect(0, 0, 10, 10)).Push(ops)
	event.Op(ops, h)
	cl.Pop()

	events(r, -1, pointer.Filter{Target: h, Kinds: pointer.Press})
	r.Frame(ops)
	r.Queue(pointer.Event{Kind: pointer.Press, Position: f32.Pt(5, 5)})

	events(r, -1, pointer.Filter{Target: h, Kinds: pointer.Press})

	for range 5 {
		r.Frame(ops)
		extras := events(r, -1, pointer.Filter{Target: h, Kinds: pointer.Press})
		for _, e := range extras {
			if pe, ok := e.(pointer.Event); ok && pe.Kind == pointer.Press {
				t.Errorf("frame %v: leaked Press event", e)
			}
		}
	}
}

func TestExecuteIgnoresUnknownCommand(t *testing.T) {
	r := new(Router)
	type unknown struct{ Command }
	r.Source().Execute(unknown{})
	r.Frame(new(op.Ops))
}

func TestEmptyFilterCallReset(t *testing.T) {
	r := new(Router)
	if _, ok := r.Event(); ok {
		t.Errorf("empty Event() returned a result")
	}
	if _, ok := r.Source().Event(); ok {
		t.Errorf("empty Source().Event() returned a result")
	}
}

func TestFocusFilterEmitsFocusFalseReset(t *testing.T) {
	r := new(Router)
	h := new(int)
	got := events(r, -1, key.FocusFilter{Target: h})

	found := false
	for _, e := range got {
		if fe, ok := e.(key.FocusEvent); ok && !fe.Focus {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an initial FocusEvent{Focus: false} reset; got %v", got)
	}
}

func TestRevealFocusNoFocusNoOp(t *testing.T) {
	r := new(Router)
	r.RevealFocus(image.Rect(0, 0, 10, 10))
	r.ScrollFocus(image.Pt(5, 5))
	r.ClickFocus()
}

func TestFrameClearsPointerEventsAcrossFrames(t *testing.T) {
	r := new(Router)
	h := new(int)
	ops := new(op.Ops)

	cl := clip.Rect(image.Rect(0, 0, 100, 100)).Push(ops)
	event.Op(ops, h)
	cl.Pop()

	events(r, -1, pointer.Filter{Target: h, Kinds: pointer.Press | pointer.Release})
	r.Frame(ops)
	r.Queue(pointer.Event{Kind: pointer.Press, Position: f32.Pt(5, 5)})

	first := events(r, -1, pointer.Filter{Target: h, Kinds: pointer.Press | pointer.Release})

	pressCount := 0
	for _, e := range first {
		if pe, ok := e.(pointer.Event); ok && pe.Kind == pointer.Press {
			pressCount++
		}
	}
	if pressCount != 1 {
		t.Errorf("first Frame: got %d press events, want 1", pressCount)
	}

	r.Frame(ops)
	second := events(r, -1, pointer.Filter{Target: h, Kinds: pointer.Press | pointer.Release})
	for _, e := range second {
		if pe, ok := e.(pointer.Event); ok && pe.Kind == pointer.Press {
			t.Errorf("second Frame: leaked Press event %v", pe)
		}
	}
}

func TestTextInputStateConsumed(t *testing.T) {
	r := new(Router)
	r.Source().Execute(key.SoftKeyboardCmd{Show: true})
	if got := r.TextInputState(); got != TextInputOpen {
		t.Errorf("first read: got %v, want Open", got)
	}
	if got := r.TextInputState(); got != TextInputKeep {
		t.Errorf("second read: got %v, want Keep (state should be consumed)", got)
	}
}

func TestQueueDataEvent(t *testing.T) {
	r := new(Router)
	h := new(int)

	r.Source().Execute(clipboard.ReadCmd{Tag: h})

	de := transfer.DataEvent{
		Type: "text/plain",
		Open: func() io.ReadCloser {
			return io.NopCloser(strings.NewReader("data"))
		},
	}
	r.Queue(de)

	got := events(r, -1, transfer.TargetFilter{Target: h, Type: "text/plain"})
	if len(got) != 1 {
		t.Fatalf("expected 1 DataEvent, got %d (%v)", len(got), got)
	}
	if _, ok := got[0].(transfer.DataEvent); !ok {
		t.Errorf("got %T, want transfer.DataEvent", got[0])
	}
}

func TestCursorReportsState(t *testing.T) {
	r := new(Router)

	c := r.Cursor()
	if c == cursorUnset {
		t.Errorf("Cursor() leaked internal cursorUnset sentinel")
	}
}

// TestRevealFocusUnregisteredTag verifies RevealFocus does not panic when the
// focused tag was never registered as a handler. This used to dereference
// q.handlers[focus] without an existence check.
func TestRevealFocusUnregisteredTag(t *testing.T) {
	r := new(Router)
	h := new(int)

	r.Source().Execute(key.FocusCmd{Tag: h})

	defer func() {
		if rec := recover(); rec != nil {
			t.Errorf("RevealFocus panicked with unregistered focus tag: %v", rec)
		}
	}()
	r.RevealFocus(image.Rect(0, 0, 10, 10))
	r.ScrollFocus(image.Pt(1, 1))
	r.ClickFocus()
}

// TestEditorStateUnregisteredFocus verifies EditorState does not panic when
// the focused tag is not present in q.handlers. The previous code accessed
// handlers[f].key.trans without an existence check, dereferencing a nil
// *handler when the focus had been recorded but the handler garbage-collected.
func TestEditorStateUnregisteredFocus(t *testing.T) {
	r := new(Router)
	h := new(int)

	r.Source().Execute(key.FocusCmd{Tag: h})

	defer func() {
		if rec := recover(); rec != nil {
			t.Errorf("EditorState panicked with unregistered focus tag: %v", rec)
		}
	}()
	_ = r.EditorState()
}
