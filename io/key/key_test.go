package key

import (
	"testing"

	"github.com/nanorele/gio/io/event"
	"github.com/nanorele/gio/op"
)

func TestModifiersContain(t *testing.T) {
	var zero Modifiers
	if !zero.Contain(0) {
		t.Errorf("empty Modifiers should contain empty Modifiers")
	}

	if !ModCtrl.Contain(ModCtrl) {
		t.Errorf("ModCtrl should contain itself")
	}
	if !ModShift.Contain(ModShift) {
		t.Errorf("ModShift should contain itself")
	}

	combined := ModCtrl | ModShift | ModAlt
	if !combined.Contain(ModCtrl) {
		t.Errorf("combined should contain ModCtrl")
	}
	if !combined.Contain(ModShift) {
		t.Errorf("combined should contain ModShift")
	}
	if !combined.Contain(ModAlt) {
		t.Errorf("combined should contain ModAlt")
	}
	if !combined.Contain(ModCtrl | ModShift) {
		t.Errorf("combined should contain ModCtrl|ModShift")
	}
	if !combined.Contain(combined) {
		t.Errorf("combined should contain itself")
	}

	if ModCtrl.Contain(ModCtrl | ModShift) {
		t.Errorf("single ModCtrl should not contain ModCtrl|ModShift")
	}
	if ModCtrl.Contain(ModShift) {
		t.Errorf("ModCtrl should not contain ModShift")
	}
	if zero.Contain(ModCtrl) {
		t.Errorf("empty Modifiers should not contain ModCtrl")
	}

	if !ModCtrl.Contain(0) {
		t.Errorf("any Modifiers should contain empty Modifiers")
	}
}

func TestModifiersString(t *testing.T) {
	tests := []struct {
		mods Modifiers
		want string
	}{
		{0, ""},
		{ModCtrl, "Ctrl"},
		{ModCommand, "⌘"},
		{ModShift, "Shift"},
		{ModAlt, "Alt"},
		{ModSuper, "Super"},
		{ModCtrl | ModShift, "Ctrl-Shift"},
		{ModCtrl | ModCommand | ModShift | ModAlt | ModSuper, "Ctrl-⌘-Shift-Alt-Super"},
		{ModAlt | ModCtrl, "Ctrl-Alt"},
		{ModSuper | ModShift, "Shift-Super"},
	}
	for _, tc := range tests {
		got := tc.mods.String()
		if got != tc.want {
			t.Errorf("Modifiers(%d).String() = %q, want %q", tc.mods, got, tc.want)
		}
	}
}

func TestStateString(t *testing.T) {
	if got := Press.String(); got != "Press" {
		t.Errorf("Press.String() = %q, want %q", got, "Press")
	}
	if got := Release.String(); got != "Release" {
		t.Errorf("Release.String() = %q, want %q", got, "Release")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("invalid State.String() should panic")
		}
	}()
	_ = State(99).String()
}

func TestEventMarkers(t *testing.T) {
	var _ event.Event = EditEvent{}
	var _ event.Event = Event{}
	var _ event.Event = FocusEvent{}
	var _ event.Event = SnippetEvent{}
	var _ event.Event = SelectionEvent{}
}

// commandMarker matches io/input.Command without importing it (avoiding cycles).
type commandMarker interface {
	ImplementsCommand()
}

func TestCommandMarkers(t *testing.T) {
	var _ commandMarker = FocusCmd{}
	var _ commandMarker = SoftKeyboardCmd{}
	var _ commandMarker = SelectionCmd{}
	var _ commandMarker = SnippetCmd{}
}

func TestFilterMarkers(t *testing.T) {
	var _ event.Filter = Filter{}
	var _ event.Filter = FocusFilter{}
}

func TestInputHintOpAddPanicsOnNilTag(t *testing.T) {
	var o op.Ops
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("InputHintOp.Add with nil Tag should panic")
		}
	}()
	InputHintOp{Tag: nil, Hint: HintAny}.Add(&o)
}

func TestInputHintOpAddOK(t *testing.T) {
	var o op.Ops
	tag := new(int)
	InputHintOp{Tag: tag, Hint: HintEmail}.Add(&o)
}

func TestModShortcutDefault(t *testing.T) {
	// On non-darwin, non-js builds (e.g. Windows), ModShortcut should be ModCtrl.
	if ModShortcut != ModCtrl {
		t.Errorf("ModShortcut = %v, want ModCtrl on this platform", ModShortcut)
	}
	if ModShortcutAlt != ModCtrl {
		t.Errorf("ModShortcutAlt = %v, want ModCtrl on this platform", ModShortcutAlt)
	}
}
