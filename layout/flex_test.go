package layout

import (
	"image"
	"testing"

	"github.com/nanorele/gio/op"
)

func mkGtx(max image.Point) Context {
	return Context{
		Ops:         new(op.Ops),
		Constraints: Constraints{Max: max},
	}
}

func mkRigid(sz image.Point) FlexChild {
	return Rigid(func(gtx Context) Dimensions {
		return Dimensions{Size: sz}
	})
}

func TestFlexAllRigid(t *testing.T) {
	gtx := mkGtx(image.Pt(200, 100))
	dims := Flex{}.Layout(gtx,
		mkRigid(image.Pt(20, 30)),
		mkRigid(image.Pt(40, 10)),
		mkRigid(image.Pt(10, 20)),
	)
	if dims.Size.X != 70 {
		t.Errorf("rigid sum X: got %d, want 70", dims.Size.X)
	}
	if dims.Size.Y != 30 {
		t.Errorf("max cross Y: got %d, want 30", dims.Size.Y)
	}
}

func TestFlexOneFlexed(t *testing.T) {
	gtx := mkGtx(image.Pt(100, 50))
	var flexMax int
	dims := Flex{}.Layout(gtx,
		mkRigid(image.Pt(20, 20)),
		Flexed(1, func(gtx Context) Dimensions {
			flexMax = gtx.Constraints.Max.X
			return Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 10)}
		}),
	)
	if flexMax != 80 {
		t.Errorf("flex got max=%d, want 80", flexMax)
	}
	if dims.Size.X != 100 {
		t.Errorf("dims X: got %d, want 100", dims.Size.X)
	}
}

func TestFlexMixedRigidFlexed(t *testing.T) {
	gtx := mkGtx(image.Pt(100, 50))
	var firstMax, secondMax int
	Flex{}.Layout(gtx,
		mkRigid(image.Pt(20, 10)),
		Flexed(1, func(gtx Context) Dimensions {
			firstMax = gtx.Constraints.Max.X
			return Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 10)}
		}),
		Flexed(3, func(gtx Context) Dimensions {
			secondMax = gtx.Constraints.Max.X
			return Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 10)}
		}),
	)
	// 80 remaining for two flex children with weights 1 and 3 => 20 and 60.
	if firstMax != 20 {
		t.Errorf("first flex max: got %d, want 20", firstMax)
	}
	if secondMax != 60 {
		t.Errorf("second flex max: got %d, want 60", secondMax)
	}
}

func TestFlexWeightSumZero(t *testing.T) {
	// Two flex children with zero weight: both should get 0 main-axis size.
	gtx := mkGtx(image.Pt(100, 50))
	var firstMax, secondMax int
	Flex{}.Layout(gtx,
		Flexed(0, func(gtx Context) Dimensions {
			firstMax = gtx.Constraints.Max.X
			return Dimensions{Size: image.Pt(0, 10)}
		}),
		Flexed(0, func(gtx Context) Dimensions {
			secondMax = gtx.Constraints.Max.X
			return Dimensions{Size: image.Pt(0, 10)}
		}),
	)
	if firstMax != 0 || secondMax != 0 {
		t.Errorf("zero-weight flex: %d, %d", firstMax, secondMax)
	}
}

func TestFlexChildZeroDimensions(t *testing.T) {
	gtx := mkGtx(image.Pt(100, 100))
	dims := Flex{}.Layout(gtx,
		mkRigid(image.Point{}),
		mkRigid(image.Point{}),
	)
	if dims.Size != (image.Point{}) {
		t.Errorf("flex of zero children: %v", dims.Size)
	}
}

func TestFlexSpacingDistribution(t *testing.T) {
	// With Min set, leftover space should not affect total width returned.
	gtx := Context{
		Ops: new(op.Ops),
		Constraints: Constraints{
			Min: image.Pt(100, 0),
			Max: image.Pt(100, 100),
		},
	}
	for _, sp := range []Spacing{SpaceEnd, SpaceStart, SpaceSides, SpaceAround, SpaceBetween, SpaceEvenly} {
		t.Run(sp.String(), func(t *testing.T) {
			dims := Flex{Spacing: sp}.Layout(gtx,
				mkRigid(image.Pt(20, 10)),
				mkRigid(image.Pt(20, 10)),
			)
			if dims.Size.X != 100 {
				t.Errorf("Spacing %s: got width %d, want 100", sp, dims.Size.X)
			}
		})
	}
}

func TestFlexSpacingNoChildren(t *testing.T) {
	// SpaceAround with zero children must not divide by zero.
	gtx := Context{
		Ops: new(op.Ops),
		Constraints: Constraints{
			Min: image.Pt(50, 0),
			Max: image.Pt(100, 100),
		},
	}
	for _, sp := range []Spacing{SpaceEnd, SpaceStart, SpaceSides, SpaceAround, SpaceBetween, SpaceEvenly} {
		t.Run(sp.String(), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Spacing %s panicked with no children: %v", sp, r)
				}
			}()
			dims := Flex{Spacing: sp}.Layout(gtx)
			// Should at least satisfy Min.
			if dims.Size.X < 50 {
				t.Errorf("Spacing %s: width %d < Min 50", sp, dims.Size.X)
			}
		})
	}
}

func TestFlexAlignmentStart(t *testing.T) {
	gtx := mkGtx(image.Pt(100, 100))
	dims := Flex{Alignment: Start}.Layout(gtx,
		mkRigid(image.Pt(10, 10)),
		mkRigid(image.Pt(10, 30)),
	)
	if dims.Size.Y != 30 {
		t.Errorf("Start alignment cross: got %d, want 30", dims.Size.Y)
	}
}

func TestFlexAlignmentMiddle(t *testing.T) {
	gtx := mkGtx(image.Pt(100, 100))
	dims := Flex{Alignment: Middle}.Layout(gtx,
		mkRigid(image.Pt(10, 10)),
		mkRigid(image.Pt(10, 30)),
	)
	if dims.Size.Y != 30 {
		t.Errorf("Middle alignment cross: got %d, want 30", dims.Size.Y)
	}
}

func TestFlexAlignmentEnd(t *testing.T) {
	gtx := mkGtx(image.Pt(100, 100))
	dims := Flex{Alignment: End}.Layout(gtx,
		mkRigid(image.Pt(10, 10)),
		mkRigid(image.Pt(10, 30)),
	)
	if dims.Size.Y != 30 {
		t.Errorf("End alignment cross: got %d, want 30", dims.Size.Y)
	}
}

func TestFlexAlignmentBaseline(t *testing.T) {
	gtx := mkGtx(image.Pt(100, 100))
	dims := Flex{Alignment: Baseline}.Layout(gtx,
		Rigid(func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(10, 20), Baseline: 5}
		}),
		Rigid(func(gtx Context) Dimensions {
			return Dimensions{Size: image.Pt(10, 30), Baseline: 10}
		}),
	)
	if dims.Size.Y == 0 {
		t.Errorf("Baseline alignment produced zero Y")
	}
}

func TestFlexVertical(t *testing.T) {
	gtx := mkGtx(image.Pt(100, 200))
	dims := Flex{Axis: Vertical}.Layout(gtx,
		mkRigid(image.Pt(10, 30)),
		mkRigid(image.Pt(20, 40)),
	)
	if dims.Size.Y != 70 {
		t.Errorf("vertical sum Y: got %d, want 70", dims.Size.Y)
	}
	if dims.Size.X != 20 {
		t.Errorf("vertical max cross X: got %d, want 20", dims.Size.X)
	}
}

func TestFlexVerticalFlexed(t *testing.T) {
	gtx := mkGtx(image.Pt(100, 200))
	var flexMax int
	Flex{Axis: Vertical}.Layout(gtx,
		mkRigid(image.Pt(10, 50)),
		Flexed(1, func(gtx Context) Dimensions {
			flexMax = gtx.Constraints.Max.Y
			return Dimensions{Size: image.Pt(10, gtx.Constraints.Max.Y)}
		}),
	)
	if flexMax != 150 {
		t.Errorf("vertical flex max: got %d, want 150", flexMax)
	}
}

func TestFlexWeightSumOverride(t *testing.T) {
	// WeightSum forces a denominator: with WeightSum=4 and one child weight=1,
	// child gets 1/4 of the available space.
	gtx := mkGtx(image.Pt(100, 50))
	var flexMax int
	Flex{WeightSum: 4}.Layout(gtx,
		Flexed(1, func(gtx Context) Dimensions {
			flexMax = gtx.Constraints.Max.X
			return Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 10)}
		}),
	)
	if flexMax != 25 {
		t.Errorf("WeightSum override: got %d, want 25", flexMax)
	}
}

func TestFlexWeightRoundingExactSum(t *testing.T) {
	// Sum of distributed flex sizes must equal the available space (within 1).
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Constraints{Min: image.Pt(100, 0), Max: image.Pt(100, 50)},
	}
	var sizes []int
	Flex{}.Layout(gtx,
		Flexed(1, func(gtx Context) Dimensions {
			sizes = append(sizes, gtx.Constraints.Max.X)
			return Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 10)}
		}),
		Flexed(1, func(gtx Context) Dimensions {
			sizes = append(sizes, gtx.Constraints.Max.X)
			return Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 10)}
		}),
		Flexed(1, func(gtx Context) Dimensions {
			sizes = append(sizes, gtx.Constraints.Max.X)
			return Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 10)}
		}),
	)
	total := 0
	for _, s := range sizes {
		total += s
	}
	// 100 / 3 with rounding correction: should sum to exactly 100.
	if total != 100 {
		t.Errorf("flex weight rounding: sum=%d, want 100", total)
	}
}

func TestFlexNoChildrenMin(t *testing.T) {
	gtx := Context{
		Ops:         new(op.Ops),
		Constraints: Constraints{Min: image.Pt(50, 50), Max: image.Pt(100, 100)},
	}
	dims := Flex{}.Layout(gtx)
	if dims.Size != image.Pt(50, 50) {
		t.Errorf("empty flex respect min: %v", dims.Size)
	}
}

func TestSpacingString(t *testing.T) {
	cases := map[Spacing]string{
		SpaceEnd:     "SpaceEnd",
		SpaceStart:   "SpaceStart",
		SpaceSides:   "SpaceSides",
		SpaceAround:  "SpaceAround",
		SpaceBetween: "SpaceBetween",
		SpaceEvenly:  "SpaceEvenly",
	}
	for sp, want := range cases {
		if got := sp.String(); got != want {
			t.Errorf("Spacing %d: got %q, want %q", sp, got, want)
		}
	}
}

func TestFlexManyChildrenScratch(t *testing.T) {
	// More than 32 children to exercise the heap-allocated scratch path.
	gtx := mkGtx(image.Pt(1000, 100))
	children := make([]FlexChild, 40)
	for i := range children {
		children[i] = mkRigid(image.Pt(10, 10))
	}
	dims := Flex{}.Layout(gtx, children...)
	if dims.Size.X != 400 {
		t.Errorf("40 rigid children: got %d, want 400", dims.Size.X)
	}
}
