package widget

import (
	"io"
	"strings"
	"testing"
)

func readAll(t *testing.T, b *editBuffer) string {
	t.Helper()
	if b.Size() == 0 {
		return ""
	}
	out := make([]byte, b.Size())
	n, err := b.ReadAt(out, 0)
	if err != nil && err != io.EOF {
		t.Fatalf("ReadAt: %v", err)
	}
	return string(out[:n])
}

func TestEditBufferEmpty(t *testing.T) {
	var b editBuffer
	if got := b.Size(); got != 0 {
		t.Errorf("Size() = %d, want 0", got)
	}
	if b.Changed() {
		t.Error("zero buffer should not report Changed")
	}
	// ReadAt at offset 0 of empty buffer should report EOF.
	buf := make([]byte, 4)
	n, err := b.ReadAt(buf, 0)
	if n != 0 {
		t.Errorf("expected n=0, got %d", n)
	}
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestEditBufferReplaceRunesInsert(t *testing.T) {
	var b editBuffer
	b.ReplaceRunes(0, 0, "hello")
	if got := readAll(t, &b); got != "hello" {
		t.Errorf("got %q want %q", got, "hello")
	}
	if !b.Changed() {
		t.Error("Changed should be true after insert")
	}
	if b.Changed() {
		t.Error("Changed should reset after first call")
	}
}

func TestEditBufferReplaceRunesAppend(t *testing.T) {
	var b editBuffer
	b.ReplaceRunes(0, 0, "abc")
	b.ReplaceRunes(b.Size(), 0, "def")
	if got := readAll(t, &b); got != "abcdef" {
		t.Errorf("got %q want %q", got, "abcdef")
	}
}

func TestEditBufferReplaceRunesPrepend(t *testing.T) {
	var b editBuffer
	b.ReplaceRunes(0, 0, "world")
	b.ReplaceRunes(0, 0, "hello ")
	if got := readAll(t, &b); got != "hello world" {
		t.Errorf("got %q want %q", got, "hello world")
	}
}

func TestEditBufferReplaceRunesDeleteForward(t *testing.T) {
	var b editBuffer
	b.ReplaceRunes(0, 0, "abcdef")
	b.Changed() // reset

	b.ReplaceRunes(2, 2, "")
	if got := readAll(t, &b); got != "abef" {
		t.Errorf("got %q want %q", got, "abef")
	}
	if !b.Changed() {
		t.Error("Changed should be true after delete")
	}
}

func TestEditBufferReplaceRunesDeleteBackward(t *testing.T) {
	var b editBuffer
	b.ReplaceRunes(0, 0, "abcdef")
	b.Changed()

	// Negative runeCount deletes runes before the caret.
	b.ReplaceRunes(4, -2, "")
	if got := readAll(t, &b); got != "abef" {
		t.Errorf("got %q want %q", got, "abef")
	}
}

func TestEditBufferReplaceRunesReplace(t *testing.T) {
	var b editBuffer
	b.ReplaceRunes(0, 0, "abcdef")
	b.ReplaceRunes(2, 2, "XYZ")
	if got := readAll(t, &b); got != "abXYZef" {
		t.Errorf("got %q want %q", got, "abXYZef")
	}
}

func TestEditBufferUTF8Multibyte(t *testing.T) {
	var b editBuffer
	b.ReplaceRunes(0, 0, "héllo") // é is 2 bytes
	if got := b.Size(); got != int64(len("héllo")) {
		t.Errorf("Size = %d want %d", got, len("héllo"))
	}
	// "h" is at byte 0, "é" at byte 1..2, "l" at byte 3.
	// Delete the é (one rune starting at byte offset 1).
	b.ReplaceRunes(1, 1, "")
	if got := readAll(t, &b); got != "hllo" {
		t.Errorf("got %q want %q", got, "hllo")
	}
}

func TestEditBufferInvalidUTF8Replaced(t *testing.T) {
	var b editBuffer
	b.ReplaceRunes(0, 0, "a\xffb")
	got := readAll(t, &b)
	if strings.Contains(got, "\xff") {
		t.Errorf("invalid byte should be replaced, got %q", got)
	}
	if !strings.HasPrefix(got, "a") || !strings.HasSuffix(got, "b") {
		t.Errorf("wrapping content lost, got %q", got)
	}
}

func TestEditBufferReadAtShortRead(t *testing.T) {
	var b editBuffer
	b.ReplaceRunes(0, 0, "hi")
	// io.ReaderAt requires a non-nil error when n < len(p).
	buf := make([]byte, 10)
	n, err := b.ReadAt(buf, 0)
	if n != 2 {
		t.Errorf("got n=%d want 2", n)
	}
	if string(buf[:n]) != "hi" {
		t.Errorf("got %q want %q", string(buf[:n]), "hi")
	}
	if err != io.EOF {
		t.Errorf("short read should return io.EOF, got %v", err)
	}
}

func TestEditBufferReadAtBeyondSize(t *testing.T) {
	var b editBuffer
	b.ReplaceRunes(0, 0, "abc")
	buf := make([]byte, 4)
	n, err := b.ReadAt(buf, 100)
	if n != 0 || err != io.EOF {
		t.Errorf("read past end: got n=%d err=%v want 0,EOF", n, err)
	}
}

func TestEditBufferReadAtOffsets(t *testing.T) {
	var b editBuffer
	b.ReplaceRunes(0, 0, "hello world")

	buf := make([]byte, 5)
	n, err := b.ReadAt(buf, 6)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(buf[:n]) != "world" {
		t.Errorf("got %q want %q", string(buf[:n]), "world")
	}

	// Read at exactly Size returns EOF.
	n, err = b.ReadAt(buf, b.Size())
	if n != 0 || err != io.EOF {
		t.Errorf("read at Size: got n=%d err=%v want 0,EOF", n, err)
	}

	// Empty p never returns EOF.
	n, err = b.ReadAt(nil, 0)
	if n != 0 || err != nil {
		t.Errorf("read with empty p: got n=%d err=%v want 0,nil", n, err)
	}
}

func TestEditBufferReadAtSpansGap(t *testing.T) {
	var b editBuffer
	b.ReplaceRunes(0, 0, "0123456789")
	// Force the gap somewhere in the middle by inserting then deleting.
	b.ReplaceRunes(5, 0, "X")
	b.ReplaceRunes(5, 1, "")
	// Now the buffer text is "0123456789" but the gap is internal.
	out := make([]byte, 10)
	n, err := b.ReadAt(out, 0)
	if err != nil && err != io.EOF {
		t.Fatalf("ReadAt error: %v", err)
	}
	if string(out[:n]) != "0123456789" {
		t.Errorf("got %q want %q", string(out[:n]), "0123456789")
	}
}

func TestEditBufferChangedFlagOnNoOp(t *testing.T) {
	var b editBuffer
	// Pure no-op ReplaceRunes (delete 0, insert nothing) should not flip Changed.
	b.ReplaceRunes(0, 0, "")
	if b.Changed() {
		t.Error("no-op ReplaceRunes should not set Changed")
	}
}

func TestEditBufferDeleteAcrossGap(t *testing.T) {
	var b editBuffer
	b.ReplaceRunes(0, 0, "abcdefghij")
	// Move gap by inserting in the middle, then issue a delete that crosses it.
	b.ReplaceRunes(3, 0, "")  // touches gap at 3
	b.ReplaceRunes(2, 4, "_") // delete 4 runes from offset 2: "cdef" -> "_"
	if got := readAll(t, &b); got != "ab_ghij" {
		t.Errorf("got %q want %q", got, "ab_ghij")
	}
}
