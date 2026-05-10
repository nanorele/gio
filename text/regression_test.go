package text

import (
	"reflect"
	"strings"
	"testing"

	"github.com/nanorele/gio/font/opentype"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"
)

// Fix 1: lru.Put with an existing key must drop the old entry instead of
// inserting a duplicate that desyncs the map and the linked list.

func TestLRUPutOverwriteReturnsLatest(t *testing.T) {
	c := new(layoutCache)
	k := layoutKey{str: "k"}
	c.Put(k, document{alignWidth: 1})
	c.Put(k, document{alignWidth: 2})
	v, ok := c.Get(k)
	if !ok {
		t.Fatalf("key missing after overwrite")
	}
	if v.alignWidth != 2 {
		t.Fatalf("expected latest value 2, got %d", v.alignWidth)
	}
}

func TestLRUPutOverwriteSingleEntry(t *testing.T) {
	c := new(layoutCache)
	k := layoutKey{str: "k"}
	for i := 0; i < 5; i++ {
		c.Put(k, document{alignWidth: i})
	}
	if got := len(c.m); got != 1 {
		t.Fatalf("expected 1 map entry, got %d", got)
	}
	// Walk the linked list and ensure exactly one real entry exists
	// between head and tail (excluding the sentinels).
	count := 0
	for e := c.tail.next; e != c.head; e = e.next {
		count++
		if count > 10 {
			t.Fatalf("linked list cycle or excess entries")
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 list entry, got %d", count)
	}
}

func TestLRUPutOverwriteFiresOnEvict(t *testing.T) {
	c := new(layoutCache)
	var evicted []int
	c.onEvict = func(d document) { evicted = append(evicted, d.alignWidth) }
	k := layoutKey{str: "k"}
	c.Put(k, document{alignWidth: 1})
	c.Put(k, document{alignWidth: 2})
	if len(evicted) != 1 || evicted[0] != 1 {
		t.Fatalf("expected onEvict to fire once with old value 1, got %v", evicted)
	}
}

func TestLRUPutOverwriteAtCapacity(t *testing.T) {
	c := new(layoutCache)
	c.capLimit = 3
	keys := []layoutKey{{str: "a"}, {str: "b"}, {str: "c"}}
	for i, k := range keys {
		c.Put(k, document{alignWidth: i + 1})
	}
	// Overwrite "a". With the pre-fix bug the old "a" entry stays in the
	// linked list; subsequent Puts that exceed capacity then evict the
	// stale entry and delete the freshly written map entry.
	c.Put(keys[0], document{alignWidth: 99})
	c.Put(layoutKey{str: "d"}, document{alignWidth: 4})
	c.Put(layoutKey{str: "e"}, document{alignWidth: 5})
	if v, ok := c.Get(keys[0]); !ok || v.alignWidth != 99 {
		t.Fatalf("overwritten key 'a' lost or stale (ok=%v, value=%d)", ok, v.alignWidth)
	}
}

// Fix 2: pararagraphStart -> paragraphStart. After the trailing newline of
// a paragraph the Shaper emits a synthetic glyph whose Flags include
// FlagParagraphStart with valid Ascent/Descent.

func TestShaperTrailingParagraphStartGlyph(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	collection := []FontFace{{Face: ltrFace}}
	shaper := NewShaper(NoSystemFonts(), WithCollection(collection))
	shaper.LayoutString(Parameters{
		PxPerEm:  fixed.I(10),
		MinWidth: 200,
		MaxWidth: 200,
		Locale:   english,
	}, "a\n")

	var glyphs []Glyph
	for g, ok := shaper.NextGlyph(); ok; g, ok = shaper.NextGlyph() {
		glyphs = append(glyphs, g)
	}
	if len(glyphs) == 0 {
		t.Fatalf("no glyphs produced")
	}
	last := glyphs[len(glyphs)-1]
	if last.Flags&FlagParagraphStart == 0 {
		t.Fatalf("expected trailing glyph to have FlagParagraphStart, flags=%s", last.Flags)
	}
	if last.Flags&(FlagLineBreak|FlagRunBreak|FlagClusterBreak) !=
		(FlagLineBreak | FlagRunBreak | FlagClusterBreak) {
		t.Fatalf("expected trailing glyph to have line/run/cluster break flags, flags=%s", last.Flags)
	}
	if last.Ascent == 0 || last.Descent == 0 {
		t.Fatalf("paragraphStart glyph expected non-zero ascent/descent, got %d/%d", last.Ascent, last.Descent)
	}
}

func TestShaperParagraphStartAfterDoubleNewline(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	collection := []FontFace{{Face: ltrFace}}
	shaper := NewShaper(NoSystemFonts(), WithCollection(collection))
	shaper.LayoutString(Parameters{
		PxPerEm:  fixed.I(10),
		MinWidth: 200,
		MaxWidth: 200,
		Locale:   english,
	}, "a\n\nb")

	var glyphs []Glyph
	for g, ok := shaper.NextGlyph(); ok; g, ok = shaper.NextGlyph() {
		glyphs = append(glyphs, g)
	}
	starts := 0
	for _, g := range glyphs {
		if g.Flags&FlagParagraphStart != 0 {
			starts++
		}
	}
	if starts == 0 {
		t.Fatalf("expected at least one FlagParagraphStart glyph in %q, found none", "a\n\nb")
	}
}

// Fix 3: shaperImpl.Shape glyph X offset sign. The formula must be
// (g.X - x) + g.Offset.X (not minus). With two glyphs of identical IDs
// but different Offset.X on the second glyph, the resulting clip path
// bounds must shift in the same direction as the offset.

func TestShapeOffsetXSign(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	collection := []FontFace{{Face: ltrFace}}
	shaper := NewShaper(NoSystemFonts(), WithCollection(collection))
	shaper.LayoutString(Parameters{
		PxPerEm:  fixed.I(40),
		MinWidth: 200,
		MaxWidth: 200,
		Locale:   english,
	}, "ab")

	var raw []Glyph
	for g, ok := shaper.NextGlyph(); ok; g, ok = shaper.NextGlyph() {
		raw = append(raw, g)
	}
	// Pick the first two real glyphs (skip any synthetic break glyphs at
	// end of text).
	var gs []Glyph
	for _, g := range raw {
		if g.Advance != 0 {
			gs = append(gs, g)
			if len(gs) == 2 {
				break
			}
		}
	}
	if len(gs) < 2 {
		t.Skip("font produced fewer than 2 advancing glyphs; cannot exercise formula")
	}

	mk := func(off fixed.Int26_6) []Glyph {
		out := make([]Glyph, len(gs))
		copy(out, gs)
		// Apply offset to the SECOND glyph so (g.X - x) is non-zero and
		// the formula's two terms can be distinguished.
		out[1].Offset.X = off
		return out
	}

	// Use a fresh shaper for each call to bypass the path cache (which
	// keys on the glyph slice — different Offset.X already produces a
	// different cache key, but a fresh shaper guarantees no cross-talk).
	freshShape := func(g []Glyph) (uint64, [4]int) {
		s := NewShaper(NoSystemFonts(), WithCollection(collection))
		spec := s.Shape(g)
		v := reflect.ValueOf(spec)
		hash := v.FieldByName("hash").Uint()
		b := v.FieldByName("bounds")
		bounds := [4]int{
			int(b.FieldByName("Min").FieldByName("X").Int()),
			int(b.FieldByName("Min").FieldByName("Y").Int()),
			int(b.FieldByName("Max").FieldByName("X").Int()),
			int(b.FieldByName("Max").FieldByName("Y").Int()),
		}
		return hash, bounds
	}

	off := fixed.I(20)
	zeroHash, zeroBounds := freshShape(mk(0))
	posHash, posBounds := freshShape(mk(off))
	negHash, negBounds := freshShape(mk(-off))

	if zeroHash == posHash || zeroHash == negHash || posHash == negHash {
		t.Fatalf("expected distinct PathSpec hashes for offsets {-N,0,+N}, got %d/%d/%d",
			negHash, zeroHash, posHash)
	}
	// With the correct (+) formula a positive Offset.X shifts the second
	// glyph's contribution to the path to the RIGHT — Max.X grows. With
	// the buggy (-) formula it would shrink instead.
	if posBounds[2] <= zeroBounds[2] {
		t.Fatalf("positive Offset.X should expand bounds Max.X (formula sign bug); zero=%v pos=%v",
			zeroBounds, posBounds)
	}
	if negBounds[2] >= zeroBounds[2] {
		t.Fatalf("negative Offset.X should reduce bounds Max.X; zero=%v neg=%v",
			zeroBounds, negBounds)
	}
}

// Bonus regression (found while reviewing shaper state): reset() did not
// clear brokeParagraph or paragraphStart. If a previous Layout was
// abandoned mid-iteration with brokeParagraph=true, the FIRST glyph of
// the next layout would incorrectly carry FlagParagraphStart.
func TestShaperResetClearsBrokeParagraph(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	collection := []FontFace{{Face: ltrFace}}
	shaper := NewShaper(NoSystemFonts(), WithCollection(collection))

	shaper.LayoutString(Parameters{
		PxPerEm: fixed.I(10), MinWidth: 100, MaxWidth: 100, Locale: english,
	}, "a\n")
	for i := 0; i < 10; i++ {
		g, ok := shaper.NextGlyph()
		if !ok {
			break
		}
		if g.Flags&FlagParagraphBreak != 0 {
			break // abandon before consuming the trailing paragraphStart
		}
	}
	shaper.LayoutString(Parameters{
		PxPerEm: fixed.I(10), MinWidth: 100, MaxWidth: 100, Locale: english,
	}, "x")
	first, ok := shaper.NextGlyph()
	if !ok {
		t.Fatalf("no glyph from fresh layout")
	}
	if first.Flags&FlagParagraphStart != 0 {
		t.Fatalf("first glyph of fresh layout must not have FlagParagraphStart, flags=%s",
			first.Flags)
	}
}

// Sanity: the bitmap path uses the same formula at gotext.go:795. Guard
// against future divergence by comparing two glyph layouts with the same
// shape input — this just exercises the code path without crashing.
func TestShapeOffsetXFormulaSmoke(t *testing.T) {
	ltrFace, _ := opentype.Parse(goregular.TTF)
	collection := []FontFace{{Face: ltrFace}}
	shaper := NewShaper(NoSystemFonts(), WithCollection(collection))
	shaper.LayoutString(Parameters{
		PxPerEm: fixed.I(20),
		Locale:  english,
	}, strings.Repeat("a", 4))
	var gs []Glyph
	for g, ok := shaper.NextGlyph(); ok; g, ok = shaper.NextGlyph() {
		gs = append(gs, g)
	}
	_ = shaper.Shape(gs)
	_ = shaper.Bitmaps(gs)
}
