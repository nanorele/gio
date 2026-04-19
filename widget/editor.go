package widget

import (
	"bufio"
	"image"
	"io"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/font"
	"github.com/nanorele/gio/gesture"
	"github.com/nanorele/gio/io/clipboard"
	"github.com/nanorele/gio/io/event"
	"github.com/nanorele/gio/io/key"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/io/semantic"
	"github.com/nanorele/gio/io/system"
	"github.com/nanorele/gio/io/transfer"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/op/clip"
	"github.com/nanorele/gio/text"
	"github.com/nanorele/gio/unit"
)

type Editor struct {
	text textView

	Alignment text.Alignment

	LineHeight unit.Sp

	LineHeightScale float32

	SingleLine bool

	ReadOnly bool

	Submit bool

	Mask rune

	InputHint key.InputHint

	MaxLen int

	Filter string

	WrapPolicy text.WrapPolicy

	buffer *editBuffer

	scratch    []byte
	blinkStart time.Time

	ime struct {
		imeState
		scratch []byte
	}

	dragging    bool
	dragger     gesture.Drag
	scroller    gesture.Scroll
	scrollCaret bool
	showCaret   bool

	clicker gesture.Click

	history []modification

	historyScratch []rune

	nextHistoryIdx int

	pending []EditorEvent
}

type offEntry struct {
	runes int
	bytes int
}

type imeState struct {
	selection struct {
		rng   key.Range
		caret key.Caret
	}
	snippet    key.Snippet
	start, end int
}

type maskReader struct {
	rr      io.RuneReader
	maskBuf [utf8.UTFMax]byte

	mask []byte

	overflow []byte
}

type selectionAction int

const (
	selectionExtend selectionAction = iota
	selectionClear
)

func (m *maskReader) Reset(r io.Reader, mr rune) {
	m.rr = bufio.NewReader(r)
	n := utf8.EncodeRune(m.maskBuf[:], mr)
	m.mask = m.maskBuf[:n]
}

func (m *maskReader) Read(b []byte) (n int, err error) {
	for len(b) > 0 {
		var replacement []byte
		if len(m.overflow) > 0 {
			replacement = m.overflow
		} else {
			var r rune
			r, _, err = m.rr.ReadRune()
			if err != nil {
				break
			}
			if r == '\n' {
				replacement = []byte{'\n'}
			} else {
				replacement = m.mask
			}
		}
		nn := copy(b, replacement)
		m.overflow = replacement[nn:]
		n += nn
		b = b[nn:]
	}
	return n, err
}

type EditorEvent interface {
	isEditorEvent()
}

type ChangeEvent struct{}

type SubmitEvent struct {
	Text string
}

type SelectEvent struct{}

const (
	blinksPerSecond  = 1
	maxBlinkDuration = 10 * time.Second
)

func (e *Editor) processEvents(gtx layout.Context) (ev EditorEvent, ok bool) {
	if len(e.pending) > 0 {
		out := e.pending[0]
		e.pending = e.pending[:copy(e.pending, e.pending[1:])]
		return out, true
	}
	selStart, selEnd := e.Selection()
	defer func() {
		afterSelStart, afterSelEnd := e.Selection()
		if selStart != afterSelStart || selEnd != afterSelEnd {
			if ok {
				e.pending = append(e.pending, SelectEvent{})
			} else {
				ev = SelectEvent{}
				ok = true
			}
		}
	}()

	ev, ok = e.processPointer(gtx)
	if ok {
		return ev, ok
	}
	ev, ok = e.processKey(gtx)
	if ok {
		return ev, ok
	}
	return nil, false
}

func (e *Editor) processPointer(gtx layout.Context) (EditorEvent, bool) {
	sbounds := e.text.ScrollBounds()
	var smin, smax int
	var axis gesture.Axis
	if e.SingleLine {
		axis = gesture.Horizontal
		smin, smax = sbounds.Min.X, sbounds.Max.X
	} else {
		axis = gesture.Vertical
		smin, smax = sbounds.Min.Y, sbounds.Max.Y
	}
	var scrollX, scrollY pointer.ScrollRange
	textDims := e.text.FullDimensions()

	visibleDims := e.text.Dimensions()
	visibleDims.Size = gtx.Constraints.Constrain(visibleDims.Size)

	if e.SingleLine {
		scrollOffX := e.text.ScrollOff().X
		scrollX.Min = min(-scrollOffX, 0)
		scrollX.Max = max(0, textDims.Size.X-(scrollOffX+visibleDims.Size.X))
	} else {
		scrollOffY := e.text.ScrollOff().Y
		scrollY.Min = -scrollOffY
		scrollY.Max = max(0, textDims.Size.Y-(scrollOffY+visibleDims.Size.Y))
	}
	sdist := e.scroller.Update(gtx.Metric, gtx.Source, gtx.Now, axis, scrollX, scrollY)
	var soff int
	if e.SingleLine {
		e.text.ScrollRel(sdist, 0)
		soff = e.text.ScrollOff().X
	} else {
		e.text.ScrollRel(0, sdist)
		soff = e.text.ScrollOff().Y
	}
	for {
		evt, ok := e.clicker.Update(gtx.Source)
		if !ok {
			break
		}
		ev, ok := e.processPointerEvent(gtx, evt)
		if ok {
			return ev, ok
		}
	}
	for {
		evt, ok := e.dragger.Update(gtx.Metric, gtx.Source, gesture.Both)
		if !ok {
			break
		}
		ev, ok := e.processPointerEvent(gtx, evt)
		if ok {
			return ev, ok
		}
	}
	if (sdist > 0 && soff >= smax) || (sdist < 0 && soff <= smin) {
		e.scroller.Stop()
	}
	return nil, false
}

func (e *Editor) processPointerEvent(gtx layout.Context, ev event.Event) (EditorEvent, bool) {
	switch evt := ev.(type) {
	case gesture.ClickEvent:
		switch {
		case evt.Kind == gesture.KindPress && evt.Source == pointer.Mouse,
			evt.Kind == gesture.KindClick && evt.Source != pointer.Mouse:
			prevCaretPos, _ := e.text.Selection()
			e.blinkStart = gtx.Now
			e.text.MoveCoord(image.Point{
				X: int(math.Round(float64(evt.Position.X))),
				Y: int(math.Round(float64(evt.Position.Y))),
			})
			gtx.Execute(key.FocusCmd{Tag: e})
			if !e.ReadOnly {
				gtx.Execute(key.SoftKeyboardCmd{Show: true})
			}
			if e.scroller.State() != gesture.StateFlinging {
				e.scrollCaret = true
			}

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
				e.blinkStart = gtx.Now
				e.text.MoveCoord(image.Point{
					X: int(math.Round(float64(evt.Position.X))),
					Y: int(math.Round(float64(evt.Position.Y))),
				})
				e.scrollCaret = true

				if release {
					e.dragging = false
				}
			}
		}
	}
	return nil, false
}

func (e *Editor) processKey(gtx layout.Context) (EditorEvent, bool) {
	if e.text.Changed() {
		return ChangeEvent{}, true
	}
	caret, _ := e.text.Selection()
	atBeginning := caret == 0
	atEnd := caret == e.text.Len()
	if gtx.Locale.Direction.Progression() != system.FromOrigin {
		atEnd, atBeginning = atBeginning, atEnd
	}
	var filtersBuf [20]event.Filter
	n := 0
	filtersBuf[n] = key.FocusFilter{Target: e}
	n++
	filtersBuf[n] = transfer.TargetFilter{Target: e, Type: "application/text"}
	n++
	filtersBuf[n] = key.Filter{Focus: e, Name: key.NameEnter, Optional: key.ModShift}
	n++
	filtersBuf[n] = key.Filter{Focus: e, Name: key.NameReturn, Optional: key.ModShift}
	n++
	filtersBuf[n] = key.Filter{Focus: e, Name: "Z", Required: key.ModShortcut, Optional: key.ModShift}
	n++
	filtersBuf[n] = key.Filter{Focus: e, Name: "C", Required: key.ModShortcut}
	n++
	filtersBuf[n] = key.Filter{Focus: e, Name: "V", Required: key.ModShortcut}
	n++
	filtersBuf[n] = key.Filter{Focus: e, Name: "X", Required: key.ModShortcut}
	n++
	filtersBuf[n] = key.Filter{Focus: e, Name: "A", Required: key.ModShortcut}
	n++
	filtersBuf[n] = key.Filter{Focus: e, Name: key.NameDeleteBackward, Optional: key.ModShortcutAlt | key.ModShift}
	n++
	filtersBuf[n] = key.Filter{Focus: e, Name: key.NameDeleteForward, Optional: key.ModShortcutAlt | key.ModShift}
	n++
	filtersBuf[n] = key.Filter{Focus: e, Name: key.NameHome, Optional: key.ModShortcut | key.ModShift}
	n++
	filtersBuf[n] = key.Filter{Focus: e, Name: key.NameEnd, Optional: key.ModShortcut | key.ModShift}
	n++
	filtersBuf[n] = key.Filter{Focus: e, Name: key.NamePageDown, Optional: key.ModShift}
	n++
	filtersBuf[n] = key.Filter{Focus: e, Name: key.NamePageUp, Optional: key.ModShift}
	n++
	if !atBeginning {
		filtersBuf[n] = key.Filter{Focus: e, Name: key.NameLeftArrow, Optional: key.ModShortcutAlt | key.ModShift}
		n++
		filtersBuf[n] = key.Filter{Focus: e, Name: key.NameUpArrow, Optional: key.ModShortcutAlt | key.ModShift}
		n++
	}
	if !atEnd {
		filtersBuf[n] = key.Filter{Focus: e, Name: key.NameRightArrow, Optional: key.ModShortcutAlt | key.ModShift}
		n++
		filtersBuf[n] = key.Filter{Focus: e, Name: key.NameDownArrow, Optional: key.ModShortcutAlt | key.ModShift}
		n++
	}
	filters := filtersBuf[:n]

	var adjust int
	for {
		ke, ok := gtx.Event(filters...)
		if !ok {
			break
		}
		e.blinkStart = gtx.Now
		switch ke := ke.(type) {
		case key.FocusEvent:

			e.ime.imeState = imeState{}
			if ke.Focus && !e.ReadOnly {
				gtx.Execute(key.SoftKeyboardCmd{Show: true})
			}
		case key.Event:
			if !gtx.Focused(e) || ke.State != key.Press {
				break
			}
			if !e.ReadOnly && e.Submit && (ke.Name == key.NameReturn || ke.Name == key.NameEnter) {
				if !ke.Modifiers.Contain(key.ModShift) {
					e.scratch = e.text.Text(e.scratch)
					return SubmitEvent{
						Text: string(e.scratch),
					}, true
				}
			}
			e.scrollCaret = true
			e.scroller.Stop()
			ev, ok := e.command(gtx, ke)
			if ok {
				return ev, ok
			}
		case key.SnippetEvent:
			e.updateSnippet(gtx, ke.Start, ke.End)
		case key.EditEvent:
			if e.ReadOnly {
				break
			}
			e.scrollCaret = true
			e.scroller.Stop()
			s := ke.Text
			moves := 0
			submit := false
			switch {
			case e.Submit:
				if i := strings.IndexByte(s, '\n'); i != -1 {
					submit = true
					moves += len(s) - i
					s = s[:i]
				}
			case e.SingleLine:
				s = strings.ReplaceAll(s, "\n", " ")
			}
			moves += e.replace(ke.Range.Start, ke.Range.End, s, true)
			adjust += utf8.RuneCountInString(ke.Text) - moves

			e.text.MoveCaret(0, 0)
			if submit {
				e.scratch = e.text.Text(e.scratch)
				submitEvent := SubmitEvent{
					Text: string(e.scratch),
				}
				if e.text.Changed() {
					e.pending = append(e.pending, submitEvent)
					return ChangeEvent{}, true
				}
				return submitEvent, true
			}

		case transfer.DataEvent:
			e.scrollCaret = true
			e.scroller.Stop()
			content, err := io.ReadAll(ke.Open())
			if err == nil {
				if e.Insert(string(content)) != 0 {
					return ChangeEvent{}, true
				}
			}
		case key.SelectionEvent:
			e.scrollCaret = true
			e.scroller.Stop()
			ke.Start -= adjust
			ke.End -= adjust
			adjust = 0
			e.text.SetCaret(ke.Start, ke.End)
		}
	}
	if e.text.Changed() {
		return ChangeEvent{}, true
	}
	return nil, false
}

func (e *Editor) command(gtx layout.Context, k key.Event) (EditorEvent, bool) {
	direction := 1
	if gtx.Locale.Direction.Progression() == system.TowardOrigin {
		direction = -1
	}
	moveByWord := k.Modifiers.Contain(key.ModShortcutAlt)
	selAct := selectionClear
	if k.Modifiers.Contain(key.ModShift) {
		selAct = selectionExtend
	}
	if k.Modifiers.Contain(key.ModShortcut) {
		switch k.Name {

		case "V":
			if !e.ReadOnly {
				gtx.Execute(clipboard.ReadCmd{Tag: e})
			}

		case "C", "X":
			e.scratch = e.text.SelectedText(e.scratch)
			if text := string(e.scratch); text != "" {
				gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(text))})
				if k.Name == "X" && !e.ReadOnly {
					if e.Delete(1) != 0 {
						return ChangeEvent{}, true
					}
				}
			}

		case "A":
			e.text.SetCaret(0, e.text.Len())
		case "Z":
			if !e.ReadOnly {
				if k.Modifiers.Contain(key.ModShift) {
					if ev, ok := e.redo(); ok {
						return ev, ok
					}
				} else {
					if ev, ok := e.undo(); ok {
						return ev, ok
					}
				}
			}
		case key.NameHome:
			e.text.MoveTextStart(selAct)
		case key.NameEnd:
			e.text.MoveTextEnd(selAct)
		}
		return nil, false
	}
	switch k.Name {
	case key.NameReturn, key.NameEnter:
		if !e.ReadOnly {
			if e.Insert("\n") != 0 {
				return ChangeEvent{}, true
			}
		}
	case key.NameDeleteBackward:
		if !e.ReadOnly {
			if moveByWord {
				if e.deleteWord(-1) != 0 {
					return ChangeEvent{}, true
				}
			} else {
				if e.Delete(-1) != 0 {
					return ChangeEvent{}, true
				}
			}
		}
	case key.NameDeleteForward:
		if !e.ReadOnly {
			if moveByWord {
				if e.deleteWord(1) != 0 {
					return ChangeEvent{}, true
				}
			} else {
				if e.Delete(1) != 0 {
					return ChangeEvent{}, true
				}
			}
		}
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
	return nil, false
}

func (e *Editor) initBuffer() {
	if e.buffer == nil {
		e.buffer = new(editBuffer)
		e.text.SetSource(e.buffer)
	}
	e.text.Alignment = e.Alignment
	e.text.LineHeight = e.LineHeight
	e.text.LineHeightScale = e.LineHeightScale
	e.text.SingleLine = e.SingleLine
	e.text.Mask = e.Mask
	e.text.WrapPolicy = e.WrapPolicy
	e.text.DisableSpaceTrim = true
}

func (e *Editor) Update(gtx layout.Context) (EditorEvent, bool) {
	e.initBuffer()
	event, ok := e.processEvents(gtx)

	newSel := e.ime.selection
	start, end := e.text.Selection()
	newSel.rng = key.Range{
		Start: start,
		End:   end,
	}
	caretPos, carAsc, carDesc := e.text.CaretInfo()
	newSel.caret = key.Caret{
		Pos:     layout.FPt(caretPos),
		Ascent:  float32(carAsc),
		Descent: float32(carDesc),
	}
	if newSel != e.ime.selection {
		e.ime.selection = newSel
		gtx.Execute(key.SelectionCmd{Tag: e, Range: newSel.rng, Caret: newSel.caret})
	}

	e.updateSnippet(gtx, e.ime.start, e.ime.end)
	return event, ok
}

func (e *Editor) Layout(gtx layout.Context, lt *text.Shaper, font font.Font, size unit.Sp, textMaterial, selectMaterial op.CallOp) layout.Dimensions {
	for {
		_, ok := e.Update(gtx)
		if !ok {
			break
		}
	}

	origMax := gtx.Constraints.Max
	if e.SingleLine {
		gtx.Constraints.Max.X = 1 << 24
	}

	e.text.Layout(gtx, lt, font, size)

	if e.SingleLine {
		gtx.Constraints.Max = origMax
	}

	return e.layout(gtx, textMaterial, selectMaterial)
}

func (e *Editor) updateSnippet(gtx layout.Context, start, end int) {
	if start > end {
		start, end = end, start
	}
	length := e.text.Len()
	if start > length {
		start = length
	}
	if end > length {
		end = length
	}
	e.ime.start = start
	e.ime.end = end
	startOff := e.text.ByteOffset(start)
	endOff := e.text.ByteOffset(end)
	n := endOff - startOff
	if n > int64(len(e.ime.scratch)) {
		e.ime.scratch = make([]byte, n)
	}
	scratch := e.ime.scratch[:n]
	read, _ := e.text.ReadAt(scratch, startOff)
	if read != len(scratch) {
		panic("e.rr.Read truncated data")
	}
	newSnip := key.Snippet{
		Range: key.Range{
			Start: e.ime.start,
			End:   e.ime.end,
		},
		Text: e.ime.snippet.Text,
	}
	if string(scratch) != newSnip.Text {
		newSnip.Text = string(scratch)
	}
	if newSnip == e.ime.snippet {
		return
	}
	e.ime.snippet = newSnip
	gtx.Execute(key.SnippetCmd{Tag: e, Snippet: newSnip})
}

func (e *Editor) layout(gtx layout.Context, textMaterial, selectMaterial op.CallOp) layout.Dimensions {
	e.text.ScrollRel(0, 0)
	if e.scrollCaret {
		e.scrollCaret = false
		e.text.ScrollToCaret()
	}

	visibleDims := e.text.Dimensions()
	visibleDims.Size = gtx.Constraints.Constrain(visibleDims.Size)

	defer clip.Rect(image.Rectangle{Max: visibleDims.Size}).Push(gtx.Ops).Pop()
	pointer.CursorText.Add(gtx.Ops)
	event.Op(gtx.Ops, e)
	key.InputHintOp{Tag: e, Hint: e.InputHint}.Add(gtx.Ops)
	e.scroller.Add(gtx.Ops)
	e.clicker.Add(gtx.Ops)
	e.dragger.Add(gtx.Ops)
	e.showCaret = false
	if gtx.Focused(e) {
		now := gtx.Now
		dt := now.Sub(e.blinkStart)
		blinking := dt < maxBlinkDuration
		const timePerBlink = time.Second / blinksPerSecond
		nextBlink := now.Add(timePerBlink/2 - dt%(timePerBlink/2))
		if blinking {
			gtx.Execute(op.InvalidateCmd{At: nextBlink})
		}
		e.showCaret = !blinking || dt%timePerBlink < timePerBlink/2
	}
	semantic.Editor.Add(gtx.Ops)
	if e.Len() > 0 {
		e.paintSelection(gtx, selectMaterial)
		e.paintText(gtx, textMaterial)
	}
	if gtx.Enabled() {
		e.paintCaret(gtx, textMaterial)
	}
	return visibleDims
}

func (e *Editor) paintSelection(gtx layout.Context, material op.CallOp) {
	e.initBuffer()
	if !gtx.Focused(e) {
		return
	}
	e.text.PaintSelection(gtx, material)
}

func (e *Editor) paintText(gtx layout.Context, material op.CallOp) {
	e.initBuffer()
	e.text.PaintText(gtx, material)
}

func (e *Editor) paintCaret(gtx layout.Context, material op.CallOp) {
	e.initBuffer()
	if !e.showCaret || e.ReadOnly {
		return
	}
	e.text.PaintCaret(gtx, material)
}

func (e *Editor) Len() int {
	e.initBuffer()
	return e.text.Len()
}

func (e *Editor) Text() string {
	e.initBuffer()
	e.scratch = e.text.Text(e.scratch)
	return string(e.scratch)
}

func (e *Editor) SetText(s string) {
	e.initBuffer()
	if e.SingleLine {
		s = strings.ReplaceAll(s, "\n", " ")
	}
	e.replace(0, e.text.Len(), s, true)

	e.SetCaret(0, 0)
}

func (e *Editor) CaretPos() (line, col int) {
	e.initBuffer()
	return e.text.CaretPos()
}

func (e *Editor) CaretCoords() f32.Point {
	e.initBuffer()
	return e.text.CaretCoords()
}

func (e *Editor) Delete(graphemeClusters int) (deletedRunes int) {
	e.initBuffer()
	if graphemeClusters == 0 {
		return 0
	}

	start, end := e.text.Selection()
	if start != end {
		graphemeClusters -= sign(graphemeClusters)
	}

	e.text.MoveCaret(0, graphemeClusters)

	start, end = e.text.Selection()
	e.replace(start, end, "", true)

	e.text.MoveCaret(0, 0)
	e.ClearSelection()
	return end - start
}

func (e *Editor) Insert(s string) (insertedRunes int) {
	e.initBuffer()
	if e.SingleLine {
		s = strings.ReplaceAll(s, "\n", " ")
	}
	start, end := e.text.Selection()
	moves := e.replace(start, end, s, true)
	if end < start {
		start = end
	}

	e.text.MoveCaret(0, 0)
	e.SetCaret(start+moves, start+moves)
	e.scrollCaret = true
	return moves
}

type modification struct {
	StartRune int

	ApplyContent string

	ReverseContent string
}

func (e *Editor) undo() (EditorEvent, bool) {
	e.initBuffer()
	if len(e.history) < 1 || e.nextHistoryIdx == 0 {
		return nil, false
	}
	mod := e.history[e.nextHistoryIdx-1]
	replaceEnd := mod.StartRune + utf8.RuneCountInString(mod.ApplyContent)
	e.replace(mod.StartRune, replaceEnd, mod.ReverseContent, false)
	caretEnd := mod.StartRune + utf8.RuneCountInString(mod.ReverseContent)
	e.SetCaret(caretEnd, mod.StartRune)
	e.nextHistoryIdx--
	return ChangeEvent{}, true
}

func (e *Editor) redo() (EditorEvent, bool) {
	e.initBuffer()
	if len(e.history) < 1 || e.nextHistoryIdx == len(e.history) {
		return nil, false
	}
	mod := e.history[e.nextHistoryIdx]
	end := mod.StartRune + utf8.RuneCountInString(mod.ReverseContent)
	e.replace(mod.StartRune, end, mod.ApplyContent, false)
	caretEnd := mod.StartRune + utf8.RuneCountInString(mod.ApplyContent)
	e.SetCaret(caretEnd, mod.StartRune)
	e.nextHistoryIdx++
	return ChangeEvent{}, true
}

func (e *Editor) replace(start, end int, s string, addHistory bool) int {
	length := e.text.Len()
	if start > end {
		start, end = end, start
	}
	start = min(start, length)
	end = min(end, length)
	replaceSize := end - start
	el := e.Len()
	var sc int
	if e.Filter != "" || e.MaxLen > 0 {
		// Use a builder to avoid O(n²) string concatenation when filtering.
		var b strings.Builder
		filtered := false // true if any rune was skipped by Filter
		idx := 0
		for idx < len(s) {
			if e.MaxLen > 0 && el-replaceSize+sc >= e.MaxLen {
				s = s[:idx]
				if filtered {
					s = b.String()
				}
				break
			}
			_, n := utf8.DecodeRuneInString(s[idx:])
			if e.Filter != "" && !strings.Contains(e.Filter, s[idx:idx+n]) {
				if !filtered {
					filtered = true
					b.Grow(len(s))
					b.WriteString(s[:idx])
				}
				idx += n
				continue
			}
			if filtered {
				b.WriteString(s[idx : idx+n])
			}
			idx += n
			sc++
		}
		if filtered && idx >= len(s) {
			s = b.String()
		}
	} else {
		sc = utf8.RuneCountInString(s)
	}

	if addHistory {
		if needed := replaceSize; needed > cap(e.historyScratch) {
			e.historyScratch = make([]rune, 0, needed)
		}
		deleted := e.historyScratch[:0]
		readPos := e.text.ByteOffset(start)
		for range replaceSize {
			ru, s, _ := e.text.ReadRuneAt(int64(readPos))
			readPos += int64(s)
			deleted = append(deleted, ru)
		}
		if e.nextHistoryIdx < len(e.history) {
			e.history = e.history[:e.nextHistoryIdx]
		}
		e.history = append(e.history, modification{
			StartRune:      start,
			ApplyContent:   s,
			ReverseContent: string(deleted),
		})
		e.nextHistoryIdx++
	}

	sc = e.text.Replace(start, end, s)
	newEnd := start + sc
	adjust := func(pos int) int {
		switch {
		case newEnd < pos && pos <= end:
			pos = newEnd
		case end < pos:
			diff := newEnd - end
			pos = pos + diff
		}
		return pos
	}
	e.ime.start = adjust(e.ime.start)
	e.ime.end = adjust(e.ime.end)
	return sc
}

func (e *Editor) MoveCaret(startDelta, endDelta int) {
	e.initBuffer()
	e.text.MoveCaret(startDelta, endDelta)
}

func (e *Editor) deleteWord(distance int) (deletedRunes int) {
	if distance == 0 {
		return
	}

	start, end := e.text.Selection()
	if start != end {
		deletedRunes = e.Delete(1)
		distance -= sign(distance)
	}
	if distance == 0 {
		return deletedRunes
	}

	words, direction := distance, 1
	if distance < 0 {
		words, direction = distance*-1, -1
	}
	caret, _ := e.text.Selection()

	atEnd := func(runes int) bool {
		idx := caret + runes*direction
		return idx <= 0 || idx >= e.Len()
	}

	next := func(runes int) rune {
		idx := caret + runes*direction
		if idx < 0 {
			idx = 0
		} else if idx > e.Len() {
			idx = e.Len()
		}
		off := e.text.ByteOffset(idx)
		var r rune
		if direction < 0 {
			r, _, _ = e.text.ReadRuneBefore(int64(off))
		} else {
			r, _, _ = e.text.ReadRuneAt(int64(off))
		}
		return r
	}
	runes := 1
	for range words {
		r := next(runes)
		wantSpace := unicode.IsSpace(r)
		for r := next(runes); unicode.IsSpace(r) == wantSpace && !atEnd(runes); r = next(runes) {
			runes += 1
		}
	}
	deletedRunes += e.Delete(runes * direction)
	return deletedRunes
}

func (e *Editor) SelectionLen() int {
	e.initBuffer()
	return e.text.SelectionLen()
}

func (e *Editor) Selection() (start, end int) {
	e.initBuffer()
	return e.text.Selection()
}

func (e *Editor) SetCaret(start, end int) {
	e.initBuffer()
	e.text.SetCaret(start, end)
	e.scrollCaret = true
	e.scroller.Stop()
}

func (e *Editor) SetScrollCaret(b bool) {
	e.initBuffer()
	e.scrollCaret = b
}

func (e *Editor) SelectedText() string {
	e.initBuffer()
	e.scratch = e.text.SelectedText(e.scratch)
	return string(e.scratch)
}

func (e *Editor) ClearSelection() {
	e.initBuffer()
	e.text.ClearSelection()
}

func (e *Editor) WriteTo(w io.Writer) (int64, error) {
	e.initBuffer()
	return e.text.WriteTo(w)
}

func (e *Editor) Seek(offset int64, whence int) (int64, error) {
	e.initBuffer()
	return e.text.Seek(offset, whence)
}

func (e *Editor) Read(p []byte) (int, error) {
	e.initBuffer()
	return e.text.Read(p)
}

func (e *Editor) Regions(start, end int, regions []Region) []Region {
	e.initBuffer()
	return e.text.Regions(start, end, regions)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}

func (s ChangeEvent) isEditorEvent() {}
func (s SubmitEvent) isEditorEvent() {}
func (s SelectEvent) isEditorEvent() {}

func (e *Editor) GetScrollY() int {
	e.initBuffer()
	return e.text.ScrollOff().Y
}

func (e *Editor) SetScrollY(y int) {
	e.initBuffer()
	current := e.text.ScrollOff().Y
	e.text.ScrollRel(0, y-current)
}

func (e *Editor) GetScrollBounds() image.Rectangle {
	e.initBuffer()
	return e.text.ScrollBounds()
}

func (e *Editor) GetScrollX() int {
	e.initBuffer()
	return e.text.ScrollOff().X
}
