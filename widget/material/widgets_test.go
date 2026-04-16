// SPDX-License-Identifier: Unlicense OR MIT

package material

import (
	"image"
	"testing"

	"github.com/nanorele/gio/io/system"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/unit"
	"github.com/nanorele/gio/widget"
)

func TestWidgets(t *testing.T) {
	th := NewTheme()
	gtx := layout.Context{
		Ops: new(op.Ops),
		Metric: unit.Metric{
			PxPerDp: 1,
			PxPerSp: 1,
		},
		Constraints: layout.Exact(image.Point{X: 100, Y: 100}),
	}

	t.Run("Editor", func(t *testing.T) {
		var ed widget.Editor
		e := Editor(th, &ed, "Hint")
		dims := e.Layout(gtx)
		if dims.Size.X == 0 || dims.Size.Y == 0 {
			t.Errorf("editor layout returned zero size: %v", dims.Size)
		}
	})

	t.Run("ProgressBar", func(t *testing.T) {
		p := ProgressBar(th, 0.5)
		dims := p.Layout(gtx)
		if dims.Size.X == 0 || dims.Size.Y == 0 {
			t.Errorf("progressbar layout returned zero size: %v", dims.Size)
		}
	})

	t.Run("Switch", func(t *testing.T) {
		var sw widget.Bool
		s := Switch(th, &sw, "Description")
		dims := s.Layout(gtx)
		if dims.Size.X == 0 || dims.Size.Y == 0 {
			t.Errorf("switch layout returned zero size: %v", dims.Size)
		}
	})

	t.Run("CheckBox", func(t *testing.T) {
		var cb widget.Bool
		c := CheckBox(th, &cb, "Label")
		dims := c.Layout(gtx)
		if dims.Size.X == 0 || dims.Size.Y == 0 {
			t.Errorf("checkbox layout returned zero size: %v", dims.Size)
		}
	})

	t.Run("RadioButton", func(t *testing.T) {
		var group widget.Enum
		r := RadioButton(th, &group, "key", "Label")
		dims := r.Layout(gtx)
		if dims.Size.X == 0 || dims.Size.Y == 0 {
			t.Errorf("radiobutton layout returned zero size: %v", dims.Size)
		}
	})

	t.Run("Slider", func(t *testing.T) {
		var val widget.Float
		s := Slider(th, &val)
		dims := s.Layout(gtx)
		if dims.Size.X == 0 || dims.Size.Y == 0 {
			t.Errorf("slider layout returned zero size: %v", dims.Size)
		}
	})

	t.Run("Loader", func(t *testing.T) {
		l := Loader(th)
		dims := l.Layout(gtx)
		if dims.Size.X == 0 || dims.Size.Y == 0 {
			t.Errorf("loader layout returned zero size: %v", dims.Size)
		}
	})

	t.Run("ProgressCircle", func(t *testing.T) {
		p := ProgressCircle(th, 0.5)
		dims := p.Layout(gtx)
		if dims.Size.X == 0 || dims.Size.Y == 0 {
			t.Errorf("progresscircle layout returned zero size: %v", dims.Size)
		}
	})

	t.Run("Scrollbar", func(t *testing.T) {
		var state widget.Scrollbar
		s := Scrollbar(th, &state)
		dims := s.Layout(gtx, layout.Vertical, 0, 0.5)
		if dims.Size.X == 0 || dims.Size.Y == 0 {
			t.Errorf("scrollbar layout returned zero size: %v", dims.Size)
		}
	})

	t.Run("List", func(t *testing.T) {
		var state widget.List
		l := List(th, &state)
		dims := l.Layout(gtx, 10, func(gtx layout.Context, i int) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(10, 10)}
		})
		if dims.Size.X == 0 || dims.Size.Y == 0 {
			t.Errorf("list layout returned zero size: %v", dims.Size)
		}
	})

	t.Run("Decorations", func(t *testing.T) {
		var deco widget.Decorations
		d := Decorations(th, &deco, system.ActionMinimize|system.ActionMaximize|system.ActionClose, "Title")
		dims := d.Layout(gtx)
		if dims.Size.X == 0 || dims.Size.Y == 0 {
			t.Errorf("decorations layout returned zero size: %v", dims.Size)
		}
	})
}
