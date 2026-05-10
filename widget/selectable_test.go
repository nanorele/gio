package widget

import (
	"fmt"
	"image"
	"strings"
	"testing"

	nsareg "eliasnaur.com/font/noto/sans/arabic/regular"
	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/font"
	"github.com/nanorele/gio/font/gofont"
	"github.com/nanorele/gio/font/opentype"
	"github.com/nanorele/gio/io/input"
	"github.com/nanorele/gio/io/key"
	"github.com/nanorele/gio/io/pointer"
	"github.com/nanorele/gio/io/system"
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

// newSelectableTestEnv builds a fully-initialized gtx + shaper for a Selectable.
func newSelectableTestEnv(t testing.TB, max image.Point) (*input.Router, layout.Context, *text.Shaper) {
	t.Helper()
	r := new(input.Router)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Locale:      english,
		Source:      r.Source(),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(max),
	}
	cache := text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	return r, gtx, cache
}

// newSelectableTestEnvFaces is like newSelectableTestEnv but accepts a custom
// font collection and locale.
func newSelectableTestEnvFaces(t testing.TB, max image.Point, locale system.Locale, faces []font.FontFace) (*input.Router, layout.Context, *text.Shaper) {
	t.Helper()
	r := new(input.Router)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Locale:      locale,
		Source:      r.Source(),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Constraints: layout.Exact(max),
	}
	cache := text.NewShaper(text.NoSystemFonts(), text.WithCollection(faces))
	return r, gtx, cache
}

// TestSelectableSetTextResetsCaret guards against state-leak between SetText
// calls: a prior selection must not survive a content swap.
func TestSelectableSetTextResetsCaret(t *testing.T) {
	r, gtx, cache := newSelectableTestEnv(t, image.Pt(200, 200))
	s := new(Selectable)
	s.SetText("0123456789")
	s.Layout(gtx, cache, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})
	r.Frame(gtx.Ops)

	s.SetCaret(2, 8)
	if start, end := s.Selection(); start != 2 || end != 8 {
		t.Fatalf("setup failed: got selection %d-%d", start, end)
	}

	// Replace with shorter text. Stale caret indices (2..8) point past the
	// new content (length 2). Selectable should reset selection.
	s.SetText("ab")
	if start, end := s.Selection(); start != 0 || end != 0 {
		t.Errorf("SetText did not reset caret/selection: got %d-%d, want 0-0", start, end)
	}
	if got := s.SelectedText(); got != "" {
		t.Errorf("SelectedText after SetText = %q, want empty", got)
	}
	if got := s.SelectionLen(); got != 0 {
		t.Errorf("SelectionLen after SetText = %d, want 0", got)
	}
}

// TestSelectableSetTextSameStringPreservesCaret verifies that the
// short-circuit on identical strings preserves caret state (no source change).
func TestSelectableSetTextSameStringPreservesCaret(t *testing.T) {
	r, gtx, cache := newSelectableTestEnv(t, image.Pt(200, 200))
	s := new(Selectable)
	s.SetText("hello world")
	s.Layout(gtx, cache, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})
	r.Frame(gtx.Ops)
	s.SetCaret(0, 5)
	s.SetText("hello world")
	if start, end := s.Selection(); start != 0 || end != 5 {
		t.Errorf("identical SetText changed selection: got %d-%d, want 0-5", start, end)
	}
}

// TestSelectableClearSelection ensures ClearSelection collapses the range.
func TestSelectableClearSelection(t *testing.T) {
	r, gtx, cache := newSelectableTestEnv(t, image.Pt(200, 200))
	s := new(Selectable)
	s.SetText("0123456789")
	s.Layout(gtx, cache, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})
	r.Frame(gtx.Ops)
	s.SetCaret(3, 7)
	s.ClearSelection()
	start, end := s.Selection()
	if start != end {
		t.Errorf("ClearSelection: got %d-%d, want collapsed", start, end)
	}
	if s.SelectionLen() != 0 {
		t.Errorf("SelectionLen after ClearSelection = %d", s.SelectionLen())
	}
}

// TestSelectableSetCaretClamping ensures out-of-range indices are clamped to
// the valid rune range.
func TestSelectableSetCaretClamping(t *testing.T) {
	r, gtx, cache := newSelectableTestEnv(t, image.Pt(200, 200))
	s := new(Selectable)
	s.SetText("abc")
	s.Layout(gtx, cache, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})
	r.Frame(gtx.Ops)

	s.SetCaret(-100, 1000)
	start, end := s.Selection()
	if start < 0 || start > 3 || end < 0 || end > 3 {
		t.Errorf("SetCaret didn't clamp: got %d-%d (text length 3 runes)", start, end)
	}
}

// TestSelectableSelectAcrossParagraphs covers selection spanning explicit
// newlines. After the recent index.go posStart/posEnd fixes, rune offsets
// must remain stable across paragraph boundaries.
func TestSelectableSelectAcrossParagraphs(t *testing.T) {
	r, gtx, cache := newSelectableTestEnv(t, image.Pt(400, 400))
	s := new(Selectable)
	const txt = "first line\nsecond line\nthird line"
	s.SetText(txt)
	s.Layout(gtx, cache, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})
	r.Frame(gtx.Ops)

	s.SetCaret(6, 28)
	got := s.SelectedText()
	want := txt[6:28]
	if got != want {
		t.Errorf("multi-paragraph selection: got %q, want %q", got, want)
	}
}

// TestSelectableSelectionAcrossSoftLineBreak exercises the recently fixed
// posEnd off-by-one for soft line breaks: text that wraps within a paragraph
// must still produce correct rune offsets across the wrap point.
func TestSelectableSelectionAcrossSoftLineBreak(t *testing.T) {
	r, gtx, cache := newSelectableTestEnv(t, image.Pt(40, 200))
	s := new(Selectable)
	const txt = "the quick brown fox jumps over the lazy dog"
	s.SetText(txt)
	s.Layout(gtx, cache, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})
	r.Frame(gtx.Ops)

	s.SetCaret(4, 25)
	want := txt[4:25]
	if got := s.SelectedText(); got != want {
		t.Errorf("soft-line-break selection: got %q, want %q", got, want)
	}

	s.SetCaret(0, len([]rune(txt)))
	if got := s.SelectedText(); got != txt {
		t.Errorf("select-all: got %q, want %q", got, txt)
	}
}

// TestSelectableBidiSelection ensures bidi text remains coherent under
// selection, sanity-checking the recent atStartOfLine/atEndOfLine fixes.
func TestSelectableBidiSelection(t *testing.T) {
	rtlFace, err := opentype.Parse(nsareg.TTF)
	if err != nil {
		t.Skipf("RTL font unavailable: %v", err)
	}
	faces := append([]font.FontFace{}, gofont.Collection()...)
	faces = append(faces, font.FontFace{Font: font.Font{Typeface: "RTL"}, Face: rtlFace})

	r, gtx, cache := newSelectableTestEnvFaces(t, image.Pt(300, 200), english, faces)
	s := new(Selectable)
	const txt = "hello سماء world"
	s.SetText(txt)
	s.Layout(gtx, cache, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})
	r.Frame(gtx.Ops)

	runes := []rune(txt)
	s.SetCaret(0, len(runes))
	if got := s.SelectedText(); got != txt {
		t.Errorf("bidi select-all (LTR locale): got %q (%d bytes), want %q (%d bytes)", got, len(got), txt, len(txt))
	}

	// Select just the RTL run by rune indices and check byte-substring match.
	startRune := 6
	endRune := startRune + 4
	startByte := byteOffsetOfRune(txt, startRune)
	endByte := byteOffsetOfRune(txt, endRune)
	s.SetCaret(startRune, endRune)
	want := txt[startByte:endByte]
	if got := s.SelectedText(); got != want {
		t.Errorf("RTL substring select: got %q, want %q", got, want)
	}

	// Repeat under RTL locale.
	r2, gtx2, cache2 := newSelectableTestEnvFaces(t, image.Pt(300, 200), arabic, faces)
	s2 := new(Selectable)
	s2.SetText(txt)
	s2.Layout(gtx2, cache2, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})
	r2.Frame(gtx2.Ops)
	s2.SetCaret(0, len(runes))
	if got := s2.SelectedText(); got != txt {
		t.Errorf("bidi select-all (RTL locale): got %q, want %q", got, txt)
	}
}

// TestSelectableTextRoundTrip ensures Text() returns whatever was set.
func TestSelectableTextRoundTrip(t *testing.T) {
	cases := []string{
		"",
		"a",
		"hello",
		"line1\nline2",
		strings.Repeat("x", 1000),
		"ünîcödé 🐍",
	}
	for _, want := range cases {
		s := new(Selectable)
		s.SetText(want)
		if got := s.Text(); got != want {
			t.Errorf("Text round-trip: got %q, want %q", got, want)
		}
	}
}

// TestSelectableRegionsEmpty handles the degenerate case of an empty
// Selectable.
func TestSelectableRegionsEmpty(t *testing.T) {
	s := new(Selectable)
	regs := s.Regions(0, 0, nil)
	if len(regs) != 0 {
		t.Errorf("Regions on empty Selectable: got %d, want 0", len(regs))
	}
}

// TestSelectableRegionsAcrossLines verifies multi-line region geometry.
func TestSelectableRegionsAcrossLines(t *testing.T) {
	r, gtx, cache := newSelectableTestEnv(t, image.Pt(400, 400))
	s := new(Selectable)
	const txt = "line one\nline two\nline three"
	s.SetText(txt)
	s.Layout(gtx, cache, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})
	r.Frame(gtx.Ops)
	regs := s.Regions(0, len([]rune(txt)), nil)
	if len(regs) < 3 {
		t.Errorf("expected at least 3 regions for 3 lines, got %d", len(regs))
	}
}

// TestSelectableTruncated covers the Truncated flag path under MaxLines=1.
func TestSelectableTruncated(t *testing.T) {
	r, gtx, cache := newSelectableTestEnv(t, image.Pt(40, 20))
	s := new(Selectable)
	s.MaxLines = 1
	s.Truncator = "..."
	s.SetText("the quick brown fox jumps over the lazy dog")
	s.Layout(gtx, cache, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})
	r.Frame(gtx.Ops)
	if !s.Truncated() {
		t.Error("expected Truncated()==true with MaxLines=1 on long text")
	}
}

// byteOffsetOfRune returns the byte index of the runeIdx-th rune in s.
func byteOffsetOfRune(s string, runeIdx int) int {
	for i := range s {
		if runeIdx == 0 {
			return i
		}
		runeIdx--
	}
	return len(s)
}
