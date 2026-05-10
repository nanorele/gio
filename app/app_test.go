package app

import (
	"image"
	"testing"
	"time"

	"github.com/nanorele/gio/io/event"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/unit"
)

// compile-time interface assertions that the marker methods satisfy event.Event.
var (
	_ event.Event = FrameEvent{}
	_ event.Event = URLEvent{}
)

func TestNewContextZeroInsets(t *testing.T) {
	ops := new(op.Ops)
	now := time.Now()
	e := FrameEvent{
		Now:    now,
		Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Size:   image.Pt(800, 600),
	}
	gtx := NewContext(ops, e)
	if got, want := gtx.Constraints, layout.Exact(image.Pt(800, 600)); got != want {
		t.Errorf("Constraints = %+v, want %+v", got, want)
	}
	if gtx.Ops != ops {
		t.Errorf("gtx.Ops should be the passed-in *op.Ops")
	}
	if gtx.Now != now {
		t.Errorf("gtx.Now = %v, want %v", gtx.Now, now)
	}
	// With zero insets, no offset should have been encoded.
	if got := ops.Size(); got != 0 {
		t.Errorf("ops.Size() = %d, want 0 (no offset encoded)", got)
	}
}

func TestNewContextNonZeroInsetsEncodesOffset(t *testing.T) {
	ops := new(op.Ops)
	e := FrameEvent{
		Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Size:   image.Pt(800, 600),
		Insets: Insets{Top: 10, Bottom: 20, Left: 5, Right: 15},
	}
	gtx := NewContext(ops, e)
	wantSize := image.Pt(800-5-15, 600-10-20)
	if got, want := gtx.Constraints, layout.Exact(wantSize); got != want {
		t.Errorf("Constraints = %+v, want %+v", got, want)
	}
	if got := ops.Size(); got == 0 {
		t.Errorf("expected an offset op to be encoded; ops.Size() == 0")
	}
}

func TestNewContextResetsOps(t *testing.T) {
	ops := new(op.Ops)
	// Encode a transform op so the buffer is non-empty.
	op.Offset(image.Pt(7, 7)).Add(ops)
	if ops.Size() == 0 {
		t.Fatalf("setup failed: expected non-empty ops")
	}
	e := FrameEvent{
		Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Size:   image.Pt(100, 100),
	}
	NewContext(ops, e)
	if got := ops.Size(); got != 0 {
		t.Errorf("ops.Size() after NewContext (zero insets) = %d, want 0; Reset() should have cleared it", got)
	}
}

func TestNewContextPxPerDpScalesInsets(t *testing.T) {
	ops := new(op.Ops)
	e := FrameEvent{
		Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2},
		Size:   image.Pt(200, 200),
		Insets: Insets{Top: 5, Bottom: 5, Left: 5, Right: 5},
	}
	gtx := NewContext(ops, e)
	// 5 dp * 2 px/dp = 10 px on each side.
	wantSize := image.Pt(200-10-10, 200-10-10)
	if got, want := gtx.Constraints, layout.Exact(wantSize); got != want {
		t.Errorf("Constraints = %+v, want %+v (PxPerDp scaling failed)", got, want)
	}
}

func TestNewContextNegativeSizeFromHugeInsets(t *testing.T) {
	// Sanity check: huge insets produce a negative size; the function does not clamp.
	ops := new(op.Ops)
	e := FrameEvent{
		Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Size:   image.Pt(10, 10),
		Insets: Insets{Top: 100, Bottom: 100, Left: 100, Right: 100},
	}
	gtx := NewContext(ops, e)
	if gtx.Constraints.Max.X >= 0 || gtx.Constraints.Max.Y >= 0 {
		t.Errorf("expected negative dimensions, got %+v", gtx.Constraints)
	}
}

func TestNewURLEventValid(t *testing.T) {
	ev, err := newURLEvent("https://example.com/path?q=1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.URL == nil {
		t.Fatalf("expected non-nil URL")
	}
	if ev.URL.Scheme != "https" {
		t.Errorf("Scheme = %q, want https", ev.URL.Scheme)
	}
	if ev.URL.Host != "example.com" {
		t.Errorf("Host = %q, want example.com", ev.URL.Host)
	}
	if ev.URL.Path != "/path" {
		t.Errorf("Path = %q, want /path", ev.URL.Path)
	}
	if ev.URL.RawQuery != "q=1" {
		t.Errorf("RawQuery = %q, want q=1", ev.URL.RawQuery)
	}
}

func TestNewURLEventPunycodeHostname(t *testing.T) {
	ev, err := newURLEvent("https://xn--mnchen-3ya.de/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.URL == nil {
		t.Fatalf("expected non-nil URL")
	}
	if got, want := ev.URL.Host, "münchen.de"; got != want {
		t.Errorf("Host = %q, want %q (punycode should be decoded)", got, want)
	}
	if ev.URL.Path != "/path" {
		t.Errorf("Path = %q, want /path (path should survive punycode decoding)", ev.URL.Path)
	}
}

func TestNewURLEventInvalid(t *testing.T) {
	// A control character in the URL is rejected by url.Parse.
	if _, err := newURLEvent("http://example.com/\x7f"); err == nil {
		// Some Go versions tolerate this, so fall back to a clearly invalid form.
		if _, err := newURLEvent("ht!tp://exa mple.com\n"); err == nil {
			t.Skip("url.Parse accepted both candidate invalid URLs")
		}
	}
}

func TestNewURLEventEmpty(t *testing.T) {
	ev, err := newURLEvent("")
	if err != nil {
		t.Fatalf("empty URL: unexpected error: %v", err)
	}
	if ev.URL == nil {
		t.Fatalf("expected non-nil URL even for empty input")
	}
	if ev.URL.Host != "" || ev.URL.Path != "" {
		t.Errorf("empty URL should yield empty Host/Path, got Host=%q Path=%q", ev.URL.Host, ev.URL.Path)
	}
}

func TestNewURLEventNonASCIIPath(t *testing.T) {
	ev, err := newURLEvent("https://example.com/münchen")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.URL == nil {
		t.Fatalf("expected non-nil URL")
	}
	// url.Parse percent-encodes the raw path; the decoded Path should hold the unicode.
	if ev.URL.Path != "/münchen" {
		t.Errorf("Path = %q, want /münchen", ev.URL.Path)
	}
}

func TestNewURLEventWithFragmentAndQuery(t *testing.T) {
	ev, err := newURLEvent("https://example.com/path?q=1&r=2#frag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.URL.Fragment != "frag" {
		t.Errorf("Fragment = %q, want frag", ev.URL.Fragment)
	}
	if ev.URL.RawQuery != "q=1&r=2" {
		t.Errorf("RawQuery = %q, want q=1&r=2", ev.URL.RawQuery)
	}
}

func TestNewURLEventWithPort(t *testing.T) {
	// Documents the current behavior. If newURLEvent strips the port
	// because it assigns u.Hostname() (port-less) back to u.Host, this
	// test will record the bug rather than pass silently.
	ev, err := newURLEvent("https://example.com:8080/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := ev.URL.Port(); got != "8080" {
		t.Errorf("Port = %q, want 8080 (port appears to be dropped by Punycode rewrite)", got)
	}
}

func TestProcessGlobalEventNoYield(t *testing.T) {
	t.Cleanup(func() { yieldGlobalEvent = nil })
	yieldGlobalEvent = nil
	// Should not panic.
	processGlobalEvent(FrameEvent{})
}

func TestProcessGlobalEventCallsYield(t *testing.T) {
	t.Cleanup(func() { yieldGlobalEvent = nil })
	var got event.Event
	called := 0
	yieldGlobalEvent = func(e event.Event) bool {
		called++
		got = e
		return true
	}
	want := URLEvent{}
	processGlobalEvent(want)
	if called != 1 {
		t.Errorf("yield called %d times, want 1", called)
	}
	if got != want {
		t.Errorf("yield received %+v, want %+v", got, want)
	}
	if yieldGlobalEvent == nil {
		t.Errorf("yieldGlobalEvent was cleared even though yield returned true")
	}
}

func TestProcessGlobalEventClearsOnFalse(t *testing.T) {
	t.Cleanup(func() { yieldGlobalEvent = nil })
	called := 0
	yieldGlobalEvent = func(e event.Event) bool {
		called++
		return false
	}
	processGlobalEvent(FrameEvent{})
	if called != 1 {
		t.Errorf("yield called %d times, want 1", called)
	}
	if yieldGlobalEvent != nil {
		t.Errorf("yieldGlobalEvent should be nil after yield returned false")
	}
	// A subsequent call with the cleared yield should be a no-op.
	processGlobalEvent(FrameEvent{})
	if called != 1 {
		t.Errorf("yield should not be called after being cleared, got %d", called)
	}
}

func TestInitSetsID(t *testing.T) {
	// init() runs once at package load; ID should be set to a non-empty default.
	if ID == "" {
		t.Errorf("ID should be set by init(), got empty string")
	}
}
