package text

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/nanorele/gio/font/gofont"
	"github.com/nanorele/gio/io/system"
	"golang.org/x/image/math/fixed"
)

func auditParams(w int) Parameters {
	return Parameters{
		PxPerEm:  fixed.I(13),
		MaxWidth: w,
		Locale:   system.Locale{Language: "en", Direction: system.LTR},
	}
}

// TestAuditDocPoolUnderPressure hammers the layout cache so documents are
// evicted and recycled constantly, and checks every layout against a shaper
// that has never recycled anything. A recycled document handed out while still
// aliased would show up here as a corrupted glyph stream.
func TestAuditDocPoolUnderPressure(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	texts := make([]string, 400)
	for i := range texts {
		var b strings.Builder
		n := 20 + rng.Intn(600)
		for j := 0; j < n; j++ {
			b.WriteByte(byte(0x20 + rng.Intn(0x5F)))
		}
		texts[i] = b.String()
	}

	stressed := NewShaper(NoSystemFonts(), WithCollection(gofont.Collection()))
	// Squeeze the cache so nearly every layout evicts something and the pool
	// stays hot.
	stressed.layoutCache.costLimit = 16 << 10

	mismatch := 0
	for round := 0; round < 6; round++ {
		for i, txt := range texts {
			w := 60 + (i*37)%700
			got := auditGlyphs(t, stressed, auditParams(w), txt)

			fresh := NewShaper(NoSystemFonts(), WithCollection(gofont.Collection()))
			want := auditGlyphs(t, fresh, auditParams(w), txt)

			if len(got) != len(want) {
				t.Fatalf("round %d text %d w=%d: %d glyphs, want %d", round, i, w, len(got), len(want))
			}
			for k := range got {
				if got[k] != want[k] {
					mismatch++
					t.Fatalf("round %d text %d w=%d: glyph %d = %+v, want %+v",
						round, i, w, k, got[k], want[k])
				}
			}
		}
	}
	fmt.Printf("doc pool: %d layouts under eviction pressure, all glyph-identical\n", 6*len(texts))
}

// TestAuditDocPoolInterleavedIteration recycles documents while a previous
// layout is still being read back glyph by glyph — the exact aliasing the
// generation counter is meant to prevent.
func TestAuditDocPoolInterleavedIteration(t *testing.T) {
	sh := NewShaper(NoSystemFonts(), WithCollection(gofont.Collection()))
	sh.layoutCache.costLimit = 8 << 10

	base := strings.Repeat("interleaved reading of a shaped paragraph ", 12)
	ref := NewShaper(NoSystemFonts(), WithCollection(gofont.Collection()))
	want := auditGlyphs(t, ref, auditParams(300), base)

	for round := 0; round < 40; round++ {
		sh.LayoutString(auditParams(300), base)
		var got []Glyph
		for i := 0; ; i++ {
			g, ok := sh.NextGlyph()
			if !ok {
				break
			}
			got = append(got, g)
			// Halfway through, run unrelated layouts that evict and recycle.
			if i == len(want)/2 {
				other := NewShaper(NoSystemFonts(), WithCollection(gofont.Collection()))
				other.layoutCache.costLimit = 8 << 10
				for j := 0; j < 20; j++ {
					other.LayoutString(auditParams(120+j), strings.Repeat("evict", 40+j))
					for {
						if _, ok := other.NextGlyph(); !ok {
							break
						}
					}
				}
			}
		}
		if len(got) != len(want) {
			t.Fatalf("round %d: %d glyphs, want %d", round, len(got), len(want))
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("round %d: glyph %d = %+v, want %+v", round, i, got[i], want[i])
			}
		}
	}
	fmt.Printf("doc pool: 40 interleaved read/evict rounds glyph-identical\n")
}

// TestAuditTruncCacheAcrossFonts checks the truncator cache is keyed finely
// enough: the same truncator at different sizes, fonts and locales must not
// serve each other's shaped output.
func TestAuditTruncCacheAcrossFonts(t *testing.T) {
	sh := NewShaper(NoSystemFonts(), WithCollection(gofont.Collection()))
	txt := strings.Repeat("truncate me please ", 40)

	type key struct {
		ppem  fixed.Int26_6
		trunc string
	}
	seen := map[key][]Glyph{}
	for _, ppem := range []fixed.Int26_6{fixed.I(8), fixed.I(13), fixed.I(32)} {
		for _, tr := range []string{"…", "...", "»"} {
			params := Parameters{
				PxPerEm: ppem, MaxWidth: 150, MaxLines: 1, Truncator: tr,
				Locale: system.Locale{Language: "en", Direction: system.LTR},
			}
			got := auditGlyphs(t, sh, params, txt)
			fresh := NewShaper(NoSystemFonts(), WithCollection(gofont.Collection()))
			want := auditGlyphs(t, fresh, params, txt)
			if len(got) != len(want) {
				t.Fatalf("ppem=%v trunc=%q: %d glyphs, want %d", ppem, tr, len(got), len(want))
			}
			for i := range got {
				if got[i] != want[i] {
					t.Fatalf("ppem=%v trunc=%q: glyph %d differs", ppem, tr, i)
				}
			}
			seen[key{ppem, tr}] = got
		}
	}
	// Different sizes must produce different truncator geometry.
	a := seen[key{fixed.I(8), "…"}]
	b := seen[key{fixed.I(32), "…"}]
	if len(a) > 0 && len(b) > 0 && a[len(a)-1] == b[len(b)-1] {
		t.Error("truncator glyph identical at 8px and 32px — cache key too coarse")
	}
	fmt.Printf("truncator cache: %d (size,truncator) pairs match fresh shapers\n", len(seen))
}
