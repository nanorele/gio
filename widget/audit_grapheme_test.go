package widget

import (
	"fmt"
	"strings"
	"testing"
)

// TestAuditGraphemeAcrossRefill drives the grapheme index over texts whose
// cluster boundaries land on and around the buffered reader's refill points.
// A refill that loses or splits a rune would shift every later cluster.
func TestAuditGraphemeAcrossRefill(t *testing.T) {
	const buf = graphemeReaderBufSize

	cases := []struct {
		name string
		txt  string
	}{
		{"ascii-long-word", strings.Repeat("w", 10*buf)},
		{"multibyte-2", strings.Repeat("é", 4*buf)},
		{"multibyte-3", strings.Repeat("世", 3*buf)},
		{"multibyte-4", strings.Repeat("🙂", 2*buf)},
		{"combining", strings.Repeat("é", 3*buf)},
		{"zwj-family", strings.Repeat("👩‍👩‍👧", 40)},
		{"flags", strings.Repeat("🇺🇦", 200)},
		{"mixed", strings.Repeat("ab é 世 🙂 ", 200)},
	}
	// Pad so a cluster starts at every offset around each refill boundary.
	for _, pad := range []int{0, 1, 2, 3, buf - 2, buf - 1, buf, buf + 1} {
		for _, c := range cases {
			txt := strings.Repeat("x", pad) + c.txt
			src := &testSource{txt: txt}
			var rd graphemeReader
			rd.SetSource(src)
			got := rd.Graphemes()

			// Reference: a reader that has never been reused, over the same
			// source, so any state carried across Reset shows up as a diff.
			fresh := &graphemeReader{}
			fresh.SetSource(&testSource{txt: txt})
			want := fresh.Graphemes()

			if len(got) != len(want) {
				t.Fatalf("%s pad=%d: %d boundaries, want %d", c.name, pad, len(got), len(want))
			}
			for i := range got {
				if got[i] != want[i] {
					t.Fatalf("%s pad=%d: boundary %d = %d, want %d", c.name, pad, i, got[i], want[i])
				}
			}
			// Boundaries must be strictly increasing and land on rune starts.
			for i := 1; i < len(got); i++ {
				if got[i] <= got[i-1] {
					t.Fatalf("%s pad=%d: boundary %d not increasing (%d after %d)",
						c.name, pad, i, got[i], got[i-1])
				}
			}
		}
	}
	fmt.Printf("grapheme reader: %d texts x 8 refill alignments consistent\n", len(cases))
}

// TestAuditGraphemeReaderReuse checks that reusing one reader across many
// sources gives the same answers as a fresh reader each time.
func TestAuditGraphemeReaderReuse(t *testing.T) {
	var reused graphemeReader
	texts := []string{
		"short",
		strings.Repeat("long ", 500),
		"héllo wörld",
		strings.Repeat("🙂", 300),
		"",
		strings.Repeat("a", graphemeReaderBufSize),
		strings.Repeat("b", graphemeReaderBufSize+1),
	}
	for round := 0; round < 3; round++ {
		for i, txt := range texts {
			reused.SetSource(&testSource{txt: txt})
			got := reused.Graphemes()

			fresh := &graphemeReader{}
			fresh.SetSource(&testSource{txt: txt})
			want := fresh.Graphemes()

			if len(got) != len(want) {
				t.Fatalf("round %d text %d: %d boundaries, want %d", round, i, len(got), len(want))
			}
			for k := range got {
				if got[k] != want[k] {
					t.Fatalf("round %d text %d: boundary %d differs", round, i, k)
				}
			}
		}
	}
	fmt.Printf("grapheme reader: %d reuse rounds consistent\n", 3*len(texts))
}

type testSource struct{ txt string }

func (s *testSource) ReadAt(b []byte, off int64) (int, error) {
	if off >= int64(len(s.txt)) {
		return 0, errAuditEOF
	}
	n := copy(b, s.txt[off:])
	if int(off)+n >= len(s.txt) {
		return n, errAuditEOF
	}
	return n, nil
}

var errAuditEOF = eofError{}

type eofError struct{}

func (eofError) Error() string { return "EOF" }
