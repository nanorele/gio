//go:build linux || freebsd || openbsd

package app

import (
	"errors"
	"unsafe"

	"github.com/nanorele/gio/io/pointer"
)

type X11ViewEvent struct {
	Display unsafe.Pointer

	Window uintptr
}

func (X11ViewEvent) implementsViewEvent() {}
func (X11ViewEvent) ImplementsEvent()     {}
func (x X11ViewEvent) Valid() bool {
	return x != (X11ViewEvent{})
}

type WaylandViewEvent struct {
	Display unsafe.Pointer

	Surface unsafe.Pointer
}

func (WaylandViewEvent) implementsViewEvent() {}
func (WaylandViewEvent) ImplementsEvent()     {}
func (w WaylandViewEvent) Valid() bool {
	return w != (WaylandViewEvent{})
}

func osMain() {
	select {}
}

type windowDriver func(*callbacks, []Option) error

var wlDriver, x11Driver windowDriver

func newWindow(window *callbacks, options []Option) {
	var errFirst error
	for _, d := range []windowDriver{wlDriver, x11Driver} {
		if d == nil {
			continue
		}
		err := d(window, options)
		if err == nil {
			return
		}
		if errFirst == nil {
			errFirst = err
		}
	}
	if errFirst == nil {
		errFirst = errors.New("app: no window driver available")
	}
	window.ProcessEvent(DestroyEvent{Err: errFirst})
}

var xCursor = [...]string{
	pointer.CursorDefault:                  "left_ptr",
	pointer.CursorNone:                     "",
	pointer.CursorText:                     "xterm",
	pointer.CursorVerticalText:             "vertical-text",
	pointer.CursorPointer:                  "hand2",
	pointer.CursorCrosshair:                "crosshair",
	pointer.CursorAllScroll:                "fleur",
	pointer.CursorColResize:                "sb_h_double_arrow",
	pointer.CursorRowResize:                "sb_v_double_arrow",
	pointer.CursorGrab:                     "hand1",
	pointer.CursorGrabbing:                 "move",
	pointer.CursorNotAllowed:               "crossed_circle",
	pointer.CursorWait:                     "watch",
	pointer.CursorProgress:                 "left_ptr_watch",
	pointer.CursorNorthWestResize:          "top_left_corner",
	pointer.CursorNorthEastResize:          "top_right_corner",
	pointer.CursorSouthWestResize:          "bottom_left_corner",
	pointer.CursorSouthEastResize:          "bottom_right_corner",
	pointer.CursorNorthSouthResize:         "sb_v_double_arrow",
	pointer.CursorEastWestResize:           "sb_h_double_arrow",
	pointer.CursorWestResize:               "left_side",
	pointer.CursorEastResize:               "right_side",
	pointer.CursorNorthResize:              "top_side",
	pointer.CursorSouthResize:              "bottom_side",
	pointer.CursorNorthEastSouthWestResize: "fd_double_arrow",
	pointer.CursorNorthWestSouthEastResize: "bd_double_arrow",
}
