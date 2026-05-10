package app

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/nanorele/gio/unit"
)

func metric1() unit.Metric {
	return unit.Metric{PxPerDp: 1, PxPerSp: 1}
}

func metric2() unit.Metric {
	return unit.Metric{PxPerDp: 2, PxPerSp: 2}
}

func applyOpts(m unit.Metric, opts ...Option) Config {
	var c Config
	c.apply(m, opts)
	return c
}

func TestTitle(t *testing.T) {
	c := applyOpts(metric1(), Title("hello"))
	if c.Title != "hello" {
		t.Errorf("Title = %q, want %q", c.Title, "hello")
	}
}

func TestSize(t *testing.T) {
	c := applyOpts(metric1(), Size(unit.Dp(100), unit.Dp(50)))
	if c.Size != (image.Point{X: 100, Y: 50}) {
		t.Errorf("Size = %v, want {100,50}", c.Size)
	}
	if c.Mode != Windowed {
		t.Errorf("Size did not set Mode=Windowed, got %v", c.Mode)
	}
}

func TestSizeResetsModeToWindowed(t *testing.T) {
	c := applyOpts(metric1(), Fullscreen.Option(), Size(unit.Dp(10), unit.Dp(10)))
	if c.Mode != Windowed {
		t.Errorf("Size should reset Mode to Windowed, got %v", c.Mode)
	}
}

func TestSizeMetricScaling(t *testing.T) {
	c := applyOpts(metric2(), Size(unit.Dp(100), unit.Dp(50)))
	if c.Size != (image.Point{X: 200, Y: 100}) {
		t.Errorf("Size with PxPerDp=2 = %v, want {200,100}", c.Size)
	}
}

func TestMaxSize(t *testing.T) {
	c := applyOpts(metric1(), MaxSize(unit.Dp(800), unit.Dp(600)))
	if c.MaxSize != (image.Point{X: 800, Y: 600}) {
		t.Errorf("MaxSize = %v, want {800,600}", c.MaxSize)
	}
}

func TestMaxSizeMetricScaling(t *testing.T) {
	c := applyOpts(metric2(), MaxSize(unit.Dp(80), unit.Dp(60)))
	if c.MaxSize != (image.Point{X: 160, Y: 120}) {
		t.Errorf("MaxSize with PxPerDp=2 = %v, want {160,120}", c.MaxSize)
	}
}

func TestMinSize(t *testing.T) {
	c := applyOpts(metric1(), MinSize(unit.Dp(200), unit.Dp(100)))
	if c.MinSize != (image.Point{X: 200, Y: 100}) {
		t.Errorf("MinSize = %v, want {200,100}", c.MinSize)
	}
}

func TestMinSizeMetricScaling(t *testing.T) {
	c := applyOpts(metric2(), MinSize(unit.Dp(20), unit.Dp(10)))
	if c.MinSize != (image.Point{X: 40, Y: 20}) {
		t.Errorf("MinSize with PxPerDp=2 = %v, want {40,20}", c.MinSize)
	}
}

func expectPanic(t *testing.T, wantSubstr string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected panic containing %q, got none", wantSubstr)
			return
		}
		msg, ok := r.(string)
		if !ok {
			t.Errorf("panic value not a string: %v", r)
			return
		}
		if !strings.Contains(msg, wantSubstr) {
			t.Errorf("panic message = %q, want substring %q", msg, wantSubstr)
		}
	}()
	fn()
}

func TestSizeBounds(t *testing.T) {
	// w=1 succeeds
	_ = Size(unit.Dp(1), unit.Dp(1))
	// w=0 panics
	expectPanic(t, "width must be larger than 0", func() {
		Size(unit.Dp(0), unit.Dp(10))
	})
	// h=0 panics
	expectPanic(t, "height must be larger than 0", func() {
		Size(unit.Dp(10), unit.Dp(0))
	})
	// w=-1 panics
	expectPanic(t, "width must be larger than 0", func() {
		Size(unit.Dp(-1), unit.Dp(10))
	})
	// h=-1 panics
	expectPanic(t, "height must be larger than 0", func() {
		Size(unit.Dp(10), unit.Dp(-1))
	})
}

func TestMaxSizeBounds(t *testing.T) {
	_ = MaxSize(unit.Dp(1), unit.Dp(1))
	expectPanic(t, "width must be larger than 0", func() {
		MaxSize(unit.Dp(0), unit.Dp(10))
	})
	expectPanic(t, "height must be larger than 0", func() {
		MaxSize(unit.Dp(10), unit.Dp(0))
	})
	expectPanic(t, "width must be larger than 0", func() {
		MaxSize(unit.Dp(-5), unit.Dp(10))
	})
	expectPanic(t, "height must be larger than 0", func() {
		MaxSize(unit.Dp(10), unit.Dp(-5))
	})
}

func TestMinSizeBounds(t *testing.T) {
	_ = MinSize(unit.Dp(1), unit.Dp(1))
	expectPanic(t, "width must be larger than 0", func() {
		MinSize(unit.Dp(0), unit.Dp(10))
	})
	expectPanic(t, "height must be larger than 0", func() {
		MinSize(unit.Dp(10), unit.Dp(0))
	})
	expectPanic(t, "width must be larger than 0", func() {
		MinSize(unit.Dp(-1), unit.Dp(10))
	})
	expectPanic(t, "height must be larger than 0", func() {
		MinSize(unit.Dp(10), unit.Dp(-1))
	})
}

func TestNavigationColor(t *testing.T) {
	want := color.NRGBA{R: 1, G: 2, B: 3, A: 4}
	c := applyOpts(metric1(), NavigationColor(want))
	if c.NavigationColor != want {
		t.Errorf("NavigationColor = %v, want %v", c.NavigationColor, want)
	}
}

func TestCustomRenderer(t *testing.T) {
	c := applyOpts(metric1(), CustomRenderer(true))
	if !c.CustomRenderer {
		t.Errorf("CustomRenderer = false, want true")
	}
	c2 := applyOpts(metric1(), CustomRenderer(false))
	if c2.CustomRenderer {
		t.Errorf("CustomRenderer = true, want false")
	}
}

func TestDecorated(t *testing.T) {
	c := applyOpts(metric1(), Decorated(true))
	if !c.Decorated {
		t.Errorf("Decorated = false, want true")
	}
	c2 := applyOpts(metric1(), Decorated(false))
	if c2.Decorated {
		t.Errorf("Decorated = true, want false")
	}
}

func TestTopMost(t *testing.T) {
	c := applyOpts(metric1(), TopMost(true))
	if !c.TopMost {
		t.Errorf("TopMost = false, want true")
	}
	c2 := applyOpts(metric1(), TopMost(false))
	if c2.TopMost {
		t.Errorf("TopMost = true, want false")
	}
}

func TestApplyComposesLastWriteWins(t *testing.T) {
	c := applyOpts(metric1(),
		Title("first"),
		Title("second"),
		Size(unit.Dp(10), unit.Dp(20)),
		Size(unit.Dp(30), unit.Dp(40)),
		TopMost(true),
		TopMost(false),
	)
	if c.Title != "second" {
		t.Errorf("Title = %q, want %q", c.Title, "second")
	}
	if c.Size != (image.Point{X: 30, Y: 40}) {
		t.Errorf("Size = %v, want {30,40}", c.Size)
	}
	if c.TopMost {
		t.Errorf("TopMost = true, want false")
	}
}

func TestApplyComposesIndependentFields(t *testing.T) {
	c := applyOpts(metric1(),
		Title("t"),
		Size(unit.Dp(100), unit.Dp(200)),
		MinSize(unit.Dp(10), unit.Dp(20)),
		MaxSize(unit.Dp(1000), unit.Dp(2000)),
		Decorated(true),
		TopMost(true),
		CustomRenderer(true),
		LandscapeOrientation.Option(),
		Maximized.Option(),
	)
	if c.Title != "t" {
		t.Errorf("Title = %q", c.Title)
	}
	if c.Size != (image.Point{X: 100, Y: 200}) {
		t.Errorf("Size = %v", c.Size)
	}
	if c.MinSize != (image.Point{X: 10, Y: 20}) {
		t.Errorf("MinSize = %v", c.MinSize)
	}
	if c.MaxSize != (image.Point{X: 1000, Y: 2000}) {
		t.Errorf("MaxSize = %v", c.MaxSize)
	}
	if !c.Decorated || !c.TopMost || !c.CustomRenderer {
		t.Errorf("bool fields not set: %+v", c)
	}
	if c.Orientation != LandscapeOrientation {
		t.Errorf("Orientation = %v, want LandscapeOrientation", c.Orientation)
	}
	if c.Mode != Maximized {
		t.Errorf("Mode = %v, want Maximized", c.Mode)
	}
}

func TestWindowModeString(t *testing.T) {
	cases := []struct {
		m    WindowMode
		want string
	}{
		{Windowed, "windowed"},
		{Fullscreen, "fullscreen"},
		{Minimized, "minimized"},
		{Maximized, "maximized"},
		{WindowMode(99), ""},
	}
	for _, tc := range cases {
		if got := tc.m.String(); got != tc.want {
			t.Errorf("WindowMode(%d).String() = %q, want %q", tc.m, got, tc.want)
		}
	}
}

func TestOrientationString(t *testing.T) {
	cases := []struct {
		o    Orientation
		want string
	}{
		{AnyOrientation, "any"},
		{LandscapeOrientation, "landscape"},
		{PortraitOrientation, "portrait"},
		{Orientation(99), ""},
	}
	for _, tc := range cases {
		if got := tc.o.String(); got != tc.want {
			t.Errorf("Orientation(%d).String() = %q, want %q", tc.o, got, tc.want)
		}
	}
}

func TestWindowModeOptionRoundTrip(t *testing.T) {
	for _, m := range []WindowMode{Windowed, Fullscreen, Minimized, Maximized} {
		c := applyOpts(metric1(), m.Option())
		if c.Mode != m {
			t.Errorf("WindowMode(%v).Option() applied -> Mode = %v", m, c.Mode)
		}
	}
}

func TestOrientationOptionRoundTrip(t *testing.T) {
	for _, o := range []Orientation{AnyOrientation, LandscapeOrientation, PortraitOrientation} {
		c := applyOpts(metric1(), o.Option())
		if c.Orientation != o {
			t.Errorf("Orientation(%v).Option() applied -> Orientation = %v", o, c.Orientation)
		}
	}
}

func TestConfigApplyEmpty(t *testing.T) {
	var c Config
	c.apply(metric1(), nil)
	if c != (Config{}) {
		t.Errorf("apply(nil) mutated config: %+v", c)
	}
}
