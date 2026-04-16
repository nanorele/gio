package widget

import (
	"fmt"
	"image"
	"testing"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/font"
	"github.com/nanorele/gio/font/gofont"
	"github.com/nanorele/gio/io/input"
	"github.com/nanorele/gio/io/key"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/text"
	"github.com/nanorele/gio/unit"
)

func TestSelectableZeroValue(t *testing.T) {
	var s Selectable
	if s.Text() != "" {
		t.Errorf("expected zero value to have no text, got %q", s.Text())
	}
	if start, end := s.Selection(); start != 0 || end != 0 {
		t.Errorf("expected start=0, end=0, got start=%d, end=%d", start, end)
	}
	if selected := s.SelectedText(); selected != "" {
		t.Errorf("expected selected text to be \"\", got %q", selected)
	}
	s.SetCaret(5, 5)
	if start, end := s.Selection(); start != 0 || end != 0 {
		t.Errorf("expected start=0, end=0, got start=%d, end=%d", start, end)
	}
}

func TestSelectableMove(t *testing.T) {
	r := new(input.Router)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Locale:      english,
		Source:      r.Source(),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(100, 100)),
	}


	cache := text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	fnt := font.Font{}
	fontSize := unit.Sp(10)

	str := `0123456789`

	s := new(Selectable)

	gtx.Execute(key.FocusCmd{Tag: s})
	s.SetText(str)

	s.Layout(gtx, cache, font.Font{}, fontSize, op.CallOp{}, op.CallOp{})
	r.Frame(gtx.Ops)
	s.SetCaret(3, 6)
	s.Layout(gtx, cache, font.Font{}, fontSize, op.CallOp{}, op.CallOp{})
	r.Frame(gtx.Ops)
	s.Layout(gtx, cache, font.Font{}, fontSize, op.CallOp{}, op.CallOp{})
	r.Frame(gtx.Ops)

	for _, keyName := range []key.Name{key.NameLeftArrow, key.NameRightArrow, key.NameUpArrow, key.NameDownArrow} {

		s.SetCaret(3, 6)
		if start, end := s.Selection(); start != 3 || end != 6 {
			t.Errorf("expected start=%d, end=%d, got start=%d, end=%d", 3, 6, start, end)
		}
		if expected, got := "345", s.SelectedText(); expected != got {
			t.Errorf("KeyName %s, expected %q, got %q", keyName, expected, got)
		}

		r.Queue(key.Event{State: key.Press, Name: keyName})
		s.SetText(str)
		s.Layout(gtx, cache, fnt, fontSize, op.CallOp{}, op.CallOp{})
		r.Frame(gtx.Ops)

		if expected, got := "", s.SelectedText(); expected != got {
			t.Errorf("KeyName %s, expected %q, got %q", keyName, expected, got)
		}
	}
}

func TestSelectable_Pointer(t *testing.T) {
	r := new(input.Router)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Locale:      english,
		Source:      r.Source(),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(image.Pt(100, 100)),
	}


	cache := text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	fontSize := unit.Sp(10)
	str := `0123456789`
	s := new(Selectable)
	s.SetText(str)

	// Register filter
	s.Layout(gtx, cache, font.Font{}, fontSize, op.CallOp{}, op.CallOp{})
	r.Frame(gtx.Ops)

	// 1. Press to focus and set caret
	r.Queue(pointer.Event{
		Kind:     pointer.Press,
		Source:   pointer.Mouse,
		Buttons:  pointer.ButtonPrimary,
		Position: f32.Pt(5, 5), // Roughly at the beginning
	})
	s.Layout(gtx, cache, font.Font{}, fontSize, op.CallOp{}, op.CallOp{})
	r.Frame(gtx.Ops)
	if !gtx.Focused(s) {
		t.Error("Selectable did not gain focus on press")
	}

	// 2. Drag to select
	r.Queue(pointer.Event{
		Kind:     pointer.Move,
		Source:   pointer.Mouse,
		Buttons:  pointer.ButtonPrimary,
		Position: f32.Pt(50, 5), // Roughly in the middle
		Priority: pointer.Grabbed,
	})
	s.Layout(gtx, cache, font.Font{}, fontSize, op.CallOp{}, op.CallOp{})
	r.Frame(gtx.Ops)
	
	start, end := s.Selection()
	if start == end {
		t.Errorf("expected non-empty selection after drag, got %d-%d", start, end)
	}
	if s.SelectionLen() == 0 {
		t.Error("SelectionLen should be non-zero")
	}
	if !s.Focused() {
		t.Error("Selectable should be focused")
	}
	if s.Truncated() {
		t.Error("Selectable should not be truncated in this test")
	}
	s.ClearSelection()
	if start, end := s.Selection(); start != end {
		t.Errorf("Selection should be empty after ClearSelection, got %d-%d", start, end)
	}
	_ = s.Regions(0, 5, nil)
}



func TestSelectableConfigurations(t *testing.T) {
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(300, 300)),
		Locale:      english,
	}
	cache := text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	fontSize := unit.Sp(10)
	font := font.Font{}
	sentence := "\n\n\n\n\n\n\n\n\n\n\n\nthe quick brown fox jumps over the lazy dog"

	for _, alignment := range []text.Alignment{text.Start, text.Middle, text.End} {
		for _, zeroMin := range []bool{true, false} {
			t.Run(fmt.Sprintf("Alignment: %v ZeroMinConstraint: %v", alignment, zeroMin), func(t *testing.T) {
				defer func() {
					if err := recover(); err != nil {
						t.Error(err)
					}
				}()
				if zeroMin {
					gtx.Constraints.Min = image.Point{}
				} else {
					gtx.Constraints.Min = gtx.Constraints.Max
				}
				s := new(Selectable)
				s.Alignment = alignment
				s.SetText(sentence)
				interactiveDims := s.Layout(gtx, cache, font, fontSize, op.CallOp{}, op.CallOp{})
				staticDims := Label{Alignment: alignment}.Layout(gtx, cache, font, fontSize, sentence, op.CallOp{})

				if interactiveDims != staticDims {
					t.Errorf("expected consistent dimensions, static returned %#+v, interactive returned %#+v", staticDims, interactiveDims)
				}
			})
		}
	}
}
