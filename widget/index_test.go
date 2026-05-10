package widget

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"math"
	"os"
	"testing"

	nsareg "eliasnaur.com/font/noto/sans/arabic/regular"
	"github.com/nanorele/gio/font"
	"github.com/nanorele/gio/font/opentype"
	"github.com/nanorele/gio/gpu/headless"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/text"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"
)

func makePosTestText(fontSize, lineWidth int, alignOpposite bool) (shaper *text.Shaper, source string, bidiLTR, bidiRTL []text.Glyph) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	rtlFace, _ := opentype.Parse(nsareg.TTF)

	shaper = text.NewShaper(text.NoSystemFonts(), text.WithCollection([]font.FontFace{
		{
			Font: font.Font{Typeface: "LTR"},
			Face: ltrFace,
		},
		{
			Font: font.Font{Typeface: "RTL"},
			Face: rtlFace,
		},
	}))

	bidiSource := "The quick سماء שלום لا fox تمط שלום غير the lazy dog."
	ltrParams := text.Parameters{
		PxPerEm:  fixed.I(fontSize),
		MaxWidth: lineWidth,
		MinWidth: lineWidth,
		Locale:   english,
	}
	rtlParams := text.Parameters{
		Alignment: text.End,
		PxPerEm:   fixed.I(fontSize),
		MaxWidth:  lineWidth,
		MinWidth:  lineWidth,
		Locale:    arabic,
	}
	if alignOpposite {
		ltrParams.Alignment = text.End
		rtlParams.Alignment = text.Start
	}
	shaper.LayoutString(ltrParams, bidiSource)
	for g, ok := shaper.NextGlyph(); ok; g, ok = shaper.NextGlyph() {
		bidiLTR = append(bidiLTR, g)
	}
	shaper.LayoutString(rtlParams, bidiSource)
	for g, ok := shaper.NextGlyph(); ok; g, ok = shaper.NextGlyph() {
		bidiRTL = append(bidiRTL, g)
	}
	return shaper, bidiSource, bidiLTR, bidiRTL
}

func makeAccountingTestText(str string, fontSize, lineWidth int) (txt []text.Glyph) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	rtlFace, _ := opentype.Parse(nsareg.TTF)

	shaper := text.NewShaper(text.NoSystemFonts(), text.WithCollection([]font.FontFace{
		{
			Font: font.Font{Typeface: "LTR"},
			Face: ltrFace,
		},
		{
			Font: font.Font{Typeface: "RTL"},
			Face: rtlFace,
		},
	}))
	params := text.Parameters{
		PxPerEm:  fixed.I(fontSize),
		MaxWidth: lineWidth,
		Locale:   english,
	}
	shaper.LayoutString(params, str)
	for g, ok := shaper.NextGlyph(); ok; g, ok = shaper.NextGlyph() {
		txt = append(txt, g)
	}
	return txt
}

func getGlyphs(fontSize, minWidth, lineWidth int, align text.Alignment, str string) (txt []text.Glyph) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	rtlFace, _ := opentype.Parse(nsareg.TTF)

	shaper := text.NewShaper(text.NoSystemFonts(), text.WithCollection([]font.FontFace{
		{
			Font: font.Font{Typeface: "LTR"},
			Face: ltrFace,
		},
		{
			Font: font.Font{Typeface: "RTL"},
			Face: rtlFace,
		},
	}))
	params := text.Parameters{
		PxPerEm:    fixed.I(fontSize),
		Alignment:  align,
		MinWidth:   minWidth,
		MaxWidth:   lineWidth,
		Locale:     english,
		WrapPolicy: text.WrapWords,
	}
	shaper.LayoutString(params, str)
	for g, ok := shaper.NextGlyph(); ok; g, ok = shaper.NextGlyph() {
		txt = append(txt, g)
	}
	return txt
}

func TestIndexPositionWhitespace(t *testing.T) {
	type testcase struct {
		name      string
		str       string
		lineWidth int
		align     text.Alignment
		expected  []combinedPos
	}
	for _, tc := range []testcase{
		{
			name:      "empty string",
			str:       "",
			lineWidth: 200,
			expected: []combinedPos{
				{x: fixed.Int26_6(0), y: 16, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216)},
			},
		},
		{
			name:      "just hard newline",
			str:       "\n",
			lineWidth: 200,
			expected: []combinedPos{
				{x: fixed.Int26_6(0), y: 16, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216)},
				{x: fixed.Int26_6(0), y: 35, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), runes: 1, lineCol: screenPos{line: 1}},
			},
		},
		{
			name:      "trailing newline",
			str:       "a\n",
			lineWidth: 200,
			expected: []combinedPos{
				{x: fixed.Int26_6(0), y: 16, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216)},
				{x: fixed.Int26_6(570), y: 16, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), runes: 1, lineCol: screenPos{col: 1}},
				{x: fixed.Int26_6(0), y: 35, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), runes: 2, lineCol: screenPos{line: 1}},
			},
		},
		{
			name:      "just blank line",
			str:       "\n\n",
			lineWidth: 200,
			expected: []combinedPos{
				{x: fixed.Int26_6(0), y: 16, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216)},
				{x: fixed.Int26_6(0), y: 35, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), runes: 1, lineCol: screenPos{line: 1}},
				{x: fixed.Int26_6(0), y: 54, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), runes: 2, lineCol: screenPos{line: 2}},
			},
		},
		{
			name:      "middle aligned blank lines",
			str:       "\n\n\nabc",
			align:     text.Middle,
			lineWidth: 200,
			expected: []combinedPos{
				{x: fixed.Int26_6(832), y: 16, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216)},
				{x: fixed.Int26_6(832), y: 35, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), runes: 1, lineCol: screenPos{line: 1}},
				{x: fixed.Int26_6(832), y: 54, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), runes: 2, lineCol: screenPos{line: 2}},
				{x: fixed.Int26_6(6), y: 73, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), runes: 3, lineCol: screenPos{line: 3}},
				{x: fixed.Int26_6(576), y: 73, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), runes: 4, lineCol: screenPos{line: 3, col: 1}},
				{x: fixed.Int26_6(1146), y: 73, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), runes: 5, lineCol: screenPos{line: 3, col: 2}},
				{x: fixed.Int26_6(1658), y: 73, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), runes: 6, lineCol: screenPos{line: 3, col: 3}},
			},
		},
		{
			name:      "blank line",
			str:       "a\n\nb",
			lineWidth: 200,
			expected: []combinedPos{
				{x: fixed.Int26_6(0), y: 16, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216)},
				{x: fixed.Int26_6(570), y: 16, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), runes: 1, lineCol: screenPos{col: 1}},
				{x: fixed.Int26_6(0), y: 35, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), runes: 2, lineCol: screenPos{line: 1}},
				{x: fixed.Int26_6(0), y: 54, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), runes: 3, lineCol: screenPos{line: 2}},
				{x: fixed.Int26_6(570), y: 54, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), runes: 4, lineCol: screenPos{line: 2, col: 1}},
			},
		},
		{
			name:      "soft wrap",
			str:       "abc def",
			lineWidth: 30,
			expected: []combinedPos{
				{runes: 0, lineCol: screenPos{line: 0, col: 0}, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), x: 0, y: 16},
				{runes: 1, lineCol: screenPos{line: 0, col: 1}, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), x: 570, y: 16},
				{runes: 2, lineCol: screenPos{line: 0, col: 2}, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), x: 1140, y: 16},
				{runes: 3, lineCol: screenPos{line: 0, col: 3}, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), x: 1652, y: 16},
				{runes: 4, lineCol: screenPos{line: 1, col: 0}, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), x: 0, y: 35},
				{runes: 5, lineCol: screenPos{line: 1, col: 1}, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), x: 570, y: 35},
				{runes: 6, lineCol: screenPos{line: 1, col: 2}, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), x: 1140, y: 35},
				{runes: 7, lineCol: screenPos{line: 1, col: 3}, ascent: fixed.Int26_6(968), descent: fixed.Int26_6(216), x: 1425, y: 35},
			},
		},
		{
			name:      "soft wrap arabic",
			str:       "ثنائي الاتجاه",
			lineWidth: 30,
			expected: []combinedPos{
				{runes: 0, lineCol: screenPos{line: 0, col: 0}, ascent: 1407, descent: 756, x: 2250, y: 22, towardOrigin: true},
				{runes: 1, lineCol: screenPos{line: 0, col: 1}, ascent: 1407, descent: 756, x: 1944, y: 22, towardOrigin: true},
				{runes: 2, lineCol: screenPos{line: 0, col: 2}, ascent: 1407, descent: 756, x: 1593, y: 22, towardOrigin: true},
				{runes: 3, lineCol: screenPos{line: 0, col: 3}, ascent: 1407, descent: 756, x: 1295, y: 22, towardOrigin: true},
				{runes: 4, lineCol: screenPos{line: 0, col: 4}, ascent: 1407, descent: 756, x: 1020, y: 22, towardOrigin: true},
				{runes: 5, lineCol: screenPos{line: 0, col: 5}, ascent: 1407, descent: 756, x: 266, y: 22, towardOrigin: true},
				{runes: 6, lineCol: screenPos{line: 1, col: 0}, ascent: 1407, descent: 756, x: 2511, y: 41, towardOrigin: true},
				{runes: 7, lineCol: screenPos{line: 1, col: 1}, ascent: 1407, descent: 756, x: 2267, y: 41, towardOrigin: true},
				{runes: 8, lineCol: screenPos{line: 1, col: 2}, ascent: 1407, descent: 756, x: 1969, y: 41, towardOrigin: true},
				{runes: 9, lineCol: screenPos{line: 1, col: 3}, ascent: 1407, descent: 756, x: 1671, y: 41, towardOrigin: true},
				{runes: 10, lineCol: screenPos{line: 1, col: 4}, ascent: 1407, descent: 756, x: 1365, y: 41, towardOrigin: true},
				{runes: 11, lineCol: screenPos{line: 1, col: 5}, ascent: 1407, descent: 756, x: 713, y: 41, towardOrigin: true},
				{runes: 12, lineCol: screenPos{line: 1, col: 6}, ascent: 1407, descent: 756, x: 415, y: 41, towardOrigin: true},
				{runes: 13, lineCol: screenPos{line: 1, col: 7}, ascent: 1407, descent: 756, x: 0, y: 41, towardOrigin: true},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			glyphs := getGlyphs(16, 0, tc.lineWidth, tc.align, tc.str)
			var gi glyphIndex
			gi.reset()
			for _, g := range glyphs {
				gi.Glyph(g)
			}
			if len(gi.positions) != len(tc.expected) {
				t.Errorf("expected %d positions, got %d", len(tc.expected), len(gi.positions))
			}
			for i := range min(len(gi.positions), len(tc.expected)) {
				actual := gi.positions[i]
				expected := tc.expected[i]
				if actual != expected {
					t.Errorf("position %d: expected:\n%#+v, got:\n%#+v", i, expected, actual)
				}
			}
			if t.Failed() {
				printPositions(t, gi.positions)
				printGlyphs(t, glyphs)
			}
		})
	}
}

func TestIndexPositionBidi(t *testing.T) {
	fontSize := 16
	lineWidth := fontSize * 10
	shaper, _, bidiLTRText, bidiRTLText := makePosTestText(fontSize, lineWidth, false)
	type testcase struct {
		name       string
		glyphs     []text.Glyph
		expectedXs []fixed.Int26_6
	}
	for _, tc := range []testcase{
		{
			name:   "bidi ltr",
			glyphs: bidiLTRText,
			expectedXs: []fixed.Int26_6{
				0, 626, 1196, 1766, 2051, 2621, 3191, 3444, 3956, 4468, 4753, 7133, 6330, 5738, 5440, 5019,

				3953, 3185, 2417, 1649, 881, 596, 298, 0, 3953, 4238, 4523, 5093, 5605, 5890, 7905, 7599, 7007, 6156,

				4660, 3892, 3124, 2356, 1588, 1303, 788, 406, 0, 4660, 4945, 5235, 5805, 6375, 6660, 6934, 7504, 8016, 8528,

				0, 570, 1140, 1710, 2034,
			},
		},
		{
			name:   "bidi rtl",
			glyphs: bidiRTLText,
			expectedXs: []fixed.Int26_6{

				5718, 6344, 6914, 7484, 7769, 8339, 8909, 9162, 9674, 10186, 5718, 5452, 4649, 4057, 3759, 3338, 3072, 2304, 1536, 768, 0,

				9170, 8872, 8574, 8308, 6941, 7226, 7796, 8308, 6941, 6675, 6369, 5777, 4926, 4660, 3892, 3124, 2356, 1588, 1303, 788, 406, 0,

				324, 614, 1184, 1754, 2039, 2313, 2883, 3395, 3907, 4192, 4762, 5332, 5902, 324, 0,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gi glyphIndex
			gi.reset()
			for _, g := range tc.glyphs {
				gi.Glyph(g)
			}
			if len(gi.positions) != len(tc.expectedXs) {
				t.Errorf("expected %d positions, got %d", len(tc.expectedXs), len(gi.positions))
			}
			lastRunes := 0
			lastLine := 0
			lastCol := -1
			lastY := 0
			for i := range min(len(gi.positions), len(tc.expectedXs)) {
				actualX := gi.positions[i].x
				expectedX := tc.expectedXs[i]
				if actualX != expectedX {
					t.Errorf("position %d: expected x=%v(%d), got x=%v(%d)", i, expectedX, expectedX, actualX, actualX)
				}
				if r := gi.positions[i].runes; r < lastRunes {
					t.Errorf("position %d: expected runes >= %d, got %d", i, lastRunes, r)
				}
				lastRunes = gi.positions[i].runes
				if y := gi.positions[i].y; y < lastY {
					t.Errorf("position %d: expected y>= %d, got %d", i, lastY, y)
				}
				lastY = gi.positions[i].y
				if y := gi.positions[i].y; y < lastY {
					t.Errorf("position %d: expected y>= %d, got %d", i, lastY, y)
				}
				lastY = gi.positions[i].y
				if lineCol := gi.positions[i].lineCol; lineCol.line == lastLine && lineCol.col < lastCol {
					t.Errorf("position %d: expected col >= %d, got %d", i, lastCol, lineCol.col)
				}
				lastCol = gi.positions[i].lineCol.col
				if line := gi.positions[i].lineCol.line; line < lastLine {
					t.Errorf("position %d: expected line >= %d, got %d", i, lastLine, line)
				}
				lastLine = gi.positions[i].lineCol.line
			}
			printPositions(t, gi.positions)
			if t.Failed() {
				printGlyphs(t, tc.glyphs)
				width := lineWidth
				height := 100
				cap := image.NewRGBA(image.Rect(0, 0, width, height))
				w, _ := headless.NewWindow(width, height)
				defer w.Release()
				ops := new(op.Ops)
				gtx := layout.Context{
					Constraints: layout.Constraints{Max: image.Pt(width, height)},
					Ops:         ops,
				}
				it := textIterator{viewport: image.Rectangle{Max: image.Point{X: width, Y: height}}}
				for _, g := range tc.glyphs {
					it.processGlyph(g, true)
				}
				var glyphs [32]text.Glyph
				line := glyphs[:0]
				for _, g := range gi.glyphs {
					var ok bool
					if line, ok = it.paintGlyph(gtx, shaper, g, line); !ok {
						break
					}
				}
				w.Frame(ops)        //nolint:errcheck // Test diagnostic; Frame errors not actionable.
				w.Screenshot(cap)   //nolint:errcheck // Test diagnostic; Screenshot errors not actionable.
				b := new(bytes.Buffer)
				_ = png.Encode(b, cap)
				screenshotName := tc.name + ".png"
				_ = os.WriteFile(screenshotName, b.Bytes(), 0o644)
				t.Logf("wrote %q", screenshotName)
			}
		})
	}
}

func TestIndexPositionLines(t *testing.T) {
	fontSize := 16
	lineWidth := fontSize * 10
	_, source1, bidiLTRText, bidiRTLText := makePosTestText(fontSize, lineWidth, false)
	_, source2, bidiLTRTextOpp, bidiRTLTextOpp := makePosTestText(fontSize, lineWidth, true)
	type testcase struct {
		name          string
		source        string
		glyphs        []text.Glyph
		expectedLines []lineInfo
	}
	for _, tc := range []testcase{
		{
			name:   "bidi ltr",
			source: source1,
			glyphs: bidiLTRText,
			expectedLines: []lineInfo{
				{
					xOff:    fixed.Int26_6(0),
					yOff:    22,
					glyphs:  15,
					width:   fixed.Int26_6(7133),
					ascent:  fixed.Int26_6(1407),
					descent: fixed.Int26_6(756),
				},
				{
					xOff:    fixed.Int26_6(0),
					yOff:    41,
					glyphs:  15,
					width:   fixed.Int26_6(7905),
					ascent:  fixed.Int26_6(1407),
					descent: fixed.Int26_6(756),
				},
				{
					xOff:    fixed.Int26_6(0),
					yOff:    60,
					glyphs:  18,
					width:   fixed.Int26_6(8528),
					ascent:  fixed.Int26_6(1407),
					descent: fixed.Int26_6(756),
				},
				{
					xOff:    fixed.Int26_6(0),
					yOff:    79,
					glyphs:  4,
					width:   fixed.Int26_6(2034),
					ascent:  fixed.Int26_6(968),
					descent: fixed.Int26_6(216),
				},
			},
		},
		{
			name:   "bidi rtl",
			source: source1,
			glyphs: bidiRTLText,
			expectedLines: []lineInfo{
				{
					xOff:    fixed.Int26_6(0),
					yOff:    22,
					glyphs:  20,
					width:   fixed.Int26_6(10186),
					ascent:  fixed.Int26_6(1407),
					descent: fixed.Int26_6(756),
				},
				{
					xOff:    fixed.Int26_6(0),
					yOff:    41,
					glyphs:  19,
					width:   fixed.Int26_6(9170),
					ascent:  fixed.Int26_6(1407),
					descent: fixed.Int26_6(756),
				},
				{
					xOff:    fixed.Int26_6(0),
					yOff:    60,
					glyphs:  13,
					width:   fixed.Int26_6(5902),
					ascent:  fixed.Int26_6(968),
					descent: fixed.Int26_6(216),
				},
			},
		},
		{
			name:   "bidi ltr opposite alignment",
			source: source2,
			glyphs: bidiLTRTextOpp,
			expectedLines: []lineInfo{
				{
					xOff:    fixed.Int26_6(3107),
					yOff:    22,
					glyphs:  15,
					width:   fixed.Int26_6(7133),
					ascent:  fixed.Int26_6(1407),
					descent: fixed.Int26_6(756),
				},
				{
					xOff:    fixed.Int26_6(2335),
					yOff:    41,
					glyphs:  15,
					width:   fixed.Int26_6(7905),
					ascent:  fixed.Int26_6(1407),
					descent: fixed.Int26_6(756),
				},
				{
					xOff:    fixed.Int26_6(1712),
					yOff:    60,
					glyphs:  18,
					width:   fixed.Int26_6(8528),
					ascent:  fixed.Int26_6(1407),
					descent: fixed.Int26_6(756),
				},
				{
					xOff:    fixed.Int26_6(8206),
					yOff:    79,
					glyphs:  4,
					width:   fixed.Int26_6(2034),
					ascent:  fixed.Int26_6(968),
					descent: fixed.Int26_6(216),
				},
			},
		},
		{
			name:   "bidi rtl opposite alignment",
			source: source2,
			glyphs: bidiRTLTextOpp,
			expectedLines: []lineInfo{
				{
					xOff:    fixed.Int26_6(54),
					yOff:    22,
					glyphs:  20,
					width:   fixed.Int26_6(10186),
					ascent:  fixed.Int26_6(1407),
					descent: fixed.Int26_6(756),
				},
				{
					xOff:    fixed.Int26_6(1070),
					yOff:    41,
					glyphs:  19,
					width:   fixed.Int26_6(9170),
					ascent:  fixed.Int26_6(1407),
					descent: fixed.Int26_6(756),
				},
				{
					xOff:    fixed.Int26_6(4338),
					yOff:    60,
					glyphs:  13,
					width:   fixed.Int26_6(5902),
					ascent:  fixed.Int26_6(968),
					descent: fixed.Int26_6(216),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gi glyphIndex
			gi.reset()
			for _, g := range tc.glyphs {
				gi.Glyph(g)
			}
			if len(gi.lines) != len(tc.expectedLines) {
				t.Errorf("expected %d lines, got %d", len(tc.expectedLines), len(gi.lines))
			}
			for i := range min(len(gi.lines), len(tc.expectedLines)) {
				actual := gi.lines[i]
				expected := tc.expectedLines[i]
				// Compare semantic fields only; posStart/posEnd are internal indices
				// tracked for fast line→positions lookup and are verified separately.
				if actual.xOff != expected.xOff || actual.yOff != expected.yOff ||
					actual.width != expected.width || actual.ascent != expected.ascent ||
					actual.descent != expected.descent || actual.glyphs != expected.glyphs {
					t.Errorf("line %d: expected:\n%#+v, got:\n%#+v", i, expected, actual)
				}
			}
			// Verify posStart/posEnd invariants: contiguous, non-decreasing, bounded.
			for i, ln := range gi.lines {
				if ln.posStart > ln.posEnd {
					t.Errorf("line %d: posStart (%d) > posEnd (%d)", i, ln.posStart, ln.posEnd)
				}
				if ln.posEnd > len(gi.positions) {
					t.Errorf("line %d: posEnd (%d) > len(positions) (%d)", i, ln.posEnd, len(gi.positions))
				}
				if i > 0 && ln.posStart != gi.lines[i-1].posEnd {
					t.Errorf("line %d: posStart (%d) != previous posEnd (%d)",
						i, ln.posStart, gi.lines[i-1].posEnd)
				}
			}
		})
	}
}

func TestIndexPositionRunes(t *testing.T) {
	fontSize := 16
	lineWidth := fontSize * 10

	source := "The\nquick سماء של\nום لا fox\nتمط של\nום."
	testText := makeAccountingTestText(source, fontSize, lineWidth)
	type testcase struct {
		name     string
		source   string
		glyphs   []text.Glyph
		expected []combinedPos
	}
	for _, tc := range []testcase{
		{
			name:   "many newlines",
			source: source,
			glyphs: testText,
			expected: []combinedPos{
				{runes: 0, lineCol: screenPos{line: 0, col: 0}, runIndex: 0, towardOrigin: false},
				{runes: 1, lineCol: screenPos{line: 0, col: 1}, runIndex: 0, towardOrigin: false},
				{runes: 2, lineCol: screenPos{line: 0, col: 2}, runIndex: 0, towardOrigin: false},
				{runes: 3, lineCol: screenPos{line: 0, col: 3}, runIndex: 0, towardOrigin: false},
				{runes: 4, lineCol: screenPos{line: 1, col: 0}, runIndex: 0, towardOrigin: false},
				{runes: 5, lineCol: screenPos{line: 1, col: 1}, runIndex: 0, towardOrigin: false},
				{runes: 6, lineCol: screenPos{line: 1, col: 2}, runIndex: 0, towardOrigin: false},
				{runes: 7, lineCol: screenPos{line: 1, col: 3}, runIndex: 0, towardOrigin: false},
				{runes: 8, lineCol: screenPos{line: 1, col: 4}, runIndex: 0, towardOrigin: false},
				{runes: 9, lineCol: screenPos{line: 1, col: 5}, runIndex: 0, towardOrigin: false},
				{runes: 10, lineCol: screenPos{line: 1, col: 6}, runIndex: 0, towardOrigin: false},
				{runes: 10, lineCol: screenPos{line: 1, col: 6}, runIndex: 1, towardOrigin: true},
				{runes: 11, lineCol: screenPos{line: 1, col: 7}, runIndex: 1, towardOrigin: true},
				{runes: 12, lineCol: screenPos{line: 1, col: 8}, runIndex: 1, towardOrigin: true},
				{runes: 13, lineCol: screenPos{line: 1, col: 9}, runIndex: 1, towardOrigin: true},
				{runes: 14, lineCol: screenPos{line: 1, col: 10}, runIndex: 1, towardOrigin: true},
				{runes: 15, lineCol: screenPos{line: 1, col: 11}, runIndex: 2, towardOrigin: true},
				{runes: 16, lineCol: screenPos{line: 1, col: 12}, runIndex: 2, towardOrigin: true},
				{runes: 17, lineCol: screenPos{line: 1, col: 13}, runIndex: 2, towardOrigin: true},
				{runes: 18, lineCol: screenPos{line: 2, col: 0}, runIndex: 0, towardOrigin: true},
				{runes: 19, lineCol: screenPos{line: 2, col: 1}, runIndex: 0, towardOrigin: true},
				{runes: 20, lineCol: screenPos{line: 2, col: 2}, runIndex: 0, towardOrigin: true},
				{runes: 21, lineCol: screenPos{line: 2, col: 3}, runIndex: 1, towardOrigin: true},
				{runes: 22, lineCol: screenPos{line: 2, col: 4}, runIndex: 1, towardOrigin: true},
				{runes: 23, lineCol: screenPos{line: 2, col: 5}, runIndex: 1, towardOrigin: true},
				{runes: 24, lineCol: screenPos{line: 2, col: 6}, runIndex: 1, towardOrigin: true},
				{runes: 24, lineCol: screenPos{line: 2, col: 6}, runIndex: 2, towardOrigin: false},
				{runes: 25, lineCol: screenPos{line: 2, col: 7}, runIndex: 2, towardOrigin: false},
				{runes: 26, lineCol: screenPos{line: 2, col: 8}, runIndex: 2, towardOrigin: false},
				{runes: 27, lineCol: screenPos{line: 2, col: 9}, runIndex: 2, towardOrigin: false},
				{runes: 28, lineCol: screenPos{line: 3, col: 0}, runIndex: 0, towardOrigin: true},
				{runes: 29, lineCol: screenPos{line: 3, col: 1}, runIndex: 0, towardOrigin: true},
				{runes: 30, lineCol: screenPos{line: 3, col: 2}, runIndex: 0, towardOrigin: true},
				{runes: 31, lineCol: screenPos{line: 3, col: 3}, runIndex: 0, towardOrigin: true},
				{runes: 32, lineCol: screenPos{line: 3, col: 4}, runIndex: 1, towardOrigin: true},
				{runes: 33, lineCol: screenPos{line: 3, col: 5}, runIndex: 1, towardOrigin: true},
				{runes: 34, lineCol: screenPos{line: 3, col: 6}, runIndex: 1, towardOrigin: true},
				{runes: 35, lineCol: screenPos{line: 4, col: 0}, runIndex: 0, towardOrigin: true},
				{runes: 36, lineCol: screenPos{line: 4, col: 1}, runIndex: 0, towardOrigin: true},
				{runes: 37, lineCol: screenPos{line: 4, col: 2}, runIndex: 0, towardOrigin: true},
				{runes: 38, lineCol: screenPos{line: 4, col: 3}, runIndex: 0, towardOrigin: true},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gi glyphIndex
			gi.reset()
			for _, g := range tc.glyphs {
				gi.Glyph(g)
			}
			if len(gi.positions) != len(tc.expected) {
				t.Errorf("expected %d positions, got %d", len(tc.expected), len(gi.positions))
			}
			for i := range min(len(gi.positions), len(tc.expected)) {
				actual := gi.positions[i]
				expected := tc.expected[i]
				if expected.runes != actual.runes {
					t.Errorf("position %d: expected runes=%d, got %d", i, expected.runes, actual.runes)
				}
				if expected.lineCol != actual.lineCol {
					t.Errorf("position %d: expected lineCol=%v, got %v", i, expected.lineCol, actual.lineCol)
				}
				if expected.runIndex != actual.runIndex {
					t.Errorf("position %d: expected runIndex=%d, got %d", i, expected.runIndex, actual.runIndex)
				}
				if expected.towardOrigin != actual.towardOrigin {
					t.Errorf("position %d: expected towardOrigin=%v, got %v", i, expected.towardOrigin, actual.towardOrigin)
				}
			}
			printPositions(t, gi.positions)
			if t.Failed() {
				printGlyphs(t, tc.glyphs)
			}
		})
	}
}

func printPositions(t *testing.T, positions []combinedPos) {
	t.Helper()
	for i, p := range positions {
		t.Logf("positions[%2d] = {runes: %2d, line: %2d, col: %2d, x: %5d, y: %3d}", i, p.runes, p.lineCol.line, p.lineCol.col, p.x, p.y)
	}
}

func printGlyphs(t *testing.T, glyphs []text.Glyph) {
	t.Helper()
	for i, g := range glyphs {
		t.Logf("glyphs[%2d] = {ID: 0x%013x, Flags: %4s, Advance: %4d(%6v), Runes: %d, Y: %3d, X: %4d(%6v)} ", i, g.ID, g.Flags, g.Advance, g.Advance, g.Runes, g.Y, g.X, g.X)
	}
}

func TestGraphemeReaderNext(t *testing.T) {
	latinDoc := bytes.NewReader([]byte(latinDocument))
	arabicDoc := bytes.NewReader([]byte(arabicDocument))
	emojiDoc := bytes.NewReader([]byte(emojiDocument))
	complexDoc := bytes.NewReader([]byte(complexDocument))
	type testcase struct {
		name  string
		input *bytes.Reader
		read  func() ([]rune, bool)
	}
	var pr graphemeReader
	for _, tc := range []testcase{
		{
			name:  "latin",
			input: latinDoc,
			read:  pr.next,
		},
		{
			name:  "arabic",
			input: arabicDoc,
			read:  pr.next,
		},
		{
			name:  "emoji",
			input: emojiDoc,
			read:  pr.next,
		},
		{
			name:  "complex",
			input: complexDoc,
			read:  pr.next,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pr.SetSource(tc.input)

			runes := []rune{}
			var paragraph []rune
			ok := true
			for ok {
				paragraph, ok = tc.read()
				for i, r := range paragraph {
					if i == len(paragraph)-1 {
						if r != '\n' && ok {
							t.Error("non-final paragraph does not end with newline")
						}
					} else if r == '\n' {
						t.Errorf("paragraph[%d] contains newline", i)
					}
				}
				runes = append(runes, paragraph...)
			}
			tc.input.Seek(0, 0) //nolint:errcheck // Seek on test in-memory reader cannot fail.
			b, _ := io.ReadAll(tc.input)
			asRunes := []rune(string(b))
			if len(asRunes) != len(runes) {
				t.Errorf("expected %d runes, got %d", len(asRunes), len(runes))
			}
			for i := range max(len(asRunes), len(runes)) {
				if i < min(len(asRunes), len(runes)) {
					if runes[i] != asRunes[i] {
						t.Errorf("expected runes[%d]=%d, got %d", i, asRunes[i], runes[i])
					}
				} else if i < len(asRunes) {
					t.Errorf("expected runes[%d]=%d, got nothing", i, asRunes[i])
				} else if i < len(runes) {
					t.Errorf("expected runes[%d]=nothing, got %d", i, runes[i])
				}
			}
		})
	}
}

func TestGraphemeReaderGraphemes(t *testing.T) {
	latinDoc := bytes.NewReader([]byte(latinDocument))
	arabicDoc := bytes.NewReader([]byte(arabicDocument))
	emojiDoc := bytes.NewReader([]byte(emojiDocument))
	complexDoc := bytes.NewReader([]byte(complexDocument))
	type testcase struct {
		name  string
		input *bytes.Reader
		read  func() []int
	}
	var pr graphemeReader
	for _, tc := range []testcase{
		{
			name:  "latin",
			input: latinDoc,
			read:  pr.Graphemes,
		},
		{
			name:  "arabic",
			input: arabicDoc,
			read:  pr.Graphemes,
		},
		{
			name:  "emoji",
			input: emojiDoc,
			read:  pr.Graphemes,
		},
		{
			name:  "complex",
			input: complexDoc,
			read:  pr.Graphemes,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pr.SetSource(tc.input)

			graphemes := []int{}
			for g := tc.read(); len(g) > 0; g = tc.read() {
				if len(graphemes) > 0 && g[0] != graphemes[len(graphemes)-1] {
					t.Errorf("expected first boundary in new paragraph %d to match final boundary in previous %d", g[0], graphemes[len(graphemes)-1])
				}
				if len(graphemes) > 0 {

					g = g[1:]
				}
				graphemes = append(graphemes, g...)
			}
			tc.input.Seek(0, 0) //nolint:errcheck // Seek on test in-memory reader cannot fail.
			b, _ := io.ReadAll(tc.input)
			asRunes := []rune(string(b))
			if len(asRunes)+1 < len(graphemes) {
				t.Errorf("expected <= %d graphemes, got %d", len(asRunes)+1, len(graphemes))
			}
			for i := range len(graphemes) - 1 {
				if graphemes[i] >= graphemes[i+1] {
					t.Errorf("graphemes[%d](%d) >= graphemes[%d](%d)", i, graphemes[i], i+1, graphemes[i+1])
				}
			}
		})
	}
}

func BenchmarkGraphemeReaderNext(b *testing.B) {
	latinDoc := bytes.NewReader([]byte(latinDocument))
	arabicDoc := bytes.NewReader([]byte(arabicDocument))
	emojiDoc := bytes.NewReader([]byte(emojiDocument))
	complexDoc := bytes.NewReader([]byte(complexDocument))
	type testcase struct {
		name  string
		input *bytes.Reader
		read  func() ([]rune, bool)
	}
	pr := &graphemeReader{}
	for _, tc := range []testcase{
		{
			name:  "latin",
			input: latinDoc,
			read:  pr.next,
		},
		{
			name:  "arabic",
			input: arabicDoc,
			read:  pr.next,
		},
		{
			name:  "emoji",
			input: emojiDoc,
			read:  pr.next,
		},
		{
			name:  "complex",
			input: complexDoc,
			read:  pr.next,
		},
	} {
		paragraph := make([]rune, 4096)
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				pr.SetSource(tc.input)

				ok := true
				for ok {
					paragraph, ok = tc.read()
					_ = paragraph
				}
				_ = paragraph
			}
		})
	}
}

func BenchmarkGraphemeReaderGraphemes(b *testing.B) {
	latinDoc := bytes.NewReader([]byte(latinDocument))
	arabicDoc := bytes.NewReader([]byte(arabicDocument))
	emojiDoc := bytes.NewReader([]byte(emojiDocument))
	complexDoc := bytes.NewReader([]byte(complexDocument))
	type testcase struct {
		name  string
		input *bytes.Reader
		read  func() []int
	}
	pr := &graphemeReader{}
	for _, tc := range []testcase{
		{
			name:  "latin",
			input: latinDoc,
			read:  pr.Graphemes,
		},
		{
			name:  "arabic",
			input: arabicDoc,
			read:  pr.Graphemes,
		},
		{
			name:  "emoji",
			input: emojiDoc,
			read:  pr.Graphemes,
		},
		{
			name:  "complex",
			input: complexDoc,
			read:  pr.Graphemes,
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				pr.SetSource(tc.input)
				for g := tc.read(); len(g) > 0; g = tc.read() {
					_ = g
				}
			}
		})
	}
}

// buildIndex feeds glyphs through glyphIndex.Glyph and returns the populated
// index. All methods/state of the resulting index are then available for
// direct inspection.
func buildIndex(glyphs []text.Glyph) *glyphIndex {
	gi := &glyphIndex{}
	gi.reset()
	for _, g := range glyphs {
		gi.Glyph(g)
	}
	return gi
}

// TestIndexEmpty exercises the methods of an empty (un-fed) glyphIndex.
// All accessors must be safe on an empty index; otherwise upstream code
// (closestToXY callers, locate) crashes with no glyphs.
func TestIndexEmpty(t *testing.T) {
	gi := &glyphIndex{}
	gi.reset()

	if got, idx := gi.closestToRune(0); (got != combinedPos{}) || idx != 0 {
		t.Errorf("empty closestToRune(0)=%+v,%d", got, idx)
	}
	if got := gi.closestToLineCol(screenPos{line: 5, col: 9}); (got != combinedPos{}) {
		t.Errorf("empty closestToLineCol=%+v", got)
	}
	if got, eol := gi.closestToXY(100, 50); (got != combinedPos{}) || eol {
		t.Errorf("empty closestToXY=%+v eol=%v", got, eol)
	}
	if !gi.atStartOfLine(combinedPos{}) {
		t.Errorf("empty atStartOfLine should be true")
	}
	if !gi.atEndOfLine(combinedPos{}) {
		t.Errorf("empty atEndOfLine should be true")
	}
	rects := gi.locate(image.Rect(0, 0, 100, 100), 0, 0, nil)
	if len(rects) != 0 {
		t.Errorf("empty locate: want 0 rects, got %d", len(rects))
	}
}

// TestIndexResetClearsState verifies that reusing a glyphIndex after reset()
// doesn't leak state from a prior layout. Important because production code
// reuses a single instance across re-layouts.
func TestIndexResetClearsState(t *testing.T) {
	first := getGlyphs(16, 0, 200, text.Start, "abc\ndef")
	gi := buildIndex(first)
	if len(gi.lines) == 0 || len(gi.positions) == 0 {
		t.Fatalf("first layout produced no state")
	}
	prevPosCap := cap(gi.positions)

	gi.reset()
	if len(gi.glyphs) != 0 || len(gi.positions) != 0 || len(gi.lines) != 0 {
		t.Fatal("reset did not clear slices")
	}
	if gi.currentLineMin != 0 || gi.currentLineMax != 0 || gi.currentLineGlyphs != 0 ||
		gi.currentLinePosStart != 0 || gi.pos != (combinedPos{}) || gi.prog != 0 ||
		gi.clusterAdvance != 0 || gi.truncated || gi.midCluster {
		t.Fatalf("reset left scalar state dirty: %+v", gi)
	}
	if cap(gi.positions) != prevPosCap {
		t.Errorf("reset reallocated positions slice (cap %d -> %d)", prevPosCap, cap(gi.positions))
	}

	// Second layout should produce identical positions to a fresh index.
	second := getGlyphs(16, 0, 200, text.Start, "x")
	for _, g := range second {
		gi.Glyph(g)
	}
	fresh := buildIndex(second)
	if len(gi.positions) != len(fresh.positions) {
		t.Fatalf("after reset+relayout: positions len %d, fresh %d", len(gi.positions), len(fresh.positions))
	}
	for i := range gi.positions {
		if gi.positions[i] != fresh.positions[i] {
			t.Errorf("position[%d] reused=%+v fresh=%+v", i, gi.positions[i], fresh.positions[i])
		}
	}
}

// TestIndexEnsureCapacity validates the pre-allocation hint logic.
func TestIndexEnsureCapacity(t *testing.T) {
	gi := &glyphIndex{}
	gi.ensureCapacity(0)
	if cap(gi.glyphs) != 0 || cap(gi.positions) != 0 || cap(gi.lines) != 0 {
		t.Errorf("hint <=0 should not allocate; got caps g=%d p=%d l=%d",
			cap(gi.glyphs), cap(gi.positions), cap(gi.lines))
	}
	gi.ensureCapacity(-5)
	if cap(gi.glyphs) != 0 {
		t.Errorf("negative hint should not allocate")
	}

	gi.ensureCapacity(100)
	if cap(gi.glyphs) < 100 {
		t.Errorf("glyph cap %d < hint 100", cap(gi.glyphs))
	}
	if cap(gi.positions) < 125 {
		t.Errorf("position cap %d < expected 125", cap(gi.positions))
	}
	if cap(gi.lines) < 1 {
		t.Errorf("line cap %d < 1", cap(gi.lines))
	}

	// Idempotent: smaller hint must not shrink existing capacity.
	prevG, prevP, prevL := cap(gi.glyphs), cap(gi.positions), cap(gi.lines)
	gi.ensureCapacity(10)
	if cap(gi.glyphs) != prevG || cap(gi.positions) != prevP || cap(gi.lines) != prevL {
		t.Errorf("ensureCapacity shrunk caps: g %d->%d p %d->%d l %d->%d",
			prevG, cap(gi.glyphs), prevP, cap(gi.positions), prevL, cap(gi.lines))
	}
}

// TestIndexClosestToRune covers the binary-search behavior on the runes axis,
// including out-of-range queries that must clamp instead of panicking.
func TestIndexClosestToRune(t *testing.T) {
	glyphs := getGlyphs(16, 0, 200, text.Start, "abc\ndef")
	gi := buildIndex(glyphs)

	if len(gi.positions) < 2 {
		t.Fatalf("expected positions, got %d", len(gi.positions))
	}

	// Exact hit at runes=0 must return the first position.
	got, idx := gi.closestToRune(0)
	if idx != 0 || got.runes != 0 {
		t.Errorf("closestToRune(0): got idx=%d runes=%d", idx, got.runes)
	}

	// runes far past the end clamps to the last position.
	last := gi.positions[len(gi.positions)-1]
	got, idx = gi.closestToRune(99999)
	if idx != len(gi.positions)-1 || got != last {
		t.Errorf("closestToRune(huge): got idx=%d runes=%d, want last (idx=%d)",
			idx, got.runes, len(gi.positions)-1)
	}

	// Negative rune index returns the first position (sort.Search returns 0).
	got, idx = gi.closestToRune(-1)
	if idx != 0 || got != gi.positions[0] {
		t.Errorf("closestToRune(-1): got idx=%d, want 0", idx)
	}

	// Every queried rune must satisfy the binary-search invariant
	// (returned position has runes >= queried, or it's the last one).
	for r := 0; r <= last.runes+2; r++ {
		got, idx = gi.closestToRune(r)
		if idx == len(gi.positions)-1 {
			continue // clamped at the end
		}
		if got.runes < r {
			t.Errorf("closestToRune(%d): returned runes=%d < query", r, got.runes)
		}
	}
}

// TestIndexClosestToLineCol checks line/column lookups.
func TestIndexClosestToLineCol(t *testing.T) {
	glyphs := getGlyphs(16, 0, 200, text.Start, "ab\ncde\nf")
	gi := buildIndex(glyphs)

	// Beginning of each line must land on a column-0 position.
	for line := 0; line < len(gi.lines); line++ {
		got := gi.closestToLineCol(screenPos{line: line, col: 0})
		if got.lineCol.line != line || got.lineCol.col != 0 {
			t.Errorf("line %d col 0: got %+v", line, got.lineCol)
		}
	}

	// A column past the end of the line should clamp to the rightmost
	// position on that line (not jump to the next line silently).
	got := gi.closestToLineCol(screenPos{line: 0, col: math.MaxInt})
	if got.lineCol.line != 0 {
		t.Errorf("col=MaxInt on line 0 jumped to line %d", got.lineCol.line)
	}

	// Beyond the last line clamps to the last position.
	got = gi.closestToLineCol(screenPos{line: 999, col: 999})
	last := gi.positions[len(gi.positions)-1]
	if got != last {
		t.Errorf("line=999: got %+v, want last %+v", got, last)
	}
}

// TestIndexClosestToXY tests the spatial query in different regions.
func TestIndexClosestToXY(t *testing.T) {
	// Multi-line layout so we exercise vertical + horizontal search.
	glyphs := getGlyphs(16, 0, 200, text.Start, "abc\ndef\nghi")
	gi := buildIndex(glyphs)

	// y way above the first line returns the first position.
	got, eol := gi.closestToXY(0, -10000)
	if got.lineCol.line != 0 {
		t.Errorf("y<<min: returned line %d, want 0", got.lineCol.line)
	}
	_ = eol

	// y way below the last line returns the last position.
	got, _ = gi.closestToXY(0, 1<<30)
	if got != gi.positions[len(gi.positions)-1] {
		t.Errorf("y>>max: got %+v, want last", got)
	}

	// x slightly right of the rightmost glyph on a line should snap
	// to either the line-end position or the start of the next line.
	if len(gi.lines) >= 2 {
		line := gi.lines[0]
		// Use a y inside the first line.
		got, _ = gi.closestToXY(line.getLineEnd()+fixed.I(50), gi.positions[0].y)
		if got.lineCol.line != 0 && got.lineCol.line != 1 {
			t.Errorf("x past end of line 0: got line %d", got.lineCol.line)
		}
	}
}

// TestAtEndOfLine_Bidi exposes a bug in atEndOfLine (and atStartOfLine):
// they index g.positions with pos.runes as if it were a position index,
// but with bidi/truncation a single rune index can map to multiple positions
// or vice-versa. With BIDI text, runes is no longer in lockstep with the
// position index.
func TestAtEndOfLine_Bidi(t *testing.T) {
	fontSize := 16
	source := "The\nquick سماء של\nום لا fox\nتمط של\nום."
	glyphs := makeAccountingTestText(source, fontSize, fontSize*10)
	gi := buildIndex(glyphs)

	// Walk through every position and verify that atEndOfLine returns true
	// only for positions that are actually the rightmost-in-line (in their
	// linear order) — i.e. the next position (by index) is on a later line.
	for i, p := range gi.positions {
		want := i == len(gi.positions)-1 || gi.positions[i+1].lineCol.line > p.lineCol.line
		got := gi.atEndOfLine(p)
		if got != want {
			t.Errorf("pos[%d] runes=%d line=%d col=%d: atEndOfLine=%v want=%v",
				i, p.runes, p.lineCol.line, p.lineCol.col, got, want)
		}
	}
}

// TestAtStartOfLine_Bidi mirrors the above for atStartOfLine.
func TestAtStartOfLine_Bidi(t *testing.T) {
	fontSize := 16
	source := "The\nquick سماء של\nום لا fox\nتمط של\nום."
	glyphs := makeAccountingTestText(source, fontSize, fontSize*10)
	gi := buildIndex(glyphs)

	for i, p := range gi.positions {
		want := i == 0 || gi.positions[i-1].lineCol.line < p.lineCol.line
		got := gi.atStartOfLine(p)
		if got != want {
			t.Errorf("pos[%d] runes=%d line=%d col=%d: atStartOfLine=%v want=%v",
				i, p.runes, p.lineCol.line, p.lineCol.col, got, want)
		}
	}
}

// TestIncrementPositionWalk ensures incrementPosition can walk every
// position front-to-back and only signals eof at the last position.
func TestIncrementPositionWalk(t *testing.T) {
	glyphs := getGlyphs(16, 0, 200, text.Start, "ab\ncd")
	gi := buildIndex(glyphs)
	if len(gi.positions) < 2 {
		t.Fatalf("want >=2 positions, got %d", len(gi.positions))
	}
	cur := gi.positions[0]
	for i := 0; i < len(gi.positions); i++ {
		next, eof := gi.incrementPosition(cur)
		if i == len(gi.positions)-1 {
			if !eof {
				t.Errorf("step %d: expected eof at last", i)
			}
			break
		}
		if eof {
			t.Errorf("step %d: unexpected eof", i)
			break
		}
		if next == cur {
			t.Errorf("step %d: next == cur, would infinite loop", i)
			break
		}
		cur = next
	}
}

// TestLocateBasic checks that locate returns sensible rectangles for a
// single-line selection and that out-of-order rune args are normalized.
func TestLocateBasic(t *testing.T) {
	glyphs := getGlyphs(16, 0, 200, text.Start, "hello")
	gi := buildIndex(glyphs)

	viewport := image.Rect(0, 0, 1000, 1000)
	rects := gi.locate(viewport, 0, 5, nil)
	if len(rects) != 1 {
		t.Fatalf("want 1 rect, got %d", len(rects))
	}
	r := rects[0]
	if r.Bounds.Min.X >= r.Bounds.Max.X {
		t.Errorf("rect has zero/negative width: %+v", r.Bounds)
	}
	if r.Bounds.Min.Y >= r.Bounds.Max.Y {
		t.Errorf("rect has zero/negative height: %+v", r.Bounds)
	}

	// Reversed range: locate must swap and produce the same result.
	rects2 := gi.locate(viewport, 5, 0, nil)
	if len(rects2) != len(rects) || rects2[0] != rects[0] {
		t.Errorf("reversed range produced different rects: %+v vs %+v", rects2, rects)
	}

	// Empty selection at runes=0: still returns one rect (start==end).
	rects3 := gi.locate(viewport, 0, 0, nil)
	if len(rects3) != 1 {
		t.Errorf("empty selection: want 1 rect, got %d", len(rects3))
	}
}

// TestLocateMultiLine ensures locate spans multiple lines correctly.
func TestLocateMultiLine(t *testing.T) {
	// 3 hard lines.
	glyphs := getGlyphs(16, 0, 200, text.Start, "abc\ndef\nghi")
	gi := buildIndex(glyphs)
	viewport := image.Rect(0, 0, 1000, 1000)

	// Select across all three lines: produces at least 3 rects.
	rects := gi.locate(viewport, 0, 11, nil)
	if len(rects) < 3 {
		t.Errorf("want >=3 rects across 3 lines, got %d", len(rects))
	}
	// Each rect must lie inside the (translated) viewport in y.
	for i, r := range rects {
		if r.Bounds.Min.Y > viewport.Max.Y || r.Bounds.Max.Y < viewport.Min.Y-viewport.Max.Y {
			t.Errorf("rect %d outside viewport: %+v", i, r.Bounds)
		}
	}
}

// TestLocateRespectsViewport: rects far outside the viewport are skipped.
func TestLocateRespectsViewport(t *testing.T) {
	glyphs := getGlyphs(16, 0, 200, text.Start, "a\nb\nc\nd\ne\nf")
	gi := buildIndex(glyphs)

	// A 1-pixel-tall viewport at y=0 should yield very few rects.
	viewport := image.Rect(0, 0, 1000, 1)
	rects := gi.locate(viewport, 0, 11, nil)
	allLines := gi.locate(image.Rect(0, 0, 1000, 100000), 0, 11, nil)
	if len(rects) >= len(allLines) {
		t.Errorf("tight viewport produced %d rects, full %d (expected fewer)",
			len(rects), len(allLines))
	}
}

// TestLineInfoPosBounds verifies the posStart/posEnd indices written into
// lineInfo are always within bounds and point at positions on that very line.
func TestLineInfoPosBounds(t *testing.T) {
	cases := []string{
		"a",
		"a\nb",
		"abc def ghi jkl mno", // soft wrap candidate
		"\n\n",
		"x\n\ny",
	}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			glyphs := getGlyphs(16, 0, 30, text.Start, src)
			gi := buildIndex(glyphs)
			for li, ln := range gi.lines {
				if ln.posStart < 0 || ln.posEnd > len(gi.positions) {
					t.Fatalf("line %d posStart=%d posEnd=%d out of [0,%d]",
						li, ln.posStart, ln.posEnd, len(gi.positions))
				}
				if ln.posStart > ln.posEnd {
					t.Fatalf("line %d posStart > posEnd", li)
				}
				for k := ln.posStart; k < ln.posEnd; k++ {
					if gi.positions[k].lineCol.line != li {
						t.Errorf("line %d: positions[%d] is on line %d",
							li, k, gi.positions[k].lineCol.line)
					}
				}
			}
		})
	}
}

// TestSingleGlyphBuilds checks behavior with exactly one glyph: positions and
// at least one line must be created when the glyph carries the line-break flag.
func TestSingleGlyphBuilds(t *testing.T) {
	glyphs := getGlyphs(16, 0, 200, text.Start, "x")
	gi := buildIndex(glyphs)
	if len(gi.positions) < 1 {
		t.Fatalf("want >=1 position, got 0")
	}
	if len(gi.lines) < 1 {
		t.Fatalf("want >=1 line, got 0")
	}
	// First position must be at column 0, line 0, runes 0.
	p := gi.positions[0]
	if p.lineCol != (screenPos{}) || p.runes != 0 {
		t.Errorf("first pos = %+v, expected zero", p)
	}
}

// TestTrailingNewlineCreatesLine ensures a trailing "\n" produces an extra
// line entry and a position on that empty line.
func TestTrailingNewlineCreatesLine(t *testing.T) {
	withoutNL := buildIndex(getGlyphs(16, 0, 200, text.Start, "ab"))
	withNL := buildIndex(getGlyphs(16, 0, 200, text.Start, "ab\n"))

	if len(withNL.lines) <= len(withoutNL.lines) {
		t.Errorf("trailing \\n should add a line: %d -> %d",
			len(withoutNL.lines), len(withNL.lines))
	}
	last := withNL.positions[len(withNL.positions)-1]
	if last.lineCol.line == 0 {
		t.Errorf("trailing-NL last position should be on a new line, got %+v", last.lineCol)
	}
	if last.lineCol.col != 0 {
		t.Errorf("trailing-NL last position col=%d, want 0", last.lineCol.col)
	}
}

// TestRuneCountMonotonic asserts that runes in stored positions never decrease
// even across bidi run boundaries (they may stay equal — that's the bidi
// flip case — but never go backwards).
func TestRuneCountMonotonic(t *testing.T) {
	fontSize := 16
	source := "The\nquick سماء של\nום لا fox\nتمط של\nום."
	glyphs := makeAccountingTestText(source, fontSize, fontSize*10)
	gi := buildIndex(glyphs)
	for i := 1; i < len(gi.positions); i++ {
		if gi.positions[i].runes < gi.positions[i-1].runes {
			t.Errorf("positions[%d].runes=%d < positions[%d].runes=%d (decreased)",
				i, gi.positions[i].runes, i-1, gi.positions[i-1].runes)
		}
	}
}

// TestPositionIndexAdvancesWithGlyphs verifies that for plain LTR text every
// rune produces a distinct position with monotonically increasing y for
// successive lines.
func TestLinesAdvanceY(t *testing.T) {
	glyphs := getGlyphs(16, 0, 200, text.Start, "a\nb\nc\nd")
	gi := buildIndex(glyphs)
	if len(gi.lines) < 4 {
		t.Fatalf("want 4+ lines, got %d", len(gi.lines))
	}
	for i := 1; i < len(gi.lines); i++ {
		if gi.lines[i].yOff <= gi.lines[i-1].yOff {
			t.Errorf("line %d yOff %d not greater than prev %d",
				i, gi.lines[i].yOff, gi.lines[i-1].yOff)
		}
	}
}
