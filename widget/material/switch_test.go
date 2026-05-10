// SPDX-License-Identifier: Unlicense OR MIT

package material

import (
	"image"
	"testing"

	"github.com/nanorele/gio/internal/f32color"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/unit"
	"github.com/nanorele/gio/widget"
)

func TestSwitchConstructor(t *testing.T) {
	th := NewTheme()
	var sw widget.Bool
	s := Switch(th, &sw, "Desc")

	if s.Switch != &sw {
		t.Errorf("Switch field not propagated: %p vs %p", s.Switch, &sw)
	}
	if s.Description != "Desc" {
		t.Errorf("Description = %q, want %q", s.Description, "Desc")
	}
	if s.Color.Enabled != th.Palette.ContrastBg {
		t.Errorf("Color.Enabled = %v, want ContrastBg %v", s.Color.Enabled, th.Palette.ContrastBg)
	}
	if s.Color.Disabled != th.Palette.Bg {
		t.Errorf("Color.Disabled = %v, want Bg %v", s.Color.Disabled, th.Palette.Bg)
	}
	wantTrack := f32color.MulAlpha(th.Palette.Fg, 0x88)
	if s.Color.Track != wantTrack {
		t.Errorf("Color.Track = %v, want %v", s.Color.Track, wantTrack)
	}
}

func TestSwitchEmptyDescription(t *testing.T) {
	th := NewTheme()
	var sw widget.Bool
	s := Switch(th, &sw, "")
	if s.Description != "" {
		t.Errorf("Description = %q, want empty", s.Description)
	}

	gtx := layout.Context{
		Ops:         new(op.Ops),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(100, 100)),
	}
	dims := s.Layout(gtx)
	if dims.Size.X == 0 || dims.Size.Y == 0 {
		t.Errorf("zero dims with empty description: %v", dims.Size)
	}
}

func TestSwitchValueLayout(t *testing.T) {
	th := NewTheme()
	for _, val := range []bool{false, true} {
		var sw widget.Bool
		sw.Value = val
		s := Switch(th, &sw, "x")
		gtx := layout.Context{
			Ops:         new(op.Ops),
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Constraints: layout.Exact(image.Pt(100, 100)),
		}
		dims := s.Layout(gtx)
		// Width is fixed track (36dp) and height is thumb (20dp) at PxPerDp=1.
		if dims.Size.X != 36 || dims.Size.Y != 20 {
			t.Errorf("value=%v dims = %v, want {36,20}", val, dims.Size)
		}
	}
}
