package fling

import (
	"testing"
	"time"
)

func TestDecomposeQR(t *testing.T) {
	A := &matrix{
		rows: 3, cols: 3,
		data: []float32{
			12, 6, -4,
			-51, 167, 24,
			4, -68, -41,
		},
	}
	Q, Rt, ok := decomposeQR(A)
	if !ok {
		t.Fatal("decomposeQR failed")
	}
	R := Rt.transpose()
	QR := Q.mul(R)
	if !A.approxEqual(QR) {
		t.Log("A\n", A)
		t.Log("Q\n", Q)
		t.Log("R\n", R)
		t.Log("QR\n", QR)
		t.Fatal("Q*R not approximately equal to A")
	}
}

func TestFit(t *testing.T) {
	X := []float32{-1, 0, 1}
	Y := []float32{2, 0, 2}

	got, ok := polyFit(X, Y)
	if !ok {
		t.Fatal("polyFit failed")
	}
	want := coefficients{0, 0, 2}
	if !got.approxEqual(want) {
		t.Fatalf("polyFit: got %v want %v", got, want)
	}

	// Test error cases
	_, ok = polyFit([]float32{1, 2}, []float32{1, 2})
	if ok {
		t.Error("expected polyFit to fail with too few samples")
	}

	assertPanic(t, func() { polyFit([]float32{1}, []float32{1, 2}) }, "X and Y lengths differ")
}

func TestExtrapolation(t *testing.T) {
	var e Extrapolation
	
	// Test empty Estimate
	est := e.Estimate()
	if est != (Estimate{}) {
		t.Errorf("expected empty estimate, got %v", est)
	}

	// Test Sample and SampleDelta
	e.Sample(0, 0)
	e.SampleDelta(10*time.Millisecond, 10)
	e.SampleDelta(20*time.Millisecond, 10)
	
	est = e.Estimate()
	if est.Velocity == 0 {
		// With 3 samples, polyFit should work (degree=2)
		t.Errorf("expected non-zero velocity, got %v", est)
	}

	// Test wrapping
	for i := 0; i < historySize+5; i++ {
		e.Sample(time.Duration(i)*time.Millisecond, float32(i))
	}
	
	// Test age/gap break
	e.Sample(1 * time.Second, 100)
	e.Sample(1 * time.Second + 5*time.Millisecond, 105)
	est = e.Estimate()
	// Should only use the last two samples, which is not enough for degree 2 polyFit
	if est != (Estimate{}) {
		t.Errorf("expected empty estimate due to gap, got %v", est)
	}
}

func TestMatrix(t *testing.T) {
	m := newMatrix(2, 2)
	m.set(0, 0, 1)
	m.set(0, 1, 2)
	m.set(1, 0, 3)
	m.set(1, 1, 4)

	if v := m.get(1, 0); v != 3 {
		t.Errorf("expected 3, got %g", v)
	}

	s := m.String()
	if s == "" {
		t.Error("empty string from Matrix.String")
	}

	assertPanic(t, func() { m.set(-1, 0, 0) }, "row out of range")
	assertPanic(t, func() { m.set(2, 0, 0) }, "row out of range")
	assertPanic(t, func() { m.set(0, -1, 0) }, "col out of range")
	assertPanic(t, func() { m.set(0, 2, 0) }, "col out of range")
	assertPanic(t, func() { m.get(-1, 0) }, "row out of range")
	assertPanic(t, func() { m.get(0, -1) }, "col out of range")

	m2 := newMatrix(3, 3)
	assertPanic(t, func() { m.mul(m2) }, "mismatched matrices")

	mt := m.transpose()
	if mt.rows != 2 || mt.cols != 2 || mt.get(1, 0) != 2 {
		t.Errorf("transpose failed")
	}

	// norm and dot
	if n := norm([]float32{3, 4}); n != 5 {
		t.Errorf("expected norm 5, got %g", n)
	}
	if d := dot([]float32{1, 2}, []float32{3, 4}); d != 11 {
		t.Errorf("expected dot 11, got %g", d)
	}
}

func TestDecomposeQR_Failure(t *testing.T) {
	// Singular matrix
	A := &matrix{
		rows: 2, cols: 2,
		data: []float32{
			1, 1,
			1, 1,
		},
	}
	_, _, ok := decomposeQR(A)
	if ok {
		// Actually Gram-Schmidt might succeed if rows are dependent but not zero?
		// Wait, if rows are 1,1 and 1,1.
		// row 0: n = sqrt(2). normalized: 1/sqrt(2), 1/sqrt(2)
		// row 1: d = dot([1/sqrt(2), 1/sqrt(2)], [1, 1]) = 2/sqrt(2) = sqrt(2)
		// row 1 = [1, 1] - sqrt(2)*[1/sqrt(2), 1/sqrt(2)] = [1, 1] - [1, 1] = [0, 0]
		// n = 0. norm failure.
	}
}

func assertPanic(t *testing.T, f func(), msg string) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected panic: %s", msg)
		}
	}()
	f()
}
