// SPDX-License-Identifier: Unlicense OR MIT

package material

import (
	"image"
	"testing"

	"github.com/nanorele/gio/internal/f32color"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/unit"
)

func TestLoaderConstructor(t *testing.T) {
	th := NewTheme()
	l := Loader(th)
	if l.Color != th.Palette.ContrastBg {
		t.Errorf("Loader.Color = %v, want ContrastBg %v", l.Color, th.Palette.ContrastBg)
	}
}

func TestLoaderDefaultDiameter(t *testing.T) {
	th := NewTheme()
	l := Loader(th)
	gtx := layout.Context{
		Ops:    new(op.Ops),
		Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{
			Min: image.Pt(0, 0),
			Max: image.Pt(200, 200),
		},
	}
	dims := l.Layout(gtx)
	// Default diameter is 24dp when min constraints are zero.
	if dims.Size.X != 24 || dims.Size.Y != 24 {
		t.Errorf("default loader size = %v, want {24,24}", dims.Size)
	}
}

func TestProgressBarConstructor(t *testing.T) {
	th := NewTheme()
	p := ProgressBar(th, 0.42)
	if p.Progress != 0.42 {
		t.Errorf("Progress = %v, want 0.42", p.Progress)
	}
	if p.Height != unit.Dp(4) {
		t.Errorf("Height = %v, want 4dp", p.Height)
	}
	if p.Radius != unit.Dp(2) {
		t.Errorf("Radius = %v, want 2dp", p.Radius)
	}
	if p.Color != th.Palette.ContrastBg {
		t.Errorf("Color = %v, want ContrastBg %v", p.Color, th.Palette.ContrastBg)
	}
	wantTrack := f32color.MulAlpha(th.Palette.Fg, 0x88)
	if p.TrackColor != wantTrack {
		t.Errorf("TrackColor = %v, want %v", p.TrackColor, wantTrack)
	}
}

func TestProgressBarLayoutWidth(t *testing.T) {
	th := NewTheme()
	p := ProgressBar(th, 0.5)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(100, 100)),
	}
	dims := p.Layout(gtx)
	if dims.Size.X != 100 {
		t.Errorf("ProgressBar width = %d, want 100", dims.Size.X)
	}
}

func TestClamp1(t *testing.T) {
	cases := []struct {
		in, want float32
	}{
		{-1, 0},
		{0, 0},
		{0.25, 0.25},
		{1, 1},
		{1.5, 1},
	}
	for _, c := range cases {
		if got := clamp1(c.in); got != c.want {
			t.Errorf("clamp1(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestProgressCircleConstructor(t *testing.T) {
	th := NewTheme()
	p := ProgressCircle(th, 0.7)
	if p.Progress != 0.7 {
		t.Errorf("Progress = %v, want 0.7", p.Progress)
	}
	if p.Color != th.Palette.ContrastBg {
		t.Errorf("Color = %v, want ContrastBg %v", p.Color, th.Palette.ContrastBg)
	}
}

func TestProgressCircleDefaultDiameter(t *testing.T) {
	th := NewTheme()
	p := ProgressCircle(th, 0.5)
	gtx := layout.Context{
		Ops:    new(op.Ops),
		Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Constraints{
			Min: image.Pt(0, 0),
			Max: image.Pt(200, 200),
		},
	}
	dims := p.Layout(gtx)
	if dims.Size.X != 24 || dims.Size.Y != 24 {
		t.Errorf("default circle size = %v, want {24,24}", dims.Size)
	}
}
