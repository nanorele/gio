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
	_ = dgtx
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

func TestConstraintsConstrainCases(t *testing.T) {
	cases := []struct {
		name string
		c    Constraints
		in   image.Point
		want image.Point
	}{
		{"within", Constraints{Min: image.Pt(10, 10), Max: image.Pt(50, 50)}, image.Pt(30, 30), image.Pt(30, 30)},
		{"belowMin", Constraints{Min: image.Pt(10, 10), Max: image.Pt(50, 50)}, image.Pt(0, 0), image.Pt(10, 10)},
		{"aboveMax", Constraints{Min: image.Pt(10, 10), Max: image.Pt(50, 50)}, image.Pt(100, 100), image.Pt(50, 50)},
		{"negative", Constraints{Min: image.Pt(0, 0), Max: image.Pt(50, 50)}, image.Pt(-10, -20), image.Pt(0, 0)},
		{"zeroRange", Constraints{Min: image.Pt(20, 20), Max: image.Pt(20, 20)}, image.Pt(5, 5), image.Pt(20, 20)},
		// Min > Max: clamp first to Min, then to Max => Max wins.
		{"minGtMax", Constraints{Min: image.Pt(50, 50), Max: image.Pt(10, 10)}, image.Pt(0, 0), image.Pt(10, 10)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.c.Constrain(tc.in); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestExact(t *testing.T) {
	c := Exact(image.Pt(42, 24))
	if c.Min != image.Pt(42, 24) || c.Max != image.Pt(42, 24) {
		t.Errorf("Exact: %+v", c)
	}
}

func TestConstraintsAddMinNegative(t *testing.T) {
	c := Constraints{Min: image.Pt(10, 10), Max: image.Pt(100, 100)}
	c2 := c.AddMin(image.Pt(-50, -50))
	if c2.Min != (image.Point{}) {
		t.Errorf("AddMin negative not clamped: %v", c2.Min)
	}
}

func TestConstraintsAddMinClampsToMax(t *testing.T) {
	c := Constraints{Min: image.Pt(0, 0), Max: image.Pt(20, 20)}
	c2 := c.AddMin(image.Pt(100, 100))
	if c2.Min != image.Pt(20, 20) {
		t.Errorf("AddMin overflow not clamped to Max: %v", c2.Min)
	}
}

func TestConstraintsSubMaxNegative(t *testing.T) {
	c := Constraints{Min: image.Pt(0, 0), Max: image.Pt(50, 50)}
	c2 := c.SubMax(image.Pt(100, 100))
	if c2.Max != (image.Point{}) {
		t.Errorf("SubMax overflow not clamped to 0: %v", c2.Max)
	}
}

func TestConstraintsSubMaxAdjustsMin(t *testing.T) {
	c := Constraints{Min: image.Pt(40, 40), Max: image.Pt(50, 50)}
	c2 := c.SubMax(image.Pt(20, 20))
	if c2.Max != image.Pt(30, 30) {
		t.Errorf("SubMax wrong Max: %v", c2.Max)
	}
	if c2.Min.X > c2.Max.X || c2.Min.Y > c2.Max.Y {
		t.Errorf("SubMax left Min > Max: %+v", c2)
	}
}

func TestInsetUniform(t *testing.T) {
	in := UniformInset(7)
	if in.Top != 7 || in.Bottom != 7 || in.Left != 7 || in.Right != 7 {
		t.Errorf("UniformInset wrong: %+v", in)
	}
}

func TestInsetZero(t *testing.T) {
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Constraints{Max: image.Pt(100, 100)},
	}
	dims := Inset{}.Layout(gtx, func(gtx Context) Dimensions {
		return Dimensions{Size: image.Pt(50, 60)}
	})
	if dims.Size != image.Pt(50, 60) {
		t.Errorf("zero inset: got %v", dims.Size)
	}
}

func TestInsetExceedsAvailable(t *testing.T) {
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Constraints{Max: image.Pt(20, 20)},
	}
	// 30+30 horizontal padding > 20 available; widget should get max=0.
	in := Inset{Top: 30, Bottom: 30, Left: 30, Right: 30}
	var childMax image.Point
	dims := in.Layout(gtx, func(gtx Context) Dimensions {
		childMax = gtx.Constraints.Max
		return Dimensions{Size: gtx.Constraints.Max}
	})
	if childMax.X != 0 || childMax.Y != 0 {
		t.Errorf("child max not clamped to 0: %v", childMax)
	}
	if dims.Size.X < 0 || dims.Size.Y < 0 {
		t.Errorf("inset returned negative size: %v", dims.Size)
	}
}

func TestInsetNegativeValues(t *testing.T) {
	// Negative insets are unusual but should not crash.
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Constraints{Max: image.Pt(100, 100)},
		Metric:      unit.Metric{PxPerDp: 1},
	}
	in := Inset{Top: -5, Bottom: -5, Left: -5, Right: -5}
	dims := in.Layout(gtx, func(gtx Context) Dimensions {
		return Dimensions{Size: image.Pt(20, 20)}
	})
	// Widget=20, plus -10 horizontally and -10 vertically => 10x10.
	if dims.Size != image.Pt(10, 10) {
		t.Errorf("negative inset: got %v, want (10,10)", dims.Size)
	}
}

func TestInsetBaseline(t *testing.T) {
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Constraints{Max: image.Pt(100, 100)},
		Metric:      unit.Metric{PxPerDp: 1},
	}
	in := Inset{Top: 5, Bottom: 7, Left: 3, Right: 4}
	dims := in.Layout(gtx, func(gtx Context) Dimensions {
		return Dimensions{Size: image.Pt(20, 20), Baseline: 8}
	})
	// Inset adds Bottom (7) to child Baseline (8) per Inset.Layout.
	if dims.Baseline != 15 {
		t.Errorf("inset baseline: got %d, want 15", dims.Baseline)
	}
}

func TestSpacerConstrained(t *testing.T) {
	gtx := Context{
		Constraints: Constraints{
			Min: image.Pt(50, 50),
			Max: image.Pt(80, 80),
		},
		Metric: unit.Metric{PxPerDp: 1},
	}
	dims := Spacer{Width: 10, Height: 10}.Layout(gtx)
	if dims.Size != image.Pt(50, 50) {
		t.Errorf("spacer below Min: %v", dims.Size)
	}
	dims = Spacer{Width: 200, Height: 200}.Layout(gtx)
	if dims.Size != image.Pt(80, 80) {
		t.Errorf("spacer above Max: %v", dims.Size)
	}
}

func TestSpacerZero(t *testing.T) {
	gtx := Context{
		Constraints: Constraints{Max: image.Pt(100, 100)},
	}
	dims := Spacer{}.Layout(gtx)
	if dims.Size != (image.Point{}) {
		t.Errorf("zero spacer not zero: %v", dims.Size)
	}
}

func TestDirectionPositionAll(t *testing.T) {
	widget := image.Pt(20, 20)
	bounds := image.Pt(100, 100)
	cases := []struct {
		d    Direction
		want image.Point
	}{
		{NW, image.Pt(0, 0)},
		{N, image.Pt(40, 0)},
		{NE, image.Pt(80, 0)},
		{E, image.Pt(80, 40)},
		{SE, image.Pt(80, 80)},
		{S, image.Pt(40, 80)},
		{SW, image.Pt(0, 80)},
		{W, image.Pt(0, 40)},
		{Center, image.Pt(40, 40)},
	}
	for _, tc := range cases {
		t.Run(tc.d.String(), func(t *testing.T) {
			if got := tc.d.Position(widget, bounds); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDirectionLayoutMinClearing(t *testing.T) {
	cases := []struct {
		d       Direction
		wantMin image.Point
	}{
		{NW, image.Pt(0, 0)},
		{NE, image.Pt(0, 0)},
		{SW, image.Pt(0, 0)},
		{SE, image.Pt(0, 0)},
		{Center, image.Pt(0, 0)},
		{N, image.Pt(100, 0)},
		{S, image.Pt(100, 0)},
		{E, image.Pt(0, 100)},
		{W, image.Pt(0, 100)},
	}
	for _, tc := range cases {
		t.Run(tc.d.String(), func(t *testing.T) {
			gtx := Context{
				Ops:         new(op.Ops),
				Constraints: Exact(image.Pt(100, 100)),
			}
			var got image.Point
			tc.d.Layout(gtx, func(gtx Context) Dimensions {
				got = gtx.Constraints.Min
				return Dimensions{Size: image.Pt(20, 20)}
			})
			if got != tc.wantMin {
				t.Errorf("Direction %s: child Min = %v, want %v", tc.d, got, tc.wantMin)
			}
		})
	}
}

func TestDirectionLayoutFillsConstraints(t *testing.T) {
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Exact(image.Pt(100, 100)),
	}
	dims := Center.Layout(gtx, func(gtx Context) Dimensions {
		return Dimensions{Size: image.Pt(20, 20)}
	})
	if dims.Size != image.Pt(100, 100) {
		t.Errorf("Direction.Layout did not fill Min: %v", dims.Size)
	}
}

func TestContextDpSp(t *testing.T) {
	gtx := Context{
		Metric: unit.Metric{PxPerDp: 2, PxPerSp: 3},
	}
	if got := gtx.Dp(10); got != 20 {
		t.Errorf("Dp: got %d, want 20", got)
	}
	if got := gtx.Sp(10); got != 30 {
		t.Errorf("Sp: got %d, want 30", got)
	}
	// Zero metric defaults to 1.
	gtx2 := Context{}
	if got := gtx2.Dp(5); got != 5 {
		t.Errorf("Dp default: got %d, want 5", got)
	}
	if got := gtx2.Sp(7); got != 7 {
		t.Errorf("Sp default: got %d, want 7", got)
	}
}

func TestAlignmentStringAll(t *testing.T) {
	cases := map[Alignment]string{
		Start:    "Start",
		End:      "End",
		Middle:   "Middle",
		Baseline: "Baseline",
	}
	for a, want := range cases {
		if got := a.String(); got != want {
			t.Errorf("Alignment %d: got %q, want %q", a, got, want)
		}
	}
}

func TestAxisCrossConstraint(t *testing.T) {
	cs := Constraints{Min: image.Pt(1, 2), Max: image.Pt(3, 4)}
	min, max := Horizontal.crossConstraint(cs)
	if min != 2 || max != 4 {
		t.Errorf("Horizontal.crossConstraint: %d,%d", min, max)
	}
	min, max = Vertical.crossConstraint(cs)
	if min != 1 || max != 3 {
		t.Errorf("Vertical.crossConstraint: %d,%d", min, max)
	}
}

func TestAxisConstraints(t *testing.T) {
	c := Horizontal.constraints(1, 2, 3, 4)
	if c.Min != image.Pt(1, 3) || c.Max != image.Pt(2, 4) {
		t.Errorf("Horizontal.constraints: %+v", c)
	}
	c = Vertical.constraints(1, 2, 3, 4)
	if c.Min != image.Pt(3, 1) || c.Max != image.Pt(4, 2) {
		t.Errorf("Vertical.constraints: %+v", c)
	}
}
