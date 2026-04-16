// SPDX-License-Identifier: Unlicense OR MIT

package material

import (
	"image"
	"testing"

	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/unit"
)

func TestLabel(t *testing.T) {
	th := NewTheme()
	gtx := layout.Context{
		Ops: new(op.Ops),
		Metric: unit.Metric{
			PxPerDp: 1,
			PxPerSp: 1,
		},
		Constraints: layout.Exact(image.Point{X: 100, Y: 100}),
	}

	labelFunctions := []func(th *Theme, txt string) LabelStyle{
		H1, H2, H3, H4, H5, H6, Subtitle1, Subtitle2, Body1, Body2, Caption, Overline,
	}

	for i, f := range labelFunctions {
		l := f(th, "Hello")
		dims := l.Layout(gtx)
		if dims.Size.X == 0 || dims.Size.Y == 0 {
			t.Errorf("label function %d returned zero size: %v", i, dims.Size)
		}
	}
}
