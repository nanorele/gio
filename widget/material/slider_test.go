// SPDX-License-Identifier: Unlicense OR MIT

package material

import (
	"image"
	"testing"

	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/unit"
	"github.com/nanorele/gio/widget"
)

func TestSliderConstructor(t *testing.T) {
	th := NewTheme()
	var f widget.Float
	s := Slider(th, &f)

	if s.Float != &f {
		t.Errorf("Float not propagated")
	}
	if s.Color != th.Palette.ContrastBg {
		t.Errorf("Color = %v, want ContrastBg %v", s.Color, th.Palette.ContrastBg)
	}
	if s.FingerSize != th.FingerSize {
		t.Errorf("FingerSize = %v, want %v", s.FingerSize, th.FingerSize)
	}
	// Default Axis is the zero value (Horizontal).
	if s.Axis != layout.Horizontal {
		t.Errorf("Axis default = %v, want Horizontal", s.Axis)
	}
}

func TestSliderAxisLayout(t *testing.T) {
	th := NewTheme()
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(100, 100)),
	}
	for _, axis := range []layout.Axis{layout.Horizontal, layout.Vertical} {
		var f widget.Float
		s := Slider(th, &f)
		s.Axis = axis
		dims := s.Layout(gtx)
		if dims.Size.X == 0 || dims.Size.Y == 0 {
			t.Errorf("axis=%v zero dims: %v", axis, dims.Size)
		}
	}
}
