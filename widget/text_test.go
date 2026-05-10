package widget

import (
	"image"
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/nanorele/gio/font"
	"github.com/nanorele/gio/font/gofont"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/text"
	"github.com/nanorele/gio/unit"
	"golang.org/x/image/math/fixed"
)

// newTestTextView builds a textView that has been laid out for the given
// string. The returned gtx is reused only for context propagation; tests
// don't paint anything.
func newTestTextView(t *testing.T, s string, opts ...func(*textView)) (*textView, layout.Context, *text.Shaper) {
	t.Helper()
	tv := &textView{}
	tv.SetSource(newStringSource(s))
	for _, o := range opts {
		o(tv)
	}
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(200, 200)),
		Locale:      english,
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}
	cache := text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	tv.Layout(gtx, cache, font.Font{}, unit.Sp(10))
	return tv, gtx, cache
}

func TestMaskReader_BasicMasking(t *testing.T) {
	var m maskReader
	m.Reset(strings.NewReader("abc"), '*')
	got, err := io.ReadAll(&m)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "***" {
		t.Errorf("expected ***, got %q", got)
	}
}

func TestMaskReader_PreservesNewlines(t *testing.T) {
	var m maskReader
	m.Reset(strings.NewReader("ab\ncd"), '*')
	got, _ := io.ReadAll(&m)
	if string(got) != "**\n**" {
		t.Errorf("expected **\\n**, got %q", got)
	}
}

func TestMaskReader_MultiByteMask(t *testing.T) {
	var m maskReader
	// Use a 3-byte rune as mask
	mask := '•' // bullet, 3 bytes in UTF-8
	maskBytes := string(mask)
	m.Reset(strings.NewReader("abc"), mask)
	got, _ := io.ReadAll(&m)
	want := strings.Repeat(maskBytes, 3)
	if string(got) != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestMaskReader_TinyBuffer(t *testing.T) {
	// Force the overflow path by reading byte by byte from a multi-byte mask.
	var m maskReader
	mask := '•' // 3 bytes
	m.Reset(strings.NewReader("ab"), mask)
	var collected []byte
	buf := make([]byte, 1)
	for {
		n, err := m.Read(buf)
		collected = append(collected, buf[:n]...)
		if err != nil {
			break
		}
	}
	want := strings.Repeat(string(mask), 2)
	if string(collected) != want {
		t.Errorf("expected %q, got %q", want, collected)
	}
}

func TestMaskReader_MultiByteRunesInInput(t *testing.T) {
	var m maskReader
	// Each rune should be replaced by exactly one mask, regardless of input
	// rune width.
	m.Reset(strings.NewReader("aαβ"), '*')
	got, _ := io.ReadAll(&m)
	if string(got) != "***" {
		t.Errorf("expected ***, got %q", got)
	}
}

func TestMaskReader_Empty(t *testing.T) {
	var m maskReader
	m.Reset(strings.NewReader(""), '*')
	got, _ := io.ReadAll(&m)
	if len(got) != 0 {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestTextView_DimensionsAndLen(t *testing.T) {
	tv, _, _ := newTestTextView(t, "hello")
	if got := tv.Len(); got != 5 {
		t.Errorf("Len: got %d, want 5", got)
	}
	dims := tv.Dimensions()
	if dims.Size.X <= 0 || dims.Size.Y <= 0 {
		t.Errorf("Dimensions empty: %+v", dims)
	}
	full := tv.FullDimensions()
	if full.Size.X <= 0 || full.Size.Y <= 0 {
		t.Errorf("FullDimensions empty: %+v", full)
	}
}

func TestTextView_EmptyDocument(t *testing.T) {
	tv, _, _ := newTestTextView(t, "")
	if got := tv.Len(); got != 0 {
		t.Errorf("Len of empty: got %d", got)
	}
	tv.SetCaret(0, 0)
	if s, e := tv.Selection(); s != 0 || e != 0 {
		t.Errorf("Selection on empty: %d/%d", s, e)
	}
	if l := tv.SelectionLen(); l != 0 {
		t.Errorf("SelectionLen on empty: %d", l)
	}
	// Should not panic.
	tv.MoveTextStart(selectionClear)
	tv.MoveTextEnd(selectionClear)
	tv.MoveLineStart(selectionClear)
	tv.MoveLineEnd(selectionClear)
	tv.MoveLines(+1, selectionClear)
}

func TestTextView_SetCaretAndSelection(t *testing.T) {
	tv, _, _ := newTestTextView(t, "0123456789")
	tv.SetCaret(2, 5)
	s, e := tv.Selection()
	if s != 2 || e != 5 {
		t.Errorf("Selection got %d/%d, want 2/5", s, e)
	}
	if l := tv.SelectionLen(); l != 3 {
		t.Errorf("SelectionLen got %d, want 3", l)
	}
	// Out-of-range clamps to document end.
	tv.SetCaret(100, 200)
	s, e = tv.Selection()
	if s != 10 || e != 10 {
		t.Errorf("clamped Selection got %d/%d, want 10/10", s, e)
	}
	// Negative clamps to 0.
	tv.SetCaret(-5, -3)
	s, e = tv.Selection()
	if s != 0 || e != 0 {
		t.Errorf("negative Selection got %d/%d, want 0/0", s, e)
	}
}

func TestTextView_ByteOffsetASCII(t *testing.T) {
	tv, _, _ := newTestTextView(t, "abcdef")
	for i := 0; i <= 6; i++ {
		if got := tv.ByteOffset(i); got != int64(i) {
			t.Errorf("ByteOffset(%d) = %d, want %d", i, got, i)
		}
	}
}

func TestTextView_ByteOffsetMultiByte(t *testing.T) {
	// "æbc" -> æ is 2 bytes, b/c 1 each. Total 4 bytes, 3 runes.
	tv, _, _ := newTestTextView(t, "æbc")
	if got := tv.ByteOffset(0); got != 0 {
		t.Errorf("ByteOffset(0) = %d, want 0", got)
	}
	if got := tv.ByteOffset(1); got != 2 {
		t.Errorf("ByteOffset(1) = %d, want 2", got)
	}
	if got := tv.ByteOffset(2); got != 3 {
		t.Errorf("ByteOffset(2) = %d, want 3", got)
	}
	if got := tv.ByteOffset(3); got != 4 {
		t.Errorf("ByteOffset(3) = %d, want 4", got)
	}
}

func TestTextView_Truncated(t *testing.T) {
	// With unconstrained layout it should not be truncated.
	tv, _, _ := newTestTextView(t, "hello")
	if tv.Truncated() {
		t.Errorf("expected not truncated for short text")
	}
}

func TestTextView_TextRoundTrip(t *testing.T) {
	const s = "hello world"
	tv, _, _ := newTestTextView(t, s)
	got := tv.Text(nil)
	if string(got) != s {
		t.Errorf("Text round-trip: got %q want %q", got, s)
	}
}

func TestTextView_MoveCaret(t *testing.T) {
	tv, _, _ := newTestTextView(t, "abcdefgh")
	tv.SetCaret(0, 0)
	tv.MoveCaret(+3, +3)
	if s, _ := tv.Selection(); s != 3 {
		t.Errorf("after MoveCaret(+3): start %d, want 3", s)
	}
	tv.MoveCaret(-1, -1)
	if s, _ := tv.Selection(); s != 2 {
		t.Errorf("after MoveCaret(-1): start %d, want 2", s)
	}
	// Clamps at end.
	tv.MoveCaret(+1000, +1000)
	if s, _ := tv.Selection(); s != tv.Len() {
		t.Errorf("MoveCaret past end: got %d, want %d", s, tv.Len())
	}
	// Clamps at start.
	tv.MoveCaret(-1000, -1000)
	if s, _ := tv.Selection(); s != 0 {
		t.Errorf("MoveCaret past start: got %d, want 0", s)
	}
}

func TestTextView_MoveLines(t *testing.T) {
	tv, _, _ := newTestTextView(t, "line1\nline2\nline3")
	tv.SetCaret(0, 0)
	tv.MoveLines(+1, selectionClear)
	line, _ := tv.CaretPos()
	if line != 1 {
		t.Errorf("MoveLines(+1) line = %d, want 1", line)
	}
	tv.MoveLines(+1, selectionClear)
	line, _ = tv.CaretPos()
	if line != 2 {
		t.Errorf("MoveLines(+1) line = %d, want 2", line)
	}
	// Clamps at last line — stays at last line.
	tv.MoveLines(+10, selectionClear)
	line, _ = tv.CaretPos()
	if line != 2 {
		t.Errorf("MoveLines past end: line = %d, want 2", line)
	}
	// Up.
	tv.MoveLines(-1, selectionClear)
	line, _ = tv.CaretPos()
	if line != 1 {
		t.Errorf("MoveLines(-1) line = %d, want 1", line)
	}
	// Clamps at first line.
	tv.MoveLines(-100, selectionClear)
	line, _ = tv.CaretPos()
	if line != 0 {
		t.Errorf("MoveLines past start: line = %d, want 0", line)
	}
}

func TestTextView_MoveStartEnd(t *testing.T) {
	tv, _, _ := newTestTextView(t, "abc\ndef")
	tv.SetCaret(2, 2)
	tv.MoveTextEnd(selectionClear)
	s, _ := tv.Selection()
	if s != tv.Len() {
		t.Errorf("MoveTextEnd: got %d, want %d", s, tv.Len())
	}
	tv.MoveTextStart(selectionClear)
	s, _ = tv.Selection()
	if s != 0 {
		t.Errorf("MoveTextStart: got %d, want 0", s)
	}
}

func TestTextView_MoveLineStartEnd(t *testing.T) {
	tv, _, _ := newTestTextView(t, "abc\ndef\nghi")
	// Place caret on line 1 ("def"), col 1.
	tv.SetCaret(5, 5)
	tv.MoveLineStart(selectionClear)
	s, _ := tv.Selection()
	if s != 4 { // start of "def"
		t.Errorf("MoveLineStart: got %d, want 4", s)
	}
	line, col := tv.CaretPos()
	if line != 1 || col != 0 {
		t.Errorf("MoveLineStart pos: %d/%d, want 1/0", line, col)
	}
	tv.MoveLineEnd(selectionClear)
	s, _ = tv.Selection()
	if s != 7 { // end of "def" (line 1)
		t.Errorf("MoveLineEnd: got %d, want 7", s)
	}
}

func TestTextView_SelectionMaintainedAcrossMove(t *testing.T) {
	tv, _, _ := newTestTextView(t, "0123456789")
	tv.SetCaret(2, 7)
	if l := tv.SelectionLen(); l != 5 {
		t.Errorf("initial SelectionLen %d, want 5", l)
	}
	// MoveCaret moves both endpoints independently.
	tv.MoveCaret(+1, +1)
	s, e := tv.Selection()
	if s != 3 || e != 8 {
		t.Errorf("after MoveCaret: %d/%d, want 3/8", s, e)
	}
	tv.ClearSelection()
	if l := tv.SelectionLen(); l != 0 {
		t.Errorf("after ClearSelection: %d, want 0", l)
	}
}

func TestTextView_RegionsSingleLine(t *testing.T) {
	tv, _, _ := newTestTextView(t, "hello world")
	regs := tv.Regions(0, 5, nil)
	if len(regs) == 0 {
		t.Fatalf("expected at least one region")
	}
	for _, r := range regs {
		if r.Bounds.Empty() {
			t.Errorf("empty region: %+v", r)
		}
	}
}

func TestTextView_RegionsEmpty(t *testing.T) {
	tv, _, _ := newTestTextView(t, "hello")
	regs := tv.Regions(3, 3, nil)
	// An empty selection produces no rectangles or only zero-width ones.
	for _, r := range regs {
		if r.Bounds.Dx() != 0 {
			t.Errorf("zero-length selection produced wide region: %+v", r)
		}
	}
}

func TestTextView_RegionsMultiLine(t *testing.T) {
	tv, _, _ := newTestTextView(t, "abc\ndef\nghi")
	regs := tv.Regions(0, tv.Len(), nil)
	if len(regs) < 2 {
		t.Errorf("expected multi-line selection to produce >=2 regions, got %d", len(regs))
	}
}

func TestTextView_RegionsReusesBuf(t *testing.T) {
	tv, _, _ := newTestTextView(t, "abc")
	buf := make([]Region, 0, 4)
	got := tv.Regions(0, 3, buf)
	// Buffer reuse: capacity should be at least the input cap.
	if cap(got) < 4 {
		t.Errorf("Regions did not reuse buffer; cap %d < 4", cap(got))
	}
}

func TestTextView_Closest_ClickAboveAndBelow(t *testing.T) {
	tv, _, _ := newTestTextView(t, "line1\nline2\nline3")
	// Click way above the text — should snap to first position.
	pos, _ := tv.closestToXY(0, -1000)
	if pos.runes != 0 {
		t.Errorf("click above: runes=%d, want 0", pos.runes)
	}
	// Click way below — should snap to last position.
	pos, _ = tv.closestToXY(fixed.I(0), 100000)
	if pos.runes != tv.Len() {
		t.Errorf("click below: runes=%d, want %d", pos.runes, tv.Len())
	}
}

func TestTextView_MoveCoord(t *testing.T) {
	tv, _, _ := newTestTextView(t, "abcdef")
	// Click before everything sets caret to 0.
	tv.MoveCoord(image.Pt(-100, 0))
	if s, _ := tv.Selection(); s != 0 {
		t.Errorf("MoveCoord(-100,0): got %d, want 0", s)
	}
	// Click far right of single line should land at end.
	tv.MoveCoord(image.Pt(10000, 0))
	if s, _ := tv.Selection(); s != tv.Len() {
		t.Errorf("MoveCoord(10000,0): got %d, want %d", s, tv.Len())
	}
}

func TestTextView_SingleLineLayout(t *testing.T) {
	tv := &textView{}
	tv.SingleLine = true
	tv.SetSource(newStringSource("abcdef ghijkl mnopqr"))
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(20, 20)),
		Locale:      english,
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}
	cache := text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	tv.Layout(gtx, cache, font.Font{}, unit.Sp(10))
	// Even with very narrow constraints, single-line should fit on one line
	// (no wrapping). Therefore CaretPos should not produce line > 0 for the
	// last position.
	tv.SetCaret(tv.Len(), tv.Len())
	if line, _ := tv.CaretPos(); line != 0 {
		t.Errorf("SingleLine caret on last line %d, want 0", line)
	}
}

func TestTextView_MaskInvalidatesLayout(t *testing.T) {
	tv := &textView{}
	tv.SetSource(newStringSource("secret"))
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(200, 200)),
		Locale:      english,
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}
	cache := text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	tv.Layout(gtx, cache, font.Font{}, unit.Sp(10))
	dimsBefore := tv.Dimensions()
	// Use a wide rune as mask — width should likely change.
	tv.Mask = 'W'
	tv.Layout(gtx, cache, font.Font{}, unit.Sp(10))
	dimsAfter := tv.Dimensions()
	// We don't assert exact width — just that layout was re-run (no panic).
	if dimsBefore.Size.Y <= 0 || dimsAfter.Size.Y <= 0 {
		t.Errorf("mask change broke layout: %+v / %+v", dimsBefore, dimsAfter)
	}
}

func TestTextView_VeryLongLine(t *testing.T) {
	long := strings.Repeat("abcdef ", 200) // 1400 chars
	tv, _, _ := newTestTextView(t, long)
	if got := tv.Len(); got != utf8.RuneCountInString(long) {
		t.Errorf("Len got %d, want %d", got, utf8.RuneCountInString(long))
	}
	// Walking ByteOffset across the entire range should not panic and should
	// monotonically increase.
	prev := int64(-1)
	for i := 0; i <= tv.Len(); i += 73 {
		off := tv.ByteOffset(i)
		if off < prev {
			t.Errorf("ByteOffset non-monotonic at %d: %d < %d", i, off, prev)
		}
		prev = off
	}
}

func TestTextView_SingleCharacter(t *testing.T) {
	tv, _, _ := newTestTextView(t, "x")
	if tv.Len() != 1 {
		t.Errorf("Len = %d, want 1", tv.Len())
	}
	tv.SetCaret(0, 1)
	if l := tv.SelectionLen(); l != 1 {
		t.Errorf("SelectionLen = %d, want 1", l)
	}
	if got := tv.ByteOffset(1); got != 1 {
		t.Errorf("ByteOffset(1) = %d, want 1", got)
	}
}

func TestTextView_SeekAndRead(t *testing.T) {
	tv, _, _ := newTestTextView(t, "abcdef")
	tv.Seek(2, io.SeekStart) //nolint:errcheck // Seek on in-memory text cannot fail.
	buf := make([]byte, 3)
	n, _ := tv.Read(buf)
	if n != 3 || string(buf[:n]) != "cde" {
		t.Errorf("Seek+Read got %q (n=%d), want \"cde\"", buf[:n], n)
	}
	tv.Seek(0, io.SeekEnd) //nolint:errcheck // Seek on in-memory text cannot fail.
	cur, _ := tv.Seek(0, io.SeekCurrent)
	if cur != 6 {
		t.Errorf("SeekEnd then SeekCurrent: got %d, want 6", cur)
	}
}

func TestTextView_ScrollBoundsMultiLine(t *testing.T) {
	tv := &textView{}
	tv.SetSource(newStringSource("a\nb\nc\nd\ne\nf\ng"))
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(50, 20)), // tiny vertical viewport
		Locale:      english,
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}
	cache := text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	tv.Layout(gtx, cache, font.Font{}, unit.Sp(10))
	b := tv.ScrollBounds()
	if b.Max.Y <= 0 {
		t.Errorf("expected positive vertical scroll range, got %+v", b)
	}
	tv.ScrollRel(0, 5)
	if off := tv.ScrollOff(); off.Y != 5 {
		t.Errorf("ScrollRel Y: got %d, want 5", off.Y)
	}
	tv.ScrollRel(0, 100000)
	if off := tv.ScrollOff(); off.Y > b.Max.Y {
		t.Errorf("ScrollRel exceeded max: %d > %d", off.Y, b.Max.Y)
	}
}

func TestTextView_CaretPosOnEmpty(t *testing.T) {
	tv, _, _ := newTestTextView(t, "")
	line, col := tv.CaretPos()
	if line != 0 || col != 0 {
		t.Errorf("empty CaretPos: %d/%d, want 0/0", line, col)
	}
}

func TestTextView_SetSourceResetsValid(t *testing.T) {
	tv, gtx, cache := newTestTextView(t, "abc")
	if tv.Len() != 3 {
		t.Fatalf("initial Len = %d", tv.Len())
	}
	tv.SetSource(newStringSource("hello world"))
	tv.Layout(gtx, cache, font.Font{}, unit.Sp(10))
	if tv.Len() != 11 {
		t.Errorf("after SetSource Len = %d, want 11", tv.Len())
	}
}

