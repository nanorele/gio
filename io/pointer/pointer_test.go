package pointer

import (
	"testing"

	"github.com/nanorele/gio/op"
)

func TestTypeString(t *testing.T) {
	for _, tc := range []struct {
		typ Kind
		res string
	}{
		{Cancel, "Cancel"},
		{Press, "Press"},
		{Release, "Release"},
		{Move, "Move"},
		{Drag, "Drag"},
		{Enter, "Enter"},
		{Leave, "Leave"},
		{Scroll, "Scroll"},
		{Enter | Leave, "Enter|Leave"},
		{Press | Release, "Press|Release"},
		{Enter | Leave | Press | Release, "Press|Release|Enter|Leave"},
		{Move | Scroll, "Move|Scroll"},
		{Cancel | Press, "Cancel|Press"},
	} {
		t.Run(tc.res, func(t *testing.T) {
			if want, got := tc.res, tc.typ.String(); want != got {
				t.Errorf("got %q; want %q", got, want)
			}
		})
	}
}

func TestPanicStrings(t *testing.T) {
	checkPanic := func(name string, f func()) {
		defer func() {
			if recover() == nil {
				t.Errorf("%s: expected panic", name)
			}
		}()
		f()
	}

	checkPanic("Kind.String()", func() { _ = (Kind(0x1000)).String() })
	checkPanic("Priority.String()", func() { _ = (Priority(100)).String() })
	checkPanic("Source.String()", func() { _ = (Source(100)).String() })
	checkPanic("Cursor.String()", func() { _ = (Cursor(100)).String() })
}

func TestScrollRange_Union(t *testing.T) {
	s1 := ScrollRange{Min: -10, Max: 5}
	s2 := ScrollRange{Min: -5, Max: 10}
	got := s1.Union(s2)
	want := ScrollRange{Min: -10, Max: 10}
	if got != want {
		t.Errorf("got %+v; want %+v", got, want)
	}
}

func TestPriority_String(t *testing.T) {
	for _, tc := range []struct {
		p   Priority
		res string
	}{
		{Shared, "Shared"},
		{Grabbed, "Grabbed"},
	} {
		if got, want := tc.p.String(), tc.res; got != want {
			t.Errorf("got %q; want %q", got, want)
		}
	}
}

func TestSource_String(t *testing.T) {
	for _, tc := range []struct {
		s   Source
		res string
	}{
		{Mouse, "Mouse"},
		{Touch, "Touch"},
	} {
		if got, want := tc.s.String(), tc.res; got != want {
			t.Errorf("got %q; want %q", got, want)
		}
	}
}

func TestButtons(t *testing.T) {
	b := ButtonPrimary | ButtonSecondary
	if !b.Contain(ButtonPrimary) {
		t.Error("expected ButtonPrimary to be contained")
	}
	if !b.Contain(ButtonSecondary) {
		t.Error("expected ButtonSecondary to be contained")
	}
	if b.Contain(ButtonTertiary) {
		t.Error("did not expect ButtonTertiary to be contained")
	}
	got := b.String()
	want := "ButtonPrimary|ButtonSecondary"
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
	all := ButtonPrimary | ButtonSecondary | ButtonTertiary | ButtonQuaternary | ButtonQuinary
	gotAll := all.String()
	wantAll := "ButtonPrimary|ButtonSecondary|ButtonTertiary|ButtonQuaternary|ButtonQuinary"
	if gotAll != wantAll {
		t.Errorf("got %q; want %q", gotAll, wantAll)
	}
}

func TestCursor_String(t *testing.T) {
	for _, tc := range []struct {
		c   Cursor
		res string
	}{
		{CursorDefault, "Default"},
		{CursorNone, "None"},
		{CursorText, "Text"},
		{CursorVerticalText, "VerticalText"},
		{CursorPointer, "Pointer"},
		{CursorCrosshair, "Crosshair"},
		{CursorAllScroll, "AllScroll"},
		{CursorColResize, "ColResize"},
		{CursorRowResize, "RowResize"},
		{CursorGrab, "Grab"},
		{CursorGrabbing, "Grabbing"},
		{CursorNotAllowed, "NotAllowed"},
		{CursorWait, "Wait"},
		{CursorProgress, "Progress"},
		{CursorNorthWestResize, "NorthWestResize"},
		{CursorNorthEastResize, "NorthEastResize"},
		{CursorSouthWestResize, "SouthWestResize"},
		{CursorSouthEastResize, "SouthEastResize"},
		{CursorNorthSouthResize, "NorthSouthResize"},
		{CursorEastWestResize, "EastWestResize"},
		{CursorWestResize, "WestResize"},
		{CursorEastResize, "EastResize"},
		{CursorNorthResize, "NorthResize"},
		{CursorSouthResize, "SouthResize"},
		{CursorNorthEastSouthWestResize, "NorthEastSouthWestResize"},
		{CursorNorthWestSouthEastResize, "NorthWestSouthEastResize"},
	} {
		if got, want := tc.c.String(), tc.res; got != want {
			t.Errorf("got %q; want %q", got, want)
		}
	}
}

func TestImplements(t *testing.T) {
	(Event{}).ImplementsEvent()
	(GrabCmd{}).ImplementsCommand()
	(Filter{}).ImplementsFilter()
}

func TestOps(t *testing.T) {
	var ops op.Ops
	PassOp{}.Push(&ops).Pop()
	CursorPointer.Add(&ops)
}
