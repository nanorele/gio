package widget

import (
	"image"
	"testing"

	"golang.org/x/image/math/fixed"
)

// regionHeight returns the pixel height of a region's bounding box.
func regionHeight(r Region) int { return r.Bounds.Max.Y - r.Bounds.Min.Y }

// TestMakeRegionCoversDescent verifies that the selection rectangle produced by
// makeRegion fully encloses the glyph row from the top of the ascent to the
// bottom of the descent. The baseline sits at y; ascent extends above it and
// descent below it. A descender such as 'j', 'g', 'p', 'q' or 'y' reaches all
// the way down to y+descent, so the selection bottom must be at least that far
// down. Previously descent was truncated with Floor(), clipping a sub-pixel row
// off the bottom of every descender.
func TestMakeRegionCoversDescent(t *testing.T) {
	const y = 16
	tests := []struct {
		name             string
		ascent, descent  fixed.Int26_6
		wantTopAboveY    int // pixels the region extends above the baseline
		wantBottomBelowY int // pixels the region extends below the baseline
	}{
		{
			// 216/64 = 3.375px of descent: must round up to 4 so the full
			// descender is covered. This is the reported "missing pixels" bug.
			name:             "fractional descent (goregular 16px)",
			ascent:           fixed.Int26_6(968), // 15.125px
			descent:          fixed.Int26_6(216), // 3.375px
			wantTopAboveY:    16,                  // ceil(15.125)
			wantBottomBelowY: 4,                   // ceil(3.375)
		},
		{
			// Exactly integral descent: ceil == floor, nothing changes.
			name:             "integral descent",
			ascent:           fixed.I(12),
			descent:          fixed.I(3),
			wantTopAboveY:    12,
			wantBottomBelowY: 3,
		},
		{
			// Just over an integer boundary: 64*3+1 = 193 -> 3.0156px.
			name:             "barely fractional descent",
			ascent:           fixed.I(10),
			descent:          fixed.Int26_6(193),
			wantTopAboveY:    10,
			wantBottomBelowY: 4,
		},
		{
			// Tiny descent must still produce at least 1px of coverage.
			name:             "sub-pixel descent",
			ascent:           fixed.I(8),
			descent:          fixed.Int26_6(1), // 0.0156px
			wantTopAboveY:    8,
			wantBottomBelowY: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			line := lineInfo{ascent: tc.ascent, descent: tc.descent}
			r := makeRegion(line, y, fixed.I(0), fixed.I(50))

			gotTop := y - r.Bounds.Min.Y
			gotBottom := r.Bounds.Max.Y - y

			if gotTop != tc.wantTopAboveY {
				t.Errorf("top coverage = %d px above baseline, want %d", gotTop, tc.wantTopAboveY)
			}
			if gotBottom != tc.wantBottomBelowY {
				t.Errorf("bottom coverage = %d px below baseline, want %d", gotBottom, tc.wantBottomBelowY)
			}

			// The bottom edge must never clip the descender: it must reach at
			// least the real (fractional) descent.
			minBottom := tc.descent.Ceil()
			if gotBottom < minBottom {
				t.Errorf("descender clipped: bottom coverage %d px < required %d px (descent %v)",
					gotBottom, minBottom, tc.descent)
			}
		})
	}
}

// TestMakeRegionHeight checks the total height equals ceil(ascent)+ceil(descent),
// i.e. no rows lost to truncation on either edge.
func TestMakeRegionHeight(t *testing.T) {
	line := lineInfo{ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216)}
	r := makeRegion(line, 16, fixed.I(0), fixed.I(100))
	want := line.ascent.Ceil() + line.descent.Ceil()
	if got := regionHeight(r); got != want {
		t.Errorf("region height = %d, want %d", got, want)
	}
}

// TestMakeRegionHorizontal verifies the horizontal bounds and the start/end
// swap behaviour are unaffected by the descent fix.
func TestMakeRegionHorizontal(t *testing.T) {
	line := lineInfo{ascent: fixed.I(10), descent: fixed.I(3)}

	r := makeRegion(line, 0, fixed.I(5), fixed.I(40))
	if r.Bounds.Min.X != 5 || r.Bounds.Max.X != 40 {
		t.Errorf("x bounds = [%d,%d], want [5,40]", r.Bounds.Min.X, r.Bounds.Max.X)
	}

	// Reversed selection must yield identical bounds.
	rev := makeRegion(line, 0, fixed.I(40), fixed.I(5))
	if rev.Bounds != r.Bounds {
		t.Errorf("reversed region bounds = %v, want %v", rev.Bounds, r.Bounds)
	}
}

// TestMakeRegionBaseline verifies the Baseline field reports the full descent
// (distance from baseline to the bottom edge of the region), consistent with
// the region bottom.
func TestMakeRegionBaseline(t *testing.T) {
	line := lineInfo{ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216)}
	r := makeRegion(line, 16, fixed.I(0), fixed.I(10))
	wantBottom := r.Bounds.Max.Y - 16
	if r.Baseline != wantBottom {
		t.Errorf("Baseline = %d, want %d (region bottom below baseline)", r.Baseline, wantBottom)
	}
}

var _ = image.Point{}
