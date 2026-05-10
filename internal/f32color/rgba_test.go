package f32color

import (
	"image/color"
	"math"
	"testing"
)

func TestNRGBAToLinearRGBA_Boundary(t *testing.T) {
	for col := 0; col <= 0xFF; col++ {
		for alpha := 0; alpha <= 0xFF; alpha++ {
			in := color.NRGBA{R: uint8(col), A: uint8(alpha)}
			premul := NRGBAToLinearRGBA(in)
			if premul.A != uint8(alpha) {
				t.Errorf("%v: got %v expected %v", in, premul.A, alpha)
			}
			if premul.R > premul.A {
				t.Errorf("%v: R=%v > A=%v", in, premul.R, premul.A)
			}
		}
	}
}

func TestLinearToRGBARoundtrip(t *testing.T) {
	for col := 0; col <= 0xFF; col++ {
		for alpha := 0; alpha <= 0xFF; alpha++ {
			want := color.NRGBA{R: uint8(col), A: uint8(alpha)}
			if alpha == 0 {
				want.R = 0
			}
			got := LinearFromSRGB(want).SRGB()
			if want != got {
				t.Errorf("got %v expected %v", got, want)
			}
		}
	}
}

var sink RGBA

func BenchmarkLinearFromSRGB(b *testing.B) {
	b.Run("opaque", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			sink = LinearFromSRGB(color.NRGBA{R: byte(i), G: byte(i >> 8), B: byte(i >> 16), A: 0xFF})
		}
	})
	b.Run("translucent", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			sink = LinearFromSRGB(color.NRGBA{R: byte(i), G: byte(i >> 8), B: byte(i >> 16), A: 0x50})
		}
	})
	b.Run("transparent", func(b *testing.B) {
		for i := 0; b.Loop(); i++ {
			sink = LinearFromSRGB(color.NRGBA{R: byte(i), G: byte(i >> 8), B: byte(i >> 16), A: 0x00})
		}
	})
}

func TestRGBA_Array_Float32(t *testing.T) {
	c := RGBA{R: 0.1, G: 0.2, B: 0.3, A: 0.4}
	arr := c.Array()
	if arr != [4]float32{0.1, 0.2, 0.3, 0.4} {
		t.Errorf("Array() got %v", arr)
	}
	r, g, b, a := c.Float32()
	if r != 0.1 || g != 0.2 || b != 0.3 || a != 0.4 {
		t.Errorf("Float32() got %v, %v, %v, %v", r, g, b, a)
	}
}

func TestRGBA_Opaque(t *testing.T) {
	c := RGBA{R: 0.1, G: 0.2, B: 0.3, A: 0.4}
	op := c.Opaque()
	if op.A != 1.0 {
		t.Errorf("Opaque() got %v", op)
	}
}

func TestSRGB_ZeroAlpha(t *testing.T) {
	c := RGBA{R: 1.0, G: 1.0, B: 1.0, A: 0.0}
	srgb := c.SRGB()
	if srgb.R != 0 || srgb.G != 0 || srgb.B != 0 || srgb.A != 0 {
		t.Errorf("SRGB() with zero alpha got %v", srgb)
	}
}

func TestLuminance(t *testing.T) {
	c := RGBA{R: 1.0, G: 1.0, B: 1.0, A: 1.0}
	lum := c.Luminance()
	if lum < 0.99 || lum > 1.01 {
		t.Errorf("Luminance() for white got %v", lum)
	}
}

func TestNRGBAToRGBA(t *testing.T) {
	c := color.NRGBA{R: 100, G: 150, B: 200, A: 128}
	rgba := NRGBAToRGBA(c)
	if rgba.A != 128 {
		t.Errorf("NRGBAToRGBA() got %v", rgba)
	}
	cFull := color.NRGBA{R: 100, G: 150, B: 200, A: 255}
	rgbaFull := NRGBAToRGBA(cFull)
	if rgbaFull.A != 255 || rgbaFull.R != 100 {
		t.Errorf("NRGBAToRGBA() full alpha got %v", rgbaFull)
	}
}

func TestRGBAToNRGBA(t *testing.T) {
	c := color.RGBA{R: 50, G: 75, B: 100, A: 128}
	nrgba := RGBAToNRGBA(c)
	if nrgba.A != 128 {
		t.Errorf("RGBAToNRGBA() got %v", nrgba)
	}
	cFull := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	nrgbaFull := RGBAToNRGBA(cFull)
	if nrgbaFull.A != 255 || nrgbaFull.R != 100 {
		t.Errorf("RGBAToNRGBA() full alpha got %v", nrgbaFull)
	}
}

func TestMulAlpha(t *testing.T) {
	c := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	m := MulAlpha(c, 128)
	if m.A != 128 {
		t.Errorf("MulAlpha() got %v", m.A)
	}
}

func TestDisabledHovered(t *testing.T) {
	c := color.NRGBA{R: 100, G: 100, B: 100, A: 255}
	d := Disabled(c)
	if d.A == 255 {
		t.Errorf("Disabled() got alpha %v", d.A)
	}
	h := Hovered(c)
	if h.R == 100 && h.G == 100 && h.B == 100 {
		t.Errorf("Hovered() did not change color")
	}
	zeroAlpha := Hovered(color.NRGBA{A: 0})
	if zeroAlpha.A != 0x44 {
		t.Errorf("Hovered(transparent) got %v", zeroAlpha)
	}
	cLight := color.NRGBA{R: 200, G: 200, B: 200, A: 255}
	hLight := Hovered(cLight)
	if hLight.R >= 200 {
		t.Errorf("Hovered(light color) got %v", hLight)
	}
}

func TestLinearTosRGB_Boundary(t *testing.T) {
	if v := linearTosRGB(-0.1); v != 0 {
		t.Errorf("linearTosRGB(-0.1) = %v, want 0", v)
	}
	if v := linearTosRGB(1.5); v != 1 {
		t.Errorf("linearTosRGB(1.5) = %v, want 1", v)
	}
}

const colorEps = 1e-5

func floatNear(a, b, eps float32) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
}

// referenceSRGBToLinear is the canonical sRGB -> linear formula.
func referenceSRGBToLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func TestSRGBToLinear_Reference(t *testing.T) {
	for _, c := range []float32{0, 0.01, 0.04045, 0.1, 0.5, 0.8, 1.0} {
		got := sRGBToLinear(c)
		want := float32(referenceSRGBToLinear(float64(c)))
		if !floatNear(got, want, colorEps) {
			t.Errorf("sRGBToLinear(%v) = %v, want %v", c, got, want)
		}
	}
}

func TestLinearTosRGB_Reference(t *testing.T) {
	// Verify low/high segments and a couple of midpoints against the formula.
	cases := []float32{0, 0.001, 0.0031308, 0.01, 0.1, 0.5, 1.0}
	for _, c := range cases {
		got := linearTosRGB(c)
		var want float32
		switch {
		case c <= 0:
			want = 0
		case c < 0.0031308:
			want = 12.92 * c
		case c < 1:
			want = float32(1.055*math.Pow(float64(c), 1.0/2.4) - 0.055)
		default:
			want = 1
		}
		// Allow slightly larger epsilon since linearTosRGB uses 0.41666 instead of 1/2.4.
		if !floatNear(got, want, 1e-3) {
			t.Errorf("linearTosRGB(%v) = %v, want %v", c, got, want)
		}
	}
}

func TestLinearFromSRGB_KnownValues(t *testing.T) {
	cases := []struct {
		in   color.NRGBA
		want RGBA
	}{
		{color.NRGBA{R: 0, G: 0, B: 0, A: 0xFF}, RGBA{0, 0, 0, 1}},
		{color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}, RGBA{1, 1, 1, 1}},
		{color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xFF}, RGBA{
			R: float32(referenceSRGBToLinear(float64(0x80) / 255)),
			G: float32(referenceSRGBToLinear(float64(0x80) / 255)),
			B: float32(referenceSRGBToLinear(float64(0x80) / 255)),
			A: 1,
		}},
		{color.NRGBA{R: 0xFF, A: 0}, RGBA{0, 0, 0, 0}},
	}
	for _, tc := range cases {
		got := LinearFromSRGB(tc.in)
		if !floatNear(got.R, tc.want.R, colorEps) ||
			!floatNear(got.G, tc.want.G, colorEps) ||
			!floatNear(got.B, tc.want.B, colorEps) ||
			!floatNear(got.A, tc.want.A, colorEps) {
			t.Errorf("LinearFromSRGB(%v) = %+v, want %+v", tc.in, got, tc.want)
		}
	}
}

func TestSRGBTable_Monotonic(t *testing.T) {
	if len(srgb8ToLinear) != 256 {
		t.Fatalf("table length = %d, want 256", len(srgb8ToLinear))
	}
	if srgb8ToLinear[0] != 0 {
		t.Errorf("table[0] = %v, want 0", srgb8ToLinear[0])
	}
	if !floatNear(srgb8ToLinear[255], 1, colorEps) {
		t.Errorf("table[255] = %v, want 1", srgb8ToLinear[255])
	}
	for i := 1; i < len(srgb8ToLinear); i++ {
		if srgb8ToLinear[i] < srgb8ToLinear[i-1] {
			t.Errorf("table not monotonic at %d: %v < %v", i, srgb8ToLinear[i], srgb8ToLinear[i-1])
		}
	}
}

func TestSRGBTable_MatchesFormula(t *testing.T) {
	for i := 0; i < 256; i++ {
		want := float32(referenceSRGBToLinear(float64(i) / 255))
		if !floatNear(srgb8ToLinear[i], want, colorEps) {
			t.Errorf("table[%d] = %v, want %v", i, srgb8ToLinear[i], want)
		}
	}
}

func TestMulAlpha_EdgeCases(t *testing.T) {
	c := color.NRGBA{R: 10, G: 20, B: 30, A: 200}
	// alpha = 0 -> A becomes 0
	if got := MulAlpha(c, 0); got.A != 0 {
		t.Errorf("MulAlpha(_, 0).A = %v, want 0", got.A)
	}
	// alpha = 0xFF -> A unchanged
	if got := MulAlpha(c, 0xFF); got.A != 200 {
		t.Errorf("MulAlpha(_, 0xFF).A = %v, want 200", got.A)
	}
	// MulAlpha must not modify RGB
	if got := MulAlpha(c, 100); got.R != c.R || got.G != c.G || got.B != c.B {
		t.Errorf("MulAlpha changed RGB: %v", got)
	}
	// 0 in alpha
	z := color.NRGBA{R: 50, G: 50, B: 50, A: 0}
	if got := MulAlpha(z, 200); got.A != 0 {
		t.Errorf("MulAlpha({A:0}, 200).A = %v, want 0", got.A)
	}
	// numerical: 128 * 128 / 255 = 64
	c2 := color.NRGBA{A: 128}
	if got := MulAlpha(c2, 128); got.A != 64 {
		t.Errorf("MulAlpha(128,128).A = %v, want 64", got.A)
	}
	// 255 * 255 / 255 = 255
	c3 := color.NRGBA{A: 255}
	if got := MulAlpha(c3, 255); got.A != 255 {
		t.Errorf("MulAlpha(255,255).A = %v, want 255", got.A)
	}
}

func TestNRGBAToRGBA_RoundTrip(t *testing.T) {
	// Round-trip via RGBAToNRGBA. With premultiplied math we accept small loss.
	cases := []color.NRGBA{
		{R: 0, G: 0, B: 0, A: 0},
		{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
		{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x80},
		{R: 0x40, G: 0x80, B: 0xC0, A: 0x80},
		{R: 100, G: 150, B: 200, A: 200},
	}
	for _, in := range cases {
		rgba := NRGBAToRGBA(in)
		got := RGBAToNRGBA(rgba)
		// alpha must round-trip exactly
		if got.A != in.A {
			t.Errorf("alpha round-trip %v -> %v -> %v: A mismatch", in, rgba, got)
		}
		// when alpha=0 the channels are by spec zeroed
		if in.A == 0 {
			continue
		}
		// allow ±1 because of premul rounding
		if absDiff(got.R, in.R) > 1 || absDiff(got.G, in.G) > 1 || absDiff(got.B, in.B) > 1 {
			t.Errorf("round-trip %v -> %v -> %v differs >1", in, rgba, got)
		}
	}
}

func absDiff(a, b uint8) int {
	d := int(a) - int(b)
	if d < 0 {
		return -d
	}
	return d
}

func TestNRGBAToRGBA_TransparentZeros(t *testing.T) {
	in := color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0}
	got := NRGBAToRGBA(in)
	if got.A != 0 {
		t.Errorf("alpha = %v, want 0", got.A)
	}
	// With A=0 the premultiplied result must have RGB=0.
	if got.R != 0 || got.G != 0 || got.B != 0 {
		t.Errorf("expected RGB=0 when alpha=0, got %v", got)
	}
}

func TestRGBA_SRGB_KnownValues(t *testing.T) {
	// White (premultiplied alpha=1)
	white := RGBA{R: 1, G: 1, B: 1, A: 1}
	if got := white.SRGB(); got != (color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}) {
		t.Errorf("white.SRGB() = %v", got)
	}
	// Black opaque
	black := RGBA{A: 1}
	if got := black.SRGB(); got != (color.NRGBA{A: 0xFF}) {
		t.Errorf("black.SRGB() = %v", got)
	}
}

func TestRGBA_Float32_Reflexive(t *testing.T) {
	// Special values should pass through Float32().
	c := RGBA{R: float32(math.NaN()), G: float32(math.Inf(1)), B: float32(math.Inf(-1)), A: 0}
	r, g, b, a := c.Float32()
	if !math.IsNaN(float64(r)) {
		t.Errorf("R: want NaN, got %v", r)
	}
	if !math.IsInf(float64(g), 1) {
		t.Errorf("G: want +Inf, got %v", g)
	}
	if !math.IsInf(float64(b), -1) {
		t.Errorf("B: want -Inf, got %v", b)
	}
	if a != 0 {
		t.Errorf("A: want 0, got %v", a)
	}
}

func TestRGBA_Opaque_PreservesRGB(t *testing.T) {
	c := RGBA{R: 0.25, G: 0.5, B: 0.75, A: 0.1}
	op := c.Opaque()
	if op.R != 0.25 || op.G != 0.5 || op.B != 0.75 {
		t.Errorf("Opaque() altered RGB: %+v", op)
	}
	if op.A != 1 {
		t.Errorf("Opaque().A = %v", op.A)
	}
	// Original unchanged
	if c.A != 0.1 {
		t.Errorf("Opaque() mutated receiver: %v", c)
	}
}

// TestLinearTosRGB_Precision verifies that linearTosRGB uses the precise
// gamma exponent 1/2.4. With the previous 0.41666 approximation, the round-
// trip linearTosRGB(sRGBToLinear(c)) drifted by ~1e-4; with 1/2.4 the drift
// must be within 1e-5.
func TestLinearTosRGB_Precision(t *testing.T) {
	const eps = 1e-5
	for _, c := range []float32{0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.95} {
		got := linearTosRGB(sRGBToLinear(c))
		if !floatNear(got, c, eps) {
			t.Errorf("linearTosRGB(sRGBToLinear(%v)) = %v, drift %v > %v", c, got, got-c, eps)
		}
	}
}

func TestLuminance_Channels(t *testing.T) {
	// Pure-channel luminance must match the documented coefficients.
	if lum := (RGBA{R: 1, A: 1}).Luminance(); !floatNear(lum, 0.2126, colorEps) {
		t.Errorf("R luminance = %v", lum)
	}
	if lum := (RGBA{G: 1, A: 1}).Luminance(); !floatNear(lum, 0.7152, colorEps) {
		t.Errorf("G luminance = %v", lum)
	}
	if lum := (RGBA{B: 1, A: 1}).Luminance(); !floatNear(lum, 0.0722, colorEps) {
		t.Errorf("B luminance = %v", lum)
	}
	// Black = 0
	if lum := (RGBA{}).Luminance(); lum != 0 {
		t.Errorf("black luminance = %v", lum)
	}
}
