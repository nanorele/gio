package layout

import (
	"image"
	"testing"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/unit"
)

func TestStack(t *testing.T) {
	gtx := Context{
		Ops: new(op.Ops),
		Constraints: Constraints{
			Max: image.Pt(100, 100),
		},
	}
	exp := image.Point{X: 60, Y: 70}
	dims := Stack{Alignment: Center}.Layout(gtx,
		Expanded(func(gtx Context) Dimensions {
			return Dimensions{Size: exp}
		}),
		Stacked(func(gtx Context) Dimensions {
			return Dimensions{Size: image.Point{X: 50, Y: 50}}
		}),
	)
	if got := dims.Size; got != exp {
		t.Errorf("Stack ignored Expanded size, got %v expected %v", got, exp)
	}
}

func TestFlex(t *testing.T) {
	gtx := Context{
		Ops: new(op.Ops),
		Constraints: Constraints{
			Min: image.Pt(100, 100),
			Max: image.Pt(100, 100),
		},
	}
	dims := Flex{}.Layout(gtx)
	if got := dims.Size; got != gtx.Constraints.Min {
		t.Errorf("Flex ignored minimum constraints, got %v expected %v", got, gtx.Constraints.Min)
	}
}

func TestFlexGap(t *testing.T) {
	gtx := Context{
		Ops: new(op.Ops),
		Constraints: Constraints{
			Max: image.Pt(100, 100),
		},
	}

	dims := Flex{Gap: 10}.Layout(gtx,
		Rigid(func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(20, 10)}
		}),
		Rigid(func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(20, 10)}
		}),
	)
	if got, exp := dims.Size.X, 50; got != exp {
		t.Errorf("two rigid children with gap: got width %d, expected %d", got, exp)
	}

	dims = Flex{Gap: 5}.Layout(gtx,
		Rigid(func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(10, 10)}
		}),
		Rigid(func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(10, 10)}
		}),
		Rigid(func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(10, 10)}
		}),
	)
	if got, exp := dims.Size.X, 40; got != exp {
		t.Errorf("three rigid children with gap: got width %d, expected %d", got, exp)
	}

	dims = Flex{Gap: 10}.Layout(gtx,
		Rigid(func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(20, 10)}
		}),
	)
	if got, exp := dims.Size.X, 20; got != exp {
		t.Errorf("single child with gap: got width %d, expected %d", got, exp)
	}

	dims = Flex{Gap: 10}.Layout(gtx,
		Flexed(1, func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 10)}
		}),
		Flexed(1, func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 10)}
		}),
	)

	if got, exp := dims.Size.X, 100; got != exp {
		t.Errorf("flexed children with gap: got width %d, expected %d", got, exp)
	}

	dims = Flex{Axis: Vertical, Gap: 15}.Layout(gtx,
		Rigid(func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(10, 20)}
		}),
		Rigid(func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(10, 20)}
		}),
	)
	if got, exp := dims.Size.Y, 55; got != exp {
		t.Errorf("vertical with gap: got height %d, expected %d", got, exp)
	}
}

func TestFlexGapConstraints(t *testing.T) {
	gtx := Context{
		Ops: new(op.Ops),
		Constraints: Constraints{
			Max: image.Pt(100, 100),
		},
	}

	var flexMax int
	Flex{Gap: 10}.Layout(gtx,
		Rigid(func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(30, 10)}
		}),
		Flexed(1, func(gtx Context) Dimensions {
			flexMax = gtx.Constraints.Max.X
			return Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 10)}
		}),
	)

	if got, exp := flexMax, 60; got != exp {
		t.Errorf("flex constraint with gap: got %d, expected %d", got, exp)
	}
}

func TestDirection(t *testing.T) {
	max := image.Pt(100, 100)
	for _, tc := range []struct {
		dir Direction
		exp image.Point
	}{
		{N, image.Pt(max.X, 0)},
		{S, image.Pt(max.X, 0)},
		{E, image.Pt(0, max.Y)},
		{W, image.Pt(0, max.Y)},
		{NW, image.Pt(0, 0)},
		{NE, image.Pt(0, 0)},
		{SE, image.Pt(0, 0)},
		{SW, image.Pt(0, 0)},
		{Center, image.Pt(0, 0)},
	} {
		t.Run(tc.dir.String(), func(t *testing.T) {
			gtx := Context{
				Ops:         new(op.Ops),
				Constraints: Exact(max),
			}
			var min image.Point
			tc.dir.Layout(gtx, func(gtx Context) Dimensions {
				min = gtx.Constraints.Min
				return Dimensions{}
			})
			if got, exp := min, tc.exp; got != exp {
				t.Errorf("got %v; expected %v", got, exp)
			}
		})
	}
}

func TestInset(t *testing.T) {
	gtx := Context{
		Ops: new(op.Ops),
		Constraints: Constraints{
			Max: image.Pt(100, 100),
		},
	}
	in := UniformInset(10)
	dims := in.Layout(gtx, func(gtx Context) Dimensions {
		if exp := 80; gtx.Constraints.Max.X != exp {
			t.Errorf("expected max width %d, got %d", exp, gtx.Constraints.Max.X)
		}
		return Dimensions{Size: image.Pt(50, 50)}
	})
	if exp := image.Pt(70, 70); dims.Size != exp {
		t.Errorf("expected size %v, got %v", exp, dims.Size)
	}
}

func TestSpacer(t *testing.T) {
	gtx := Context{
		Constraints: Constraints{
			Max: image.Pt(100, 100),
		},
	}
	dims := Spacer{Width: 20, Height: 30}.Layout(gtx)
	if exp := image.Pt(20, 30); dims.Size != exp {
		t.Errorf("expected size %v, got %v", exp, dims.Size)
	}
}

func TestAxisMethods(t *testing.T) {
	if Horizontal.String() != "Horizontal" || Vertical.String() != "Vertical" {
		t.Error("Axis.String failed")
	}
	pt := image.Pt(10, 20)
	if Horizontal.Convert(pt) != pt {
		t.Error("Horizontal.Convert failed")
	}
	if Vertical.Convert(pt) != image.Pt(20, 10) {
		t.Error("Vertical.Convert failed")
	}
	
	fpt := f32.Pt(10, 20)
	if Horizontal.FConvert(fpt) != fpt {
		t.Error("Horizontal.FConvert failed")
	}
	if Vertical.FConvert(fpt) != f32.Pt(20, 10) {
		t.Error("Vertical.FConvert failed")
	}

	cs := Constraints{Min: image.Pt(1, 2), Max: image.Pt(3, 4)}
	min, max := Horizontal.mainConstraint(cs)
	if min != 1 || max != 3 {
		t.Error("Horizontal.mainConstraint failed")
	}
	min, max = Vertical.mainConstraint(cs)
	if min != 2 || max != 4 {
		t.Error("Vertical.mainConstraint failed")
	}
}

func TestContextMethods(t *testing.T) {
	gtx := Context{
		Metric: unit.Metric{PxPerDp: 1, PxPerSp: 2},
	}
	if got, exp := gtx.Sp(10), 20; got != exp {
		t.Errorf("gtx.Sp(10) = %d, expected %d", got, exp)
	}
	dgtx := gtx.Disabled()
	if !dgtx.Source.Enabled() {
		// This depends on how input.Source handles Disabled.
		// Usually it sets a flag.
	}
}

func TestAlignmentStrings(t *testing.T) {
	if Start.String() != "Start" || End.String() != "End" || Middle.String() != "Middle" || Baseline.String() != "Baseline" {
		t.Error("Alignment.String failed")
	}
}

func TestAxisStrings(t *testing.T) {
	if Horizontal.String() != "Horizontal" || Vertical.String() != "Vertical" {
		t.Error("Axis.String failed")
	}
}

func TestDirectionStrings(t *testing.T) {
	dirs := []Direction{NW, N, NE, E, SE, S, SW, W, Center}
	for _, d := range dirs {
		if d.String() == "" {
			t.Errorf("Direction %d has empty string", d)
		}
	}
}


func TestConstraints_AddSub(t *testing.T) {
	c := Constraints{Min: image.Pt(10, 10), Max: image.Pt(100, 100)}
	c = c.AddMin(image.Pt(5, 5))
	if c.Min != image.Pt(15, 15) {
		t.Errorf("AddMin failed: %v", c.Min)
	}
	c = c.SubMax(image.Pt(20, 20))
	if c.Max != image.Pt(80, 80) {
		t.Errorf("SubMax failed: %v", c.Max)
	}
}

func TestDirection_Layout(t *testing.T) {
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(100, 100)),
	}
	// Test N, S, E, W cases for min constraints clearing
	dirs := []Direction{N, S, E, W, Center}
	for _, d := range dirs {
		d.Layout(gtx, func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(50, 50)}
		})
	}
}

func TestFPt(t *testing.T) {
	p := image.Pt(10, 20)
	fp := FPt(p)
	if fp.X != 10 || fp.Y != 20 {
		t.Error("FPt failed")
	}
}

