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

func TestStackSingleChild(t *testing.T) {
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Constraints{Max: image.Pt(100, 100)},
	}
	dims := Stack{}.Layout(gtx,
		Stacked(func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(40, 30)}
		}),
	)
	if dims.Size != image.Pt(40, 30) {
		t.Errorf("single-stacked: got %v, want (40,30)", dims.Size)
	}
}

func TestStackEmpty(t *testing.T) {
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Constraints{Min: image.Pt(20, 30), Max: image.Pt(100, 100)},
	}
	dims := Stack{}.Layout(gtx)
	// Empty stack should be constrained by Min.
	if dims.Size != image.Pt(20, 30) {
		t.Errorf("empty stack: got %v, want (20,30)", dims.Size)
	}
}

func TestStackExpandedFollowsStackedMax(t *testing.T) {
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Constraints{Max: image.Pt(100, 100)},
	}
	var expandedMin image.Point
	Stack{}.Layout(gtx,
		Stacked(func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(60, 70)}
		}),
		Expanded(func(gtx Context) Dimensions {
			expandedMin = gtx.Constraints.Min
			return Dimensions{Size: image.Pt(0, 0)}
		}),
	)
	if expandedMin != image.Pt(60, 70) {
		t.Errorf("expanded Min should match stacked size: got %v", expandedMin)
	}
}

func TestStackChildLargerThanParent(t *testing.T) {
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Constraints{Min: image.Pt(0, 0), Max: image.Pt(50, 50)},
	}
	dims := Stack{}.Layout(gtx,
		Stacked(func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(200, 200)}
		}),
	)
	// Final size is constrained.
	if dims.Size != image.Pt(50, 50) {
		t.Errorf("oversized child: got %v, want (50,50)", dims.Size)
	}
}

func TestStackAlignmentAll(t *testing.T) {
	dirs := []Direction{NW, N, NE, E, SE, S, SW, W, Center}
	for _, d := range dirs {
		t.Run(d.String(), func(t *testing.T) {
			gtx := Context{
				Ops:         new(op.Ops),
				Constraints: Constraints{Max: image.Pt(100, 100)},
			}
			dims := Stack{Alignment: d}.Layout(gtx,
				Stacked(func(gtx Context) Dimensions {
					return Dimensions{Size: image.Pt(80, 80)}
				}),
				Stacked(func(gtx Context) Dimensions {
					return Dimensions{Size: image.Pt(20, 20)}
				}),
			)
			if dims.Size != image.Pt(80, 80) {
				t.Errorf("Alignment %s: got %v, want (80,80)", d, dims.Size)
			}
		})
	}
}

func TestStackBaselineFromFirst(t *testing.T) {
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Constraints{Max: image.Pt(100, 100)},
	}
	dims := Stack{Alignment: Center}.Layout(gtx,
		Stacked(func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(50, 50), Baseline: 10}
		}),
	)
	if dims.Baseline == 0 {
		t.Errorf("baseline should propagate from child")
	}
}

func TestStackManyChildren(t *testing.T) {
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Constraints{Max: image.Pt(1000, 1000)},
	}
	children := make([]StackChild, 40)
	for i := range children {
		children[i] = Stacked(func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(10, 10)}
		})
	}
	dims := Stack{}.Layout(gtx, children...)
	if dims.Size != image.Pt(10, 10) {
		t.Errorf("many stacked children: %v", dims.Size)
	}
}

func TestStackExpandedOnly(t *testing.T) {
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Constraints{Min: image.Pt(30, 40), Max: image.Pt(100, 100)},
	}
	dims := Stack{}.Layout(gtx,
		Expanded(func(gtx Context) Dimensions {
			return Dimensions{Size: gtx.Constraints.Min}
		}),
	)
	// Only expanded, no Stacked: Expanded uses Min=image.Point{} from cgtx
	// reset path. Final size constrained by parent.
	if dims.Size.X < 30 || dims.Size.Y < 40 {
		t.Errorf("expanded-only respects parent Min: %v", dims.Size)
	}
}

func TestBackgroundLargerBackground(t *testing.T) {
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Constraints{Min: image.Pt(80, 80), Max: image.Pt(100, 100)},
	}
	dims := Background{}.Layout(gtx,
		func(gtx Context) Dimensions {
			// background grows to Min provided by background path = constrain(widget).
			return Dimensions{Size: gtx.Constraints.Min}
		},
		func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(40, 40)}
		},
	)
	// widget=40,40 -> bg.Min = constrain(40)=80. bg returns 80,80.
	if dims.Size != image.Pt(80, 80) {
		t.Errorf("Background size: got %v, want (80,80)", dims.Size)
	}
}
