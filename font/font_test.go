package font

import (
	"testing"
)

func TestStyleString(t *testing.T) {
	cases := []struct {
		s    Style
		want string
	}{
		{Regular, "Regular"},
		{Italic, "Italic"},
	}
	for _, c := range cases {
		if got := c.s.String(); got != c.want {
			t.Errorf("Style(%d).String() = %q, want %q", int(c.s), got, c.want)
		}
	}
}

func TestStyleStringInvalidPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Style(99).String() did not panic")
		}
	}()
	_ = Style(99).String()
}

func TestWeightString(t *testing.T) {
	cases := []struct {
		w    Weight
		want string
	}{
		{Thin, "Thin"},
		{ExtraLight, "ExtraLight"},
		{Light, "Light"},
		{Normal, "Normal"},
		{Medium, "Medium"},
		{SemiBold, "SemiBold"},
		{Bold, "Bold"},
		{ExtraBold, "ExtraBold"},
		{Black, "Black"},
	}
	for _, c := range cases {
		if got := c.w.String(); got != c.want {
			t.Errorf("Weight(%d).String() = %q, want %q", int(c.w), got, c.want)
		}
	}
}

func TestWeightStringInvalidPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Weight(12345).String() did not panic")
		}
	}()
	_ = Weight(12345).String()
}
