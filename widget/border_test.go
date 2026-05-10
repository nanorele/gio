package widget_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/widget"
)

func TestBorderPassesThroughChildSize(t *testing.T) {
	cases := []struct {
		name  string
		width float32
		size  image.Point
	}{
		{"zero width", 0, image.Pt(50, 50)},
		{"width 1", 1, image.Pt(80, 20)},
		{"width 10", 10, image.Pt(200, 100)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gtx := layout.Context{
				Ops:         new(op.Ops),
				Constraints: layout.Exact(image.Pt(300, 300)),
			}
			b := widget.Border{
				Color: color.NRGBA{A: 255},
				Width: 2,
			}
			dims := b.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{Size: tc.size}
			})
			if dims.Size != tc.size {
				t.Errorf("expected %v, got %v", tc.size, dims.Size)
			}
		})
	}
}

func TestBorderZeroSizeChild(t *testing.T) {
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(100, 100)),
	}
	b := widget.Border{
		Color:        color.NRGBA{R: 1, G: 2, B: 3, A: 255},
		Width:        4,
		CornerRadius: 5,
	}
	dims := b.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: image.Pt(0, 0)}
	})
	if dims.Size != (image.Point{}) {
		t.Errorf("expected zero size, got %v", dims.Size)
	}
}

func TestBorderFieldsZeroValue(t *testing.T) {
	var b widget.Border
	if b.Width != 0 || b.CornerRadius != 0 {
		t.Errorf("expected zero values, got Width=%v Radius=%v", b.Width, b.CornerRadius)
	}
	if (b.Color != color.NRGBA{}) {
		t.Errorf("expected zero color, got %v", b.Color)
	}
}
