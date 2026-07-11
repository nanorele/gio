package widget

import (
	"image"
	"strings"
	"testing"

	"github.com/nanorele/gio/font"
	"github.com/nanorele/gio/font/gofont"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/text"
	"github.com/nanorele/gio/unit"
	"golang.org/x/image/math/fixed"
)

// TestEditorSingleLineScrollsToCaret ensures a single-line editor keeps the
// caret visible by scrolling horizontally. The SingleLine shaping width
// (1<<24) must never leak into viewSize: that collapses ScrollBounds to
// zero, freezing the viewport at scroll offset 0 while the caret moves past
// the right edge.
func TestEditorSingleLineScrollsToCaret(t *testing.T) {
	e := new(Editor)
	e.SingleLine = true
	e.SetText(strings.Repeat("abc def ", 30))
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(100, 20)),
		Locale:      english,
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}
	cache := text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	e.Layout(gtx, cache, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})

	if got := e.text.viewSize.X; got != 100 {
		t.Fatalf("viewSize.X = %d, want 100 (widget width); shaping width leaked into the viewport", got)
	}
	if b := e.text.ScrollBounds(); b.Max.X <= 0 {
		t.Fatalf("ScrollBounds = %v, want positive Max.X for overflowing single-line text", b)
	}

	e.SetCaret(e.Len(), e.Len())
	gtx.Ops.Reset()
	e.Layout(gtx, cache, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})

	if got := e.text.ScrollOff().X; got <= 0 {
		t.Errorf("scroll offset = %d after moving caret to the end; viewport never scrolled, caret is invisible", got)
	}
	if coords := e.CaretCoords(); coords.X < 0 || coords.X > 100 {
		t.Errorf("caret X = %v, want within viewport [0, 100]", coords.X)
	}
}

// TestEditorUndoAfterReadOnlyEdit ensures that edits applied while the
// editor is read-only do not leave the undo history pointing at stale text.
// Recording is skipped in read-only mode for performance, so the history
// must be dropped; otherwise undo replays old modifications against the new
// text and corrupts it.
func TestEditorUndoAfterReadOnlyEdit(t *testing.T) {
	e := new(Editor)
	e.Insert("hello")
	e.ReadOnly = true
	e.SetText("universe")
	e.ReadOnly = false
	e.undo()
	if got := e.Text(); got != "universe" {
		t.Errorf("undo after read-only SetText produced %q, want %q (stale history applied)", got, "universe")
	}
}

// TestEditorSetTextResetsScroll ensures SetText returns the viewport to the
// caret (rune 0), matching upstream behavior: after replacing the content of
// a scrolled editor the user must see the start of the new text, not a
// stale offset into the middle of it.
func TestEditorSetTextResetsScroll(t *testing.T) {
	e := new(Editor)
	e.SetText(strings.Repeat("line\n", 100))
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(100, 20)),
		Locale:      english,
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}
	cache := text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	e.Layout(gtx, cache, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})
	e.SetScrollY(500)
	if e.GetScrollY() == 0 {
		t.Fatal("test setup failed: editor did not scroll")
	}

	e.SetText(strings.Repeat("other\n", 100))
	gtx.Ops.Reset()
	e.Layout(gtx, cache, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})
	if got := e.GetScrollY(); got != 0 {
		t.Errorf("scroll offset = %d after SetText, want 0 (caret is at rune 0)", got)
	}
}

// TestGlyphIndexLastLinePosEnd ensures the final caret position of the text
// (after the last rune) is included in the last line's position range. The
// soft-break posEnd decrement assumes a following line overwrites the
// trailing slot; on the last line no further glyphs arrive, so without a
// final fix-up closestToXY can never return the end-of-text position.
func TestGlyphIndexLastLinePosEnd(t *testing.T) {
	tv, _, _ := newTestTextView(t, "hello")
	lines := tv.index.lines
	if len(lines) == 0 {
		t.Fatal("no lines indexed")
	}
	last := lines[len(lines)-1]
	if got, want := last.posEnd, len(tv.index.positions); got != want {
		t.Errorf("last line posEnd = %d, want %d: end-of-text caret position excluded from its line", got, want)
	}

	baseline := tv.index.positions[0].y
	pos, _ := tv.index.closestToXY(fixed.I(10000), baseline)
	if pos.runes != tv.Len() {
		t.Errorf("closestToXY far right of last line returned rune %d, want %d (end of text)", pos.runes, tv.Len())
	}
}
