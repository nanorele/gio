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

func TestButton(t *testing.T) {
	th := NewTheme()
	var click widget.Clickable
	b := Button(th, &click, "Click Me")
	
	gtx := layout.Context{
		Ops: new(op.Ops),
		Metric: unit.Metric{
			PxPerDp: 1,
			PxPerSp: 1,
		},
		Constraints: layout.Exact(image.Point{X: 100, Y: 100}),
	}
	
	dims := b.Layout(gtx)
	if dims.Size.X == 0 || dims.Size.Y == 0 {
		t.Errorf("button layout returned zero size: %v", dims.Size)
	}
}

func TestIconButton(t *testing.T) {
	th := NewTheme()
	var click widget.Clickable
	ic, _ := widget.NewIcon(nil) // empty icon
	b := IconButton(th, &click, ic, "description")
	
	gtx := layout.Context{
		Ops: new(op.Ops),
		Metric: unit.Metric{
			PxPerDp: 1,
			PxPerSp: 1,
		},
		Constraints: layout.Exact(image.Point{X: 100, Y: 100}),
	}
	
	dims := b.Layout(gtx)
	if dims.Size.X == 0 || dims.Size.Y == 0 {
		t.Errorf("icon button layout returned zero size: %v", dims.Size)
	}
}

func TestButtonLayout(t *testing.T) {
	th := NewTheme()
	var click widget.Clickable
	bl := ButtonLayout(th, &click)
	
	gtx := layout.Context{
		Ops: new(op.Ops),
		Metric: unit.Metric{
			PxPerDp: 1,
			PxPerSp: 1,
		},
		Constraints: layout.Exact(image.Point{X: 100, Y: 100}),
	}
	
	dims := bl.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Point{X: 50, Y: 50}}
	})
	if dims.Size.X == 0 || dims.Size.Y == 0 {
		t.Errorf("button layout returned zero size: %v", dims.Size)
	}
}

func TestClickable(t *testing.T) {
	var click widget.Clickable
	gtx := layout.Context{
		Ops: new(op.Ops),
		Metric: unit.Metric{
			PxPerDp: 1,
			PxPerSp: 1,
		},
		Constraints: layout.Exact(image.Point{X: 100, Y: 100}),
	}
	
	dims := Clickable(gtx, &click, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Point{X: 50, Y: 50}}
	})
	if dims.Size.X == 0 || dims.Size.Y == 0 {
		t.Errorf("clickable layout returned zero size: %v", dims.Size)
	}
}
