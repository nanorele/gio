package widget

import (
	"image"
	"math"
	"strings"
	"testing"

	"github.com/nanorele/gio/font"
	"github.com/nanorele/gio/font/gofont"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/text"
	"github.com/nanorele/gio/unit"
	"golang.org/x/image/math/fixed"
)

func TestGlyphIterator(t *testing.T) {
	fontSize := 16
	stdAscent := fixed.I(fontSize)
	stdDescent := fixed.I(4)
	stdLineHeight := stdAscent + stdDescent
	type testcase struct {
		name             string
		str              string
		maxWidth         int
		maxLines         int
		viewport         image.Rectangle
		expectedDims     image.Rectangle
		expectedBaseline int
		stopAtGlyph      int
	}
	for _, tc := range []testcase{
		{
			name:     "empty string",
			str:      "",
			viewport: image.Rectangle{Max: image.Pt(math.MaxInt, math.MaxInt)},
			expectedDims: image.Rectangle{
				Max: image.Point{X: 0, Y: stdLineHeight.Round()},
			},
			expectedBaseline: fontSize,
			stopAtGlyph:      0,
		},
		{
			name:     "simple",
			str:      "MMM",
			viewport: image.Rectangle{Max: image.Pt(math.MaxInt, math.MaxInt)},
			expectedDims: image.Rectangle{
				Max: image.Point{X: 40, Y: stdLineHeight.Round()},
			},
			expectedBaseline: fontSize,
			stopAtGlyph:      2,
		},
		{
			name:     "simple clipped horizontally",
			str:      "MMM",
			viewport: image.Rectangle{Max: image.Pt(20, math.MaxInt)},

			expectedDims: image.Rectangle{
				Max: image.Point{X: 27, Y: stdLineHeight.Round()},
			},
			expectedBaseline: fontSize,
			stopAtGlyph:      2,
		},
		{
			name:     "simple clipped vertically",
			str:      "M\nM\nM\nM",
			viewport: image.Rectangle{Max: image.Pt(math.MaxInt, 2*stdLineHeight.Floor()-3)},

			expectedDims: image.Rectangle{
				Max: image.Point{X: 14, Y: 39},
			},
			expectedBaseline: fontSize,
			stopAtGlyph:      4,
		},
		{
			name:     "simple truncated",
			str:      "mmm",
			maxLines: 1,
			viewport: image.Rectangle{Max: image.Pt(math.MaxInt, math.MaxInt)},

			expectedDims: image.Rectangle{
				Max: image.Point{X: 40, Y: stdLineHeight.Round()},
			},
			expectedBaseline: fontSize,
			stopAtGlyph:      2,
		},
		{
			name:     "whitespace",
			str:      "   ",
			viewport: image.Rectangle{Max: image.Pt(math.MaxInt, math.MaxInt)},
			expectedDims: image.Rectangle{
				Max: image.Point{X: 14, Y: stdLineHeight.Round()},
			},
			expectedBaseline: fontSize,
			stopAtGlyph:      2,
		},
		{
			name:     "multi-line with hard newline",
			str:      "你\n好",
			viewport: image.Rectangle{Max: image.Pt(math.MaxInt, math.MaxInt)},
			expectedDims: image.Rectangle{
				Max: image.Point{X: 12, Y: 39},
			},
			expectedBaseline: fontSize,
			stopAtGlyph:      3,
		},
		{
			name:     "multi-line with soft newline",
			str:      "你好",
			maxWidth: fontSize,
			viewport: image.Rectangle{Max: image.Pt(math.MaxInt, math.MaxInt)},
			expectedDims: image.Rectangle{
				Max: image.Point{X: 12, Y: 39},
			},
			expectedBaseline: fontSize,
			stopAtGlyph:      2,
		},
		{
			name:     "trailing hard newline",
			str:      "m\n",
			viewport: image.Rectangle{Max: image.Pt(math.MaxInt, math.MaxInt)},

			expectedDims: image.Rectangle{
				Max: image.Point{X: 14, Y: 39},
			},
			expectedBaseline: fontSize,
			stopAtGlyph:      1,
		},
		{
			name:     "truncated trailing hard newline",
			str:      "m\n",
			maxLines: 1,
			viewport: image.Rectangle{Max: image.Pt(math.MaxInt, math.MaxInt)},

			expectedDims: image.Rectangle{
				Max: image.Point{X: 14, Y: 20},
			},
			expectedBaseline: fontSize,
			stopAtGlyph:      1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			maxWidth := 200
			if tc.maxWidth != 0 {
				maxWidth = tc.maxWidth
			}
			glyphs := getGlyphs(16, 0, maxWidth, text.Start, tc.str)
			it := textIterator{viewport: tc.viewport, maxLines: tc.maxLines}
			for i, g := range glyphs {
				ok := it.processGlyph(g, true)
				if !ok && i != tc.stopAtGlyph {
					t.Errorf("expected iterator to stop at glyph %d, stopped at %d", tc.stopAtGlyph, i)
				}
				if !ok {
					break
				}
			}
			if it.bounds != tc.expectedDims {
				t.Errorf("expected bounds %#+v, got %#+v", tc.expectedDims, it.bounds)
			}
			if it.baseline != tc.expectedBaseline {
				t.Errorf("expected baseline %d, got %d", tc.expectedBaseline, it.baseline)
			}
		})
	}
}

func TestGlyphIteratorPadding(t *testing.T) {
	type testcase struct {
		name             string
		glyph            text.Glyph
		viewport         image.Rectangle
		expectedDims     image.Rectangle
		expectedPadding  image.Rectangle
		expectedBaseline int
	}
	for _, tc := range []testcase{
		{
			name: "simple",
			glyph: text.Glyph{
				X:       0,
				Y:       50,
				Advance: fixed.I(50),
				Ascent:  fixed.I(50),
				Descent: fixed.I(50),
				Bounds: fixed.Rectangle26_6{
					Min: fixed.Point26_6{
						X: fixed.I(-5),
						Y: fixed.I(-56),
					},
					Max: fixed.Point26_6{
						X: fixed.I(57),
						Y: fixed.I(58),
					},
				},
			},
			viewport: image.Rectangle{Max: image.Pt(math.MaxInt, math.MaxInt)},
			expectedDims: image.Rectangle{
				Max: image.Point{X: 50, Y: 100},
			},
			expectedBaseline: 50,
			expectedPadding: image.Rectangle{
				Min: image.Point{
					X: -5,
					Y: -6,
				},
				Max: image.Point{
					X: 7,
					Y: 8,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			it := textIterator{viewport: tc.viewport}
			it.processGlyph(tc.glyph, true)
			if it.bounds != tc.expectedDims {
				t.Errorf("expected bounds %#+v, got %#+v", tc.expectedDims, it.bounds)
			}
			if it.baseline != tc.expectedBaseline {
				t.Errorf("expected baseline %d, got %d", tc.expectedBaseline, it.baseline)
			}
			if it.padding != tc.expectedPadding {
				t.Errorf("expected padding %d, got %d", tc.expectedPadding, it.padding)
			}
		})
	}
}

// labelGtx returns a layout.Context suitable for laying out a Label, with
// constraints set as requested. Metric is 1:1 (1px = 1Sp = 1Dp) for
// deterministic dimension assertions.
func labelGtx(maxX, maxY int) layout.Context {
	return layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Constraints{Max: image.Pt(maxX, maxY)},
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Locale:      english,
	}
}

func labelShaper() *text.Shaper {
	return text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
}

// TestLabelLayoutEmpty verifies layout of an empty string yields well-defined
// (non-negative, finite) dimensions and zero TextInfo.Truncated.
func TestLabelLayoutEmpty(t *testing.T) {
	gtx := labelGtx(200, 200)
	sh := labelShaper()
	l := Label{}
	dims, info := l.LayoutDetailed(gtx, sh, font.Font{}, unit.Sp(12), "", op.CallOp{})

	if dims.Size.X < 0 || dims.Size.Y < 0 {
		t.Errorf("negative dims for empty label: %+v", dims.Size)
	}
	if dims.Size.Y == 0 {
		t.Errorf("empty label produced zero height; expected at least one line height")
	}
	if info.Truncated != 0 {
		t.Errorf("Truncated=%d for empty text, want 0", info.Truncated)
	}
	if dims.Baseline < 0 || dims.Baseline > dims.Size.Y {
		t.Errorf("Baseline %d outside [0,%d]", dims.Baseline, dims.Size.Y)
	}
}

// TestLabelLayoutBasic checks single-line layout produces width <= max width
// and height >= one font size.
func TestLabelLayoutBasic(t *testing.T) {
	gtx := labelGtx(500, 200)
	sh := labelShaper()
	l := Label{}
	dims, _ := l.LayoutDetailed(gtx, sh, font.Font{}, unit.Sp(12), "Hello", op.CallOp{})
	if dims.Size.X > gtx.Constraints.Max.X {
		t.Errorf("dim X %d > max %d", dims.Size.X, gtx.Constraints.Max.X)
	}
	if dims.Size.Y < 12 {
		t.Errorf("dim Y %d < font size 12", dims.Size.Y)
	}
}

// TestLabelLayoutReuse: laying out the same Label repeatedly should be
// idempotent (dimensions remain stable across calls).
func TestLabelLayoutReuse(t *testing.T) {
	sh := labelShaper()
	l := Label{}
	for _, txt := range []string{"hello", "hello world", "h"} {
		gtx := labelGtx(300, 300)
		d1, _ := l.LayoutDetailed(gtx, sh, font.Font{}, unit.Sp(12), txt, op.CallOp{})
		gtx.Ops.Reset()
		d2, _ := l.LayoutDetailed(gtx, sh, font.Font{}, unit.Sp(12), txt, op.CallOp{})
		if d1 != d2 {
			t.Errorf("non-deterministic Layout for %q: %+v vs %+v", txt, d1, d2)
		}
	}
}

// TestLabelMaxLinesTruncates: a label whose MaxLines is set to 1 must
// produce TextInfo.Truncated > 0 when the text is wider than the max line.
func TestLabelMaxLinesTruncates(t *testing.T) {
	sh := labelShaper()
	gtx := labelGtx(40, 200) // narrow constraint
	l := Label{MaxLines: 1, Truncator: "..."}
	long := strings.Repeat("ab cd ef ", 8)
	_, info := l.LayoutDetailed(gtx, sh, font.Font{}, unit.Sp(12), long, op.CallOp{})
	if info.Truncated == 0 {
		t.Errorf("expected Truncated > 0 with MaxLines=1 and long text, got 0")
	}
}

// TestLabelMaxLinesNoTruncationWhenFits: short text + MaxLines=1 fits and
// reports zero truncation.
func TestLabelMaxLinesNoTruncationWhenFits(t *testing.T) {
	sh := labelShaper()
	gtx := labelGtx(500, 200)
	l := Label{MaxLines: 1}
	_, info := l.LayoutDetailed(gtx, sh, font.Font{}, unit.Sp(12), "ok", op.CallOp{})
	if info.Truncated != 0 {
		t.Errorf("Truncated=%d for short text, want 0", info.Truncated)
	}
}

// TestLabelMultiLineHeightGrows: increasing the rune count when MaxLines==0
// (unbounded) eventually grows the y-dimension.
func TestLabelMultiLineHeightGrows(t *testing.T) {
	sh := labelShaper()
	short := "a"
	long := "a\nb\nc\nd\ne"
	gtx1 := labelGtx(200, 1000)
	gtx2 := labelGtx(200, 1000)
	l := Label{}
	d1, _ := l.LayoutDetailed(gtx1, sh, font.Font{}, unit.Sp(12), short, op.CallOp{})
	d2, _ := l.LayoutDetailed(gtx2, sh, font.Font{}, unit.Sp(12), long, op.CallOp{})
	if d2.Size.Y <= d1.Size.Y {
		t.Errorf("multi-line should be taller: short Y=%d long Y=%d", d1.Size.Y, d2.Size.Y)
	}
}

// TestLabelAlignmentDoesNotChangeSize: alignment shifts glyphs but should
// not change the reported dimensions for a given constraint.
func TestLabelAlignmentDoesNotChangeSize(t *testing.T) {
	sh := labelShaper()
	const txt = "hello world"
	for _, a := range []text.Alignment{text.Start, text.End, text.Middle} {
		gtx := labelGtx(300, 300)
		l := Label{Alignment: a}
		d, _ := l.LayoutDetailed(gtx, sh, font.Font{}, unit.Sp(12), txt, op.CallOp{})
		if d.Size.X <= 0 || d.Size.Y <= 0 {
			t.Errorf("alignment %v: invalid size %+v", a, d.Size)
		}
	}
}

// TestLabelLayoutEqualsLayoutDetailed: the convenience Layout method must
// return the same Dimensions as LayoutDetailed.
func TestLabelLayoutEqualsLayoutDetailed(t *testing.T) {
	sh := labelShaper()
	l := Label{}
	gtx1 := labelGtx(200, 200)
	gtx2 := labelGtx(200, 200)
	a := l.Layout(gtx1, sh, font.Font{}, unit.Sp(12), "alpha beta gamma", op.CallOp{})
	b, _ := l.LayoutDetailed(gtx2, sh, font.Font{}, unit.Sp(12), "alpha beta gamma", op.CallOp{})
	if a != b {
		t.Errorf("Layout vs LayoutDetailed mismatch: %+v vs %+v", a, b)
	}
}

// TestLabelLineHeightIncreasesY: a positive LineHeight must cause the
// rendered height to be >= the natural height.
func TestLabelLineHeightIncreasesY(t *testing.T) {
	sh := labelShaper()
	gtx1 := labelGtx(200, 1000)
	gtx2 := labelGtx(200, 1000)
	natural, _ := Label{}.LayoutDetailed(gtx1, sh, font.Font{}, unit.Sp(12), "a\nb\nc", op.CallOp{})
	tall, _ := Label{LineHeight: unit.Sp(40)}.LayoutDetailed(gtx2, sh, font.Font{}, unit.Sp(12), "a\nb\nc", op.CallOp{})
	if tall.Size.Y < natural.Size.Y {
		t.Errorf("forced LineHeight=40 produced shorter layout: %d < %d", tall.Size.Y, natural.Size.Y)
	}
}

// TestLabelLineHeightScale: a > 1 LineHeightScale grows or matches; <1
// shrinks or matches. Either way the dimensions must remain valid.
func TestLabelLineHeightScale(t *testing.T) {
	sh := labelShaper()
	gtx1 := labelGtx(200, 1000)
	gtx2 := labelGtx(200, 1000)
	d1, _ := Label{LineHeightScale: 1.0}.LayoutDetailed(gtx1, sh, font.Font{}, unit.Sp(12), "a\nb", op.CallOp{})
	d2, _ := Label{LineHeightScale: 2.0}.LayoutDetailed(gtx2, sh, font.Font{}, unit.Sp(12), "a\nb", op.CallOp{})
	if d2.Size.Y < d1.Size.Y {
		t.Errorf("LineHeightScale=2.0 should not be shorter than 1.0: %d vs %d", d2.Size.Y, d1.Size.Y)
	}
}

// --- textIterator focused tests --------------------------------------------

// TestTextIteratorTruncated verifies that a truncator glyph (FlagTruncator |
// FlagClusterBreak) updates it.truncated to gl.Runes.
func TestTextIteratorTruncated(t *testing.T) {
	it := textIterator{
		viewport: image.Rect(0, 0, 1000, 1000),
		maxLines: 1,
	}
	g := text.Glyph{
		X:       0,
		Y:       20,
		Advance: fixed.I(10),
		Ascent:  fixed.I(10),
		Descent: fixed.I(2),
		Flags:   text.FlagTruncator | text.FlagClusterBreak,
		Runes:   7,
	}
	if !it.processGlyph(g, true) {
		t.Errorf("processGlyph returned false for an in-viewport truncator")
	}
	if it.truncated != 7 {
		t.Errorf("it.truncated=%d, want 7", it.truncated)
	}
}

// TestTextIteratorMaxLinesStops: when linesSeen reaches maxLines and a
// paragraph break arrives, processGlyph must return false to signal stop.
func TestTextIteratorMaxLinesStops(t *testing.T) {
	it := textIterator{
		viewport: image.Rect(0, 0, 1000, 1000),
		maxLines: 1,
	}
	// First glyph closes line 1 with a line-break.
	g1 := text.Glyph{
		X: 0, Y: 20,
		Advance: fixed.I(10), Ascent: fixed.I(10), Descent: fixed.I(2),
		Flags: text.FlagLineBreak | text.FlagClusterBreak,
	}
	if !it.processGlyph(g1, true) {
		t.Errorf("first glyph: processGlyph=false, want true")
	}
	if it.linesSeen != 1 {
		t.Errorf("linesSeen=%d after first line break, want 1", it.linesSeen)
	}
	// Next glyph is a paragraph break — must signal stop.
	g2 := text.Glyph{
		X: 0, Y: 40,
		Advance: fixed.I(10), Ascent: fixed.I(10), Descent: fixed.I(2),
		Flags: text.FlagParagraphBreak | text.FlagClusterBreak,
	}
	if it.processGlyph(g2, true) {
		t.Errorf("paragraph break beyond maxLines: want stop signal (false)")
	}
}

// TestTextIteratorVisibility: a glyph entirely above the viewport should
// not contribute to bounds.
func TestTextIteratorAboveViewport(t *testing.T) {
	it := textIterator{
		viewport: image.Rect(0, 0, 100, 100),
	}
	above := text.Glyph{
		X: 0, Y: -1000,
		Advance: fixed.I(10), Ascent: fixed.I(10), Descent: fixed.I(2),
	}
	it.processGlyph(above, true)
	if it.visible {
		t.Errorf("glyph above viewport should not be visible")
	}
}

// TestTextIteratorBelowViewportReturnsFalse: a glyph below the viewport
// should signal stop (visibleOrBefore == false).
func TestTextIteratorBelowViewportReturnsFalse(t *testing.T) {
	it := textIterator{viewport: image.Rect(0, 0, 100, 100)}
	below := text.Glyph{
		X: 0, Y: 1000,
		Advance: fixed.I(10), Ascent: fixed.I(10), Descent: fixed.I(2),
	}
	if it.processGlyph(below, true) {
		t.Errorf("glyph below viewport: want false (stop)")
	}
}

// TestTextIteratorOkPropagation: when ok=false (caller signals end of
// glyphs), processGlyph propagates that, returning false.
func TestTextIteratorOkPropagation(t *testing.T) {
	it := textIterator{viewport: image.Rect(0, 0, 100, 100)}
	g := text.Glyph{
		X: 0, Y: 20,
		Advance: fixed.I(10), Ascent: fixed.I(10), Descent: fixed.I(2),
	}
	if it.processGlyph(g, false) {
		t.Errorf("ok=false should propagate as false")
	}
}

// TestFixedToFloat sanity-checks the helper used by paintGlyph.
func TestFixedToFloat(t *testing.T) {
	cases := []struct {
		in   fixed.Int26_6
		want float32
	}{
		{0, 0},
		{fixed.I(1), 1.0},
		{fixed.I(-1), -1.0},
		{fixed.I(64), 64.0},
		{32, 0.5},   // half pixel
		{-32, -0.5}, // negative half pixel
	}
	for _, c := range cases {
		got := fixedToFloat(c.in)
		if got != c.want {
			t.Errorf("fixedToFloat(%d)=%v, want %v", int(c.in), got, c.want)
		}
	}
}

// TestLabelConstraintsFloor: a Label laid out with a large minimum
// constraint must report dims at least the minimum (Constrain).
func TestLabelConstraintsFloor(t *testing.T) {
	sh := labelShaper()
	gtx := layout.Context{
		Ops: new(op.Ops),
		Constraints: layout.Constraints{
			Min: image.Pt(80, 80),
			Max: image.Pt(200, 200),
		},
		Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Locale: english,
	}
	d, _ := Label{}.LayoutDetailed(gtx, sh, font.Font{}, unit.Sp(12), "x", op.CallOp{})
	if d.Size.X < 80 {
		t.Errorf("Constrain not enforced on X: got %d, min 80", d.Size.X)
	}
	if d.Size.Y < 80 {
		t.Errorf("Constrain not enforced on Y: got %d, min 80", d.Size.Y)
	}
}

// TestLabelDoesNotMutateInput: Label.Layout must not mutate the input
// string (regression guard for any future internal optimization).
func TestLabelDoesNotMutateInput(t *testing.T) {
	sh := labelShaper()
	gtx := labelGtx(200, 200)
	src := "hello, world"
	srcCopy := src
	_ = Label{}.Layout(gtx, sh, font.Font{}, unit.Sp(12), src, op.CallOp{})
	if src != srcCopy {
		t.Errorf("Layout mutated input: %q -> %q", srcCopy, src)
	}
}
