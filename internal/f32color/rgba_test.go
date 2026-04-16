package f32color

import (
	"image/color"
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
