package widget

import (
	"image"
	"io"
	"math"
	"strings"

	"github.com/nanorele/gio/font"
	"github.com/nanorele/gio/gesture"
	"github.com/nanorele/gio/io/clipboard"
	"github.com/nanorele/gio/io/event"
	"github.com/nanorele/gio/io/key"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/io/system"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/op/clip"
	"github.com/nanorele/gio/text"
	"github.com/nanorele/gio/unit"
)

type stringSource struct {
	reader *strings.Reader
}

var _ textSource = stringSource{}

func newStringSource(str string) stringSource {
	return stringSource{
		reader: strings.NewReader(str),
	}
}

func (s stringSource) Changed() bool {
	return false
}

func (s stringSource) Size() int64 {
	return s.reader.Size()
}

func (s stringSource) ReadAt(b []byte, offset int64) (int, error) {
	return s.reader.ReadAt(b, offset)
}

func (s stringSource) ReplaceRunes(byteOffset, runeCount int64, str string) {
}

type Selectable struct {
	Alignment text.Alignment

	MaxLines int

	Truncator string

	WrapPolicy text.WrapPolicy

	LineHeight unit.Sp

	LineHeightScale float32
	initialized     bool
	source          stringSource

	scratch   []byte
	lastValue string
	text      textView
	focused   bool
	dragging  bool
	dragger   gesture.Drag

	clicker gesture.Click
}

func (l *Selectable) initialize() {
	if !l.initialized {
		l.source = newStringSource("")
		l.text.SetSource(l.source)
		l.initialized = true
	}
}

func (l *Selectable) Focused() bool {
	return l.focused
}

func (l *Selectable) paintSelection(gtx layout.Context, material op.CallOp) {
	l.initialize()
	if !l.focused {
		return
	}
	l.text.PaintSelection(gtx, material)
}

func (l *Selectable) paintText(gtx layout.Context, material op.CallOp) {
	l.initialize()
	l.text.PaintText(gtx, material)
}

func (l *Selectable) SelectionLen() int {
	l.initialize()
	return l.text.SelectionLen()
}

func (l *Selectable) Selection() (start, end int) {
	l.initialize()
	return l.text.Selection()
}

func (l *Selectable) SetCaret(start, end int) {
	l.initialize()
	l.text.SetCaret(start, end)
}

func (l *Selectable) SelectedText() string {
	l.initialize()
	l.scratch = l.text.SelectedText(l.scratch)
	return string(l.scratch)
}

func (l *Selectable) ClearSelection() {
	l.initialize()
	l.text.ClearSelection()
}

func (l *Selectable) Text() string {
	l.initialize()
	l.scratch = l.text.Text(l.scratch)
	return string(l.scratch)
}

func (l *Selectable) SetText(s string) {
	l.initialize()
	if l.lastValue != s {
		l.source = newStringSource(s)
		l.lastValue = s
		l.text.SetSource(l.source)
	}
}

func (l *Selectable) Truncated() bool {
	return l.text.Truncated()
}

func (l *Selectable) Update(gtx layout.Context) bool {
	l.initialize()
	return l.handleEvents(gtx)
}

func (l *Selectable) Layout(gtx layout.Context, lt *text.Shaper, font font.Font, size unit.Sp, textMaterial, selectionMaterial op.CallOp) layout.Dimensions {
	l.Update(gtx)
	l.text.LineHeight = l.LineHeight
	l.text.LineHeightScale = l.LineHeightScale
	l.text.Alignment = l.Alignment
	l.text.MaxLines = l.MaxLines
	l.text.Truncator = l.Truncator
	l.text.WrapPolicy = l.WrapPolicy
	l.text.Layout(gtx, lt, font, size)
	dims := l.text.Dimensions()
	defer clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
	pointer.CursorText.Add(gtx.Ops)
	event.Op(gtx.Ops, l)

	l.clicker.Add(gtx.Ops)
	l.dragger.Add(gtx.Ops)

	l.paintSelection(gtx, selectionMaterial)
	l.paintText(gtx, textMaterial)
	return dims
}

func (l *Selectable) handleEvents(gtx layout.Context) (selectionChanged bool) {
	oldStart, oldLen := min(l.text.Selection()), l.text.SelectionLen()
	defer func() {
		if newStart, newLen := min(l.text.Selection()), l.text.SelectionLen(); oldStart != newStart || oldLen != newLen {
			selectionChanged = true
		}
	}()
	l.processPointer(gtx)
	l.processKey(gtx)
	return selectionChanged
}

func (e *Selectable) processPointer(gtx layout.Context) {
	for _, evt := range e.clickDragEvents(gtx) {
		switch evt := evt.(type) {
		case gesture.ClickEvent:
			switch {
			case evt.Kind == gesture.KindPress && evt.Source == pointer.Mouse,
				evt.Kind == gesture.KindClick && evt.Source != pointer.Mouse:
				prevCaretPos, _ := e.text.Selection()
				e.text.MoveCoord(image.Point{
					X: int(math.Round(float64(evt.Position.X))),
					Y: int(math.Round(float64(evt.Position.Y))),
				})
				gtx.Execute(key.FocusCmd{Tag: e})
				if evt.Modifiers == key.ModShift {
					start, end := e.text.Selection()

					if abs(end-start) < abs(start-prevCaretPos) {
						e.text.SetCaret(start, prevCaretPos)
					}
				} else {
					e.text.ClearSelection()
				}
				e.dragging = true

				switch {
				case evt.NumClicks == 2:
					e.text.MoveWord(-1, selectionClear)
					e.text.MoveWord(1, selectionExtend)
					e.dragging = false
				case evt.NumClicks >= 3:
					e.text.MoveLineStart(selectionClear)
					e.text.MoveLineEnd(selectionExtend)
					e.dragging = false
				}
			}
		case pointer.Event:
			release := false
			switch {
			case evt.Kind == pointer.Release && evt.Source == pointer.Mouse:
				release = true
				fallthrough
			case evt.Kind == pointer.Drag && evt.Source == pointer.Mouse:
				if e.dragging {
					e.text.MoveCoord(image.Point{
						X: int(math.Round(float64(evt.Position.X))),
						Y: int(math.Round(float64(evt.Position.Y))),
					})

					if release {
						e.dragging = false
					}
				}
			}
		}
	}
}

func (e *Selectable) clickDragEvents(gtx layout.Context) []event.Event {
	var combinedEvents []event.Event
	for {
		evt, ok := e.clicker.Update(gtx.Source)
		if !ok {
			break
		}
		combinedEvents = append(combinedEvents, evt)
	}
	for {
		evt, ok := e.dragger.Update(gtx.Metric, gtx.Source, gesture.Both)
		if !ok {
			break
		}
		combinedEvents = append(combinedEvents, evt)
	}
	return combinedEvents
}

func (e *Selectable) processKey(gtx layout.Context) {
	for {
		ke, ok := gtx.Event(
			key.FocusFilter{Target: e},
			key.Filter{Focus: e, Name: key.NameLeftArrow, Optional: key.ModShortcutAlt | key.ModShift},
			key.Filter{Focus: e, Name: key.NameRightArrow, Optional: key.ModShortcutAlt | key.ModShift},
			key.Filter{Focus: e, Name: key.NameUpArrow, Optional: key.ModShortcutAlt | key.ModShift},
			key.Filter{Focus: e, Name: key.NameDownArrow, Optional: key.ModShortcutAlt | key.ModShift},

			key.Filter{Focus: e, Name: key.NamePageUp, Optional: key.ModShift},
			key.Filter{Focus: e, Name: key.NamePageDown, Optional: key.ModShift},
			key.Filter{Focus: e, Name: key.NameEnd, Optional: key.ModShift},
			key.Filter{Focus: e, Name: key.NameHome, Optional: key.ModShift},

			key.Filter{Focus: e, Name: "C", Required: key.ModShortcut},
			key.Filter{Focus: e, Name: "X", Required: key.ModShortcut},
			key.Filter{Focus: e, Name: "A", Required: key.ModShortcut},
		)
		if !ok {
			break
		}
		switch ke := ke.(type) {
		case key.FocusEvent:
			e.focused = ke.Focus
		case key.Event:
			if !e.focused || ke.State != key.Press {
				break
			}
			e.command(gtx, ke)
		}
	}
}

func (e *Selectable) command(gtx layout.Context, k key.Event) {
	direction := 1
	if gtx.Locale.Direction.Progression() == system.TowardOrigin {
		direction = -1
	}
	moveByWord := k.Modifiers.Contain(key.ModShortcutAlt)
	selAct := selectionClear
	if k.Modifiers.Contain(key.ModShift) {
		selAct = selectionExtend
	}
	if k.Modifiers == key.ModShortcut {
		switch k.Name {

		case "C", "X":
			e.scratch = e.text.SelectedText(e.scratch)
			if text := string(e.scratch); text != "" {
				gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(text))})
			}

		case "A":
			e.text.SetCaret(0, e.text.Len())
		}
		return
	}
	switch k.Name {
	case key.NameUpArrow:
		e.text.MoveLines(-1, selAct)
	case key.NameDownArrow:
		e.text.MoveLines(+1, selAct)
	case key.NameLeftArrow:
		if moveByWord {
			e.text.MoveWord(-1*direction, selAct)
		} else {
			if selAct == selectionClear {
				e.text.ClearSelection()
			}
			e.text.MoveCaret(-1*direction, -1*direction*int(selAct))
		}
	case key.NameRightArrow:
		if moveByWord {
			e.text.MoveWord(1*direction, selAct)
		} else {
			if selAct == selectionClear {
				e.text.ClearSelection()
			}
			e.text.MoveCaret(1*direction, int(selAct)*direction)
		}
	case key.NamePageUp:
		e.text.MovePages(-1, selAct)
	case key.NamePageDown:
		e.text.MovePages(+1, selAct)
	case key.NameHome:
		e.text.MoveLineStart(selAct)
	case key.NameEnd:
		e.text.MoveLineEnd(selAct)
	}
}

func (l *Selectable) Regions(start, end int, regions []Region) []Region {
	l.initialize()
	return l.text.Regions(start, end, regions)
}
