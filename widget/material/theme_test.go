// SPDX-License-Identifier: Unlicense OR MIT

package material

import (
	"image/color"
	"testing"
)

func TestNewTheme(t *testing.T) {
	th := NewTheme()
	if th == nil {
		t.Fatal("NewTheme() returned nil")
	}
	if th.Shaper == nil {
		t.Error("th.Shaper is nil")
	}
	if th.Palette.Fg != rgb(0x000000) {
		t.Errorf("expected Fg 0x000000, got %v", th.Palette.Fg)
	}
}

func TestWithPalette(t *testing.T) {
	th := NewTheme()
	p := Palette{
		Fg: color.NRGBA{R: 1, G: 2, B: 3, A: 255},
	}
	th2 := th.WithPalette(p)
	if th2.Palette.Fg != p.Fg {
		t.Errorf("expected Fg %v, got %v", p.Fg, th2.Palette.Fg)
	}
}

func TestRgb(t *testing.T) {
	c := rgb(0x112233)
	expected := color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff}
	if c != expected {
		t.Errorf("expected %v, got %v", expected, c)
	}
}

func TestArgb(t *testing.T) {
	c := argb(0x11223344)
	expected := color.NRGBA{R: 0x22, G: 0x33, B: 0x44, A: 0x11}
	if c != expected {
		t.Errorf("expected %v, got %v", expected, c)
	}
}
