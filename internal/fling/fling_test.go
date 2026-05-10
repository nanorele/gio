package fling

import (
	"math"
	"runtime"
	"testing"
	"time"

	"github.com/nanorele/gio/unit"
)

const epsilon = 1e-4

func TestAnimationStartBoundary(t *testing.T) {
	var a Animation
	now := time.Now()
	m := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	min := float32(m.Dp(minFlingVelocity))

	// Exactly at min should NOT start (boundary inclusive in the rejection range).
	if a.Start(m, now, min) {
		t.Errorf("Start at +min velocity %f should return false", min)
	}
	if a.Start(m, now, -min) {
		t.Errorf("Start at -min velocity should return false")
	}
	if a.Active() {
		t.Error("Animation must not be active after rejected starts")
	}

	// Just above min should start.
	if !a.Start(m, now, min+1) {
		t.Error("Start just above min should return true")
	}
}

func TestAnimationZeroVelocity(t *testing.T) {
	var a Animation
	now := time.Now()
	m := unit.Metric{PxPerDp: 1, PxPerSp: 1}

	if a.Start(m, now, 0) {
		t.Error("Start with zero velocity should return false")
	}
	if a.Active() {
		t.Error("Animation should not be active after zero-velocity Start")
	}
	if d := a.Tick(now.Add(50 * time.Millisecond)); d != 0 {
		t.Errorf("Tick on inactive (zero v) should be 0, got %d", d)
	}
}

func TestAnimationDecayMonotonic(t *testing.T) {
	var a Animation
	now := time.Now()
	m := unit.Metric{PxPerDp: 1, PxPerSp: 1}

	if !a.Start(m, now, 4000) {
		t.Fatal("Start should succeed")
	}

	// Sample per-tick velocity (delta distance per unit time) and ensure it
	// decays monotonically (non-increasing in absolute value).
	var prev int = math.MaxInt32
	t0 := now
	for i := 1; i <= 30 && a.Active(); i++ {
		d := a.Tick(t0.Add(time.Duration(i) * 16 * time.Millisecond))
		if d < 0 {
			t.Errorf("positive velocity should yield non-negative ticks, got %d at i=%d", d, i)
		}
		if d > prev {
			t.Errorf("decay not monotonic: prev=%d cur=%d at i=%d", prev, d, i)
		}
		prev = d
	}
}

func TestAnimationNegativeDirection(t *testing.T) {
	var a Animation
	now := time.Now()
	m := unit.Metric{PxPerDp: 1, PxPerSp: 1}

	if !a.Start(m, now, -2000) {
		t.Fatal("Start with negative velocity failed")
	}
	total := 0
	for i := 1; i <= 60 && a.Active(); i++ {
		d := a.Tick(now.Add(time.Duration(i) * 16 * time.Millisecond))
		if d > 0 {
			t.Errorf("negative velocity must produce non-positive ticks, got %d at i=%d", d, i)
		}
		total += d
	}
	if total >= 0 {
		t.Errorf("expected negative total displacement, got %d", total)
	}
}

func TestAnimationLargeVelocityNoOverflow(t *testing.T) {
	var a Animation
	now := time.Now()
	m := unit.Metric{PxPerDp: 1, PxPerSp: 1}

	// Way above max — should clamp.
	if !a.Start(m, now, 1e9) {
		t.Fatal("Start with huge velocity should succeed (clamped)")
	}
	max := float32(m.Dp(maxFlingVelocity))
	if a.v0 != max {
		t.Errorf("v0 not clamped: got %f want %f", a.v0, max)
	}

	// Run to completion; ensure no NaN/Inf and finite total distance.
	total := int64(0)
	for i := 1; i <= 1000 && a.Active(); i++ {
		d := a.Tick(now.Add(time.Duration(i) * 16 * time.Millisecond))
		total += int64(d)
	}
	if total <= 0 {
		t.Errorf("expected positive total distance, got %d", total)
	}
	if a.Active() {
		t.Error("animation should have stopped within a reasonable horizon")
	}
}

func TestAnimationRestartResetsState(t *testing.T) {
	var a Animation
	now := time.Now()
	m := unit.Metric{PxPerDp: 1, PxPerSp: 1}

	a.Start(m, now, 1500)
	for i := 1; i <= 5; i++ {
		a.Tick(now.Add(time.Duration(i) * 10 * time.Millisecond))
	}
	if a.x == 0 {
		t.Error("expected accumulated x after ticks")
	}

	// Restart with opposite-sign velocity; init must reset x and t0.
	newNow := now.Add(1 * time.Second)
	a.Start(m, newNow, -1500)
	if a.x != 0 {
		t.Errorf("Start did not reset x, got %f", a.x)
	}
	if !a.t0.Equal(newNow) {
		t.Errorf("Start did not reset t0")
	}
	if a.v0 != -1500 {
		t.Errorf("expected v0 = -1500, got %f", a.v0)
	}
}

func TestAnimationTickSameTime(t *testing.T) {
	var a Animation
	now := time.Now()
	m := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	a.Start(m, now, 2000)
	// Tick at t0: dt=0 should yield zero distance, no NaN.
	if d := a.Tick(now); d != 0 {
		t.Errorf("Tick at t0 should return 0, got %d", d)
	}
	if !a.Active() {
		t.Error("animation must remain active after zero-dt tick")
	}
}

func TestAnimationDarwinConstant(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only constant path")
	}
	// Just confirm execution path is exercised; correctness covered by
	// other monotonic/sign tests.
	var a Animation
	now := time.Now()
	m := unit.Metric{PxPerDp: 1, PxPerSp: 1}
	a.Start(m, now, 1000)
	a.Tick(now.Add(50 * time.Millisecond))
}

// --- extrapolation tests ---

func TestExtrapolationEmpty(t *testing.T) {
	var e Extrapolation
	if got := e.Estimate(); got != (Estimate{}) {
		t.Errorf("expected zero Estimate from empty, got %+v", got)
	}
}

func TestExtrapolationSingleSample(t *testing.T) {
	var e Extrapolation
	e.Sample(0, 100)
	got := e.Estimate()
	if got.Velocity != 0 {
		t.Errorf("single sample must yield zero velocity, got %f", got.Velocity)
	}
}

func TestExtrapolationTwoSamples(t *testing.T) {
	var e Extrapolation
	e.Sample(0, 0)
	e.Sample(10*time.Millisecond, 10)
	got := e.Estimate()
	// degree=2 polyFit needs >2 points; should fall back to zero.
	if got.Velocity != 0 {
		t.Errorf("two samples must yield zero velocity (polyFit fail), got %f", got.Velocity)
	}
}

func TestExtrapolationConstantValue(t *testing.T) {
	var e Extrapolation
	// All same value -> zero velocity.
	for i := 0; i < 5; i++ {
		e.Sample(time.Duration(i)*10*time.Millisecond, 50)
	}
	got := e.Estimate()
	if math.Abs(float64(got.Velocity)) > epsilon {
		t.Errorf("constant samples must yield ~0 velocity, got %f", got.Velocity)
	}
	if math.Abs(float64(got.Distance)) > epsilon {
		t.Errorf("constant samples must yield ~0 distance, got %f", got.Distance)
	}
}

func TestExtrapolationLinearVelocity(t *testing.T) {
	var e Extrapolation
	// position v(t) = 500 px/s * t, so velocity should be ~500.
	const v = 500.0
	for i := 0; i < 6; i++ {
		dt := time.Duration(i) * 10 * time.Millisecond
		pos := float32(v * dt.Seconds())
		e.Sample(dt, pos)
	}
	got := e.Estimate()
	// Allow loose tolerance — quadratic fit on linear data should still recover slope.
	if math.Abs(float64(got.Velocity)-v) > 5 {
		t.Errorf("expected velocity ~%f, got %f", v, got.Velocity)
	}
}

func TestExtrapolationSampleDeltaAccumulates(t *testing.T) {
	var e Extrapolation
	e.Sample(0, 0)
	e.SampleDelta(10*time.Millisecond, 5)
	e.SampleDelta(20*time.Millisecond, 5)
	e.SampleDelta(30*time.Millisecond, 5)
	if e.lastValue != 15 {
		t.Errorf("expected lastValue=15, got %f", e.lastValue)
	}
	got := e.Estimate()
	// Linear slope: 5 px / 10 ms = 500 px/s.
	if math.Abs(float64(got.Velocity)-500) > 50 {
		t.Errorf("expected velocity ~500, got %f", got.Velocity)
	}
}

func TestExtrapolationNegativeDirection(t *testing.T) {
	var e Extrapolation
	const v = -300.0
	for i := 0; i < 6; i++ {
		dt := time.Duration(i) * 10 * time.Millisecond
		pos := float32(v * dt.Seconds())
		e.Sample(dt, pos)
	}
	got := e.Estimate()
	if got.Velocity >= 0 {
		t.Errorf("expected negative velocity, got %f", got.Velocity)
	}
	if math.Abs(float64(got.Velocity)-v) > 5 {
		t.Errorf("expected velocity ~%f, got %f", v, got.Velocity)
	}
}

func TestExtrapolationRingBufferWrap(t *testing.T) {
	var e Extrapolation
	// Fill past historySize. Use small inter-sample gap and small total span
	// so age and gap checks don't trigger.
	const v = 200.0
	n := historySize + 7
	for i := 0; i < n; i++ {
		// 2ms steps -> total span 2*(n-1) ms < maxAge (100ms).
		dt := time.Duration(i) * 2 * time.Millisecond
		pos := float32(v * dt.Seconds())
		e.Sample(dt, pos)
	}
	if len(e.samples) != historySize {
		t.Errorf("expected samples len=%d, got %d", historySize, len(e.samples))
	}
	if e.idx >= historySize {
		t.Errorf("idx out of range after wrap: %d", e.idx)
	}
	got := e.Estimate()
	if math.Abs(float64(got.Velocity)-v) > 10 {
		t.Errorf("expected velocity ~%f after wrap, got %f", v, got.Velocity)
	}
}

func TestExtrapolationGapBreaks(t *testing.T) {
	var e Extrapolation
	// One old sample, then a big gap, then two recent samples — only the
	// recent two are usable, which is fewer than degree+1 -> Estimate{}.
	e.Sample(0, 0)
	e.Sample(500*time.Millisecond, 50)
	e.Sample(505*time.Millisecond, 55)
	got := e.Estimate()
	if got != (Estimate{}) {
		t.Errorf("expected zero Estimate when gap isolates few samples, got %+v", got)
	}
}

func TestExtrapolationVerySmallDt(t *testing.T) {
	var e Extrapolation
	// Very small but distinct dt — should not produce NaN/Inf.
	const v = 100.0
	for i := 0; i < 6; i++ {
		dt := time.Duration(i) * 100 * time.Microsecond // 0.1ms steps
		pos := float32(v * dt.Seconds())
		e.Sample(dt, pos)
	}
	got := e.Estimate()
	if math.IsNaN(float64(got.Velocity)) || math.IsInf(float64(got.Velocity), 0) {
		t.Errorf("velocity not finite: %f", got.Velocity)
	}
}

func TestExtrapolationDuplicateTimestamps(t *testing.T) {
	var e Extrapolation
	// All samples at the same timestamp -> degenerate (collinear in t).
	// polyFit must either return zero estimate or finite, not NaN.
	for i := 0; i < 5; i++ {
		e.Sample(10*time.Millisecond, float32(i))
	}
	got := e.Estimate()
	if math.IsNaN(float64(got.Velocity)) || math.IsInf(float64(got.Velocity), 0) {
		t.Errorf("velocity must be finite for degenerate input, got %f", got.Velocity)
	}
}

func TestExtrapolationDistanceField(t *testing.T) {
	var e Extrapolation
	// Make sure Distance reflects last - first within the considered window.
	e.Sample(0, 0)
	e.Sample(10*time.Millisecond, 1)
	e.Sample(20*time.Millisecond, 4)
	e.Sample(30*time.Millisecond, 9)
	got := e.Estimate()
	// Window covers all 4 samples; first.v=9, oldest p.v=0.
	// values = [9-9, 9-4, 9-1, 9-0] = [0, 5, 8, 9].
	// Distance = values[last] - values[0] = 9 - 0 = 9.
	if math.Abs(float64(got.Distance)-9) > epsilon {
		t.Errorf("expected distance 9, got %f", got.Distance)
	}
}

func TestPolyFitQuadratic(t *testing.T) {
	// y = 1 + 2x + 3x^2 — exact recovery expected.
	X := []float32{-2, -1, 0, 1, 2}
	Y := make([]float32, len(X))
	for i, x := range X {
		Y[i] = 1 + 2*x + 3*x*x
	}
	got, ok := polyFit(X, Y)
	if !ok {
		t.Fatal("polyFit failed")
	}
	want := coefficients{1, 2, 3}
	if !got.approxEqual(want) {
		t.Errorf("polyFit: got %v want %v", got, want)
	}
}

func TestPolyFitDegreeBoundary(t *testing.T) {
	// `len(X) <= degree` returns false; degree=2 so len=3 succeeds, len=2 fails.
	if _, ok := polyFit([]float32{0, 1, 2}, []float32{0, 1, 4}); !ok {
		t.Error("polyFit should succeed for len(X) == degree+1")
	}
	if _, ok := polyFit([]float32{0, 1}, []float32{0, 1}); ok {
		t.Error("polyFit should fail for len(X) <= degree")
	}
}

func TestMatrixColMethodLayout(t *testing.T) {
	// Confirm col() returns a contiguous slice of the underlying buffer
	// (it's actually a row in row-major layout — internal convention).
	m := newMatrix(3, 3)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			m.set(i, j, float32(i*3+j))
		}
	}
	c := m.col(1)
	// Row 1 in row-major = elements (1,0), (1,1), (1,2) = 3, 4, 5.
	if len(c) != 3 || c[0] != 3 || c[1] != 4 || c[2] != 5 {
		t.Errorf("unexpected col() contents: %v", c)
	}
}
