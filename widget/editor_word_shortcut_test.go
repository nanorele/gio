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

type wordShortcutRig struct {
	r     *input.Router
	gtx   layout.Context
	e     *Editor
	cache *text.Shaper
}

func newWordShortcutRig(t *testing.T, content string) *wordShortcutRig {
	t.Helper()
	if key.ModShortcutAlt != key.ModShortcut {
		t.Skip("word chords share the shortcut modifier only on this platform")
	}
	r := new(input.Router)
	rig := &wordShortcutRig{
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
	rig.e.SetText(content)
	rig.gtx.Execute(key.FocusCmd{Tag: rig.e})
	for range 3 {
		rig.frame()
	}
	return rig
}

func (rig *wordShortcutRig) frame() {
	rig.e.Layout(rig.gtx, rig.cache, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})
	rig.r.Frame(rig.gtx.Ops)
	rig.gtx.Ops.Reset()
}

func (rig *wordShortcutRig) press(name key.Name, mods key.Modifiers) {
	rig.r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Press})
	rig.frame()
}

func TestEditorShortcutArrowMovesByWord(t *testing.T) {
	rig := newWordShortcutRig(t, "alpha beta gamma")
	rig.e.SetCaret(16, 16)
	rig.frame()

	rig.press(key.NameLeftArrow, key.ModShortcut)
	if s, e := rig.e.Selection(); s != 11 || e != 11 {
		t.Errorf("shortcut+Left: caret = (%d,%d), want (11,11)", s, e)
	}

	rig.e.SetCaret(0, 0)
	rig.frame()
	rig.press(key.NameRightArrow, key.ModShortcut)
	if s, e := rig.e.Selection(); s != 5 || e != 5 {
		t.Errorf("shortcut+Right: caret = (%d,%d), want (5,5)", s, e)
	}
}

func TestEditorShortcutShiftArrowExtendsByWord(t *testing.T) {
	rig := newWordShortcutRig(t, "alpha beta gamma")
	rig.e.SetCaret(16, 16)
	rig.frame()

	rig.press(key.NameLeftArrow, key.ModShortcut|key.ModShift)
	s, e := rig.e.Selection()
	lo, hi := s, e
	if lo > hi {
		lo, hi = hi, lo
	}
	if lo != 11 || hi != 16 {
		t.Errorf("shortcut+Shift+Left: selection = (%d,%d), want covering [11,16]", s, e)
	}
}

func TestEditorShortcutDeleteRemovesWord(t *testing.T) {
	rig := newWordShortcutRig(t, "alpha beta gamma")
	rig.e.SetCaret(16, 16)
	rig.frame()
	rig.press(key.NameDeleteBackward, key.ModShortcut)
	if got := rig.e.Text(); got != "alpha beta " {
		t.Errorf("shortcut+Backspace: text = %q, want %q", got, "alpha beta ")
	}

	rig = newWordShortcutRig(t, "alpha beta gamma")
	rig.e.SetCaret(0, 0)
	rig.frame()
	rig.press(key.NameDeleteForward, key.ModShortcut)
	if got := rig.e.Text(); got != " beta gamma" {
		t.Errorf("shortcut+Delete: text = %q, want %q", got, " beta gamma")
	}
}

func TestEditorShortcutClipboardChordsStillHandled(t *testing.T) {
	rig := newWordShortcutRig(t, "alpha beta gamma")
	rig.press("A", key.ModShortcut)
	s, e := rig.e.Selection()
	if s > e {
		s, e = e, s
	}
	if s != 0 || e != rig.e.Len() {
		t.Errorf("shortcut+A: selection = (%d,%d), want (0,%d)", s, e, rig.e.Len())
	}
}
