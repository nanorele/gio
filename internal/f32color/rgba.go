package f32color

import (
	"image/color"
	"math"
)

//go:generate go run ./f32colorgen -out tables.go

type RGBA struct {
	R, G, B, A float32
}

func (rgba RGBA) Array() [4]float32 {
	return [4]float32{rgba.R, rgba.G, rgba.B, rgba.A}
}

func (col RGBA) Float32() (r, g, b, a float32) {
	return col.R, col.G, col.B, col.A
}

func (col RGBA) SRGB() color.NRGBA {
	if col.A == 0 {
		return color.NRGBA{}
	}
	return color.NRGBA{
		R: uint8(linearTosRGB(col.R/col.A)*255 + .5),
		G: uint8(linearTosRGB(col.G/col.A)*255 + .5),
		B: uint8(linearTosRGB(col.B/col.A)*255 + .5),
		A: uint8(col.A*255 + .5),
	}
}

func (col RGBA) Luminance() float32 {
	return 0.2126*col.R + 0.7152*col.G + 0.0722*col.B
}

func (col RGBA) Opaque() RGBA {
	col.A = 1.0
	return col
}

func LinearFromSRGB(col color.NRGBA) RGBA {
	af := float32(col.A) / 0xFF
	return RGBA{
		R: srgb8ToLinear[col.R] * af,
		G: srgb8ToLinear[col.G] * af,
		B: srgb8ToLinear[col.B] * af,
		A: af,
	}
}

func NRGBAToRGBA(col color.NRGBA) color.RGBA {
	if col.A == 0xFF {
		return color.RGBA(col)
	}
	c := LinearFromSRGB(col)
	return color.RGBA{
		R: uint8(linearTosRGB(c.R)*255 + .5),
		G: uint8(linearTosRGB(c.G)*255 + .5),
		B: uint8(linearTosRGB(c.B)*255 + .5),
		A: col.A,
	}
}

func NRGBAToLinearRGBA(col color.NRGBA) color.RGBA {
	if col.A == 0xFF {
		return color.RGBA(col)
	}
	c := LinearFromSRGB(col)
	return color.RGBA{
		R: uint8(c.R*255 + .5),
		G: uint8(c.G*255 + .5),
		B: uint8(c.B*255 + .5),
		A: col.A,
	}
}

func RGBAToNRGBA(col color.RGBA) color.NRGBA {
	if col.A == 0xFF {
		return color.NRGBA(col)
	}

	linear := RGBA{
		R: sRGBToLinear(float32(col.R) / 0xff),
		G: sRGBToLinear(float32(col.G) / 0xff),
		B: sRGBToLinear(float32(col.B) / 0xff),
		A: float32(col.A) / 0xff,
	}

	return linear.SRGB()
}

func linearTosRGB(c float32) float32 {
	switch {
	case c <= 0:
		return 0
	case c < 0.0031308:
		return 12.92 * c
	case c < 1:
		return 1.055*float32(math.Pow(float64(c), 0.41666)) - 0.055
	}
	return 1
}

func sRGBToLinear(c float32) float32 {
	if c <= 0.04045 {
		return c / 12.92
	} else {
		return float32(math.Pow(float64((c+0.055)/1.055), 2.4))
	}
}

func MulAlpha(c color.NRGBA, alpha uint8) color.NRGBA {
	c.A = uint8(uint32(c.A) * uint32(alpha) / 0xFF)
	return c
}

func Disabled(c color.NRGBA) (d color.NRGBA) {
	const r = 80
	lum := approxLuminance(c)
	d = mix(c, color.NRGBA{A: c.A, R: lum, G: lum, B: lum}, r)
	d = MulAlpha(d, 128+32)
	return
}

func Hovered(c color.NRGBA) (h color.NRGBA) {
	if c.A == 0 {

		return color.NRGBA{A: 0x44, R: 0x88, G: 0x88, B: 0x88}
	}
	const ratio = 0x20
	m := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: c.A}
	if approxLuminance(c) > 128 {
		m = color.NRGBA{A: c.A}
	}
	return mix(m, c, ratio)
}

func mix(c1, c2 color.NRGBA, a uint8) color.NRGBA {
	ai := int(a)
	return color.NRGBA{
		R: byte((int(c1.R)*ai + int(c2.R)*(256-ai)) / 256),
		G: byte((int(c1.G)*ai + int(c2.G)*(256-ai)) / 256),
		B: byte((int(c1.B)*ai + int(c2.B)*(256-ai)) / 256),
		A: byte((int(c1.A)*ai + int(c2.A)*(256-ai)) / 256),
	}
}

func approxLuminance(c color.NRGBA) byte {
	const (
		r = 13933
		g = 46871
		b = 4732
		t = r + g + b
	)
	return byte((r*int(c.R) + g*int(c.G) + b*int(c.B)) / t)
}
