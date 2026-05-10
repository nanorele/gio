// SPDX-License-Identifier: Unlicense OR MIT

package material

import (
	"testing"

	"github.com/nanorele/gio/widget"
)

func TestCheckBoxConstructor(t *testing.T) {
	th := NewTheme()
	var b widget.Bool
	c := CheckBox(th, &b, "Label")

	if c.CheckBox != &b {
		t.Errorf("CheckBox not propagated")
	}
	if c.Label != "Label" {
		t.Errorf("Label = %q, want %q", c.Label, "Label")
	}
	if c.Color != th.Palette.Fg {
		t.Errorf("Color = %v, want Fg %v", c.Color, th.Palette.Fg)
	}
	if c.IconColor != th.Palette.ContrastBg {
		t.Errorf("IconColor = %v, want ContrastBg %v", c.IconColor, th.Palette.ContrastBg)
	}
	wantSize := th.TextSize * 14.0 / 16.0
	if c.TextSize != wantSize {
		t.Errorf("TextSize = %v, want %v", c.TextSize, wantSize)
	}
	if c.Size != 26 {
		t.Errorf("Size = %v, want 26", c.Size)
	}
	if c.shaper != th.Shaper {
		t.Errorf("shaper not propagated")
	}
	if c.checkedStateIcon != th.Icon.CheckBoxChecked {
		t.Errorf("checkedStateIcon mismatch")
	}
	if c.uncheckedStateIcon != th.Icon.CheckBoxUnchecked {
		t.Errorf("uncheckedStateIcon mismatch")
	}
	if c.Font.Typeface != th.Face {
		t.Errorf("Font.Typeface = %v, want %v", c.Font.Typeface, th.Face)
	}
}

func TestRadioButtonConstructor(t *testing.T) {
	th := NewTheme()
	var g widget.Enum
	r := RadioButton(th, &g, "key", "Label")

	if r.Group != &g {
		t.Errorf("Group not propagated")
	}
	if r.Key != "key" {
		t.Errorf("Key = %q, want %q", r.Key, "key")
	}
	if r.Label != "Label" {
		t.Errorf("Label = %q, want %q", r.Label, "Label")
	}
	if r.Color != th.Palette.Fg {
		t.Errorf("Color = %v, want Fg %v", r.Color, th.Palette.Fg)
	}
	if r.IconColor != th.Palette.ContrastBg {
		t.Errorf("IconColor = %v, want ContrastBg %v", r.IconColor, th.Palette.ContrastBg)
	}
	wantSize := th.TextSize * 14.0 / 16.0
	if r.TextSize != wantSize {
		t.Errorf("TextSize = %v, want %v", r.TextSize, wantSize)
	}
	if r.Size != 26 {
		t.Errorf("Size = %v, want 26", r.Size)
	}
	if r.shaper != th.Shaper {
		t.Errorf("shaper not propagated")
	}
	if r.checkedStateIcon != th.Icon.RadioChecked {
		t.Errorf("checkedStateIcon mismatch")
	}
	if r.uncheckedStateIcon != th.Icon.RadioUnchecked {
		t.Errorf("uncheckedStateIcon mismatch")
	}
	if r.Font.Typeface != th.Face {
		t.Errorf("Font.Typeface = %v, want %v", r.Font.Typeface, th.Face)
	}
}

func TestCheckBoxPaletteOverride(t *testing.T) {
	th := NewTheme()
	th.Palette.Fg = rgb(0xaabbcc)
	th.Palette.ContrastBg = rgb(0x112233)
	var b widget.Bool
	c := CheckBox(th, &b, "x")
	if c.Color != th.Palette.Fg {
		t.Errorf("Color did not pick up updated Fg: %v", c.Color)
	}
	if c.IconColor != th.Palette.ContrastBg {
		t.Errorf("IconColor did not pick up updated ContrastBg: %v", c.IconColor)
	}
}
