package text

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"testing"

	nsareg "eliasnaur.com/font/noto/sans/arabic/regular"
	"github.com/nanorele/typesetting/di"
	"github.com/nanorele/typesetting/font"
	"github.com/nanorele/typesetting/language"
	"github.com/nanorele/typesetting/shaping"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"

	giofont "github.com/nanorele/gio/font"
	"github.com/nanorele/gio/font/opentype"
	"github.com/nanorele/gio/io/system"
)

var english = system.Locale{
	Language:  "EN",
	Direction: system.LTR,
}

var arabic = system.Locale{
	Language:  "AR",
	Direction: system.RTL,
}

func testShaper(faces ...giofont.Face) *shaperImpl {
	ff := make([]FontFace, 0, len(faces))
	for _, face := range faces {
		ff = append(ff, FontFace{Face: face})
	}
	shaper := newShaperImpl(false, ff)
	return shaper
}

func TestEmptyString(t *testing.T) {
	ppem := fixed.I(200)
	ltrFace, _ := opentype.Parse(goregular.TTF)
	shaper := testShaper(ltrFace)

	lines := shaper.LayoutRunes(Parameters{
		PxPerEm:  ppem,
		MaxWidth: 2000,
		Locale:   english,
	}, []rune{})
	if len(lines.lines) == 0 {
		t.Fatalf("Layout returned no lines for empty string; expected 1")
	}
	l := lines.lines[0]
	if expected := fixed.Int26_6(12094); l.ascent != expected {
		t.Errorf("unexpected ascent for empty string: %v, expected %v", l.ascent, expected)
	}
	if expected := fixed.Int26_6(2700); l.descent != expected {
		t.Errorf("unexpected descent for empty string: %v, expected %v", l.descent, expected)
	}
}

func TestNoFaces(t *testing.T) {
	ppem := fixed.I(200)
	shaper := testShaper()

	shaper.LayoutRunes(Parameters{
		PxPerEm:  ppem,
		MaxWidth: 2000,
		Locale:   english,
	}, []rune("✨ⷽℎ↞⋇ⱜ⪫⢡⽛⣦␆Ⱨⳏ⳯⒛⭣╎⌞⟻⢇┃➡⬎⩱⸇ⷎ⟅▤⼶⇺⩳⎏⤬⬞ⴈ⋠⿶⢒₍☟⽂ⶦ⫰⭢⌹∼▀⾯⧂❽⩏ⓖ⟅⤔⍇␋⽓ₑ⢳⠑❂⊪⢘⽨⃯▴ⷿ"))
}

func TestAlignWidth(t *testing.T) {
	lines := []line{
		{width: fixed.I(50)},
		{width: fixed.I(75)},
		{width: fixed.I(25)},
	}
	for _, minWidth := range []int{0, 50, 100} {
		width := alignWidth(minWidth, lines)
		if width < minWidth {
			t.Errorf("expected width >= %d, got %d", minWidth, width)
		}
	}
}

func TestShapingAlignWidth(t *testing.T) {
	ppem := fixed.I(10)
	ltrFace, _ := opentype.Parse(goregular.TTF)
	shaper := testShaper(ltrFace)

	type testcase struct {
		name               string
		minWidth, maxWidth int
		expected           int
		str                string
	}
	for _, tc := range []testcase{
		{
			name:     "zero min",
			maxWidth: 100,
			str:      "a\nb\nc",
			expected: 22,
		},
		{
			name:     "min == max",
			minWidth: 100,
			maxWidth: 100,
			str:      "a\nb\nc",
			expected: 100,
		},
		{
			name:     "min < max",
			minWidth: 50,
			maxWidth: 100,
			str:      "a\nb\nc",
			expected: 50,
		},
		{
			name:     "min < max, text > min",
			minWidth: 50,
			maxWidth: 100,
			str:      "aphabetic\nb\nc",
			expected: 60,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lines := shaper.LayoutString(Parameters{
				PxPerEm:  ppem,
				MinWidth: tc.minWidth,
				MaxWidth: tc.maxWidth,
				Locale:   english,
			}, tc.str)
			if lines.alignWidth != tc.expected {
				t.Errorf("expected line alignWidth to be %d, got %d", tc.expected, lines.alignWidth)
			}
		})
	}
}

func TestNewlineSynthesis(t *testing.T) {
	ppem := fixed.I(10)
	ltrFace, _ := opentype.Parse(goregular.TTF)
	rtlFace, _ := opentype.Parse(nsareg.TTF)
	shaper := testShaper(ltrFace, rtlFace)

	type testcase struct {
		name   string
		locale system.Locale
		txt    string
	}
	for _, tc := range []testcase{
		{
			name:   "ltr bidi newline in rtl segment",
			locale: english,
			txt:    "The quick سماء שלום لا fox تمط שלום\n",
		},
		{
			name:   "ltr bidi newline in ltr segment",
			locale: english,
			txt:    "The quick سماء שלום لا fox\n",
		},
		{
			name:   "rtl bidi newline in ltr segment",
			locale: arabic,
			txt:    "الحب سماء brown привет fox تمط jumps\n",
		},
		{
			name:   "rtl bidi newline in rtl segment",
			locale: arabic,
			txt:    "الحب سماء brown привет fox تمط\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := shaper.LayoutRunes(Parameters{
				PxPerEm:  ppem,
				MaxWidth: 200,
				Locale:   tc.locale,
			}, []rune(tc.txt))
			for lineIdx, line := range doc.lines {
				lastRunIdx := len(line.runs) - 1
				lastRun := line.runs[lastRunIdx]
				lastGlyphIdx := len(lastRun.Glyphs) - 1
				if lastRun.Direction.Progression() == system.TowardOrigin {
					lastGlyphIdx = 0
				}
				glyph := lastRun.Glyphs[lastGlyphIdx]
				if glyph.glyphCount != 0 {
					t.Errorf("expected synthetic newline on line %d, run %d, glyph %d", lineIdx, lastRunIdx, lastGlyphIdx)
				}
				for runIdx, run := range line.runs {
					for glyphIdx, glyph := range run.Glyphs {
						if runIdx == lastRunIdx && glyphIdx == lastGlyphIdx {
							continue
						}
						if glyph.glyphCount == 0 {
							t.Errorf("found invalid synthetic newline on line %d, run %d, glyph %d", lineIdx, runIdx, glyphIdx)
						}
					}
				}
			}
			if t.Failed() {
				printLinePositioning(t, doc.lines, nil)
			}
		})
	}
}

func copyLines(lines []shaping.Line) []shaping.Line {
	out := make([]shaping.Line, len(lines))
	for lineIdx, line := range lines {
		lineCopy := make([]shaping.Output, len(line))
		for runIdx, run := range line {
			lineCopy[runIdx] = run
			lineCopy[runIdx].Glyphs = slices.Clone(run.Glyphs)
		}
		out[lineIdx] = lineCopy
	}
	return out
}

func makeTestText(shaper *shaperImpl, primaryDir system.TextDirection, fontSize, lineWidth, runeLimit int) (simpleSample, complexSample []shaping.Line) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	rtlFace, _ := opentype.Parse(nsareg.TTF)
	if shaper == nil {
		shaper = testShaper(ltrFace, rtlFace)
	}

	ltrSource := "The quick brown fox jumps over the lazy dog."
	rtlSource := "الحب سماء لا تمط غير الأحلام"

	bidiSource := "The quick سماء שלום لا fox تمط שלום غير the lazy dog."

	bidi2Source := "الحب سماء brown привет fox تمط jumps привет over غير الأحلام"

	locale := english
	simpleSource := ltrSource
	complexSource := bidiSource
	if primaryDir == system.RTL {
		simpleSource = rtlSource
		complexSource = bidi2Source
		locale = arabic
	}
	if runeLimit != 0 {
		simpleRunes := []rune(simpleSource)
		complexRunes := []rune(complexSource)
		if runeLimit < len(simpleRunes) {
			simpleSource = string(simpleRunes[:runeLimit])
		}
		if runeLimit < len(complexRunes) {
			complexSource = string(complexRunes[:runeLimit])
		}
	}
	simpleText, _ := shaper.shapeAndWrapText(Parameters{
		PxPerEm:  fixed.I(fontSize),
		MaxWidth: lineWidth,
		Locale:   locale,
	}, []rune(simpleSource))
	simpleText = copyLines(simpleText)
	complexText, _ := shaper.shapeAndWrapText(Parameters{
		PxPerEm:  fixed.I(fontSize),
		MaxWidth: lineWidth,
		Locale:   locale,
	}, []rune(complexSource))
	complexText = copyLines(complexText)
	testShaper(rtlFace, ltrFace)
	return simpleText, complexText
}

func fixedAbs(a fixed.Int26_6) fixed.Int26_6 {
	if a < 0 {
		a = -a
	}
	return a
}

func TestToLine(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	rtlFace, _ := opentype.Parse(nsareg.TTF)
	shaper := testShaper(ltrFace, rtlFace)
	ltr, bidi := makeTestText(shaper, system.LTR, 16, 100, 0)
	rtl, bidi2 := makeTestText(shaper, system.RTL, 16, 100, 0)
	_, bidiWide := makeTestText(shaper, system.LTR, 16, 200, 0)
	_, bidi2Wide := makeTestText(shaper, system.RTL, 16, 200, 0)
	type testcase struct {
		name  string
		lines []shaping.Line

		dir system.TextDirection
	}
	for _, tc := range []testcase{
		{
			name:  "ltr",
			lines: ltr,
			dir:   system.LTR,
		},
		{
			name:  "rtl",
			lines: rtl,
			dir:   system.RTL,
		},
		{
			name:  "bidi",
			lines: bidi,
			dir:   system.LTR,
		},
		{
			name:  "bidi2",
			lines: bidi2,
			dir:   system.RTL,
		},
		{
			name:  "bidi_wide",
			lines: bidiWide,
			dir:   system.LTR,
		},
		{
			name:  "bidi2_wide",
			lines: bidi2Wide,
			dir:   system.RTL,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {

			runesSeen := Range{}
			shaper := testShaper(ltrFace, rtlFace)
			for i, input := range tc.lines {
				seenRun := make([]bool, len(input))
				inputLowestRuneOffset := math.MaxInt
				totalInputGlyphs := 0
				totalInputRunes := 0
				for _, run := range input {
					if run.Runes.Offset < inputLowestRuneOffset {
						inputLowestRuneOffset = run.Runes.Offset
					}
					totalInputGlyphs += len(run.Glyphs)
					totalInputRunes += run.Runes.Count
				}
				output := toLine(shaper.faceToIndex, input, tc.dir)
				if output.direction != tc.dir {
					t.Errorf("line %d: expected direction %v, got %v", i, tc.dir, output.direction)
				}
				totalRunWidth := fixed.I(0)
				totalLineGlyphs := 0
				totalLineRunes := 0
				for k, run := range output.runs {
					seenRun[run.VisualPosition] = true
					if output.visualOrder[run.VisualPosition] != k {
						t.Errorf("line %d, run %d: run.VisualPosition=%d, but line.VisualOrder[%d]=%d(should be %d)", i, k, run.VisualPosition, run.VisualPosition, output.visualOrder[run.VisualPosition], k)
					}
					if run.Runes.Offset != totalLineRunes {
						t.Errorf("line %d, run %d: expected Runes.Offset to be %d, got %d", i, k, totalLineRunes, run.Runes.Offset)
					}
					runGlyphCount := len(run.Glyphs)
					if inputGlyphs := len(input[k].Glyphs); runGlyphCount != inputGlyphs {
						t.Errorf("line %d, run %d: expected %d glyphs, found %d", i, k, inputGlyphs, runGlyphCount)
					}
					runRuneCount := 0
					currentCluster := -1
					for _, g := range run.Glyphs {
						if g.clusterIndex != currentCluster {
							runRuneCount += g.runeCount
							currentCluster = g.clusterIndex
						}
					}
					if run.Runes.Count != runRuneCount {
						t.Errorf("line %d, run %d: expected %d runes, counted %d", i, k, run.Runes.Count, runRuneCount)
					}
					runesSeen.Count += run.Runes.Count
					totalRunWidth += fixedAbs(run.Advance)
					totalLineGlyphs += len(run.Glyphs)
					totalLineRunes += run.Runes.Count
				}
				if output.runeCount != totalInputRunes {
					t.Errorf("line %d: input had %d runes, only counted %d", i, totalInputRunes, output.runeCount)
				}
				if totalLineGlyphs != totalInputGlyphs {
					t.Errorf("line %d: input had %d glyphs, only counted %d", i, totalInputRunes, totalLineGlyphs)
				}
				if totalRunWidth != output.width {
					t.Errorf("line %d: expected width %d, got %d", i, totalRunWidth, output.width)
				}
				for runIndex, seen := range seenRun {
					if !seen {
						t.Errorf("line %d, run %d missing from runs VisualPosition fields", i, runIndex)
					}
				}
			}
			lastLine := tc.lines[len(tc.lines)-1]
			maxRunes := 0
			for _, run := range lastLine {
				if run.Runes.Count+run.Runes.Offset > maxRunes {
					maxRunes = run.Runes.Count + run.Runes.Offset
				}
			}
			if runesSeen.Count != maxRunes {
				t.Errorf("input covered %d runes, output only covers %d", maxRunes, runesSeen.Count)
			}
		})
	}
}

func FuzzLayout(f *testing.F) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	rtlFace, _ := opentype.Parse(nsareg.TTF)
	f.Add("د عرمثال dstي met لم aqل جدmوpمg lرe dرd  لو عل ميrةsdiduntut lab renنيتذدagلaaiua.ئPocttأior رادرsاي mيrbلmnonaيdتد ماةعcلخ.", true, false, uint8(10), uint16(200))

	shaper := testShaper(ltrFace, rtlFace)
	f.Fuzz(func(t *testing.T, txt string, rtl bool, truncate bool, fontSize uint8, width uint16) {
		locale := system.Locale{
			Direction: system.LTR,
		}
		if rtl {
			locale.Direction = system.RTL
		}
		if fontSize < 1 {
			fontSize = 1
		}
		maxLines := 0
		if truncate {
			maxLines = 1
		}
		lines := shaper.LayoutRunes(Parameters{
			PxPerEm:  fixed.I(int(fontSize)),
			MaxWidth: int(width),
			MaxLines: maxLines,
			Locale:   locale,
		}, []rune(txt))
		validateLines(t, lines.lines, len([]rune(txt)))
	})
}

func validateLines(t *testing.T, lines []line, expectedRuneCount int) {
	t.Helper()
	runesSeen := 0
	for i, line := range lines {
		totalRunWidth := fixed.I(0)
		totalLineGlyphs := 0
		lineRunesSeen := 0
		for k, run := range line.runs {
			if line.visualOrder[run.VisualPosition] != k {
				t.Errorf("line %d, run %d: run.VisualPosition=%d, but line.VisualOrder[%d]=%d(should be %d)", i, k, run.VisualPosition, run.VisualPosition, line.visualOrder[run.VisualPosition], k)
			}
			if run.Runes.Offset != lineRunesSeen {
				t.Errorf("line %d, run %d: expected Runes.Offset to be %d, got %d", i, k, lineRunesSeen, run.Runes.Offset)
			}
			runRuneCount := 0
			currentCluster := -1
			for _, g := range run.Glyphs {
				if g.clusterIndex != currentCluster {
					runRuneCount += g.runeCount
					currentCluster = g.clusterIndex
				}
			}
			if run.Runes.Count != runRuneCount {
				t.Errorf("line %d, run %d: expected %d runes, counted %d", i, k, run.Runes.Count, runRuneCount)
			}
			lineRunesSeen += run.Runes.Count
			totalRunWidth += fixedAbs(run.Advance)
			totalLineGlyphs += len(run.Glyphs)
		}
		if totalRunWidth != line.width {
			t.Errorf("line %d: expected width %d, got %d", i, line.width, totalRunWidth)
		}
		runesSeen += lineRunesSeen
	}
	if runesSeen != expectedRuneCount {
		t.Errorf("input covered %d runes, output only covers %d", expectedRuneCount, runesSeen)
	}
}

func TestTextAppend(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	rtlFace, _ := opentype.Parse(nsareg.TTF)

	shaper := testShaper(ltrFace, rtlFace)

	text1 := shaper.LayoutString(Parameters{
		PxPerEm:  fixed.I(14),
		MaxWidth: 200,
		Locale:   english,
	}, "د عرمثال dstي met لم aqل جدmوpمg lرe dرd  لو عل ميrةsdiduntut lab renنيتذدagلaaiua.ئPocttأior رادرsاي mيrbلmnonaيdتد ماةعcلخ.")
	text2 := shaper.LayoutString(Parameters{
		PxPerEm:  fixed.I(14),
		MaxWidth: 200,
		Locale:   english,
	}, "د عرمثال dstي met لم aqل جدmوpمg lرe dرd  لو عل ميrةsdiduntut lab renنيتذدagلaaiua.ئPocttأior رادرsاي mيrbلmnonaيdتد ماةعcلخ.")

	text1.append(text2)
	curY := math.MinInt
	for lineNum, line := range text1.lines {
		yOff := line.yOffset
		if yOff <= curY {
			t.Errorf("lines[%d] has y offset %d, <= to previous %d", lineNum, yOff, curY)
		}
		curY = yOff
	}
}

func TestGlyphIDPacking(t *testing.T) {
	const maxPPem = fixed.Int26_6((1 << sizebits) - 1)
	type testcase struct {
		name      string
		ppem      fixed.Int26_6
		faceIndex int
		gid       font.GID
		expected  GlyphID
	}
	for _, tc := range []testcase{
		{
			name: "zero value",
		},
		{
			name:      "10 ppem faceIdx 1 GID 5",
			ppem:      fixed.I(10),
			faceIndex: 1,
			gid:       5,
			expected:  284223755780101,
		},
		{
			name:      maxPPem.String() + " ppem faceIdx " + strconv.Itoa(math.MaxUint16) + " GID " + fmt.Sprintf("%d", int64(math.MaxUint32)),
			ppem:      maxPPem,
			faceIndex: math.MaxUint16,
			gid:       math.MaxUint32,
			expected:  18446744073709551615,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual := newGlyphID(tc.ppem, tc.faceIndex, tc.gid)
			if actual != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, actual)
			}
			actualPPEM, actualFaceIdx, actualGID := splitGlyphID(actual)
			if actualPPEM != tc.ppem {
				t.Errorf("expected ppem %d, got %d", tc.ppem, actualPPEM)
			}
			if actualFaceIdx != tc.faceIndex {
				t.Errorf("expected faceIdx %d, got %d", tc.faceIndex, actualFaceIdx)
			}
			if actualGID != tc.gid {
				t.Errorf("expected gid %d, got %d", tc.gid, actualGID)
			}
		})
	}
}

func TestArabicDiacriticClustering(t *testing.T) {
	tests := []struct {
		name          string
		input         []rune
		wantRuns      int
		wantScript    language.Script
		wantDirection di.Direction
	}{
		{
			name: "Arabic Letter + Diacritic",

			input:         []rune{'\u0628', '\u0650'},
			wantRuns:      1,
			wantScript:    language.Arabic,
			wantDirection: di.DirectionRTL,
		},
		{
			name: "Arabic Word with Multiple Diacritics",
			input: []rune{
				'\u0628',
				'\u0650',
				'\u0633',
				'\u0652',
				'\u0645',
				'\u0650',
			},
			wantRuns:      1,
			wantScript:    language.Arabic,
			wantDirection: di.DirectionRTL,
		},
		{
			name: "Mixed Script (CONTROL Case) #1",

			input:         []rune{'\u0628', 'A'},
			wantRuns:      2,
			wantScript:    language.Arabic,
			wantDirection: di.DirectionRTL,
		},
		{
			name: "Mixed Script (CONTROL Case) #2",

			input:         []rune{'\u0628', '\u0651', '\u0650', 'A', '\u0628', '\u0650'},
			wantRuns:      3,
			wantScript:    language.Arabic,
			wantDirection: di.DirectionRTL,
		},
		{
			name: "Mixed Script (A little 'stress' test)",

			input:         []rune{'s', '\u0651', '\u0650', 'r', '\u064E'},
			wantRuns:      1,
			wantScript:    language.Latin,
			wantDirection: di.DirectionLTR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputs := []shaping.Input{{
				Text:      tt.input,
				RunStart:  0,
				RunEnd:    len(tt.input),
				Direction: tt.wantDirection,
				Script:    language.Arabic,
				Face:      nil,
				Size:      fixed.I(10),
			}}

			got := splitByScript(inputs, tt.wantDirection, nil)

			if len(got) != tt.wantRuns {
				t.Fatalf("splitByScript produced %d runs, expected %d. \nRun details: %+v", len(got), tt.wantRuns, got)
			}

			if tt.wantRuns == 1 {
				run := got[0]
				if run.RunEnd != len(tt.input) {
					t.Errorf("Run truncated early. End = %d, expected %d", run.RunEnd, len(tt.input))
				}
				if run.Script != tt.wantScript {
					t.Errorf("Run assigned wrong script. Got %s, expected %s", run.Script, tt.wantScript)
				}
			}
		})
	}
}

func TestFixedFloatRoundTrip(t *testing.T) {
	cases := []fixed.Int26_6{
		0,
		1,
		-1,
		64,
		-64,
		fixed.I(1),
		fixed.I(-1),
		fixed.I(100),
		fixed.I(-100),
		fixed.Int26_6(1<<20) - 1,
		-(fixed.Int26_6(1<<20) - 1),
	}
	for _, c := range cases {
		f := fixedToFloat(c)
		got := floatToFixed(f)
		if got != c {
			t.Errorf("fixedToFloat/floatToFixed round trip failed: in=%d float=%f out=%d", c, f, got)
		}
	}
}

func TestFixedToFloatScale(t *testing.T) {
	if got, want := fixedToFloat(fixed.I(1)), float32(1.0); got != want {
		t.Errorf("fixedToFloat(1.0) = %f, want %f", got, want)
	}
	if got, want := fixedToFloat(fixed.Int26_6(32)), float32(0.5); got != want {
		t.Errorf("fixedToFloat(0.5) = %f, want %f", got, want)
	}
	if got, want := fixedToFloat(fixed.Int26_6(-32)), float32(-0.5); got != want {
		t.Errorf("fixedToFloat(-0.5) = %f, want %f", got, want)
	}
}

func TestFloatToFixedScale(t *testing.T) {
	if got, want := floatToFixed(1.0), fixed.I(1); got != want {
		t.Errorf("floatToFixed(1.0) = %d, want %d", got, want)
	}
	if got, want := floatToFixed(0.5), fixed.Int26_6(32); got != want {
		t.Errorf("floatToFixed(0.5) = %d, want %d", got, want)
	}
	if got, want := floatToFixed(-0.5), fixed.Int26_6(-32); got != want {
		t.Errorf("floatToFixed(-0.5) = %d, want %d", got, want)
	}
}

func TestSplitBidiPureASCIIFastPath(t *testing.T) {
	shaper := testShaper()
	txt := []rune("hello world")
	in := shaping.Input{
		Text:      txt,
		RunStart:  0,
		RunEnd:    len(txt),
		Direction: di.DirectionLTR,
		Size:      fixed.I(10),
	}
	out := shaper.splitBidi(in)
	if len(out) != 1 {
		t.Fatalf("expected 1 run for pure-ASCII LTR fast path, got %d", len(out))
	}
	if out[0].Direction != di.DirectionLTR {
		t.Errorf("expected LTR, got %v", out[0].Direction)
	}
	if out[0].RunStart != 0 || out[0].RunEnd != len(txt) {
		t.Errorf("range modified by ASCII fast path: %d..%d", out[0].RunStart, out[0].RunEnd)
	}
}

func TestSplitBidiEmptyInput(t *testing.T) {
	shaper := testShaper()
	in := shaping.Input{
		Text:      []rune{},
		RunStart:  0,
		RunEnd:    0,
		Direction: di.DirectionLTR,
		Size:      fixed.I(10),
	}
	out := shaper.splitBidi(in)
	if len(out) != 1 {
		t.Fatalf("expected 1 input back for empty text, got %d", len(out))
	}
}

func TestSplitBidiMixedLTRRTL(t *testing.T) {
	shaper := testShaper()
	txt := []rune("abc אבג def")
	in := shaping.Input{
		Text:      txt,
		RunStart:  0,
		RunEnd:    len(txt),
		Direction: di.DirectionLTR,
		Size:      fixed.I(10),
	}
	out := shaper.splitBidi(in)
	if len(out) < 2 {
		t.Fatalf("expected multiple runs from mixed bidi text, got %d", len(out))
	}
	sawLTR, sawRTL := false, false
	for _, r := range out {
		if r.Direction == di.DirectionLTR {
			sawLTR = true
		}
		if r.Direction == di.DirectionRTL {
			sawRTL = true
		}
	}
	if !sawLTR || !sawRTL {
		t.Errorf("expected both LTR and RTL runs, got LTR=%v RTL=%v", sawLTR, sawRTL)
	}
	// Runs should be contiguous with no gaps and cover the full input range.
	prevEnd := 0
	for i, r := range out {
		if r.RunStart != prevEnd {
			t.Errorf("run %d: expected RunStart=%d (after previous), got %d", i, prevEnd, r.RunStart)
		}
		prevEnd = r.RunEnd
	}
	if prevEnd != len(txt) {
		t.Errorf("runs ended at %d, expected %d", prevEnd, len(txt))
	}
}

func TestSplitByScriptEmptyInput(t *testing.T) {
	in := []shaping.Input{{Text: []rune{}, RunStart: 0, RunEnd: 0}}
	out := splitByScript(in, di.DirectionLTR, nil)
	if len(out) != 1 {
		t.Fatalf("expected 1 input from empty splitByScript, got %d", len(out))
	}
}

func TestSplitByScriptLatinOnly(t *testing.T) {
	txt := []rune("hello world")
	in := []shaping.Input{{
		Text:     txt,
		RunStart: 0,
		RunEnd:   len(txt),
	}}
	out := splitByScript(in, di.DirectionLTR, nil)
	if len(out) != 1 {
		t.Fatalf("expected 1 run for latin-only text, got %d", len(out))
	}
	if out[0].Script != language.Latin {
		t.Errorf("expected Latin script, got %v", out[0].Script)
	}
}

func TestIsASCII(t *testing.T) {
	cases := []struct {
		name string
		in   []rune
		want bool
	}{
		{"empty", []rune(""), true},
		{"plain", []rune("Hello, World!"), true},
		{"control chars", []rune("\t\n\r"), true},
		{"high ascii boundary", []rune{0x7F}, true},
		{"first non-ascii", []rune{0x80}, false},
		{"arabic", []rune("الحب"), false},
		{"mixed", []rune("abcé"), false},
	}
	for _, c := range cases {
		if got := isASCII(c.in); got != c.want {
			t.Errorf("%s: isASCII = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestReplaceControlCharacters(t *testing.T) {
	in := []rune("hello\tworld​zero⁠width")
	out := replaceControlCharacters(append([]rune(nil), in...))
	for i, r := range out {
		if r == '\t' {
			t.Errorf("tab not replaced at index %d", i)
		}
		if r == '​' || r == '⁠' {
			t.Errorf("zero-width char not replaced at index %d", i)
		}
	}
	// tab gets replaced with em-space  .
	if out[5] != ' ' {
		t.Errorf("expected tab -> em space, got %U", out[5])
	}
}

func TestReplaceControlCharactersNewline(t *testing.T) {
	// '\n' is a unicode space; replaceControlCharacters substitutes it with ' '.
	// LayoutRunes strips the trailing newline before calling replaceControlCharacters.
	in := []rune("a\nb")
	out := replaceControlCharacters(append([]rune(nil), in...))
	if out[1] != ' ' {
		t.Errorf("expected newline -> space, got %U", out[1])
	}
}

func TestWrapPolicyToGoText(t *testing.T) {
	if wrapPolicyToGoText(WrapGraphemes) != shaping.Always {
		t.Errorf("WrapGraphemes should map to shaping.Always")
	}
	if wrapPolicyToGoText(WrapWords) != shaping.Never {
		t.Errorf("WrapWords should map to shaping.Never")
	}
	if wrapPolicyToGoText(WrapHeuristically) != shaping.WhenNecessary {
		t.Errorf("WrapHeuristically should map to shaping.WhenNecessary")
	}
}

func TestMapDirectionRoundTrip(t *testing.T) {
	cases := []system.TextDirection{system.LTR, system.RTL}
	for _, d := range cases {
		got := unmapDirection(mapDirection(d))
		if got != d {
			t.Errorf("round trip failed: %v -> %v", d, got)
		}
	}
	// Unknown di.Direction maps back to LTR (default).
	if unmapDirection(di.Direction(0xFF)) != system.LTR {
		t.Errorf("unknown direction should default to LTR")
	}
}

func TestCalculateYOffsetsEmpty(t *testing.T) {
	// Should not panic on empty input.
	calculateYOffsets(nil)
	calculateYOffsets([]line{})
	calculateYOffsetsFrom(nil, 0)
	calculateYOffsetsFrom([]line{}, 5)
}

func TestCalculateYOffsetsMonotonic(t *testing.T) {
	lines := []line{
		{ascent: fixed.I(10), lineHeight: fixed.I(15)},
		{ascent: fixed.I(10), lineHeight: fixed.I(15)},
		{ascent: fixed.I(10), lineHeight: fixed.I(15)},
	}
	calculateYOffsets(lines)
	if lines[0].yOffset != 10 {
		t.Errorf("first line yOffset = %d, want 10", lines[0].yOffset)
	}
	for i := 1; i < len(lines); i++ {
		if lines[i].yOffset <= lines[i-1].yOffset {
			t.Errorf("yOffsets not monotonic: lines[%d].yOffset=%d <= lines[%d].yOffset=%d",
				i, lines[i].yOffset, i-1, lines[i-1].yOffset)
		}
	}
}

func TestCalculateYOffsetsFromCarry(t *testing.T) {
	lines := []line{
		{ascent: fixed.I(10), lineHeight: fixed.I(15), yOffset: 100},
		{ascent: fixed.I(10), lineHeight: fixed.I(15)},
		{ascent: fixed.I(10), lineHeight: fixed.I(15)},
	}
	// startIdx=1 should treat lines[0].yOffset as the carry.
	calculateYOffsetsFrom(lines, 1)
	if lines[0].yOffset != 100 {
		t.Errorf("startIdx=1 should not modify lines[0].yOffset; got %d", lines[0].yOffset)
	}
	if lines[1].yOffset != 115 {
		t.Errorf("lines[1].yOffset = %d, want 115", lines[1].yOffset)
	}
	if lines[2].yOffset != 130 {
		t.Errorf("lines[2].yOffset = %d, want 130", lines[2].yOffset)
	}
}

func TestLayoutEmptyHasOneLine(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	shaper := testShaper(ltrFace)
	doc := shaper.LayoutString(Parameters{
		PxPerEm:  fixed.I(10),
		MaxWidth: 200,
		Locale:   english,
	}, "")
	if len(doc.lines) != 1 {
		t.Fatalf("expected 1 line for empty string, got %d", len(doc.lines))
	}
}

func TestLayoutSingleCharacter(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	shaper := testShaper(ltrFace)
	doc := shaper.LayoutString(Parameters{
		PxPerEm:  fixed.I(10),
		MaxWidth: 200,
		Locale:   english,
	}, "x")
	if len(doc.lines) != 1 {
		t.Fatalf("expected 1 line for single char, got %d", len(doc.lines))
	}
	if doc.lines[0].runeCount < 1 {
		t.Errorf("expected runeCount >= 1, got %d", doc.lines[0].runeCount)
	}
}

func TestLayoutAllWhitespace(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	shaper := testShaper(ltrFace)
	doc := shaper.LayoutString(Parameters{
		PxPerEm:  fixed.I(10),
		MaxWidth: 200,
		Locale:   english,
	}, "     ")
	if len(doc.lines) < 1 {
		t.Fatalf("expected at least 1 line for whitespace, got %d", len(doc.lines))
	}
}

func TestLayoutAllNewlines(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	shaper := testShaper(ltrFace)
	doc := shaper.LayoutString(Parameters{
		PxPerEm:  fixed.I(10),
		MaxWidth: 200,
		Locale:   english,
	}, "\n\n\n")
	if len(doc.lines) < 1 {
		t.Fatalf("expected at least 1 line for newline-only text, got %d", len(doc.lines))
	}
	// Y offsets must be monotonically non-decreasing across lines.
	prev := doc.lines[0].yOffset
	for i := 1; i < len(doc.lines); i++ {
		if doc.lines[i].yOffset < prev {
			t.Errorf("non-monotonic yOffset at line %d: %d < %d", i, doc.lines[i].yOffset, prev)
		}
		prev = doc.lines[i].yOffset
	}
}

func TestLayoutMultiParagraph(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	collection := []FontFace{{Face: ltrFace}}
	shaper := NewShaper(NoSystemFonts(), WithCollection(collection))
	shaper.LayoutString(Parameters{
		PxPerEm:  fixed.I(10),
		MaxWidth: 200,
		Locale:   english,
	}, "para one\npara two\npara three")
	var glyphs []Glyph
	for g, ok := shaper.NextGlyph(); ok; g, ok = shaper.NextGlyph() {
		glyphs = append(glyphs, g)
	}
	// Expect at least two ParagraphStart flags (one for "para two", one for "para three").
	starts := 0
	for _, g := range glyphs {
		if g.Flags&FlagParagraphStart != 0 {
			starts++
		}
	}
	if starts < 2 {
		t.Errorf("expected >= 2 ParagraphStart glyphs across 3 paragraphs, got %d", starts)
	}
}

func TestLayoutTinyConstraintWraps(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	shaper := testShaper(ltrFace)
	doc := shaper.LayoutString(Parameters{
		PxPerEm:    fixed.I(10),
		MaxWidth:   1,
		Locale:     english,
		WrapPolicy: WrapGraphemes,
	}, "abcde")
	if len(doc.lines) < 5 {
		t.Errorf("MaxWidth=1 with WrapGraphemes should produce >= 5 lines for 5 chars, got %d", len(doc.lines))
	}
}

func TestLayoutConstraintExactlyLineWidth(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	shaper := testShaper(ltrFace)
	// First measure with a generous width.
	wide := shaper.LayoutString(Parameters{
		PxPerEm:  fixed.I(10),
		MaxWidth: 10000,
		Locale:   english,
	}, "hello")
	if len(wide.lines) != 1 {
		t.Fatalf("expected 1 line for wide layout, got %d", len(wide.lines))
	}
	exact := wide.lines[0].width.Ceil()
	got := shaper.LayoutString(Parameters{
		PxPerEm:  fixed.I(10),
		MaxWidth: exact,
		Locale:   english,
	}, "hello")
	if len(got.lines) != 1 {
		t.Errorf("MaxWidth = exact line width should keep text on 1 line, got %d", len(got.lines))
	}
}

func TestLayoutTruncatorAppears(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	collection := []FontFace{{Face: ltrFace}}
	shaper := NewShaper(NoSystemFonts(), WithCollection(collection))
	shaper.LayoutString(Parameters{
		PxPerEm:  fixed.I(10),
		MaxWidth: 30,
		MaxLines: 1,
		Locale:   english,
	}, "the quick brown fox jumps over the lazy dog")
	sawTruncator := false
	for g, ok := shaper.NextGlyph(); ok; g, ok = shaper.NextGlyph() {
		if g.Flags&FlagTruncator != 0 {
			sawTruncator = true
		}
	}
	if !sawTruncator {
		t.Errorf("expected truncator glyph to be emitted when MaxLines<text length")
	}
}

func TestLayoutCustomTruncator(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	collection := []FontFace{{Face: ltrFace}}
	shaper := NewShaper(NoSystemFonts(), WithCollection(collection))
	shaper.LayoutString(Parameters{
		PxPerEm:   fixed.I(10),
		MaxWidth:  30,
		MaxLines:  1,
		Truncator: ">>",
		Locale:    english,
	}, "the quick brown fox jumps over the lazy dog")
	sawTruncator := false
	for g, ok := shaper.NextGlyph(); ok; g, ok = shaper.NextGlyph() {
		if g.Flags&FlagTruncator != 0 {
			sawTruncator = true
		}
	}
	if !sawTruncator {
		t.Errorf("expected truncator glyph to be emitted with custom truncator")
	}
}

func TestShapeOffsetXMultipleGlyphs(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	collection := []FontFace{{Face: ltrFace}}
	shaper := NewShaper(NoSystemFonts(), WithCollection(collection))
	shaper.LayoutString(Parameters{
		PxPerEm:  fixed.I(40),
		MinWidth: 200,
		MaxWidth: 200,
		Locale:   english,
	}, "abcd")
	var raw []Glyph
	for g, ok := shaper.NextGlyph(); ok; g, ok = shaper.NextGlyph() {
		raw = append(raw, g)
	}
	var gs []Glyph
	for _, g := range raw {
		if g.Advance != 0 {
			gs = append(gs, g)
			if len(gs) == 4 {
				break
			}
		}
	}
	if len(gs) < 4 {
		t.Skip("font produced fewer than 4 advancing glyphs")
	}
	// Apply a different offset to each glyph.
	offsets := []fixed.Int26_6{0, fixed.I(5), fixed.I(-5), fixed.I(10)}
	for i := range gs {
		gs[i].Offset.X = offsets[i]
	}
	// Should not panic. The Shape call exercises the Offset.X path.
	_ = shaper.Shape(gs)
}

func TestShapeBitmapsEmptyGlyphs(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	collection := []FontFace{{Face: ltrFace}}
	shaper := NewShaper(NoSystemFonts(), WithCollection(collection))
	// Both should handle empty input without crashing.
	_ = shaper.Shape(nil)
	_ = shaper.Bitmaps(nil)
	_ = shaper.Shape([]Glyph{})
	_ = shaper.Bitmaps([]Glyph{})
}

func TestShapeInvalidFaceIdx(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	collection := []FontFace{{Face: ltrFace}}
	shaper := NewShaper(NoSystemFonts(), WithCollection(collection))
	// Build a glyph with a face index way beyond what the shaper has loaded.
	bogus := Glyph{
		ID:      newGlyphID(fixed.I(10), 1000, 0),
		Advance: fixed.I(10),
	}
	// Should be skipped silently, no panic.
	_ = shaper.Shape([]Glyph{bogus})
	_ = shaper.Bitmaps([]Glyph{bogus})
}

func TestSetTruncatedCountClearsNonFinalGlyphs(t *testing.T) {
	// Regression: setTruncatedCount used to write to runs[finalRunIdx].Glyphs[finalGlyphIdx]
	// for every iteration, never zeroing the non-final glyphs. Construct a line whose final
	// run has multiple glyphs and verify only the final one carries the count.
	l := line{
		runs: []runLayout{{
			Glyphs: []glyph{
				{runeCount: 5},
				{runeCount: 7},
				{runeCount: 9},
			},
		}},
	}
	l.setTruncatedCount(42)
	final := len(l.runs[0].Glyphs) - 1
	for i, g := range l.runs[0].Glyphs {
		if i == final {
			if g.runeCount != 42 {
				t.Errorf("final glyph runeCount = %d, want 42", g.runeCount)
			}
		} else {
			if g.runeCount != 0 {
				t.Errorf("non-final glyph %d runeCount = %d, want 0 (write-to-wrong-index bug)", i, g.runeCount)
			}
		}
	}
	if !l.runs[0].truncator {
		t.Errorf("setTruncatedCount must mark final run as truncator")
	}
	if l.runs[0].Runes.Count != 42 {
		t.Errorf("setTruncatedCount must set Runes.Count = 42, got %d", l.runs[0].Runes.Count)
	}
}

func TestInsertTrailingSyntheticNewlineLTR(t *testing.T) {
	l := line{
		runs: []runLayout{{
			Direction: system.LTR,
			Glyphs: []glyph{
				{id: 1, glyphCount: 1, runeCount: 1, xAdvance: fixed.I(5)},
			},
			Runes: Range{Count: 1},
		}},
		runeCount: 1,
	}
	l.insertTrailingSyntheticNewline(99)
	if l.runeCount != 2 {
		t.Errorf("expected runeCount=2, got %d", l.runeCount)
	}
	gs := l.runs[0].Glyphs
	// LTR (FromOrigin) appends to the end.
	if gs[len(gs)-1].clusterIndex != 99 {
		t.Errorf("synthetic glyph cluster idx = %d, want 99", gs[len(gs)-1].clusterIndex)
	}
	if gs[len(gs)-1].glyphCount != 0 {
		t.Errorf("synthetic glyph must have glyphCount=0")
	}
	if gs[len(gs)-1].xAdvance != 0 {
		t.Errorf("synthetic glyph must have zero xAdvance")
	}
}

func TestInsertTrailingSyntheticNewlineRTL(t *testing.T) {
	l := line{
		runs: []runLayout{{
			Direction: system.RTL,
			Glyphs: []glyph{
				{id: 1, glyphCount: 1, runeCount: 1, xAdvance: fixed.I(5)},
			},
			Runes: Range{Count: 1},
		}},
		runeCount: 1,
	}
	l.insertTrailingSyntheticNewline(99)
	gs := l.runs[0].Glyphs
	// RTL (TowardOrigin) prepends to position 0.
	if gs[0].clusterIndex != 99 {
		t.Errorf("RTL synthetic glyph should be at index 0, cluster=%d", gs[0].clusterIndex)
	}
	if gs[0].glyphCount != 0 {
		t.Errorf("RTL synthetic glyph must have glyphCount=0")
	}
}

func TestStateLeakAcrossLayouts(t *testing.T) {
	// Layouts in sequence must not leak document state.
	ltrFace, _ := opentype.Parse(goregular.TTF)
	shaper := testShaper(ltrFace)
	d1 := shaper.LayoutString(Parameters{
		PxPerEm: fixed.I(10), MaxWidth: 200, Locale: english,
	}, "first paragraph here")
	d2 := shaper.LayoutString(Parameters{
		PxPerEm: fixed.I(10), MaxWidth: 200, Locale: english,
	}, "x")
	if d2.lines[0].runeCount != 1 {
		t.Errorf("second layout leaked rune count from first; got %d, want 1", d2.lines[0].runeCount)
	}
	// Confirm d1 unchanged.
	if d1.lines[0].runeCount < 5 {
		t.Errorf("first layout document mutated by second; got runeCount=%d", d1.lines[0].runeCount)
	}
}

func TestDocumentResetClears(t *testing.T) {
	d := document{
		lines:           []line{{}},
		runs:            []runLayout{{}},
		glyphs:          []glyph{{}},
		visual:          []int{0},
		alignment:       End,
		alignWidth:      123,
		unreadRuneCount: 5,
	}
	d.reset()
	if len(d.lines) != 0 || len(d.runs) != 0 || len(d.glyphs) != 0 || len(d.visual) != 0 {
		t.Errorf("reset should empty slices: lines=%d runs=%d glyphs=%d visual=%d",
			len(d.lines), len(d.runs), len(d.glyphs), len(d.visual))
	}
	if d.alignment != Start {
		t.Errorf("reset should restore Start alignment, got %v", d.alignment)
	}
	if d.alignWidth != 0 {
		t.Errorf("reset should clear alignWidth, got %d", d.alignWidth)
	}
	if d.unreadRuneCount != 0 {
		t.Errorf("reset should clear unreadRuneCount, got %d", d.unreadRuneCount)
	}
}

func TestToInputDefaults(t *testing.T) {
	runes := []rune("abc")
	lc := langConfig{
		Language:  language.NewLanguage("en"),
		Direction: di.DirectionLTR,
	}
	in := toInput(nil, fixed.I(12), lc, runes)
	if in.RunStart != 0 || in.RunEnd != len(runes) {
		t.Errorf("toInput RunStart/RunEnd = %d..%d, want 0..%d", in.RunStart, in.RunEnd, len(runes))
	}
	if in.Direction != di.DirectionLTR {
		t.Errorf("toInput direction not propagated, got %v", in.Direction)
	}
	if in.Size != fixed.I(12) {
		t.Errorf("toInput size not propagated, got %v", in.Size)
	}
}
