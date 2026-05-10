package opentype

import (
	"testing"

	giofont "github.com/nanorele/gio/font"
	fontapi "github.com/nanorele/typesetting/font"
)

func TestGioStyleRoundTrip(t *testing.T) {
	if got := gioStyle(fontapi.StyleItalic); got != giofont.Italic {
		t.Errorf("gioStyle(StyleItalic) = %v, want Italic", got)
	}
	if got := gioStyle(fontapi.StyleNormal); got != giofont.Regular {
		t.Errorf("gioStyle(StyleNormal) = %v, want Regular", got)
	}
	// Unknown style should fall through to Regular.
	if got := gioStyle(fontapi.Style(99)); got != giofont.Regular {
		t.Errorf("gioStyle(unknown) = %v, want Regular", got)
	}
}

func TestMdStyleRoundTrip(t *testing.T) {
	if got := mdStyle(giofont.Italic); got != fontapi.StyleItalic {
		t.Errorf("mdStyle(Italic) = %v, want StyleItalic", got)
	}
	if got := mdStyle(giofont.Regular); got != fontapi.StyleNormal {
		t.Errorf("mdStyle(Regular) = %v, want StyleNormal", got)
	}
	// Unknown style should fall through to StyleNormal.
	if got := mdStyle(giofont.Style(99)); got != fontapi.StyleNormal {
		t.Errorf("mdStyle(unknown) = %v, want StyleNormal", got)
	}
}

func TestStyleRoundTrip(t *testing.T) {
	for _, s := range []giofont.Style{giofont.Regular, giofont.Italic} {
		if got := gioStyle(mdStyle(s)); got != s {
			t.Errorf("round-trip Style %v: got %v", s, got)
		}
	}
}

func TestGioWeight(t *testing.T) {
	cases := []struct {
		in   fontapi.Weight
		want giofont.Weight
	}{
		{fontapi.WeightThin, giofont.Thin},
		{fontapi.WeightExtraLight, giofont.ExtraLight},
		{fontapi.WeightLight, giofont.Light},
		{fontapi.WeightNormal, giofont.Normal},
		{fontapi.WeightMedium, giofont.Medium},
		{fontapi.WeightSemibold, giofont.SemiBold},
		{fontapi.WeightBold, giofont.Bold},
		{fontapi.WeightExtraBold, giofont.ExtraBold},
		{fontapi.WeightBlack, giofont.Black},
	}
	for _, c := range cases {
		if got := gioWeight(c.in); got != c.want {
			t.Errorf("gioWeight(%v) = %v, want %v", c.in, got, c.want)
		}
	}
	// Unknown weight defaults to Normal.
	if got := gioWeight(fontapi.Weight(9999)); got != giofont.Normal {
		t.Errorf("gioWeight(unknown) = %v, want Normal", got)
	}
}

func TestMdWeight(t *testing.T) {
	cases := []struct {
		in   giofont.Weight
		want fontapi.Weight
	}{
		{giofont.Thin, fontapi.WeightThin},
		{giofont.ExtraLight, fontapi.WeightExtraLight},
		{giofont.Light, fontapi.WeightLight},
		{giofont.Normal, fontapi.WeightNormal},
		{giofont.Medium, fontapi.WeightMedium},
		{giofont.SemiBold, fontapi.WeightSemibold},
		{giofont.Bold, fontapi.WeightBold},
		{giofont.ExtraBold, fontapi.WeightExtraBold},
		{giofont.Black, fontapi.WeightBlack},
	}
	for _, c := range cases {
		if got := mdWeight(c.in); got != c.want {
			t.Errorf("mdWeight(%v) = %v, want %v", c.in, got, c.want)
		}
	}
	// Unknown weight defaults to Normal.
	if got := mdWeight(giofont.Weight(9999)); got != fontapi.WeightNormal {
		t.Errorf("mdWeight(unknown) = %v, want WeightNormal", got)
	}
}

func TestWeightRoundTrip(t *testing.T) {
	weights := []giofont.Weight{
		giofont.Thin, giofont.ExtraLight, giofont.Light, giofont.Normal,
		giofont.Medium, giofont.SemiBold, giofont.Bold, giofont.ExtraBold,
		giofont.Black,
	}
	for _, w := range weights {
		if got := gioWeight(mdWeight(w)); got != w {
			t.Errorf("round-trip Weight %v: got %v", w, got)
		}
	}
}

func TestDescriptionFontRoundTrip(t *testing.T) {
	in := giofont.Font{
		Typeface: "Foo",
		Style:    giofont.Italic,
		Weight:   giofont.Bold,
	}
	desc := FontToDescription(in)
	if desc.Family != "Foo" {
		t.Errorf("Family = %q, want %q", desc.Family, "Foo")
	}
	if desc.Aspect.Style != fontapi.StyleItalic {
		t.Errorf("Aspect.Style = %v, want StyleItalic", desc.Aspect.Style)
	}
	if desc.Aspect.Weight != fontapi.WeightBold {
		t.Errorf("Aspect.Weight = %v, want WeightBold", desc.Aspect.Weight)
	}
	out := DescriptionToFont(desc)
	if out != in {
		t.Errorf("round-trip: got %+v, want %+v", out, in)
	}
}

func TestParseInvalid(t *testing.T) {
	if _, err := Parse(nil); err == nil {
		t.Errorf("Parse(nil) returned nil error, want error")
	}
	if _, err := Parse([]byte{0, 0, 0, 0}); err == nil {
		t.Errorf("Parse([4 zero bytes]) returned nil error, want error")
	}
}

func TestParseCollectionInvalid(t *testing.T) {
	if _, err := ParseCollection(nil); err == nil {
		t.Errorf("ParseCollection(nil) returned nil error, want error")
	}
}
