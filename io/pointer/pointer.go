package pointer

import (
	"strings"
	"time"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/internal/ops"
	"github.com/nanorele/gio/io/event"
	"github.com/nanorele/gio/io/key"
	"github.com/nanorele/gio/op"
)

type Event struct {
	Kind   Kind
	Source Source

	PointerID ID

	Priority Priority

	Time time.Duration

	Buttons Buttons

	Position f32.Point

	Scroll f32.Point

	Modifiers key.Modifiers
}

type PassOp struct{}

type PassStack struct {
	ops     *ops.Ops
	id      ops.StackID
	macroID uint32
}

type Filter struct {
	Target event.Tag

	Kinds Kind

	ScrollX ScrollRange
	ScrollY ScrollRange
}

type ScrollRange struct {
	Min, Max int
}

type GrabCmd struct {
	Tag event.Tag
	ID  ID
}

type ID uint16

type Kind uint

type Priority uint8

type Source uint8

type Buttons uint8

type Cursor byte

const (
	CursorDefault Cursor = iota

	CursorNone

	CursorText

	CursorVerticalText

	CursorPointer

	CursorCrosshair

	CursorAllScroll

	CursorColResize

	CursorRowResize

	CursorGrab

	CursorGrabbing

	CursorNotAllowed

	CursorWait

	CursorProgress

	CursorNorthWestResize

	CursorNorthEastResize

	CursorSouthWestResize

	CursorSouthEastResize

	CursorNorthSouthResize

	CursorEastWestResize

	CursorWestResize

	CursorEastResize

	CursorNorthResize

	CursorSouthResize

	CursorNorthEastSouthWestResize

	CursorNorthWestSouthEastResize
)

const (
	Cancel Kind = 1 << iota

	Press

	Release

	Move

	Drag

	Enter

	Leave

	Scroll
)

const (
	Mouse Source = iota

	Touch
)

const (
	Shared Priority = iota

	Grabbed
)

const (
	ButtonPrimary Buttons = 1 << iota

	ButtonSecondary

	ButtonTertiary

	ButtonQuaternary

	ButtonQuinary
)

func (s ScrollRange) Union(s2 ScrollRange) ScrollRange {
	return ScrollRange{
		Min: min(s.Min, s2.Min),
		Max: max(s.Max, s2.Max),
	}
}

func (p PassOp) Push(o *op.Ops) PassStack {
	id, mid := ops.PushOp(&o.Internal, ops.PassStack)
	data := ops.Write(&o.Internal, ops.TypePassLen)
	data[0] = byte(ops.TypePass)
	return PassStack{ops: &o.Internal, id: id, macroID: mid}
}

func (p PassStack) Pop() {
	ops.PopOp(p.ops, ops.PassStack, p.id, p.macroID)
	data := ops.Write(p.ops, ops.TypePopPassLen)
	data[0] = byte(ops.TypePopPass)
}

func (op Cursor) Add(o *op.Ops) {
	data := ops.Write(&o.Internal, ops.TypeCursorLen)
	data[0] = byte(ops.TypeCursor)
	data[1] = byte(op)
}

func (t Kind) String() string {
	if t == Cancel {
		return "Cancel"
	}
	var buf strings.Builder
	for tt := Kind(1); tt > 0; tt <<= 1 {
		if t&tt > 0 {
			if buf.Len() > 0 {
				buf.WriteByte('|')
			}
			buf.WriteString((t & tt).string())
		}
	}
	return buf.String()
}

func (t Kind) string() string {
	switch t {
	case Press:
		return "Press"
	case Release:
		return "Release"
	case Cancel:
		return "Cancel"
	case Move:
		return "Move"
	case Drag:
		return "Drag"
	case Enter:
		return "Enter"
	case Leave:
		return "Leave"
	case Scroll:
		return "Scroll"
	default:
		panic("unknown Type")
	}
}

func (p Priority) String() string {
	switch p {
	case Shared:
		return "Shared"
	case Grabbed:
		return "Grabbed"
	default:
		panic("unknown priority")
	}
}

func (s Source) String() string {
	switch s {
	case Mouse:
		return "Mouse"
	case Touch:
		return "Touch"
	default:
		panic("unknown source")
	}
}

func (b Buttons) Contain(buttons Buttons) bool {
	return b&buttons == buttons
}

func (b Buttons) String() string {
	var strs []string
	if b.Contain(ButtonPrimary) {
		strs = append(strs, "ButtonPrimary")
	}
	if b.Contain(ButtonSecondary) {
		strs = append(strs, "ButtonSecondary")
	}
	if b.Contain(ButtonTertiary) {
		strs = append(strs, "ButtonTertiary")
	}
	if b.Contain(ButtonQuaternary) {
		strs = append(strs, "ButtonQuaternary")
	}
	if b.Contain(ButtonQuinary) {
		strs = append(strs, "ButtonQuinary")
	}
	return strings.Join(strs, "|")
}

func (c Cursor) String() string {
	switch c {
	case CursorDefault:
		return "Default"
	case CursorNone:
		return "None"
	case CursorText:
		return "Text"
	case CursorVerticalText:
		return "VerticalText"
	case CursorPointer:
		return "Pointer"
	case CursorCrosshair:
		return "Crosshair"
	case CursorAllScroll:
		return "AllScroll"
	case CursorColResize:
		return "ColResize"
	case CursorRowResize:
		return "RowResize"
	case CursorGrab:
		return "Grab"
	case CursorGrabbing:
		return "Grabbing"
	case CursorNotAllowed:
		return "NotAllowed"
	case CursorWait:
		return "Wait"
	case CursorProgress:
		return "Progress"
	case CursorNorthWestResize:
		return "NorthWestResize"
	case CursorNorthEastResize:
		return "NorthEastResize"
	case CursorSouthWestResize:
		return "SouthWestResize"
	case CursorSouthEastResize:
		return "SouthEastResize"
	case CursorNorthSouthResize:
		return "NorthSouthResize"
	case CursorEastWestResize:
		return "EastWestResize"
	case CursorWestResize:
		return "WestResize"
	case CursorEastResize:
		return "EastResize"
	case CursorNorthResize:
		return "NorthResize"
	case CursorSouthResize:
		return "SouthResize"
	case CursorNorthEastSouthWestResize:
		return "NorthEastSouthWestResize"
	case CursorNorthWestSouthEastResize:
		return "NorthWestSouthEastResize"
	default:
		panic("unknown Type")
	}
}

func (Event) ImplementsEvent() {}

func (GrabCmd) ImplementsCommand() {}

func (Filter) ImplementsFilter() {}
