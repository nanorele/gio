package gpu

import (
	"image"
	"math"
	"testing"

	"github.com/nanorele/gio/internal/f32"
)

// fakeResource counts release() calls for cache regression tests.
type fakeResource struct {
	released int
}

func (f *fakeResource) release() { f.released++ }

// Fix 1: textureCache.put must release the previous resource when overwriting
// an unused entry, otherwise the GPU handle leaks.
func TestTextureCachePutReleasesEvictedResource(t *testing.T) {
	c := newTextureCache()
	key := textureCacheKey{filter: 0, handle: "k"}

	first := &fakeResource{}
	c.put(key, first)

	// Mark unused so the next put() takes the eviction branch.
	c.frame()

	second := &fakeResource{}
	c.put(key, second)

	if first.released != 1 {
		t.Fatalf("expected first resource released exactly once, got %d", first.released)
	}
	if second.released != 0 {
		t.Fatalf("expected second resource not released, got %d", second.released)
	}

	// The new resource must be retrievable.
	got, ok := c.get(key)
	if !ok || got != second {
		t.Fatalf("expected to get back second resource, got %v ok=%v", got, ok)
	}
}

// A fresh put() into an empty slot must not call release() on anything.
func TestTextureCachePutFreshNoRelease(t *testing.T) {
	c := newTextureCache()
	key := textureCacheKey{filter: 1, handle: "fresh"}
	r := &fakeResource{}
	c.put(key, r)
	if r.released != 0 {
		t.Fatalf("fresh put should not release, got %d", r.released)
	}
}

// Putting twice without an intervening frame() while the entry is still "used"
// must panic — the contract documented by the existing code.
func TestTextureCachePutUsedKeyPanics(t *testing.T) {
	c := newTextureCache()
	key := textureCacheKey{filter: 0, handle: "dup"}
	c.put(key, &fakeResource{})
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when putting a used key")
		}
	}()
	c.put(key, &fakeResource{})
}

// Fix 2: texSpaceTransform must guard against zero bounds to avoid div-by-zero
// producing NaN/Inf in shader uniforms.
func TestTexSpaceTransformZeroBoundsX(t *testing.T) {
	scale, off := texSpaceTransform(f32.Rectangle{Max: f32.Point{X: 1, Y: 1}}, image.Pt(0, 50))
	if scale != (f32.Point{}) || off != (f32.Point{}) {
		t.Fatalf("expected zero scale/offset, got scale=%v off=%v", scale, off)
	}
}

func TestTexSpaceTransformZeroBoundsY(t *testing.T) {
	scale, off := texSpaceTransform(f32.Rectangle{Max: f32.Point{X: 1, Y: 1}}, image.Pt(50, 0))
	if scale != (f32.Point{}) || off != (f32.Point{}) {
		t.Fatalf("expected zero scale/offset, got scale=%v off=%v", scale, off)
	}
}

func TestTexSpaceTransformZeroBoundsBoth(t *testing.T) {
	scale, off := texSpaceTransform(f32.Rectangle{Max: f32.Point{X: 1, Y: 1}}, image.Pt(0, 0))
	if scale != (f32.Point{}) || off != (f32.Point{}) {
		t.Fatalf("expected zero scale/offset, got scale=%v off=%v", scale, off)
	}
}

func TestTexSpaceTransformValid(t *testing.T) {
	r := f32.Rectangle{
		Min: f32.Point{X: 10, Y: 5},
		Max: f32.Point{X: 60, Y: 30},
	}
	bounds := image.Pt(100, 50)
	scale, off := texSpaceTransform(r, bounds)

	wantScale := f32.Point{X: 0.5, Y: 0.5}      // (50/100, 25/50)
	wantOff := f32.Point{X: 0.1, Y: 0.1}        // (10/100, 5/50)

	if !nearPt(scale, wantScale) {
		t.Errorf("scale: got %v, want %v", scale, wantScale)
	}
	if !nearPt(off, wantOff) {
		t.Errorf("offset: got %v, want %v", off, wantOff)
	}

	if isBadFloat(scale.X) || isBadFloat(scale.Y) || isBadFloat(off.X) || isBadFloat(off.Y) {
		t.Fatalf("unexpected NaN/Inf: scale=%v off=%v", scale, off)
	}
}

// clipSpaceTransform had the same div-by-zero hazard as texSpaceTransform when
// the viewport collapses to 0 (e.g. minimized window). Fixed in this pass.
func TestClipSpaceTransformZeroViewport(t *testing.T) {
	cases := []image.Point{{X: 0, Y: 100}, {X: 100, Y: 0}, {X: 0, Y: 0}}
	for _, vp := range cases {
		scale, off := clipSpaceTransform(image.Rect(0, 0, 10, 10), vp)
		if scale != (f32.Point{}) || off != (f32.Point{}) {
			t.Errorf("viewport %v: expected zero, got scale=%v off=%v", vp, scale, off)
		}
		if isBadFloat(scale.X) || isBadFloat(scale.Y) || isBadFloat(off.X) || isBadFloat(off.Y) {
			t.Errorf("viewport %v: produced NaN/Inf", vp)
		}
	}
}

// Fix 3 (stencilPath zero-bounds): NOT unit-tested here.
// stencilPath is a method on *stenciler whose body (after the early return)
// touches s.ctx (driver.Device), s.pipeline, and issues GPU calls
// (Viewport, BindVertexBuffer, DrawElements, UploadUniforms). Constructing a
// stenciler with a real driver.Device requires a full GPU/window context that
// is unavailable in headless `go test` runs and would dominate this regression
// suite with platform-specific setup. The early return predicate
// (bounds.Dx() == 0 || bounds.Dy() == 0) is trivial and covered by manual
// inspection; we instead exercise the analogous predicate in texSpaceTransform
// above.

func nearPt(a, b f32.Point) bool {
	const eps = 1e-6
	return math.Abs(float64(a.X-b.X)) < eps && math.Abs(float64(a.Y-b.Y)) < eps
}

func isBadFloat(f float32) bool {
	d := float64(f)
	return math.IsNaN(d) || math.IsInf(d, 0)
}
