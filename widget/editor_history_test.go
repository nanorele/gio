package widget

import (
	"image"
	"testing"

	"github.com/nanorele/gio/font"
	"github.com/nanorele/gio/font/gofont"
	"github.com/nanorele/gio/io/input"
	"github.com/nanorele/gio/io/key"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/text"
	"github.com/nanorele/gio/unit"
)

type historyRig struct {
	r     *input.Router
	gtx   layout.Context
	e     *Editor
	cache *text.Shaper
}

func newHistoryRig() *historyRig {
	r := new(input.Router)
	rig := &historyRig{
		r: r,
		gtx: layout.Context{
			Ops:         new(op.Ops),
			Constraints: layout.Constraints{Max: image.Pt(300, 100)},
			Locale:      english,
			Source:      r.Source(),
		},
		e:     new(Editor),
		cache: text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection())),
	}
	return rig
}

func (rig *historyRig) frame() {
	rig.e.Layout(rig.gtx, rig.cache, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})
	rig.r.Frame(rig.gtx.Ops)
	rig.gtx.Ops.Reset()
}

func (rig *historyRig) focus() {
	rig.gtx.Execute(key.FocusCmd{Tag: rig.e})
	for range 3 {
		rig.frame()
	}
}

func (rig *historyRig) press(name key.Name, mods key.Modifiers) {
	rig.r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Press})
	rig.frame()
}

func TestEditorSetTextOnFreshEditorIsNotUndoable(t *testing.T) {
	rig := newHistoryRig()
	rig.e.SetText("123")
	rig.focus()

	rig.press("Z", key.ModShortcut)
	if got := rig.e.Text(); got != "123" {
		t.Fatalf("Ctrl+Z right after loading text must keep it, got %q", got)
	}
}

func TestEditorTypedEditsUndoBackToLoadedText(t *testing.T) {
	rig := newHistoryRig()
	rig.e.SetText("123")
	rig.focus()
	rig.e.SetCaret(3, 3)
	rig.frame()

	rig.r.Queue(key.EditEvent{Range: key.Range{Start: 3, End: 3}, Text: "4"})
	rig.frame()
	if got := rig.e.Text(); got != "1234" {
		t.Fatalf("setup: got %q", got)
	}

	rig.press("Z", key.ModShortcut)
	if got := rig.e.Text(); got != "123" {
		t.Fatalf("undo should revert to the loaded text, got %q", got)
	}
	rig.press("Z", key.ModShortcut)
	if got := rig.e.Text(); got != "123" {
		t.Fatalf("a second undo must stop at the loaded text, got %q", got)
	}
}

func TestEditorSetTextWhileEditingStaysUndoable(t *testing.T) {
	rig := newHistoryRig()
	rig.e.SetText("abc")
	rig.focus()
	rig.e.SetCaret(3, 3)
	rig.frame()
	rig.r.Queue(key.EditEvent{Range: key.Range{Start: 3, End: 3}, Text: "d"})
	rig.frame()

	rig.e.SetText("replaced")
	rig.frame()
	rig.press("Z", key.ModShortcut)
	if got := rig.e.Text(); got != "abcd" {
		t.Fatalf("programmatic replacement during a session should undo to the previous text, got %q", got)
	}
}

func TestEditorCtrlYRedoes(t *testing.T) {
	rig := newHistoryRig()
	rig.e.SetText("abc")
	rig.focus()
	rig.e.SetCaret(3, 3)
	rig.frame()
	rig.r.Queue(key.EditEvent{Range: key.Range{Start: 3, End: 3}, Text: "d"})
	rig.frame()

	rig.press("Z", key.ModShortcut)
	if got := rig.e.Text(); got != "abc" {
		t.Fatalf("undo: got %q", got)
	}
	rig.press("Y", key.ModShortcut)
	if got := rig.e.Text(); got != "abcd" {
		t.Fatalf("Ctrl+Y should redo, got %q", got)
	}
}

func TestEditorUndoRedoLeaveCollapsedCaret(t *testing.T) {
	rig := newHistoryRig()
	rig.e.SetText("abc")
	rig.focus()
	rig.e.SetCaret(3, 3)
	rig.frame()
	rig.r.Queue(key.EditEvent{Range: key.Range{Start: 3, End: 3}, Text: "def"})
	rig.frame()
	rig.e.SetCaret(1, 3)
	rig.frame()
	rig.r.Queue(key.EditEvent{Range: key.Range{Start: 1, End: 3}, Text: ""})
	rig.frame()
	if got := rig.e.Text(); got != "adef" {
		t.Fatalf("setup: got %q", got)
	}

	rig.press("Z", key.ModShortcut)
	if s, e := rig.e.Selection(); s != e || s != 3 {
		t.Fatalf("undo of a deletion must leave a collapsed caret after the restored text, got (%d,%d)", s, e)
	}
	rig.press("Y", key.ModShortcut)
	if got := rig.e.Text(); got != "adef" {
		t.Fatalf("redo: got %q", got)
	}
	if s, e := rig.e.Selection(); s != e {
		t.Fatalf("redo must not select the reapplied text, got (%d,%d)", s, e)
	}
	rig.press("Z", key.ModShortcut)
	rig.press("Z", key.ModShortcut)
	rig.press("Y", key.ModShortcut)
	if s, e := rig.e.Selection(); s != e || s != 6 {
		t.Fatalf("redo of an insertion must leave the caret after it, got (%d,%d)", s, e)
	}
}
