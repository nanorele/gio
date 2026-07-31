package widget

import (
	"image"
	"runtime"
	"testing"

	"github.com/nanorele/gio/font"
	"github.com/nanorele/gio/font/gofont"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/text"
	"github.com/nanorele/gio/unit"
)

// editorSink keeps measured editors alive until process exit so that
// -memprofile attributes their retained memory.
var editorSink [][]*Editor

// measureRetained reports retained bytes per editor for n editors after
// running stage on each.
func measureRetained(t *testing.T, n int, stage func(e *Editor)) float64 {
	t.Helper()
	editors := make([]*Editor, n)
	for i := range editors {
		editors[i] = new(Editor)
	}
	editorSink = append(editorSink, editors)
	runtime.GC()
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	for _, e := range editors {
		stage(e)
	}
	runtime.GC()
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(editors)
	return float64(after.HeapAlloc-before.HeapAlloc) / float64(n)
}

// TestEditorStageCosts prints the retained cost of an editor by stage. Run
// with -v to see the numbers; it fails only on the hard acceptance bound for
// SetText-only editors (T3: display-only cells must stay cheap).
func TestEditorStageCosts(t *testing.T) {
	const n = 2000
	shaper := text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(120, 20)),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}

	setText := measureRetained(t, n, func(e *Editor) {
		e.SingleLine = true
		e.SetText("VARIABLE_VALUE_00042")
	})
	t.Logf("after SetText: %.0f B/editor", setText)

	laidOut := measureRetained(t, n, func(e *Editor) {
		e.SingleLine = true
		e.SetText("VARIABLE_VALUE_00042")
		gtx.Ops.Reset()
		e.Layout(gtx, shaper, font.Font{}, unit.Sp(12), op.CallOp{}, op.CallOp{})
	})
	t.Logf("after SetText+Layout: %.0f B/editor", laidOut)

	// Regression bounds with ~2x headroom over measured values (496 B and
	// 6.8 KB as of the graphemeReader buffer fix). A display-only cell
	// (SetText, never laid out) must stay near-free: the 4KB bufio.Reader
	// retained per editor used to dominate here.
	if setText > 1024 {
		t.Errorf("SetText retains %.0f B/editor, want <= 1024", setText)
	}
	if laidOut > 13<<10 {
		t.Errorf("SetText+Layout retains %.0f B/editor, want <= 13KB", laidOut)
	}
}
