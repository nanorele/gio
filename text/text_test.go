package text

import (
	"testing"

	"github.com/nanorele/gio/io/system"
	"golang.org/x/image/math/fixed"
)

func TestAlignmentString(t *testing.T) {
	cases := []struct {
		a    Alignment
		want string
	}{
		{Start, "Start"},
		{End, "End"},
		{Middle, "Middle"},
	}
	for _, c := range cases {
		if got := c.a.String(); got != c.want {
			t.Errorf("Alignment(%d).String() = %q, want %q", c.a, got, c.want)
		}
	}
}

func TestAlignmentStringInvalidPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for invalid Alignment")
		}
	}()
	_ = Alignment(99).String()
}

func TestAlignmentAlignLTR(t *testing.T) {

	cases := []struct {
		name     string
		a        Alignment
		width    fixed.Int26_6
		maxWidth int
		want     fixed.Int26_6
	}{
		{"start fits", Start, fixed.I(50), 100, 0},
		{"start zero width", Start, 0, 100, 0},
		{"start equal", Start, fixed.I(100), 100, 0},
		{"start overflows", Start, fixed.I(150), 100, 0},
		{"end fits", End, fixed.I(50), 100, fixed.I(50)},
		{"end zero width", End, 0, 100, fixed.I(100)},
		{"end equal", End, fixed.I(100), 100, 0},
		{"end overflows", End, fixed.I(150), 100, fixed.I(-50)},
		{"middle fits", Middle, fixed.I(50), 100, fixed.I(25)},
		{"middle zero width", Middle, 0, 100, fixed.I(50)},
		{"middle equal", Middle, fixed.I(100), 100, 0},
		{"middle overflows", Middle, fixed.I(150), 100, fixed.I(-25)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.a.Align(system.LTR, c.width, c.maxWidth)
			if got != c.want {
				t.Errorf("Align(LTR, %v, %d) = %v, want %v", c.width, c.maxWidth, got, c.want)
			}
		})
	}
}

func TestAlignmentAlignRTL(t *testing.T) {

	cases := []struct {
		name     string
		a        Alignment
		width    fixed.Int26_6
		maxWidth int
		want     fixed.Int26_6
	}{
		{"start (right edge) fits", Start, fixed.I(50), 100, fixed.I(50)},
		{"start zero width", Start, 0, 100, fixed.I(100)},
		{"end (left edge) fits", End, fixed.I(50), 100, 0},
		{"end zero width", End, 0, 100, 0},
		{"middle fits", Middle, fixed.I(50), 100, fixed.I(25)},
		{"middle zero width", Middle, 0, 100, fixed.I(50)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.a.Align(system.RTL, c.width, c.maxWidth)
			if got != c.want {
				t.Errorf("Align(RTL, %v, %d) = %v, want %v", c.width, c.maxWidth, got, c.want)
			}
		})
	}
}

func TestAlignmentAlignNegativeContentWidth(t *testing.T) {

	a := Start
	if got := a.Align(system.LTR, fixed.I(-10), 100); got != 0 {
		t.Errorf("Start with negative width should still return 0, got %v", got)
	}

	a = End
	if got := a.Align(system.LTR, fixed.I(-10), 100); got != fixed.I(110) {
		t.Errorf("End with negative width: got %v, want %v", got, fixed.I(110))
	}

	a = Middle
	if got := a.Align(system.LTR, fixed.I(-10), 100); got != fixed.I(55) {
		t.Errorf("Middle with negative width: got %v, want %v", got, fixed.I(55))
	}
}

func TestAlignmentAlignZeroMaxWidth(t *testing.T) {

	if got := Start.Align(system.LTR, fixed.I(50), 0); got != 0 {
		t.Errorf("Start zero maxWidth: got %v, want 0", got)
	}
	if got := End.Align(system.LTR, fixed.I(50), 0); got != fixed.I(-50) {
		t.Errorf("End zero maxWidth: got %v, want %v", got, fixed.I(-50))
	}
	if got := Middle.Align(system.LTR, fixed.I(50), 0); got != fixed.I(-25) {
		t.Errorf("Middle zero maxWidth: got %v, want %v", got, fixed.I(-25))
	}
}

func TestAlignmentAlignInvalidPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for invalid Alignment")
		}
	}()
	_ = Alignment(99).Align(system.LTR, 0, 100)
}

func TestWrapPolicyConstants(t *testing.T) {

	if WrapHeuristically != 0 {
		t.Errorf("WrapHeuristically = %d, want 0", WrapHeuristically)
	}
	if WrapWords != 1 {
		t.Errorf("WrapWords = %d, want 1", WrapWords)
	}
	if WrapGraphemes != 2 {
		t.Errorf("WrapGraphemes = %d, want 2", WrapGraphemes)
	}
}

func TestAlignmentConstants(t *testing.T) {
	if Start != 0 {
		t.Errorf("Start = %d, want 0", Start)
	}
	if End != 1 {
		t.Errorf("End = %d, want 1", End)
	}
	if Middle != 2 {
		t.Errorf("Middle = %d, want 2", Middle)
	}
}

func TestParametersZeroValue(t *testing.T) {

	var p Parameters
	if p.Alignment != Start {
		t.Errorf("zero Parameters.Alignment = %v, want Start", p.Alignment)
	}
	if p.WrapPolicy != WrapHeuristically {
		t.Errorf("zero Parameters.WrapPolicy = %v, want WrapHeuristically", p.WrapPolicy)
	}
	if p.PxPerEm != 0 {
		t.Errorf("zero Parameters.PxPerEm = %v, want 0", p.PxPerEm)
	}
	if p.MaxLines != 0 {
		t.Errorf("zero Parameters.MaxLines = %v, want 0", p.MaxLines)
	}
	if p.Truncator != "" {
		t.Errorf("zero Parameters.Truncator = %q, want empty", p.Truncator)
	}
	if p.MinWidth != 0 || p.MaxWidth != 0 {
		t.Errorf("zero Parameters width = (%d,%d), want (0,0)", p.MinWidth, p.MaxWidth)
	}
	if p.LineHeightScale != 0 {
		t.Errorf("zero Parameters.LineHeightScale = %v, want 0", p.LineHeightScale)
	}
	if p.LineHeight != 0 {
		t.Errorf("zero Parameters.LineHeight = %v, want 0", p.LineHeight)
	}
	if p.DisableSpaceTrim {
		t.Errorf("zero Parameters.DisableSpaceTrim = true, want false")
	}
	if p.Locale.Language != "" || p.Locale.Direction != system.LTR {
		t.Errorf("zero Parameters.Locale = %+v, want zero/LTR", p.Locale)
	}
}
