package text

import (
	"fmt"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"

	"github.com/nanorele/gio/font/opentype"
	"github.com/nanorele/gio/io/system"
)

// benchShaper builds a Shaper the way the application does: no system fonts,
// a single collection face.
func benchShaper(b testing.TB) *Shaper {
	face, err := opentype.Parse(goregular.TTF)
	if err != nil {
		b.Fatal(err)
	}
	return NewShaper(NoSystemFonts(), WithCollection([]FontFace{{Face: face}}))
}

// benchStrings returns n distinct strings of the given rune length. Cycling
// through more strings than the layout cache holds keeps every LayoutString
// call a cache miss.
func benchStrings(n, strLen int) []string {
	out := make([]string, n)
	for i := range out {
		s := fmt.Sprintf("label %06d ", i)
		for len(s) < strLen {
			s += "abcdefgh "
		}
		out[i] = s[:strLen]
	}
	return out
}

func benchDrain(l *Shaper) int {
	n := 0
	for {
		_, ok := l.NextGlyph()
		if !ok {
			return n
		}
		n++
	}
}

// BenchmarkLayoutMissShort measures the allocation cost of laying out a short
// label whose text is not in the layout cache (T1 acceptance: <=10KB).
func BenchmarkLayoutMissShort(b *testing.B) {
	l := benchShaper(b)
	strs := benchStrings(4096, 13)
	params := Parameters{
		PxPerEm:  fixed.I(14),
		MaxWidth: 1200,
		MinWidth: 0,
		Locale:   system.Locale{Language: "EN", Direction: system.LTR},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		l.LayoutString(params, strs[i%len(strs)])
		benchDrain(l)
	}
}

// BenchmarkLayoutMissTruncated measures an 80-char label with MaxLines: 1 in
// a narrow cell, the list-cell case (T2 acceptance: cost proportional to the
// visible part; T1 acceptance: <=25KB).
func BenchmarkLayoutMissTruncated(b *testing.B) {
	l := benchShaper(b)
	strs := benchStrings(4096, 80)
	params := Parameters{
		PxPerEm:  fixed.I(14),
		MaxWidth: 200,
		MaxLines: 1,
		Locale:   system.Locale{Language: "EN", Direction: system.LTR},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		l.LayoutString(params, strs[i%len(strs)])
		benchDrain(l)
	}
}

// BenchmarkLayoutMissLongTruncated is a 500-char string in a 200px cell with
// MaxLines: 1. T2 acceptance: not more expensive than the 40-char version.
func BenchmarkLayoutMissLongTruncated(b *testing.B) {
	l := benchShaper(b)
	strs := benchStrings(4096, 500)
	params := Parameters{
		PxPerEm:  fixed.I(14),
		MaxWidth: 200,
		MaxLines: 1,
		Locale:   system.Locale{Language: "EN", Direction: system.LTR},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		l.LayoutString(params, strs[i%len(strs)])
		benchDrain(l)
	}
}

// BenchmarkLayoutMissShortTruncated40 is the 40-char reference point for
// BenchmarkLayoutMissLongTruncated.
func BenchmarkLayoutMissShortTruncated40(b *testing.B) {
	l := benchShaper(b)
	strs := benchStrings(4096, 40)
	params := Parameters{
		PxPerEm:  fixed.I(14),
		MaxWidth: 200,
		MaxLines: 1,
		Locale:   system.Locale{Language: "EN", Direction: system.LTR},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		l.LayoutString(params, strs[i%len(strs)])
		benchDrain(l)
	}
}

// BenchmarkLayoutHit is the steady-state reference: repeated layout of the
// same string must stay at ~zero allocations.
func BenchmarkLayoutHit(b *testing.B) {
	l := benchShaper(b)
	params := Parameters{
		PxPerEm:  fixed.I(14),
		MaxWidth: 1200,
		Locale:   system.Locale{Language: "EN", Direction: system.LTR},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		l.LayoutString(params, "hello, world!")
		benchDrain(l)
	}
}
