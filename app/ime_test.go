package app

import (
	"testing"
	"unicode/utf8"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/font"
	"github.com/nanorele/gio/font/gofont"
	"github.com/nanorele/gio/io/input"
	"github.com/nanorele/gio/io/key"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/text"
	"github.com/nanorele/gio/unit"
	"github.com/nanorele/gio/widget"
)

func FuzzIME(f *testing.F) {
	runes := []rune("Hello, 世界! 🤬 علي،الحسنب北查爾斯頓工廠的安全漏洞已")
	f.Add([]byte("20\x0010"))
	f.Add([]byte("80000"))
	f.Add([]byte("2008\"80\r00"))
	f.Add([]byte("20007900002\x02000"))
	f.Add([]byte("20007800002\x02000"))
	f.Add([]byte("200A02000990\x19002\x17\x0200"))
	f.Fuzz(func(t *testing.T, cmds []byte) {
		cache := text.NewShaper(text.WithCollection(gofont.Collection()))
		e := new(widget.Editor)

		var r input.Router
		gtx := layout.Context{Ops: new(op.Ops), Source: r.Source()}
		gtx.Execute(key.FocusCmd{Tag: e})

		e.Layout(gtx, cache, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})
		r.Frame(gtx.Ops)

		var state editorState
		state.Selection.Transform = f32.AffineId()
		const (
			cmdReplace = iota
			cmdSelect
			cmdSnip
			maxCmd
		)
		const cmdLen = 5
		for len(cmds) >= cmdLen {
			n := e.Len()
			rng := key.Range{
				Start: int(cmds[1]) % (n + 1),
				End:   int(cmds[2]) % (n + 1),
			}
			switch cmds[0] % cmdLen {
			case cmdReplace:
				rstart := int(cmds[3]) % len(runes)
				rend := int(cmds[4]) % len(runes)
				if rstart > rend {
					rstart, rend = rend, rstart
				}
				replacement := string(runes[rstart:rend])
				state.Replace(rng, replacement)
				r.Queue(key.EditEvent{Range: rng, Text: replacement})
				r.Queue(key.SnippetEvent(state.Snippet.Range))
			case cmdSelect:
				r.Queue(key.SelectionEvent(rng))
				runes := []rune(e.Text())
				if rng.Start < 0 {
					rng.Start = 0
				}
				if rng.End < 0 {
					rng.End = 0
				}
				if rng.Start > len(runes) {
					rng.Start = len(runes)
				}
				if rng.End > len(runes) {
					rng.End = len(runes)
				}
				state.Selection.Range = rng
			case cmdSnip:
				r.Queue(key.SnippetEvent(rng))
				runes := []rune(e.Text())
				if rng.Start > rng.End {
					rng.Start, rng.End = rng.End, rng.Start
				}
				if rng.Start < 0 {
					rng.Start = 0
				}
				if rng.End < 0 {
					rng.End = 0
				}
				if rng.Start > len(runes) {
					rng.Start = len(runes)
				}
				if rng.End > len(runes) {
					rng.End = len(runes)
				}
				state.Snippet = key.Snippet{
					Range: rng,
					Text:  string(runes[rng.Start:rng.End]),
				}
			}
			cmds = cmds[cmdLen:]
			e.Layout(gtx, cache, font.Font{}, unit.Sp(10), op.CallOp{}, op.CallOp{})
			r.Frame(gtx.Ops)
			newState := r.EditorState()

			state.Selection.Caret = newState.Selection.Caret

			their, our := newState.Snippet, state.EditorState.Snippet
			beforeLen := 0
			for before := our.Start - their.Start; before > 0; before-- {
				_, n := utf8.DecodeRuneInString(their.Text[beforeLen:])
				beforeLen += n
			}
			afterLen := 0
			for after := their.End - our.End; after > 0; after-- {
				_, n := utf8.DecodeLastRuneInString(their.Text[:len(their.Text)-afterLen])
				afterLen += n
			}
			if beforeLen > 0 {
				our.Text = their.Text[:beforeLen] + our.Text
				our.Start = their.Start
			}
			if afterLen > 0 {
				our.Text = our.Text + their.Text[len(their.Text)-afterLen:]
				our.End = their.End
			}
			state.EditorState.Snippet = our
			if newState != state.EditorState {
				t.Errorf("IME state: %+v\neditor state: %+v", state.EditorState, newState)
			}
		}
	})
}

func TestEditorIndices(t *testing.T) {
	var s editorState
	s.Selection.Transform = f32.AffineId()
	const str = "Hello, 😀"
	s.Snippet = key.Snippet{
		Text: str,
		Range: key.Range{
			Start: 10,
			End:   utf8.RuneCountInString(str),
		},
	}
	utf16Indices := [...]struct {
		Runes, UTF16 int
	}{
		{0, 0}, {10, 10}, {17, 17}, {18, 19}, {30, 31},
	}
	for _, p := range utf16Indices {
		if want, got := p.UTF16, s.UTF16Index(p.Runes); want != got {
			t.Errorf("UTF16Index(%d) = %d, wanted %d", p.Runes, got, want)
		}
		if want, got := p.Runes, s.RunesIndex(p.UTF16); want != got {
			t.Errorf("RunesIndex(%d) = %d, wanted %d", p.UTF16, got, want)
		}
	}
}

func newEditorState(text string, start int) *editorState {
	s := &editorState{}
	s.compose.Start = -1
	s.compose.End = -1
	s.Snippet = key.Snippet{
		Text: text,
		Range: key.Range{
			Start: start,
			End:   start + utf8.RuneCountInString(text),
		},
	}
	return s
}

func TestReplaceInsertAtEnd(t *testing.T) {
	s := newEditorState("abc", 0)
	s.Selection.Start, s.Selection.End = 3, 3
	s.Replace(key.Range{Start: 3, End: 3}, "de")
	if s.Snippet.Text != "abcde" {
		t.Errorf("Snippet.Text = %q, want abcde", s.Snippet.Text)
	}
	if s.Snippet.Start != 0 || s.Snippet.End != 5 {
		t.Errorf("Snippet range = [%d,%d], want [0,5]", s.Snippet.Start, s.Snippet.End)
	}
	// adjust() only shifts positions strictly past r.End; selection at r.End stays put.
	if s.Selection.Start != 3 || s.Selection.End != 3 {
		t.Errorf("Selection = [%d,%d], want [3,3]", s.Selection.Start, s.Selection.End)
	}
}

func TestReplaceInsertAtStart(t *testing.T) {
	s := newEditorState("abc", 0)
	s.Selection.Start, s.Selection.End = 1, 1
	s.Replace(key.Range{Start: 0, End: 0}, "XY")
	if s.Snippet.Text != "XYabc" {
		t.Errorf("Snippet.Text = %q, want XYabc", s.Snippet.Text)
	}
	// Selection should shift by the inserted length.
	if s.Selection.Start != 3 || s.Selection.End != 3 {
		t.Errorf("Selection = [%d,%d], want [3,3]", s.Selection.Start, s.Selection.End)
	}
}

func TestReplaceDeleteRangeContainingSelection(t *testing.T) {
	s := newEditorState("abcdef", 0)
	s.Selection.Start, s.Selection.End = 2, 4
	s.Replace(key.Range{Start: 1, End: 5}, "")
	// After replace, selection collapses to the new end position (1).
	if s.Selection.Start != 1 || s.Selection.End != 1 {
		t.Errorf("Selection = [%d,%d], want [1,1]", s.Selection.Start, s.Selection.End)
	}
	if s.Snippet.Text != "af" {
		t.Errorf("Snippet.Text = %q, want af", s.Snippet.Text)
	}
}

func TestReplaceSwapsReversedRange(t *testing.T) {
	s := newEditorState("abcdef", 0)
	s.Selection.Start, s.Selection.End = 0, 0
	// End < Start should be swapped.
	s.Replace(key.Range{Start: 4, End: 1}, "XX")
	if s.Snippet.Text != "aXXef" {
		t.Errorf("Snippet.Text = %q, want aXXef", s.Snippet.Text)
	}
}

func TestReplaceUpdatesCompose(t *testing.T) {
	s := newEditorState("abcdef", 0)
	s.compose.Start, s.compose.End = 2, 4
	s.Replace(key.Range{Start: 0, End: 0}, "XX")
	// Compose range should shift by +2.
	if s.compose.Start != 4 || s.compose.End != 6 {
		t.Errorf("compose = [%d,%d], want [4,6]", s.compose.Start, s.compose.End)
	}
}

func TestReplaceUnicodeRunes(t *testing.T) {
	s := newEditorState("ab", 0)
	s.Selection.Start, s.Selection.End = 2, 2
	s.Replace(key.Range{Start: 2, End: 2}, "😀世")
	// Snippet.End is in runes, so 2 + 2 runes = 4.
	if s.Snippet.End != 4 {
		t.Errorf("Snippet.End = %d, want 4", s.Snippet.End)
	}
	if s.Snippet.Text != "ab😀世" {
		t.Errorf("Snippet.Text = %q, want ab😀世", s.Snippet.Text)
	}
}

func TestReplaceOutsideSnippetResets(t *testing.T) {
	s := newEditorState("abc", 10) // snippet at runes [10,13]
	s.Selection.Start, s.Selection.End = 10, 10
	// Replace far outside the snippet — should reset snippet to a fresh window.
	s.Replace(key.Range{Start: 100, End: 100}, "X")
	if s.Snippet.Text != "X" {
		t.Errorf("Snippet.Text = %q, want X", s.Snippet.Text)
	}
	if s.Snippet.Start != 100 || s.Snippet.End != 101 {
		t.Errorf("Snippet range = [%d,%d], want [100,101]", s.Snippet.Start, s.Snippet.End)
	}
}

func TestUTF16IndexAndRunesIndexNegOne(t *testing.T) {
	s := newEditorState("hello", 0)
	if got := s.UTF16Index(-1); got != -1 {
		t.Errorf("UTF16Index(-1) = %d, want -1", got)
	}
	if got := s.RunesIndex(-1); got != -1 {
		t.Errorf("RunesIndex(-1) = %d, want -1", got)
	}
}

func TestUTF16IndexBeforeSnippet(t *testing.T) {
	s := newEditorState("xyz", 10)
	// For positions before the snippet, the function returns the input unchanged.
	if got := s.UTF16Index(5); got != 5 {
		t.Errorf("UTF16Index(5) = %d, want 5", got)
	}
	if got := s.RunesIndex(5); got != 5 {
		t.Errorf("RunesIndex(5) = %d, want 5", got)
	}
}

func TestUTF16IndexAfterSurrogatePair(t *testing.T) {
	// 😀 is a surrogate pair in UTF-16 (2 chars), 1 rune.
	s := newEditorState("a😀b", 0)
	// rune index 0 -> utf16 0
	// rune index 1 -> utf16 1 (after 'a')
	// rune index 2 -> utf16 3 (after surrogate pair)
	// rune index 3 -> utf16 4 (after 'b')
	cases := []struct{ runes, utf16 int }{
		{0, 0}, {1, 1}, {2, 3}, {3, 4},
	}
	for _, c := range cases {
		if got := s.UTF16Index(c.runes); got != c.utf16 {
			t.Errorf("UTF16Index(%d) = %d, want %d", c.runes, got, c.utf16)
		}
		if got := s.RunesIndex(c.utf16); got != c.runes {
			t.Errorf("RunesIndex(%d) = %d, want %d", c.utf16, got, c.runes)
		}
	}
}

func TestUTF16IndexBeyondSnippet(t *testing.T) {
	// Indices past the snippet's end should be returned offset by the snippet length.
	s := newEditorState("ab", 5) // runes 5..7
	// rune 7 -> utf16 7 (no surrogate). rune 10 -> utf16 10.
	if got := s.UTF16Index(10); got != 10 {
		t.Errorf("UTF16Index(10) = %d, want 10", got)
	}
}

func TestSnippetSubstringFullRange(t *testing.T) {
	s := key.Snippet{Text: "abcdef", Range: key.Range{Start: 0, End: 6}}
	got := snippetSubstring(s, key.Range{Start: 0, End: 6})
	if got != "abcdef" {
		t.Errorf("snippetSubstring full = %q, want abcdef", got)
	}
}

func TestSnippetSubstringPartial(t *testing.T) {
	s := key.Snippet{Text: "abcdef", Range: key.Range{Start: 0, End: 6}}
	got := snippetSubstring(s, key.Range{Start: 2, End: 5})
	if got != "cde" {
		t.Errorf("snippetSubstring [2,5] = %q, want cde", got)
	}
}

func TestSnippetSubstringWithUnicode(t *testing.T) {
	s := key.Snippet{Text: "a😀b", Range: key.Range{Start: 0, End: 3}}
	got := snippetSubstring(s, key.Range{Start: 1, End: 2})
	if got != "😀" {
		t.Errorf("snippetSubstring rune range = %q, want 😀", got)
	}
}

func TestSnippetSubstringOffsetStart(t *testing.T) {
	s := key.Snippet{Text: "abcdef", Range: key.Range{Start: 10, End: 16}}
	got := snippetSubstring(s, key.Range{Start: 12, End: 15})
	if got != "cde" {
		t.Errorf("snippetSubstring offset = %q, want cde", got)
	}
}

func TestAreSnippetsConsistentSame(t *testing.T) {
	a := key.Snippet{Text: "abcdef", Range: key.Range{Start: 0, End: 6}}
	if !areSnippetsConsistent(a, a) {
		t.Errorf("snippets identical to themselves should be consistent")
	}
}

func TestAreSnippetsConsistentOverlap(t *testing.T) {
	old := key.Snippet{Text: "abcdef", Range: key.Range{Start: 0, End: 6}}
	new := key.Snippet{Text: "cdefgh", Range: key.Range{Start: 2, End: 8}}
	if !areSnippetsConsistent(old, new) {
		t.Errorf("overlapping consistent snippets should be consistent")
	}
}

func TestAreSnippetsConsistentDiverge(t *testing.T) {
	old := key.Snippet{Text: "abcdef", Range: key.Range{Start: 0, End: 6}}
	new := key.Snippet{Text: "XYZdef", Range: key.Range{Start: 0, End: 6}}
	if areSnippetsConsistent(old, new) {
		t.Errorf("divergent snippets should NOT be consistent")
	}
}

func TestAreSnippetsConsistentNoOverlap(t *testing.T) {
	// Disjoint ranges: intersection is empty, both yield empty string => consistent.
	old := key.Snippet{Text: "abc", Range: key.Range{Start: 0, End: 3}}
	new := key.Snippet{Text: "xyz", Range: key.Range{Start: 10, End: 13}}
	if !areSnippetsConsistent(old, new) {
		t.Errorf("non-overlapping snippets should be trivially consistent (empty intersection)")
	}
}
