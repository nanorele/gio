package layout

import (
	"image"
	"testing"

	"github.com/nanorele/gio/op"
)

func TestBackground(t *testing.T) {
	gtx := Context{
		Ops: new(op.Ops),
		Constraints: Constraints{
			Max: image.Pt(100, 100),
		},
	}
	dims := Background{}.Layout(gtx,
		func(gtx Context) Dimensions {
			return Dimensions{Size: gtx.Constraints.Min}
		},
		func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(50, 50)}
		},
	)
	if dims.Size != image.Pt(50, 50) {
		t.Errorf("expected size (50, 50), got %v", dims.Size)
	}
}

func BenchmarkStack(b *testing.B) {
	gtx := Context{
		Ops: new(op.Ops),
		Constraints: Constraints{
			Max: image.Point{X: 100, Y: 100},
		},
	}
	b.ReportAllocs()

	for b.Loop() {
		gtx.Ops.Reset()

		Stack{}.Layout(gtx,
			Expanded(emptyWidget{
				Size: image.Point{X: 60, Y: 60},
			}.Layout),
			Stacked(emptyWidget{
				Size: image.Point{X: 30, Y: 30},
			}.Layout),
		)
	}
}

func BenchmarkBackground(b *testing.B) {
	gtx := Context{
		Ops: new(op.Ops),
		Constraints: Constraints{
			Max: image.Point{X: 100, Y: 100},
		},
	}
	b.ReportAllocs()

	for b.Loop() {
		gtx.Ops.Reset()

		Background{}.Layout(gtx,
			emptyWidget{
				Size: image.Point{X: 60, Y: 60},
			}.Layout,
			emptyWidget{
				Size: image.Point{X: 30, Y: 30},
			}.Layout,
		)
	}
}

type emptyWidget struct {
	Size image.Point
}

func (w emptyWidget) Layout(gtx Context) Dimensions {
	return Dimensions{Size: w.Size}
}
