package f32

import (
	"image"
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
