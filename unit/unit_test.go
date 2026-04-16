package unit_test

import (
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
