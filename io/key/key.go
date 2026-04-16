package key

import (
	"strings"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/internal/ops"
	"github.com/nanorele/gio/io/event"
	"github.com/nanorele/gio/op"
)

type Filter struct {
	Focus event.Tag

	Required Modifiers

	Optional Modifiers

	Name Name
}

type InputHintOp struct {
	Tag  event.Tag
	Hint InputHint
}

type SoftKeyboardCmd struct {
	Show bool
}

type SelectionCmd struct {
	Tag event.Tag
	Range
	Caret
}

type SnippetCmd struct {
	Tag event.Tag
	Snippet
}

type Range struct {
	Start int
	End   int
}

type Snippet struct {
	Range
	Text string
}

type Caret struct {
	Pos f32.Point

	Ascent float32

	Descent float32
}

type SelectionEvent Range

type SnippetEvent Range

type FocusEvent struct {
	Focus bool
}

type Event struct {
	Name Name

	Modifiers Modifiers

	State State
}

type EditEvent struct {
	Range Range
	Text  string
}

type FocusFilter struct {
	Target event.Tag
}

type InputHint uint8

const (
	HintAny InputHint = iota

	HintText

	HintNumeric

	HintEmail

	HintURL

	HintTelephone

	HintPassword
)

type State uint8

const (
	Press State = iota

	Release
)

type Modifiers uint32

const (
	ModCtrl Modifiers = 1 << iota

	ModCommand

	ModShift

	ModAlt

	ModSuper
)

type Name string

const (
	NameLeftArrow      Name = "←"
	NameRightArrow     Name = "→"
	NameUpArrow        Name = "↑"
	NameDownArrow      Name = "↓"
	NameReturn         Name = "⏎"
	NameEnter          Name = "⌤"
	NameEscape         Name = "⎋"
	NameHome           Name = "⇱"
	NameEnd            Name = "⇲"
	NameDeleteBackward Name = "⌫"
	NameDeleteForward  Name = "⌦"
	NamePageUp         Name = "⇞"
	NamePageDown       Name = "⇟"
	NameTab            Name = "Tab"
	NameSpace          Name = "Space"
	NameCtrl           Name = "Ctrl"
	NameShift          Name = "Shift"
	NameAlt            Name = "Alt"
	NameSuper          Name = "Super"
	NameCommand        Name = "⌘"
	NameF1             Name = "F1"
	NameF2             Name = "F2"
	NameF3             Name = "F3"
	NameF4             Name = "F4"
	NameF5             Name = "F5"
	NameF6             Name = "F6"
	NameF7             Name = "F7"
	NameF8             Name = "F8"
	NameF9             Name = "F9"
	NameF10            Name = "F10"
	NameF11            Name = "F11"
	NameF12            Name = "F12"
	NameBack           Name = "Back"
)

type FocusDirection int

const (
	FocusRight FocusDirection = iota
	FocusLeft
	FocusUp
	FocusDown
	FocusForward
	FocusBackward
)

func (m Modifiers) Contain(m2 Modifiers) bool {
	return m&m2 == m2
}

type FocusCmd struct {
	Tag event.Tag
}

func (h InputHintOp) Add(o *op.Ops) {
	if h.Tag == nil {
		panic("Tag must be non-nil")
	}
	data := ops.Write1(&o.Internal, ops.TypeKeyInputHintLen, h.Tag)
	data[0] = byte(ops.TypeKeyInputHint)
	data[1] = byte(h.Hint)
}

func (EditEvent) ImplementsEvent()      {}
func (Event) ImplementsEvent()          {}
func (FocusEvent) ImplementsEvent()     {}
func (SnippetEvent) ImplementsEvent()   {}
func (SelectionEvent) ImplementsEvent() {}

func (FocusCmd) ImplementsCommand()        {}
func (SoftKeyboardCmd) ImplementsCommand() {}
func (SelectionCmd) ImplementsCommand()    {}
func (SnippetCmd) ImplementsCommand()      {}

func (Filter) ImplementsFilter()      {}
func (FocusFilter) ImplementsFilter() {}

func (m Modifiers) String() string {
	var strs []string
	if m.Contain(ModCtrl) {
		strs = append(strs, string(NameCtrl))
	}
	if m.Contain(ModCommand) {
		strs = append(strs, string(NameCommand))
	}
	if m.Contain(ModShift) {
		strs = append(strs, string(NameShift))
	}
	if m.Contain(ModAlt) {
		strs = append(strs, string(NameAlt))
	}
	if m.Contain(ModSuper) {
		strs = append(strs, string(NameSuper))
	}
	return strings.Join(strs, "-")
}

func (s State) String() string {
	switch s {
	case Press:
		return "Press"
	case Release:
		return "Release"
	default:
		panic("invalid State")
	}
}
