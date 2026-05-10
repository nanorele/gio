// SPDX-License-Identifier: Unlicense OR MIT

package material

import (
	"image"
	"testing"

	"github.com/nanorele/gio/io/system"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/unit"
	"github.com/nanorele/gio/widget"
)

func TestDecorationsConstructor(t *testing.T) {
	th := NewTheme()
	var deco widget.Decorations
	actions := system.ActionMinimize | system.ActionMaximize | system.ActionClose
	d := Decorations(th, &deco, actions, "Window")

	if d.Decorations != &deco {
		t.Errorf("Decorations not propagated")
	}
	if d.Actions != actions {
		t.Errorf("Actions = %v, want %v", d.Actions, actions)
	}
	if d.Background != th.Palette.ContrastBg {
		t.Errorf("Background = %v, want ContrastBg %v", d.Background, th.Palette.ContrastBg)
	}
	if d.Foreground != th.Palette.ContrastFg {
		t.Errorf("Foreground = %v, want ContrastFg %v", d.Foreground, th.Palette.ContrastFg)
	}
	if d.Title.Text != "Window" {
		t.Errorf("Title.Text = %q, want %q", d.Title.Text, "Window")
	}
	if d.Title.Color != th.Palette.ContrastFg {
		t.Errorf("Title.Color = %v, want ContrastFg %v", d.Title.Color, th.Palette.ContrastFg)
	}
	// Title is built from Body1 -> uses theme TextSize.
	if d.Title.TextSize != th.TextSize {
		t.Errorf("Title.TextSize = %v, want %v", d.Title.TextSize, th.TextSize)
	}
}

func TestDecorationsPaletteOverride(t *testing.T) {
	th := NewTheme()
	th.Palette.ContrastBg = rgb(0x010203)
	th.Palette.ContrastFg = rgb(0xfefefe)
	var deco widget.Decorations
	d := Decorations(th, &deco, 0, "")
	if d.Background != th.Palette.ContrastBg {
		t.Errorf("Background did not pick up palette override: %v", d.Background)
	}
	if d.Foreground != th.Palette.ContrastFg {
		t.Errorf("Foreground did not pick up palette override: %v", d.Foreground)
	}
	if d.Title.Color != th.Palette.ContrastFg {
		t.Errorf("Title.Color did not pick up palette override: %v", d.Title.Color)
	}
}

func TestDecorationsNoActionsLayout(t *testing.T) {
	th := NewTheme()
	var deco widget.Decorations
	d := Decorations(th, &deco, 0, "Title")
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(200, 40)),
	}
	dims := d.Layout(gtx)
	if dims.Size.X == 0 || dims.Size.Y == 0 {
		t.Errorf("zero-size layout with no actions: %v", dims.Size)
	}
}

func TestDecorationsIconWidgets(t *testing.T) {
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(40, 40)),
	}
	icons := []layout.Widget{minimizeWindow, maximizeWindow, maximizedWindow, closeWindow}
	for i, w := range icons {
		dims := w(gtx)
		// Each glyph should be square at winIconSize (20dp) with PxPerDp=1.
		if dims.Size.X != 20 || dims.Size.Y != 20 {
			t.Errorf("icon %d size = %v, want {20,20}", i, dims.Size)
		}
	}
}
