package system

import (
	"strings"
	"testing"
	"time"
)

func timeAfter() <-chan time.Time {
	return time.After(2 * time.Second)
}

func TestActionStringSingleBits(t *testing.T) {
	cases := []struct {
		a    Action
		want string
	}{
		{ActionMinimize, "ActionMinimize"},
		{ActionMaximize, "ActionMaximize"},
		{ActionUnmaximize, "ActionUnmaximize"},
		{ActionFullscreen, "ActionFullscreen"},
		{ActionRaise, "ActionRaise"},
		{ActionCenter, "ActionCenter"},
		{ActionClose, "ActionClose"},
		{ActionMove, "ActionMove"},
	}
	for _, c := range cases {
		got := c.a.String()
		if got != c.want {
			t.Errorf("Action(%d).String() = %q, want %q", c.a, got, c.want)
		}
	}
}

func TestActionStringLowercaseSingleBits(t *testing.T) {
	cases := []struct {
		a    Action
		want string
	}{
		{ActionMinimize, "ActionMinimize"},
		{ActionMaximize, "ActionMaximize"},
		{ActionUnmaximize, "ActionUnmaximize"},
		{ActionFullscreen, "ActionFullscreen"},
		{ActionRaise, "ActionRaise"},
		{ActionCenter, "ActionCenter"},
		{ActionClose, "ActionClose"},
		{ActionMove, "ActionMove"},
	}
	for _, c := range cases {
		got := c.a.string()
		if got != c.want {
			t.Errorf("Action(%d).string() = %q, want %q", c.a, got, c.want)
		}
	}
}

func TestActionStringCombinations(t *testing.T) {
	cases := []struct {
		a    Action
		want string
	}{
		{ActionMinimize | ActionClose, "ActionMinimize|ActionClose"},
		{ActionMaximize | ActionUnmaximize, "ActionMaximize|ActionUnmaximize"},
		{ActionFullscreen | ActionRaise | ActionCenter, "ActionFullscreen|ActionRaise|ActionCenter"},
		{ActionMinimize | ActionMaximize | ActionUnmaximize | ActionFullscreen | ActionRaise | ActionCenter | ActionClose | ActionMove,
			"ActionMinimize|ActionMaximize|ActionUnmaximize|ActionFullscreen|ActionRaise|ActionCenter|ActionClose|ActionMove"},
	}
	for _, c := range cases {
		got := c.a.String()
		if got != c.want {
			t.Errorf("Action(%b).String() = %q, want %q", c.a, got, c.want)
		}
	}
}

func TestActionStringSeparator(t *testing.T) {
	got := (ActionMinimize | ActionClose).String()
	if !strings.Contains(got, "|") {
		t.Errorf("expected '|' separator in %q", got)
	}
	parts := strings.Split(got, "|")
	if len(parts) != 2 {
		t.Errorf("expected 2 parts separated by '|', got %d in %q", len(parts), got)
	}
}

func TestActionStringZero(t *testing.T) {
	if got := Action(0).String(); got != "" {
		t.Errorf("Action(0).String() = %q, want empty string", got)
	}
}

func TestActionStringAllDefinedBitsTerminates(t *testing.T) {
	all := ActionMinimize | ActionMaximize | ActionUnmaximize | ActionFullscreen |
		ActionRaise | ActionCenter | ActionClose | ActionMove
	done := make(chan string, 1)
	go func() {
		done <- all.String()
	}()
	select {
	case s := <-done:
		if s == "" {
			t.Errorf("expected non-empty string, got empty")
		}
	case <-timeAfter():
		t.Fatal("Action.String() did not terminate with all defined bits set")
	}
}

func TestActionStringAllBitsTerminates(t *testing.T) {
	a := Action(^uint(0))
	done := make(chan string, 1)
	go func() {
		done <- a.String()
	}()
	select {
	case <-done:
	case <-timeAfter():
		t.Fatal("Action.String() did not terminate with all bits set (^uint(0))")
	}
}

func TestLTRConstantValue(t *testing.T) {
	if byte(LTR) != 0x00 {
		t.Errorf("LTR raw byte = %#x, want 0x00", byte(LTR))
	}
}

func TestRTLConstantValue(t *testing.T) {
	if byte(RTL) != 0x02 {
		t.Errorf("RTL raw byte = %#x, want 0x02", byte(RTL))
	}
}

func TestTextDirectionAxis(t *testing.T) {
	if got := LTR.Axis(); got != Horizontal {
		t.Errorf("LTR.Axis() = %d, want %d (Horizontal)", got, Horizontal)
	}
	if got := RTL.Axis(); got != Horizontal {
		t.Errorf("RTL.Axis() = %d, want %d (Horizontal)", got, Horizontal)
	}
}

func TestTextDirectionProgression(t *testing.T) {
	if got := LTR.Progression(); got != FromOrigin {
		t.Errorf("LTR.Progression() = %d, want %d (FromOrigin)", got, FromOrigin)
	}
	if got := RTL.Progression(); got != TowardOrigin {
		t.Errorf("RTL.Progression() = %d, want %d (TowardOrigin)", got, TowardOrigin)
	}
}

func TestTextDirectionString(t *testing.T) {
	if got := LTR.String(); got != "LTR" {
		t.Errorf("LTR.String() = %q, want %q", got, "LTR")
	}
	if got := RTL.String(); got != "RTL" {
		t.Errorf("RTL.String() = %q, want %q", got, "RTL")
	}
}
