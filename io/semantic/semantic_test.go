package semantic

import "testing"

func TestClassOpString(t *testing.T) {
	tests := []struct {
		c    ClassOp
		want string
	}{
		{Unknown, "Unknown"},
		{Button, "Button"},
		{CheckBox, "CheckBox"},
		{Editor, "Editor"},
		{RadioButton, "RadioButton"},
		{Switch, "Switch"},
	}
	for _, tc := range tests {
		if got := tc.c.String(); got != tc.want {
			t.Errorf("ClassOp(%d).String() = %q, want %q", tc.c, got, tc.want)
		}
	}
}

func TestClassOpStringInvalidPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("invalid ClassOp.String() should panic")
		}
	}()
	_ = ClassOp(99).String()
}

func TestSelectedOpType(t *testing.T) {
	// Construction-only smoke test: ensure SelectedOp acts as a bool wrapper.
	var s SelectedOp = true
	if !bool(s) {
		t.Errorf("SelectedOp(true) should convert to bool true")
	}
	s = false
	if bool(s) {
		t.Errorf("SelectedOp(false) should convert to bool false")
	}
}

func TestEnabledOpType(t *testing.T) {
	var e EnabledOp = true
	if !bool(e) {
		t.Errorf("EnabledOp(true) should convert to bool true")
	}
	e = false
	if bool(e) {
		t.Errorf("EnabledOp(false) should convert to bool false")
	}
}

func TestLabelDescTypes(t *testing.T) {
	var l LabelOp = "hello"
	if string(l) != "hello" {
		t.Errorf("LabelOp string conversion mismatch")
	}
	var d DescriptionOp = "desc"
	if string(d) != "desc" {
		t.Errorf("DescriptionOp string conversion mismatch")
	}
}
