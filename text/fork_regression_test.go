package text

import (
	"slices"
	"strings"
	"testing"

	"github.com/nanorele/gio/font/opentype"
	"github.com/nanorele/gio/io/system"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"
)

// TestASCIIWithRTLLocaleKeepsLTROrder ensures that pure-ASCII text shaped
// under an RTL locale still resolves to LTR runs. The isASCII fast path in
// splitBidi must not skip bidi resolution for RTL paragraphs: latin text and
// digits are strong-LTR characters and must render left-to-right even inside
// an RTL paragraph.
func TestASCIIWithRTLLocaleKeepsLTROrder(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	shaper := testShaper(ltrFace)

	doc := shaper.LayoutRunes(Parameters{
		PxPerEm:  fixed.I(16),
		MaxWidth: 2000,
		Locale:   arabic,
	}, []rune("hello 123"))
	if len(doc.lines) == 0 {
		t.Fatal("no lines returned")
	}
	for i, run := range doc.lines[0].runs {
		if run.Direction.Progression() == system.TowardOrigin {
			t.Errorf("run %d: ASCII text shaped as RTL run under RTL locale; visual glyph order is reversed", i)
		}
	}
}

// TestReplaceControlCharactersPreservesShapingControls ensures the
// pre-shaping character substitution touches only line/paragraph separators
// and tabs. Zero-width characters control emoji sequences and cursive
// joining, and typographic spaces carry width and line-break semantics; all
// must reach the shaper untouched.
func TestReplaceControlCharactersPreservesShapingControls(t *testing.T) {
	preserved := []rune{
		0x200B, // ZWSP: invisible break opportunity
		0x200C, // ZWNJ: blocks cursive joining (Persian/Arabic orthography)
		0x200D, // ZWJ: emoji sequences, forces joining
		0x2060, // word joiner
		0xFEFF, // BOM / zero-width no-break space
		0x00A0, // NBSP: forbids line break
		0x202F, // narrow NBSP
		0x2003, // em space: distinct width
		0x2009, // thin space: distinct width
	}
	for _, r := range preserved {
		out := replaceControlCharacters([]rune{'a', r, 'b'})
		if out[1] != r {
			t.Errorf("%U must be preserved, got %U", r, out[1])
		}
	}
	replaced := []rune{0x001C, 0x001D, 0x001E, '\v', '\f', '\r', '\n', 0x0085, 0x2029}
	for _, r := range replaced {
		out := replaceControlCharacters([]rune{'a', r, 'b'})
		if out[1] != ' ' {
			t.Errorf("%U must be replaced with space, got %U", r, out[1])
		}
	}
	// U+2028 LINE SEPARATOR deliberately passes through: it is a mandatory
	// break and the wrapper must honor it.
	if out := replaceControlCharacters([]rune{'a', 0x2028, 'b'}); out[1] != 0x2028 {
		t.Errorf("U+2028 must be preserved as a mandatory break, got %U", out[1])
	}
	if out := replaceControlCharacters([]rune{'\t'}); out[0] != 0x2003 {
		t.Errorf("tab must map to em space, got %U", out[0])
	}
}

// TestNBSPPreventsLineBreak verifies end-to-end that a non-breaking space
// survives to the wrapper: "aaa bbb" with a regular space breaks under a
// narrow max width, while the same text with NBSP must stay on one line.
func TestNBSPPreventsLineBreak(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	shaper := testShaper(ltrFace)
	params := Parameters{
		PxPerEm: fixed.I(10),
		Locale:  english,
	}

	// "aa bb cc" with a regular space between "bb" and "cc" breaks after
	// "bb" when only "aa bb" fits; with an NBSP joining "bb cc" the break
	// must move before "bb", leaving only "aa " on the first line.
	withSpace := []rune("aa bb cc")
	withNBSP := slices.Clone(withSpace)
	withNBSP[5] = 0x00A0

	params.MaxWidth = 2000
	fit := shaper.LayoutRunes(params, []rune("aa bb"))
	params.MaxWidth = fit.lines[0].width.Ceil() + 2

	spaced := shaper.LayoutRunes(params, withSpace)
	if got := len(spaced.lines); got != 2 {
		t.Fatalf("expected 2 lines with regular spaces, got %d", got)
	}
	if got := spaced.lines[0].runeCount; got != 6 {
		t.Fatalf(`expected "aa bb " (6 runes) on line 0, got %d`, got)
	}
	nonBreaking := shaper.LayoutRunes(params, withNBSP)
	if got := len(nonBreaking.lines); got != 2 {
		t.Fatalf("expected 2 lines with NBSP, got %d", got)
	}
	if got := nonBreaking.lines[0].runeCount; got != 3 {
		t.Errorf(`NBSP between "bb" and "cc" was treated as a break opportunity: line 0 has %d runes, want 3 ("aa ")`, got)
	}
}

// TestLayoutCacheKeyIncludesSpaceTrim ensures that two layouts of the same
// string differing only in DisableSpaceTrim do not share a cache entry. The
// Editor always sets DisableSpaceTrim while Label does not, and both share
// one Shaper, so a missing key field serves stale layouts across widgets.
func TestLayoutCacheKeyIncludesSpaceTrim(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	collection := []FontFace{{Face: ltrFace}}
	cache := NewShaper(NoSystemFonts(), WithCollection(collection))

	params := Parameters{
		PxPerEm:  fixed.I(10),
		MaxWidth: 2000,
		Locale:   english,
	}
	// Force a wrap between the words so the first line ends with a space.
	oneWord := cache.layoutParagraph(params, "word", nil)
	params.MaxWidth = oneWord.lines[0].width.Ceil() + 2

	params.DisableSpaceTrim = false
	trimmed := cache.layoutParagraph(params, "word word", nil)
	params.DisableSpaceTrim = true
	untrimmed := cache.layoutParagraph(params, "word word", nil)

	if len(trimmed.lines) < 2 || len(untrimmed.lines) < 2 {
		t.Fatalf("expected wrapped text, got %d/%d lines", len(trimmed.lines), len(untrimmed.lines))
	}
	trimmedSpace := trailingGlyphAdvance(t, trimmed)
	untrimmedSpace := trailingGlyphAdvance(t, untrimmed)
	if trimmedSpace == untrimmedSpace {
		t.Errorf("DisableSpaceTrim had no effect: trailing space advance %v in both layouts (stale cache entry served)", trimmedSpace)
	}
}

func trailingGlyphAdvance(t *testing.T, doc document) fixed.Int26_6 {
	t.Helper()
	run := doc.lines[0].runs[len(doc.lines[0].runs)-1]
	if len(run.Glyphs) == 0 {
		t.Fatal("no glyphs in first line's final run")
	}
	return run.Glyphs[len(run.Glyphs)-1].xAdvance
}

// TestGlyphRunesNoOverflowOnLargeTruncation ensures the per-glyph rune count
// survives truncating texts longer than 65535 runes. The truncator glyph
// carries the entire hidden rune count; a uint16 field wraps around and
// corrupts caret/position mapping in the widget layer.
func TestGlyphRunesNoOverflowOnLargeTruncation(t *testing.T) {
	const runeCount = 70000
	ltrFace, _ := opentype.Parse(goregular.TTF)
	collection := []FontFace{{Face: ltrFace}}
	cache := NewShaper(NoSystemFonts(), WithCollection(collection))

	cache.LayoutString(Parameters{
		PxPerEm:  fixed.I(10),
		MaxWidth: 200,
		MaxLines: 1,
		Locale:   english,
	}, strings.Repeat("a b ", runeCount/4))

	total := 0
	for g, ok := cache.NextGlyph(); ok; g, ok = cache.NextGlyph() {
		total += int(g.Runes)
	}
	if total != runeCount {
		t.Errorf("glyph rune counts sum to %d, want %d (uint16 overflow?)", total, runeCount)
	}
}

// TestGlyphCacheKeyIncludesOffset ensures two glyph sequences that differ
// only in Offset do not collide in the glyph path/bitmap caches: the cached
// path geometry embeds the offsets.
func TestGlyphCacheKeyIncludesOffset(t *testing.T) {
	c := &glyphLRU[int]{}
	base := []Glyph{{ID: 42, X: 0}}
	shifted := []Glyph{{ID: 42, X: 0, Offset: fixed.Point26_6{X: 64, Y: -32}}}

	c.Put(c.hashGlyphs(base), base, 1)
	if v, ok := c.Get(c.hashGlyphs(shifted), shifted); ok {
		t.Errorf("glyphs differing only in Offset hit the same cache entry (got %d)", v)
	}
}
