package app

import (
	"image"
	"sync"
	"testing"
	"time"

	"github.com/nanorele/gio/io/event"
	"github.com/nanorele/gio/io/key"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/io/system"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/unit"
)

// stubDriver is a minimal driver implementation that records SetCursor calls
// and otherwise no-ops. Methods that should not be invoked from the tests in
// this file panic so misuse is loud.
type stubDriver struct {
	mu          sync.Mutex
	setCursors  []pointer.Cursor
	invalidated int
	performed   []system.Action
}

func (d *stubDriver) SetCursor(c pointer.Cursor) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.setCursors = append(d.setCursors, c)
}

func (d *stubDriver) cursorCalls() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.setCursors)
}

func (d *stubDriver) lastCursor() pointer.Cursor {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.setCursors) == 0 {
		return pointer.CursorDefault
	}
	return d.setCursors[len(d.setCursors)-1]
}

func (d *stubDriver) Event() event.Event                      { panic("stubDriver.Event not used") }
func (d *stubDriver) Invalidate()                             { d.invalidated++ }
func (d *stubDriver) SetAnimating(bool)                       {}
func (d *stubDriver) ShowTextInput(bool)                      {}
func (d *stubDriver) SetInputHint(key.InputHint)              {}
func (d *stubDriver) NewContext() (context, error)            { return nil, nil }
func (d *stubDriver) ReadClipboard()                          {}
func (d *stubDriver) WriteClipboard(string, []byte)           {}
func (d *stubDriver) Configure([]Option)                      {}
func (d *stubDriver) Perform(a system.Action)                 { d.performed = append(d.performed, a) }
func (d *stubDriver) EditorStateChanged(_, _ editorState)     {}
func (d *stubDriver) Run(f func())                            { f() }
func (d *stubDriver) Frame(*op.Ops)                           {}
func (d *stubDriver) ProcessEvent(event.Event)                {}

// newTestWindow builds a Window value bare enough to exercise the pure-logic
// methods without going through init() / driver setup.
func newTestWindow() *Window {
	return &Window{}
}

func TestDecoHeightOpt(t *testing.T) {
	c := applyOpts(metric1(), decoHeightOpt(unit.Dp(42)))
	if c.decoHeight != unit.Dp(42) {
		t.Errorf("decoHeight = %v, want 42", c.decoHeight)
	}
}

func TestDecoHeightOptOverwrites(t *testing.T) {
	// Last write wins: matches the same semantics as the public Options.
	c := applyOpts(metric1(),
		decoHeightOpt(unit.Dp(10)),
		decoHeightOpt(unit.Dp(20)),
	)
	if c.decoHeight != unit.Dp(20) {
		t.Errorf("decoHeight = %v, want 20", c.decoHeight)
	}
}

func TestDecoHeightOptZero(t *testing.T) {
	c := applyOpts(metric1(), decoHeightOpt(0))
	if c.decoHeight != 0 {
		t.Errorf("decoHeight = %v, want 0", c.decoHeight)
	}
}

func TestDecoHeightOptNegativeIsAccepted(t *testing.T) {
	// Documents the absence of validation: decoHeightOpt accepts negative
	// values verbatim. effectiveConfig() subtracts currentHeight from
	// Size.Y, so a negative value would inflate the reported size — this
	// is a latent bug, not a fix in this wave.
	c := applyOpts(metric1(), decoHeightOpt(unit.Dp(-5)))
	if c.decoHeight != unit.Dp(-5) {
		t.Errorf("decoHeight = %v, want -5 (no clamp expected)", c.decoHeight)
	}
}

func TestSetDriverNilPanics(t *testing.T) {
	w := newTestWindow()
	cb := &callbacks{w: w}
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on SetDriver(nil), got none")
		}
	}()
	cb.SetDriver(nil)
}

func TestSetDriverStoresDriver(t *testing.T) {
	w := newTestWindow()
	cb := &callbacks{w: w}
	d := &stubDriver{}
	cb.SetDriver(d)
	if w.driver != d {
		t.Errorf("driver not stored on window")
	}
}

func TestMarkCursorDirtySetsFlag(t *testing.T) {
	w := newTestWindow()
	cb := &callbacks{w: w}
	if w.cursorDirty {
		t.Fatalf("cursorDirty should start false")
	}
	cb.MarkCursorDirty()
	if !w.cursorDirty {
		t.Errorf("MarkCursorDirty did not set cursorDirty=true")
	}
}

func TestMarkCursorDirtyIsIdempotent(t *testing.T) {
	w := newTestWindow()
	cb := &callbacks{w: w}
	cb.MarkCursorDirty()
	cb.MarkCursorDirty()
	cb.MarkCursorDirty()
	if !w.cursorDirty {
		t.Errorf("cursorDirty should remain true after repeated calls")
	}
}

func TestUpdateCursorSameCursorNotDirtySkipsDriver(t *testing.T) {
	w := newTestWindow()
	d := &stubDriver{}
	(&callbacks{w: w}).SetDriver(d)
	// w.cursor and queue.Cursor() are both pointer.CursorDefault here.
	w.cursor = pointer.CursorDefault
	w.cursorDirty = false
	w.updateCursor()
	if d.cursorCalls() != 0 {
		t.Errorf("SetCursor should NOT be called when cursor matches and not dirty; got %d calls", d.cursorCalls())
	}
	if w.cursorDirty {
		t.Errorf("cursorDirty should still be false after no-op updateCursor")
	}
}

func TestUpdateCursorSameCursorDirtyCallsDriverAndClears(t *testing.T) {
	w := newTestWindow()
	d := &stubDriver{}
	(&callbacks{w: w}).SetDriver(d)
	w.cursor = pointer.CursorDefault
	w.cursorDirty = true
	w.updateCursor()
	if d.cursorCalls() != 1 {
		t.Errorf("SetCursor should be called once when dirty; got %d", d.cursorCalls())
	}
	if got := d.lastCursor(); got != pointer.CursorDefault {
		t.Errorf("SetCursor argument = %v, want %v", got, pointer.CursorDefault)
	}
	if w.cursorDirty {
		t.Errorf("cursorDirty should be cleared after updateCursor")
	}
}

func TestUpdateCursorDifferentCursorClearsDirty(t *testing.T) {
	w := newTestWindow()
	d := &stubDriver{}
	(&callbacks{w: w}).SetDriver(d)
	// Force a mismatch: cached cursor is Pointer; the queue (untouched)
	// reports CursorDefault. updateCursor must call the driver and update
	// the cached value.
	w.cursor = pointer.CursorPointer
	w.cursorDirty = true
	w.updateCursor()
	if d.cursorCalls() != 1 {
		t.Errorf("expected 1 SetCursor call, got %d", d.cursorCalls())
	}
	if got := d.lastCursor(); got != pointer.CursorDefault {
		t.Errorf("SetCursor argument = %v, want %v", got, pointer.CursorDefault)
	}
	if w.cursor != pointer.CursorDefault {
		t.Errorf("w.cursor not updated to %v, got %v", pointer.CursorDefault, w.cursor)
	}
	if w.cursorDirty {
		t.Errorf("cursorDirty should be cleared")
	}
}

func TestUpdateCursorDifferentCursorNotDirty(t *testing.T) {
	w := newTestWindow()
	d := &stubDriver{}
	(&callbacks{w: w}).SetDriver(d)
	// Mismatch alone (without dirty flag) must still drive an update.
	w.cursor = pointer.CursorPointer
	w.cursorDirty = false
	w.updateCursor()
	if d.cursorCalls() != 1 {
		t.Errorf("expected 1 SetCursor call on cursor change, got %d", d.cursorCalls())
	}
	if w.cursor != pointer.CursorDefault {
		t.Errorf("w.cursor = %v, want %v", w.cursor, pointer.CursorDefault)
	}
}

func TestUpdateCursorRepeatedCallsDoNotResend(t *testing.T) {
	w := newTestWindow()
	d := &stubDriver{}
	(&callbacks{w: w}).SetDriver(d)
	w.cursor = pointer.CursorPointer
	w.cursorDirty = false
	w.updateCursor() // pushes CursorDefault, clears mismatch
	w.updateCursor() // no-op
	w.updateCursor() // no-op
	if d.cursorCalls() != 1 {
		t.Errorf("expected 1 SetCursor call total, got %d", d.cursorCalls())
	}
}

func TestUpdateCursorMarkDirtyForcesResend(t *testing.T) {
	w := newTestWindow()
	d := &stubDriver{}
	cb := &callbacks{w: w}
	cb.SetDriver(d)
	w.cursor = pointer.CursorPointer
	w.updateCursor() // first push: 1 call, cursor cached at CursorDefault
	if d.cursorCalls() != 1 {
		t.Fatalf("setup: expected 1 call, got %d", d.cursorCalls())
	}
	cb.MarkCursorDirty()
	w.updateCursor() // dirty => must re-push even though value didn't change
	if d.cursorCalls() != 2 {
		t.Errorf("expected 2 SetCursor calls after MarkCursorDirty, got %d", d.cursorCalls())
	}
	if w.cursorDirty {
		t.Errorf("cursorDirty should be cleared")
	}
}

func TestSetNextFrameFirstCallSetsTime(t *testing.T) {
	w := newTestWindow()
	t0 := time.Now()
	w.setNextFrame(t0)
	if !w.hasNextFrame {
		t.Errorf("hasNextFrame should be true after first call")
	}
	if !w.nextFrame.Equal(t0) {
		t.Errorf("nextFrame = %v, want %v", w.nextFrame, t0)
	}
}

func TestSetNextFrameKeepsEarliest(t *testing.T) {
	w := newTestWindow()
	later := time.Now().Add(1 * time.Hour)
	earlier := later.Add(-30 * time.Minute)
	w.setNextFrame(later)
	w.setNextFrame(earlier)
	if !w.nextFrame.Equal(earlier) {
		t.Errorf("nextFrame = %v, want earliest %v", w.nextFrame, earlier)
	}
}

func TestSetNextFrameIgnoresLater(t *testing.T) {
	w := newTestWindow()
	earlier := time.Now()
	later := earlier.Add(1 * time.Hour)
	w.setNextFrame(earlier)
	w.setNextFrame(later)
	if !w.nextFrame.Equal(earlier) {
		t.Errorf("nextFrame = %v, want %v (later should not overwrite earlier)", w.nextFrame, earlier)
	}
}

func TestEffectiveConfigSubtractsDecoHeight(t *testing.T) {
	w := newTestWindow()
	w.decorations.Config = Config{Size: image.Pt(800, 600)}
	w.decorations.currentHeight = 30
	w.decorations.enabled = true
	got := w.effectiveConfig()
	if got.Size.Y != 570 {
		t.Errorf("effectiveConfig().Size.Y = %d, want 570 (600 - 30)", got.Size.Y)
	}
	if got.Size.X != 800 {
		t.Errorf("effectiveConfig().Size.X = %d, want 800", got.Size.X)
	}
	if !got.Decorated {
		t.Errorf("effectiveConfig().Decorated should be true when fallback decorations are enabled")
	}
}

func TestEffectiveConfigDecoratedFlagOR(t *testing.T) {
	// Decorated is enabled || cnf.Decorated.
	w := newTestWindow()
	w.decorations.Config = Config{Decorated: true, Size: image.Pt(100, 100)}
	w.decorations.enabled = false
	if !w.effectiveConfig().Decorated {
		t.Errorf("Decorated should be true when cnf.Decorated=true")
	}
	w.decorations.Config.Decorated = false
	w.decorations.enabled = true
	if !w.effectiveConfig().Decorated {
		t.Errorf("Decorated should be true when w.decorations.enabled=true")
	}
	w.decorations.enabled = false
	if w.effectiveConfig().Decorated {
		t.Errorf("Decorated should be false when both sources are false")
	}
}

func TestFallbackDecorateRequiresEnabled(t *testing.T) {
	w := newTestWindow()
	w.decorations.enabled = false
	w.decorations.Config = Config{Decorated: false, Mode: Windowed}
	if w.fallbackDecorate() {
		t.Errorf("fallbackDecorate should be false when decorations.enabled=false")
	}
}

func TestFallbackDecorateRequiresClientSide(t *testing.T) {
	w := newTestWindow()
	w.decorations.enabled = true
	// Config.Decorated = true means the OS draws decorations -> no fallback.
	w.decorations.Config = Config{Decorated: true, Mode: Windowed}
	if w.fallbackDecorate() {
		t.Errorf("fallbackDecorate should be false when OS decorates the window")
	}
}

func TestFallbackDecorateSkippedInFullscreen(t *testing.T) {
	w := newTestWindow()
	w.decorations.enabled = true
	w.decorations.Config = Config{Decorated: false, Mode: Fullscreen}
	if w.fallbackDecorate() {
		t.Errorf("fallbackDecorate should be false in Fullscreen mode")
	}
}

func TestFallbackDecorateSkippedWithCustomRenderer(t *testing.T) {
	w := newTestWindow()
	w.decorations.enabled = true
	w.decorations.Config = Config{Decorated: false, Mode: Windowed}
	w.nocontext = true
	if w.fallbackDecorate() {
		t.Errorf("fallbackDecorate should be false when nocontext=true (custom renderer)")
	}
}

func TestFallbackDecorateActiveCase(t *testing.T) {
	w := newTestWindow()
	w.decorations.enabled = true
	w.decorations.Config = Config{Decorated: false, Mode: Windowed}
	if !w.fallbackDecorate() {
		t.Errorf("fallbackDecorate should be true when client-side decorations are needed")
	}
}

func TestSplitActionsMode(t *testing.T) {
	cases := []struct {
		in   system.Action
		mode WindowMode
	}{
		{system.ActionMinimize, Minimized},
		{system.ActionMaximize, Maximized},
		{system.ActionUnmaximize, Windowed},
		{system.ActionFullscreen, Fullscreen},
	}
	for _, tc := range cases {
		opts, leftover := splitActions(tc.in)
		if len(opts) != 1 {
			t.Errorf("splitActions(%v): got %d opts, want 1", tc.in, len(opts))
			continue
		}
		if leftover != 0 {
			t.Errorf("splitActions(%v): leftover = %v, want 0", tc.in, leftover)
		}
		var c Config
		opts[0](metric1(), &c)
		if c.Mode != tc.mode {
			t.Errorf("splitActions(%v) opt set Mode=%v, want %v", tc.in, c.Mode, tc.mode)
		}
	}
}

func TestSplitActionsPassesThroughUnknown(t *testing.T) {
	in := system.ActionClose | system.ActionMove
	opts, leftover := splitActions(in)
	if len(opts) != 0 {
		t.Errorf("expected no opts for non-mode actions, got %d", len(opts))
	}
	if leftover != in {
		t.Errorf("leftover = %v, want %v", leftover, in)
	}
}

func TestSplitActionsMixed(t *testing.T) {
	in := system.ActionMaximize | system.ActionClose
	opts, leftover := splitActions(in)
	if len(opts) != 1 {
		t.Errorf("expected 1 opt for mixed actions, got %d", len(opts))
	}
	if leftover != system.ActionClose {
		t.Errorf("leftover = %v, want ActionClose", leftover)
	}
}

func TestWalkActionsAll(t *testing.T) {
	in := system.ActionMinimize | system.ActionMaximize | system.ActionClose
	var seen []system.Action
	walkActions(in, func(a system.Action) {
		seen = append(seen, a)
	})
	want := map[system.Action]bool{
		system.ActionMinimize: true,
		system.ActionMaximize: true,
		system.ActionClose:    true,
	}
	if len(seen) != len(want) {
		t.Errorf("walkActions visited %d actions, want %d (%v)", len(seen), len(want), seen)
	}
	for _, a := range seen {
		if !want[a] {
			t.Errorf("walkActions visited unexpected action %v", a)
		}
	}
}

func TestWalkActionsZero(t *testing.T) {
	called := 0
	walkActions(0, func(system.Action) { called++ })
	if called != 0 {
		t.Errorf("walkActions(0) called callback %d times, want 0", called)
	}
}

func TestWindowOptionDeferredBeforeDriver(t *testing.T) {
	w := newTestWindow()
	// With no driver attached, Option must stash the opts in initialOpts.
	w.Option(Title("queued"))
	if len(w.initialOpts) != 1 {
		t.Errorf("initialOpts len = %d, want 1", len(w.initialOpts))
	}
	w.Option(Size(unit.Dp(100), unit.Dp(50)), TopMost(true))
	if len(w.initialOpts) != 3 {
		t.Errorf("initialOpts len = %d, want 3 after second call", len(w.initialOpts))
	}
}

func TestWindowOptionEmptyIsNoOp(t *testing.T) {
	w := newTestWindow()
	w.Option()
	if len(w.initialOpts) != 0 {
		t.Errorf("Option() with no args should not allocate; got %d", len(w.initialOpts))
	}
}

func TestPerformDeferredBeforeDriver(t *testing.T) {
	w := newTestWindow()
	// Non-Mode actions go to initialActions when no driver is set.
	w.Perform(system.ActionClose)
	if len(w.initialActions) != 1 {
		t.Errorf("initialActions len = %d, want 1", len(w.initialActions))
	}
	if w.initialActions[0] != system.ActionClose {
		t.Errorf("initialActions[0] = %v, want ActionClose", w.initialActions[0])
	}
}

func TestPerformModeOnlyDoesNotQueueAction(t *testing.T) {
	w := newTestWindow()
	w.Perform(system.ActionMaximize)
	if len(w.initialActions) != 0 {
		t.Errorf("ActionMaximize is consumed by splitActions; expected no initialActions, got %v", w.initialActions)
	}
	if len(w.initialOpts) != 1 {
		t.Errorf("expected the Maximized opt to be queued, got %d initialOpts", len(w.initialOpts))
	}
}

func TestRunWithNilDriverRunsInline(t *testing.T) {
	w := newTestWindow()
	called := false
	w.Run(func() { called = true })
	if !called {
		t.Errorf("Run should invoke f synchronously when driver is nil")
	}
}

// DestroyEvent is defined in system.go. A minimal smoke test that the
// marker interface is satisfied and the Err field round-trips.
func TestDestroyEventImplementsEvent(t *testing.T) {
	var _ event.Event = DestroyEvent{}
}

func TestDestroyEventCarriesError(t *testing.T) {
	want := errFakeForTest{}
	e := DestroyEvent{Err: want}
	if e.Err != want {
		t.Errorf("DestroyEvent.Err round-trip failed: got %v", e.Err)
	}
}

type errFakeForTest struct{}

func (errFakeForTest) Error() string { return "fake" }

func TestCallbacksEditorState(t *testing.T) {
	w := newTestWindow()
	cb := &callbacks{w: w}
	w.imeState.compose = key.Range{Start: 1, End: 5}
	got := cb.EditorState()
	if got.compose != (key.Range{Start: 1, End: 5}) {
		t.Errorf("EditorState().compose = %v, want {1,5}", got.compose)
	}
}

func TestCallbacksSetComposingRegion(t *testing.T) {
	w := newTestWindow()
	cb := &callbacks{w: w}
	r := key.Range{Start: 7, End: 11}
	cb.SetComposingRegion(r)
	if w.imeState.compose != r {
		t.Errorf("imeState.compose = %v, want %v", w.imeState.compose, r)
	}
}

func TestProcessEventWakeupSetsCoalesced(t *testing.T) {
	w := newTestWindow()
	if w.coalesced.wakeup {
		t.Fatalf("coalesced.wakeup should start false")
	}
	w.processEvent(wakeupEvent{})
	if !w.coalesced.wakeup {
		t.Errorf("processEvent(wakeupEvent) should set coalesced.wakeup=true")
	}
}

func TestNextEventEmptyReturnsFalse(t *testing.T) {
	w := newTestWindow()
	e, ok := w.nextEvent()
	if ok {
		t.Errorf("nextEvent on fresh window should return ok=false, got %v", e)
	}
}

func TestNextEventWakeupCoalesces(t *testing.T) {
	w := newTestWindow()
	w.processEvent(wakeupEvent{})
	w.processEvent(wakeupEvent{})
	w.processEvent(wakeupEvent{})
	e, ok := w.nextEvent()
	if !ok {
		t.Fatalf("expected wakeupEvent to be returned")
	}
	if _, isWakeup := e.(wakeupEvent); !isWakeup {
		t.Errorf("expected wakeupEvent, got %T", e)
	}
	// Wakeup is coalesced: a follow-up call without re-injection returns nothing.
	_, ok2 := w.nextEvent()
	if ok2 {
		t.Errorf("second nextEvent should return ok=false (wakeup was consumed)")
	}
}

func TestNextEventSetsMayInvalidate(t *testing.T) {
	w := newTestWindow()
	d := &stubDriver{}
	(&callbacks{w: w}).SetDriver(d)
	if w.mayInvalidate {
		t.Fatalf("mayInvalidate should start false")
	}
	_, ok := w.nextEvent()
	if ok {
		t.Fatalf("expected no event")
	}
	if !w.mayInvalidate {
		t.Errorf("nextEvent should set mayInvalidate=true when driver is non-nil and queue empty")
	}
}

func TestNextEventNoDriverDoesNotSetMayInvalidate(t *testing.T) {
	w := newTestWindow()
	_, ok := w.nextEvent()
	if ok {
		t.Fatalf("expected no event")
	}
	if w.mayInvalidate {
		t.Errorf("mayInvalidate should remain false when driver is nil")
	}
}

func TestWindowInvalidateNoDriverDoesNotPanic(t *testing.T) {
	w := newTestWindow()
	// mayInvalidate is false, so the nil driver should never be touched.
	w.Invalidate()
}

func TestWindowInvalidateGatedByMayInvalidate(t *testing.T) {
	w := newTestWindow()
	d := &stubDriver{}
	(&callbacks{w: w}).SetDriver(d)
	// Without mayInvalidate set, Invalidate should be a no-op.
	w.Invalidate()
	if d.invalidated != 0 {
		t.Errorf("driver.Invalidate called %d times before mayInvalidate=true; want 0", d.invalidated)
	}
	w.mayInvalidate = true
	w.Invalidate()
	if d.invalidated != 1 {
		t.Errorf("driver.Invalidate called %d times after mayInvalidate=true; want 1", d.invalidated)
	}
	if w.mayInvalidate {
		t.Errorf("Invalidate should clear mayInvalidate")
	}
	// Subsequent call should now be gated again.
	w.Invalidate()
	if d.invalidated != 1 {
		t.Errorf("driver.Invalidate called %d times after second call; want still 1", d.invalidated)
	}
}

func TestCallbacksProcessEventDispatches(t *testing.T) {
	w := newTestWindow()
	cb := &callbacks{w: w}
	cb.ProcessEvent(wakeupEvent{})
	if !w.coalesced.wakeup {
		t.Errorf("ProcessEvent should dispatch to processEvent")
	}
}

func TestCallbacksInvalidateMarksNextFrame(t *testing.T) {
	w := newTestWindow()
	cb := &callbacks{w: w}
	cb.Invalidate()
	if !w.hasNextFrame {
		t.Errorf("callbacks.Invalidate should set hasNextFrame=true")
	}
	if !w.coalesced.wakeup {
		t.Errorf("callbacks.Invalidate should produce a wakeup")
	}
}

func TestCursorDirtyStartsFalseOnNewWindow(t *testing.T) {
	w := newTestWindow()
	if w.cursorDirty {
		t.Errorf("cursorDirty should be false on a freshly constructed Window")
	}
}

func TestSetNextFrameZeroTimeAlwaysWins(t *testing.T) {
	// time.Time{} is the zero value; setNextFrame(time.Time{}) is the
	// idiom Invalidate() uses to schedule "as soon as possible".
	w := newTestWindow()
	future := time.Now().Add(time.Hour)
	w.setNextFrame(future)
	w.setNextFrame(time.Time{})
	if !w.nextFrame.IsZero() {
		t.Errorf("setNextFrame(zero) should overwrite a later time; got %v", w.nextFrame)
	}
}

func TestUpdateCursorClearsDirtyEvenIfDriverNoop(t *testing.T) {
	// Regression: the dirty flag must be cleared regardless of which branch
	// of the equality check triggered the SetCursor call.
	w := newTestWindow()
	d := &stubDriver{}
	(&callbacks{w: w}).SetDriver(d)
	w.cursor = pointer.CursorPointer
	w.cursorDirty = true
	w.updateCursor()
	if w.cursorDirty {
		t.Errorf("cursorDirty should be cleared after updateCursor (different cursor + dirty)")
	}
}

// Compile-time assertion that the stubDriver satisfies the driver interface.
// If SetCursor or any other method drifts, this fails to build.
var _ driver = (*stubDriver)(nil)

// Compile-time check that DestroyEvent satisfies event.Event from system.go.
var _ event.Event = DestroyEvent{Err: nil}
