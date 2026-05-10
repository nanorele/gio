package f32

import (
	"image"
	"math"
	"testing"
)

func TestRectangle(t *testing.T) {
	r := Rect(10, 20, 30, 40)
	if got, want := r.Min, (Point{X: 10, Y: 20}); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := r.Max, (Point{X: 30, Y: 40}); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	r2 := Rect(30, 40, 10, 20)
	if r != r2 {
		t.Errorf("Rect(30, 40, 10, 20) should be same as Rect(10, 20, 30, 40)")
	}

	if got, want := r.String(), "(10,20)-(30,40)"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}

	if got, want := r.Size(), (Point{X: 20, Y: 20}); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := r.Dx(), float32(20); got != want {
		t.Errorf("got %f, want %f", got, want)
	}

	if got, want := r.Dy(), float32(20); got != want {
		t.Errorf("got %f, want %f", got, want)
	}

	// Intersect
	tests1 := []struct {
		r, s Rectangle
		want Rectangle
	}{
		{Rect(10, 10, 30, 30), Rect(20, 20, 40, 40), Rect(20, 20, 30, 30)},
		{Rect(20, 20, 40, 40), Rect(10, 10, 30, 30), Rect(20, 20, 30, 30)},
		{Rect(10, 10, 20, 20), Rect(30, 30, 40, 40), Rectangle{}},
	}
	for _, tc := range tests1 {
		if got := tc.r.Intersect(tc.s); got != tc.want {
			t.Errorf("%v Intersect %v: got %v, want %v", tc.r, tc.s, got, tc.want)
		}
	}

	// Union
	tests2 := []struct {
		r, s Rectangle
		want Rectangle
	}{
		{Rect(10, 10, 30, 30), Rect(20, 20, 40, 40), Rect(10, 10, 40, 40)},
		{Rect(20, 20, 40, 40), Rect(10, 10, 30, 30), Rect(10, 10, 40, 40)},
		{Rectangle{}, Rect(10, 10, 20, 20), Rect(10, 10, 20, 20)},
		{Rect(10, 10, 20, 20), Rectangle{}, Rect(10, 10, 20, 20)},
	}
	for _, tc := range tests2 {
		if got := tc.r.Union(tc.s); got != tc.want {
			t.Errorf("%v Union %v: got %v, want %v", tc.r, tc.s, got, tc.want)
		}
	}

	// Canon
	tests3 := []struct {
		r, want Rectangle
	}{
		{Rectangle{Point{X: 30, Y: 40}, Point{X: 10, Y: 20}}, Rect(10, 20, 30, 40)},
		{Rectangle{Point{X: 10, Y: 40}, Point{X: 30, Y: 20}}, Rect(10, 20, 30, 40)},
	}
	for _, tc := range tests3 {
		if got := tc.r.Canon(); got != tc.want {
			t.Errorf("Canon %v: got %v, want %v", tc.r, got, tc.want)
		}
	}

	// Empty
	if !Rect(10, 10, 10, 20).Empty() {
		t.Error("Rect(10, 10, 10, 20) should be empty")
	}
	if !Rect(10, 10, 20, 10).Empty() {
		t.Error("Rect(10, 10, 20, 10) should be empty")
	}

	// Add/Sub
	p := Point{X: 5, Y: 5}
	if got, want := r.Add(p), Rect(15, 25, 35, 45); got != want {
		t.Errorf("Add: got %v, want %v", got, want)
	}
	if got, want := r.Sub(p), Rect(5, 15, 25, 35); got != want {
		t.Errorf("Sub: got %v, want %v", got, want)
	}

	// Round
	rr := Rect(10.1, 20.9, 30.1, 40.9).Round()
	if got, want := rr, image.Rect(10, 20, 31, 41); got != want {
		t.Errorf("Round: got %v, want %v", got, want)
	}

	// FRect / FPt
	ir := image.Rect(10, 20, 30, 40)
	if got, want := FRect(ir), r; got != want {
		t.Errorf("FRect: got %v, want %v", got, want)
	}
}

func TestAliases(t *testing.T) {
	_ = Point{}
	_ = Affine2D{}
	_ = NewAffine2D(1, 0, 0, 1, 0, 0)
	_ = AffineId()
	_ = Pt(0, 0)
}

const eps = 1e-5

func almostEq(a, b float32) bool {
	return math.Abs(float64(a-b)) < eps
}

func affEq(t *testing.T, a Affine2D, sx, hx, ox, hy, sy, oy float32) {
	t.Helper()
	gsx, ghx, gox, ghy, gsy, goy := a.Elems()
	if !almostEq(gsx, sx) || !almostEq(ghx, hx) || !almostEq(gox, ox) ||
		!almostEq(ghy, hy) || !almostEq(gsy, sy) || !almostEq(goy, oy) {
		t.Errorf("Affine2D mismatch: got (%v,%v,%v,%v,%v,%v) want (%v,%v,%v,%v,%v,%v)",
			gsx, ghx, gox, ghy, gsy, goy, sx, hx, ox, hy, sy, oy)
	}
}

func ptEq(t *testing.T, got, want Point) {
	t.Helper()
	if !almostEq(got.X, want.X) || !almostEq(got.Y, want.Y) {
		t.Errorf("Point mismatch: got %v want %v", got, want)
	}
}

func TestAffineIdentityElems(t *testing.T) {
	affEq(t, AffineId(), 1, 0, 0, 0, 1, 0)
}

func TestAffineMulIdentity(t *testing.T) {
	id := AffineId()
	a := NewAffine2D(2, 3, 5, 7, 11, 13)
	affEq(t, id.Mul(a), 2, 3, 5, 7, 11, 13)
	affEq(t, a.Mul(id), 2, 3, 5, 7, 11, 13)
}

func TestAffineMulAssociative(t *testing.T) {
	a := NewAffine2D(2, 1, 3, 0, 1, 4)
	b := NewAffine2D(1, 2, 0, 3, 1, 1)
	c := NewAffine2D(0.5, 0, 1, 0, 2, 2)
	left := a.Mul(b).Mul(c)
	right := a.Mul(b.Mul(c))
	p := Pt(1.5, -2.5)
	ptEq(t, left.Transform(p), right.Transform(p))
}

func TestAffineInvertOfInvert(t *testing.T) {
	a := NewAffine2D(2, 1, 3, 0, 4, -1)
	got := a.Invert().Invert()
	gsx, ghx, gox, ghy, gsy, goy := got.Elems()
	affEq(t, a, gsx, ghx, gox, ghy, gsy, goy)
}

func TestAffineInvertOffsetOnly(t *testing.T) {
	a := AffineId().Offset(Pt(7, -9))
	inv := a.Invert()
	affEq(t, inv, 1, 0, -7, 0, 1, 9)
	ptEq(t, inv.Transform(a.Transform(Pt(3, 4))), Pt(3, 4))
}

func TestAffineInvertSingular(t *testing.T) {
	// Scale by 0 in X collapses every point onto x=0.
	// The contract of Invert on a singular matrix is undocumented;
	// it must not panic and may produce non-finite values.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Invert of singular matrix panicked: %v", r)
		}
	}()
	a := AffineId().Scale(Point{}, Pt(0, 1))
	_ = a.Invert()
}

func TestAffineScaleZero(t *testing.T) {
	a := AffineId().Scale(Point{}, Pt(0, 0))
	ptEq(t, a.Transform(Pt(5, 7)), Pt(0, 0))
}

func TestAffineScaleAroundOrigin(t *testing.T) {
	origin := Pt(2, 3)
	a := AffineId().Scale(origin, Pt(2, 3))
	ptEq(t, a.Transform(origin), origin)
	ptEq(t, a.Transform(Pt(3, 4)), Pt(4, 6))
}

func TestAffineRotateZero(t *testing.T) {
	a := AffineId().Rotate(Point{}, 0)
	p := Pt(2.5, -1.25)
	ptEq(t, a.Transform(p), p)
}

func TestAffineRotateFullTurn(t *testing.T) {
	a := AffineId().Rotate(Point{}, 2*math.Pi)
	p := Pt(1, 2)
	ptEq(t, a.Transform(p), p)
}

func TestAffineRotateHalfTurn(t *testing.T) {
	a := AffineId().Rotate(Point{}, math.Pi)
	ptEq(t, a.Transform(Pt(1, 2)), Pt(-1, -2))
}

func TestAffineRotateAroundOrigin(t *testing.T) {
	origin := Pt(1, 1)
	a := AffineId().Rotate(origin, math.Pi)
	ptEq(t, a.Transform(origin), origin)
	ptEq(t, a.Transform(Pt(2, 1)), Pt(0, 1))
}

func TestAffineOffsetCompose(t *testing.T) {
	a := AffineId().Offset(Pt(1, 2)).Offset(Pt(3, 4))
	affEq(t, a, 1, 0, 4, 0, 1, 6)
}

func TestAffineShearXOnly(t *testing.T) {
	// Shear in X by tan(π/4)=1: (x,y) -> (x+y, y).
	a := AffineId().Shear(Point{}, math.Pi/4, 0)
	ptEq(t, a.Transform(Pt(2, 3)), Pt(5, 3))
}

func TestAffineShearYOnly(t *testing.T) {
	// Shear in Y by tan(π/4)=1: (x,y) -> (x, y+x).
	a := AffineId().Shear(Point{}, 0, math.Pi/4)
	ptEq(t, a.Transform(Pt(2, 3)), Pt(2, 5))
}

func TestAffineShearAfterOffset(t *testing.T) {
	// Regression: shear must use the matrix's c (X translation) when
	// computing the new f (Y translation), not its own f. With offset
	// (-2,-1) followed by Y-shear of tan(π/4)=1, mapping (0,0) gives
	// p1 = (-2,-1) then p2 = (-2, -1 + (-2)*1) = (-2,-3).
	a := AffineId().Offset(Pt(-2, -1)).Shear(Point{}, 0, math.Pi/4)
	ptEq(t, a.Transform(Pt(0, 0)), Pt(-2, -3))
}

func TestAffineTransformVerySmall(t *testing.T) {
	a := AffineId().Scale(Point{}, Pt(1e-20, 1e-20))
	got := a.Transform(Pt(1, 1))
	if !almostEq(got.X, 0) || !almostEq(got.Y, 0) {
		t.Errorf("Tiny scale: got %v", got)
	}
}

func TestAffineTransformVeryLarge(t *testing.T) {
	a := AffineId().Scale(Point{}, Pt(1e10, 1e10))
	got := a.Transform(Pt(1, 1))
	if math.IsInf(float64(got.X), 0) || math.IsInf(float64(got.Y), 0) {
		t.Errorf("Large scale produced Inf: %v", got)
	}
	if got.X != 1e10 || got.Y != 1e10 {
		t.Errorf("Large scale: got %v want (1e10,1e10)", got)
	}
}

func TestAliasesIdentityFromValues(t *testing.T) {
	a := NewAffine2D(1, 0, 0, 0, 1, 0)
	id := AffineId()
	asx, ahx, aox, ahy, asy, aoy := a.Elems()
	isx, ihx, iox, ihy, isy, ioy := id.Elems()
	if asx != isx || ahx != ihx || aox != iox || ahy != ihy || asy != isy || aoy != ioy {
		t.Errorf("NewAffine2D(identity) != AffineId(): %v vs %v", a, id)
	}
}
