package pointer

import (
	"testing"
	"time"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/io/key"
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

func TestButtons_ContainEmpty(t *testing.T) {
	var b Buttons
	if !b.Contain(0) {
		t.Error("empty Buttons should contain 0")
	}
	if b.Contain(ButtonPrimary) {
		t.Error("empty Buttons should not contain ButtonPrimary")
	}
	if b.String() != "" {
		t.Errorf("empty Buttons string = %q; want \"\"", b.String())
	}
}

func TestButtons_ContainCombined(t *testing.T) {
	b := ButtonPrimary | ButtonTertiary | ButtonQuinary
	combo := ButtonPrimary | ButtonQuinary
	if !b.Contain(combo) {
		t.Errorf("%s should contain %s", b, combo)
	}
	if b.Contain(ButtonPrimary | ButtonSecondary) {
		t.Errorf("%s should not contain %s", b, ButtonPrimary|ButtonSecondary)
	}
	if !b.Contain(ButtonTertiary) {
		t.Errorf("%s should contain ButtonTertiary", b)
	}
}

func TestButtons_AddRemoveBits(t *testing.T) {
	var b Buttons
	b |= ButtonPrimary
	if !b.Contain(ButtonPrimary) {
		t.Error("after OR, ButtonPrimary not contained")
	}
	b |= ButtonSecondary
	if !b.Contain(ButtonPrimary | ButtonSecondary) {
		t.Error("after OR ButtonSecondary, not all contained")
	}
	b &^= ButtonPrimary
	if b.Contain(ButtonPrimary) {
		t.Error("after AND-NOT, ButtonPrimary still contained")
	}
	if !b.Contain(ButtonSecondary) {
		t.Error("after AND-NOT, ButtonSecondary missing")
	}
}

func TestButtons_AllOnly(t *testing.T) {
	all := ButtonPrimary | ButtonSecondary | ButtonTertiary | ButtonQuaternary | ButtonQuinary
	for _, b := range []Buttons{ButtonPrimary, ButtonSecondary, ButtonTertiary, ButtonQuaternary, ButtonQuinary} {
		if !all.Contain(b) {
			t.Errorf("all should contain %s", b)
		}
	}
}

func TestButtons_StringSingle(t *testing.T) {
	for _, tc := range []struct {
		b    Buttons
		want string
	}{
		{ButtonPrimary, "ButtonPrimary"},
		{ButtonSecondary, "ButtonSecondary"},
		{ButtonTertiary, "ButtonTertiary"},
		{ButtonQuaternary, "ButtonQuaternary"},
		{ButtonQuinary, "ButtonQuinary"},
	} {
		if got := tc.b.String(); got != tc.want {
			t.Errorf("got %q; want %q", got, tc.want)
		}
	}
}

func TestKind_StringSingleBits(t *testing.T) {
	for _, tc := range []struct {
		k    Kind
		want string
	}{
		{Press, "Press"},
		{Release, "Release"},
		{Move, "Move"},
		{Drag, "Drag"},
		{Enter, "Enter"},
		{Leave, "Leave"},
		{Scroll, "Scroll"},
	} {
		if got := tc.k.String(); got != tc.want {
			t.Errorf("Kind(%d).String() = %q; want %q", tc.k, got, tc.want)
		}
	}
}

func TestKind_StringAllSet(t *testing.T) {
	all := Cancel | Press | Release | Move | Drag | Enter | Leave | Scroll
	got := all.String()
	want := "Cancel|Press|Release|Move|Drag|Enter|Leave|Scroll"
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestKind_StringZero(t *testing.T) {
	if got := Kind(0).String(); got != "" {
		t.Errorf("Kind(0).String() = %q; want empty", got)
	}
}

func TestScrollRange_UnionEdges(t *testing.T) {
	for _, tc := range []struct {
		a, b, want ScrollRange
	}{
		{ScrollRange{0, 0}, ScrollRange{0, 0}, ScrollRange{0, 0}},
		{ScrollRange{1, 5}, ScrollRange{1, 5}, ScrollRange{1, 5}},
		{ScrollRange{-100, -50}, ScrollRange{50, 100}, ScrollRange{-100, 100}},
		{ScrollRange{5, 10}, ScrollRange{1, 3}, ScrollRange{1, 10}},
	} {
		got := tc.a.Union(tc.b)
		if got != tc.want {
			t.Errorf("Union(%+v, %+v) = %+v; want %+v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestFilterConstruction(t *testing.T) {
	tag := new(int)
	f := Filter{
		Target:  tag,
		Kinds:   Press | Release,
		ScrollX: ScrollRange{Min: -10, Max: 10},
		ScrollY: ScrollRange{Min: -20, Max: 20},
	}
	if f.Target != tag {
		t.Errorf("Target = %v; want %v", f.Target, tag)
	}
	if f.Kinds != Press|Release {
		t.Errorf("Kinds = %v; want Press|Release", f.Kinds)
	}
	if f.ScrollX != (ScrollRange{Min: -10, Max: 10}) {
		t.Errorf("ScrollX = %+v", f.ScrollX)
	}
	if f.ScrollY != (ScrollRange{Min: -20, Max: 20}) {
		t.Errorf("ScrollY = %+v", f.ScrollY)
	}
}

func TestEventStruct(t *testing.T) {
	e := Event{
		Kind:      Press,
		Source:    Mouse,
		PointerID: ID(7),
		Priority:  Grabbed,
		Time:      100 * time.Millisecond,
		Buttons:   ButtonPrimary,
		Position:  f32.Pt(1, 2),
		Scroll:    f32.Pt(3, 4),
		Modifiers: key.ModCtrl,
	}
	if e.Kind != Press || e.Source != Mouse || e.PointerID != 7 ||
		e.Priority != Grabbed || e.Time != 100*time.Millisecond ||
		e.Buttons != ButtonPrimary || e.Position != f32.Pt(1, 2) ||
		e.Scroll != f32.Pt(3, 4) || e.Modifiers != key.ModCtrl {
		t.Errorf("Event field round-trip failed: %+v", e)
	}
}

func TestGrabCmd(t *testing.T) {
	tag := new(int)
	g := GrabCmd{Tag: tag, ID: ID(42)}
	if g.Tag != tag || g.ID != 42 {
		t.Errorf("GrabCmd round-trip failed: %+v", g)
	}
}

func TestPassOp_PushPopMultiple(t *testing.T) {
	var ops op.Ops
	s1 := PassOp{}.Push(&ops)
	s2 := PassOp{}.Push(&ops)
	s2.Pop()
	s1.Pop()
}

func TestCursor_AddAll(t *testing.T) {
	var ops op.Ops
	for c := CursorDefault; c <= CursorNorthWestSouthEastResize; c++ {
		c.Add(&ops)
	}
}

func TestKind_PanicOnUnknownSingleBit(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic on unknown Kind bit")
		}
	}()
	// Set a bit beyond the defined Scroll bit.
	_ = (Kind(1 << 10)).String()
}
