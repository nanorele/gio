//go:build (linux || freebsd) && !nowayland
// +build linux freebsd
// +build !nowayland

package app

import (
	"testing"

	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/io/system"
)

// The positive path needs a real compositor, because move calls xdg_toplevel_move.
// A window without a topLvl proves the guard returned before reaching C.
func TestWaylandPerformMove(t *testing.T) {
	newWindow := func(btns pointer.Buttons, mode WindowMode) *window {
		seat := &wlSeat{}
		w := &window{disp: &wlDisplay{seat: seat}, pointerBtns: btns}
		w.config.Mode = mode
		seat.pointerFocus = w
		return w
	}
	t.Run("return without move when primary button is not pressed", func(t *testing.T) {
		w := newWindow(0, Windowed)
		w.Perform(system.ActionMove)
		if w.inCompositor {
			t.Fatal("move started without pressed primary button")
		}
	})
	t.Run("return without move when only secondary button is pressed", func(t *testing.T) {
		w := newWindow(pointer.ButtonSecondary, Windowed)
		w.Perform(system.ActionMove)
		if w.inCompositor {
			t.Fatal("move started with secondary button")
		}
	})
	t.Run("return without move when window is maximized", func(t *testing.T) {
		w := newWindow(pointer.ButtonPrimary, Maximized)
		w.Perform(system.ActionMove)
		if w.inCompositor {
			t.Fatal("move started in maximized mode")
		}
	})
	t.Run("return without move when action is not move", func(t *testing.T) {
		w := newWindow(pointer.ButtonPrimary, Windowed)
		w.Perform(system.ActionRaise)
		if w.inCompositor {
			t.Fatal("move started by unrelated action")
		}
	})
}

func TestWaylandSetDecorated(t *testing.T) {
	t.Run("return without set_mode when decor is nil", func(t *testing.T) {
		w := &window{}
		w.setDecorated(Config{Decorated: false}, Config{Decorated: true})
	})
}
