package unit_test

import (
	"math"
	"testing"

	"github.com/nanorele/gio/unit"
)

func TestMetric_DpToSp(t *testing.T) {
	m := unit.Metric{
		PxPerDp: 2,
		PxPerSp: 3,
	}

	{
		exp := m.Dp(5)
		got := m.Sp(m.DpToSp(5))
		if got != exp {
			t.Errorf("DpToSp conversion mismatch %v != %v", exp, got)
		}
	}

	{
		exp := m.Sp(5)
		got := m.Dp(m.SpToDp(5))
		if got != exp {
			t.Errorf("SpToDp conversion mismatch %v != %v", exp, got)
		}
	}

	{
		exp := unit.Dp(5)
		got := m.PxToDp(m.Dp(5))
		if got != exp {
			t.Errorf("PxToDp conversion mismatch %v != %v", exp, got)
		}
	}

	{
		exp := unit.Sp(5)
		got := m.PxToSp(m.Sp(5))
		if got != exp {
			t.Errorf("PxToSp conversion mismatch %v != %v", exp, got)
		}
	}
}

func TestMetric_Zero(t *testing.T) {
	m := unit.Metric{}
	if got := m.Dp(5); got != 5 {
		t.Errorf("expected 5, got %d", got)
	}
	if got := m.Sp(5); got != 5 {
		t.Errorf("expected 5, got %d", got)
	}
	if got := m.DpToSp(5); got != 5 {
		t.Errorf("expected 5, got %v", got)
	}
	if got := m.SpToDp(5); got != 5 {
		t.Errorf("expected 5, got %v", got)
	}
	if got := m.PxToSp(5); got != 5 {
		t.Errorf("expected 5, got %v", got)
	}
	if got := m.PxToDp(5); got != 5 {
		t.Errorf("expected 5, got %v", got)
	}
}

func TestMetric_Dp_Rounding(t *testing.T) {
	cases := []struct {
		pxPerDp float32
		v       unit.Dp
		want    int
	}{
		{1, 0, 0},
		{1, 1, 1},
		{1, -1, -1},
		{2, 1, 2},
		{2.5, 2, 5},
		{1.5, 1, 2}, // 1.5 -> 2 (half away from zero)
		{1.5, -1, -2},
		{1, 0.4, 0},
		{1, 0.6, 1},
		{1, -0.5, -1},
	}
	for _, tc := range cases {
		m := unit.Metric{PxPerDp: tc.pxPerDp, PxPerSp: tc.pxPerDp}
		if got := m.Dp(tc.v); got != tc.want {
			t.Errorf("Metric{%v}.Dp(%v) = %v, want %v", tc.pxPerDp, tc.v, got, tc.want)
		}
	}
}

func TestMetric_Sp_Rounding(t *testing.T) {
	cases := []struct {
		pxPerSp float32
		v       unit.Sp
		want    int
	}{
		{1, 0, 0},
		{2, 3, 6},
		{0.5, 4, 2},
		{2, -3, -6},
		{1.49, 2, 3},
	}
	for _, tc := range cases {
		m := unit.Metric{PxPerDp: 1, PxPerSp: tc.pxPerSp}
		if got := m.Sp(tc.v); got != tc.want {
			t.Errorf("Metric{Sp:%v}.Sp(%v) = %v, want %v", tc.pxPerSp, tc.v, got, tc.want)
		}
	}
}

func TestMetric_NonZero_Treats_Zero_As_One(t *testing.T) {
	// PxPerDp == 0 means scale 1 (no division by zero).
	m := unit.Metric{}
	if got := m.Dp(7); got != 7 {
		t.Errorf("zero PxPerDp: Dp(7) = %v, want 7", got)
	}
	if got := m.Sp(7); got != 7 {
		t.Errorf("zero PxPerSp: Sp(7) = %v, want 7", got)
	}
	if got := m.PxToDp(7); got != 7 {
		t.Errorf("zero PxPerDp: PxToDp(7) = %v, want 7", got)
	}
	if got := m.PxToSp(7); got != 7 {
		t.Errorf("zero PxPerSp: PxToSp(7) = %v, want 7", got)
	}
	// Conversions must not divide by zero either.
	if got := m.DpToSp(5); math.IsNaN(float64(got)) || math.IsInf(float64(got), 0) {
		t.Errorf("zero metrics: DpToSp(5) = %v", got)
	}
	if got := m.SpToDp(5); math.IsNaN(float64(got)) || math.IsInf(float64(got), 0) {
		t.Errorf("zero metrics: SpToDp(5) = %v", got)
	}
}

func TestMetric_OnlyPxPerDpSet(t *testing.T) {
	// PxPerSp = 0 must be treated as 1 in DpToSp/SpToDp/PxToSp.
	m := unit.Metric{PxPerDp: 2}
	// DpToSp: v * 2 / 1 = 2v
	if got := m.DpToSp(3); got != 6 {
		t.Errorf("DpToSp(3) = %v, want 6", got)
	}
	if got := m.SpToDp(6); got != 3 {
		t.Errorf("SpToDp(6) = %v, want 3", got)
	}
	if got := m.PxToSp(5); got != 5 {
		t.Errorf("PxToSp(5) = %v, want 5", got)
	}
}

func TestMetric_DpToSp_RoundTrip(t *testing.T) {
	for _, m := range []unit.Metric{
		{PxPerDp: 1, PxPerSp: 1},
		{PxPerDp: 2, PxPerSp: 3},
		{PxPerDp: 1.5, PxPerSp: 2.5},
		{PxPerDp: 4, PxPerSp: 4},
	} {
		for _, v := range []unit.Dp{0, 1, 5, 10, 100, -5, -100} {
			got := m.SpToDp(m.DpToSp(v))
			// Allow tiny floating error
			diff := float32(got - v)
			if diff < 0 {
				diff = -diff
			}
			if diff > 1e-3 {
				t.Errorf("Metric%+v Dp->Sp->Dp(%v) = %v", m, v, got)
			}
		}
	}
}

func TestMetric_PxRoundTrip(t *testing.T) {
	for _, m := range []unit.Metric{
		{PxPerDp: 1, PxPerSp: 1},
		{PxPerDp: 2, PxPerSp: 3},
		{PxPerDp: 0.5, PxPerSp: 4},
	} {
		for _, v := range []int{0, 1, 5, 100, -5} {
			// PxToDp -> Dp must give back v (when rounding doesn't lose info)
			gotDp := m.Dp(m.PxToDp(v))
			if gotDp != v {
				t.Errorf("Metric%+v Px->Dp->Px(%v) = %v", m, v, gotDp)
			}
			gotSp := m.Sp(m.PxToSp(v))
			if gotSp != v {
				t.Errorf("Metric%+v Px->Sp->Px(%v) = %v", m, v, gotSp)
			}
		}
	}
}

func TestMetric_LargeValues(t *testing.T) {
	m := unit.Metric{PxPerDp: 3, PxPerSp: 3}
	const big unit.Dp = 1_000_000
	if got := m.Dp(big); got != 3_000_000 {
		t.Errorf("Dp(%v) = %v, want 3_000_000", big, got)
	}
	if got := m.Dp(-big); got != -3_000_000 {
		t.Errorf("Dp(%v) = %v, want -3_000_000", -big, got)
	}
}

func TestMetric_Negative(t *testing.T) {
	m := unit.Metric{PxPerDp: 2, PxPerSp: 2}
	if got := m.Dp(-3); got != -6 {
		t.Errorf("Dp(-3) = %v, want -6", got)
	}
	if got := m.PxToDp(-4); got != -2 {
		t.Errorf("PxToDp(-4) = %v, want -2", got)
	}
	if got := m.SpToDp(-4); got != -4 {
		t.Errorf("SpToDp(-4) = %v, want -4", got)
	}
}
