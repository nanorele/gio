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

// auditGlyphs collects the full glyph stream a Shaper produces for params/txt.
func auditGlyphs(t *testing.T, sh *Shaper, params Parameters, txt string) []Glyph {
	t.Helper()
	sh.LayoutString(params, txt)
	var out []Glyph
	for {
		g, ok := sh.NextGlyph()
		if !ok {
			return out
		}
		out = append(out, g)
	}
}

// TestAuditSingleLineTrimEquivalence is an independent check of the
// MaxLines==1 shaping cutoff: for an adversarial corpus it compares the glyph
// stream against a shaper with the cutoff disabled. Any difference is a
// visible truncation bug.
func TestAuditSingleLineTrimEquivalence(t *testing.T) {
	rng := rand.New(rand.NewSource(20260729))
	randASCII := func(n int) string {
		var b strings.Builder
		for i := 0; i < n; i++ {
			b.WriteByte(byte(0x20 + rng.Intn(0x5F)))
		}
		return b.String()
	}

	corpus := []string{
		"",
		" ",
		strings.Repeat(" ", 300),
		"short",
		strings.Repeat("i", 4000),
		strings.Repeat("W", 4000),
		strings.Repeat("iW", 2000),
		strings.Repeat("fi fl ffi ", 400),
		"AV AW To Ta Wa " + strings.Repeat("AV", 1000),
		strings.Repeat("word ", 800),
		"lead" + strings.Repeat(" ", 400) + "tail",
		strings.Repeat("a", 500) + strings.Repeat(" ", 500) + strings.Repeat("b", 500),
		"https://api.example.com/v2/resources/1234567890?expand=items,owner&limit=50&page=2",
		strings.Repeat("...", 1500),
		strings.Repeat("\t", 200) + "after tabs",
		randASCII(3000),
		randASCII(64),
		randASCII(65),
	}

	shTrim := NewShaper(NoSystemFonts(), WithCollection(gofont.Collection()))
	shFull := NewShaper(NoSystemFonts(), WithCollection(gofont.Collection()))
	shFull.shaper.disableSingleLineTrim = true

	widths := []int{1, 2, 7, 13, 40, 97, 200, 313, 800, 4000}
	sizes := []fixed.Int26_6{fixed.I(1), fixed.I(8), fixed.I(13), fixed.I(64)}
	truncators := []string{"", "…", "...", ">>>>>>>>"}

	checked := 0
	for ci, txt := range corpus {
		for _, w := range widths {
			for _, sz := range sizes {
				for _, tr := range truncators {
					params := Parameters{
						PxPerEm:   sz,
						MaxWidth:  w,
						MaxLines:  1,
						Truncator: tr,
						Locale:    system.Locale{Language: "en", Direction: system.LTR},
					}
					got := auditGlyphs(t, shTrim, params, txt)
					want := auditGlyphs(t, shFull, params, txt)
					checked++
					if len(got) != len(want) {
						t.Fatalf("corpus[%d] w=%d ppem=%v trunc=%q: %d glyphs, want %d",
							ci, w, sz, tr, len(got), len(want))
					}
					for i := range got {
						if got[i] != want[i] {
							t.Fatalf("corpus[%d] w=%d ppem=%v trunc=%q: glyph %d = %+v, want %+v",
								ci, w, sz, tr, i, got[i], want[i])
						}
					}
				}
			}
		}
	}
	fmt.Printf("single-line trim: %d (text,width,size,truncator) combinations glyph-identical\n", checked)
}

// TestAuditTrimRuneAccounting checks that the runes the cutoff never shapes are
// still accounted for, so caret mapping over a truncated field stays correct.
func TestAuditTrimRuneAccounting(t *testing.T) {
	sh := NewShaper(NoSystemFonts(), WithCollection(gofont.Collection()))
	for _, n := range []int{10, 100, 1000, 20000} {
		txt := strings.Repeat("x", n)
		params := Parameters{
			PxPerEm: fixed.I(13), MaxWidth: 120, MaxLines: 1, Truncator: "…",
			Locale: system.Locale{Language: "en", Direction: system.LTR},
		}
		var runes int
		sh.LayoutString(params, txt)
		for {
			g, ok := sh.NextGlyph()
			if !ok {
				break
			}
			runes += int(g.Runes)
		}
		if runes != n {
			t.Errorf("len=%d: glyph stream accounts for %d runes, want %d", n, runes, len(txt))
		}
	}
}

// TestAuditNonASCIINotTrimmed pins the guard: anything the cutoff cannot
// reason about must go through full shaping.
func TestAuditNonASCIINotTrimmed(t *testing.T) {
	sh := NewShaper(NoSystemFonts(), WithCollection(gofont.Collection()))
	full := NewShaper(NoSystemFonts(), WithCollection(gofont.Collection()))
	full.shaper.disableSingleLineTrim = true

	cases := []struct {
		name string
		txt  string
		dir  system.TextDirection
	}{
		{"cyrillic", strings.Repeat("привет ", 500), system.LTR},
		{"cjk", strings.Repeat("你好世界", 500), system.LTR},
		{"rtl-locale-ascii", strings.Repeat("hello ", 500), system.RTL},
		{"combining", strings.Repeat("é", 1000), system.LTR},
		{"nbsp", strings.Repeat("a b ", 500), system.LTR},
	}
	for _, c := range cases {
		params := Parameters{
			PxPerEm: fixed.I(13), MaxWidth: 200, MaxLines: 1, Truncator: "…",
			Locale: system.Locale{Language: "en", Direction: c.dir},
		}
		got := auditGlyphs(t, sh, params, c.txt)
		want := auditGlyphs(t, full, params, c.txt)
		if len(got) != len(want) {
			t.Fatalf("%s: %d glyphs, want %d", c.name, len(got), len(want))
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("%s: glyph %d differs", c.name, i)
			}
		}
	}
}
