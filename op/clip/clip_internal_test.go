package clip

import (
	"image"
	"math"
	"testing"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/internal/ops"
	"github.com/nanorele/gio/op"
)

func countOps(o *op.Ops) map[ops.OpType]int {
	counts := make(map[ops.OpType]int)
	var r ops.Reader
	r.Reset(&o.Internal)
	for {
		enc, ok := r.Decode()
		if !ok {
			break
		}
		counts[ops.OpType(enc.Data[0])]++
	}
	return counts
}

func TestPath_MoveTo_LineTo(t *testing.T) {
	var ops op.Ops
	p := Path{}
	p.Begin(&ops)
	startPoint := f32.Pt(32, 32)
	endPoint := f32.Pt(64, 64)
	p.MoveTo(startPoint)
	p.LineTo(endPoint)
	pathSpec := p.End()

	minPoint := f32.Pt(32, 32)
	maxPoint := f32.Pt(64, 64)
	if pathSpec.bounds.Min == pathSpec.bounds.Max {
		t.Errorf("zero path")
	}
	if pathSpec.bounds.Min != minPoint.Round() {
		t.Errorf("pathSpec.bounds.Min = %v, want %v", pathSpec.bounds.Min, minPoint)
	}
	if pathSpec.bounds.Max != maxPoint.Round() {
		t.Errorf("pathSpec.bounds.Max = %v, want %v", pathSpec.bounds.Max, maxPoint)
	}
}

func TestPath_MoveTo_QuadTo(t *testing.T) {
	var ops op.Ops
	p := Path{}
	p.Begin(&ops)
	startPoint := f32.Pt(32, 32)
	midPoint := f32.Pt(60, 60)
	p.MoveTo(startPoint)
	p.QuadTo(midPoint.Sub(f32.Pt(-4, 0)), midPoint.Sub(f32.Pt(0, -4)))
	pathSpec := p.End()

	minPoint := f32.Pt(32, 32)
	maxPoint := f32.Pt(64, 64)
	if pathSpec.bounds.Min == pathSpec.bounds.Max {
		t.Errorf("zero path")
	}
	if pathSpec.bounds.Min != minPoint.Round() {
		t.Errorf("pathSpec.bounds.Min = %v, want %v", pathSpec.bounds.Min, minPoint)
	}
	if pathSpec.bounds.Max != maxPoint.Round() {
		t.Errorf("pathSpec.bounds.Max = %v, want %v", pathSpec.bounds.Max, maxPoint)
	}
}

func TestPath_MoveTo_ArcTo(t *testing.T) {

	tolerance := f32.Pt(1, 1)

	var ops op.Ops
	p := Path{}
	p.Begin(&ops)
	arcStartPoint := f32.Pt(48, 32)
	arcCenterPoint := f32.Pt(48, 48)
	p.MoveTo(arcStartPoint)
	p.ArcTo(arcCenterPoint, arcCenterPoint, math.Pi*2)
	pathSpec := p.End()

	minPoint := f32.Pt(32, 32).Sub(tolerance).Round()
	maxPoint := f32.Pt(64, 64).Add(tolerance).Round()
	if pathSpec.bounds.Min == pathSpec.bounds.Max {
		t.Errorf("zero path")
	}
	if pathSpec.bounds.Min.X < minPoint.X || pathSpec.bounds.Min.Y < minPoint.Y {
		t.Errorf("pathSpec.bounds.Min = %v, want %v", pathSpec.bounds.Min, minPoint)
	}
	if pathSpec.bounds.Max.X > maxPoint.X || pathSpec.bounds.Max.Y > maxPoint.Y {
		t.Errorf("pathSpec.bounds.Max = %v, want %v", pathSpec.bounds.Max, maxPoint)
	}
}

func TestPath_MoveTo_CubeTo(t *testing.T) {
	var ops op.Ops
	p := Path{}
	p.Begin(&ops)
	startPoint := f32.Pt(32, 32)
	midPoint := f32.Pt(48, 48)
	endPoint := f32.Pt(64, 64)
	p.MoveTo(startPoint)
	p.CubeTo(midPoint.Sub(f32.Pt(-4, 0)), midPoint.Sub(f32.Pt(0, -4)), endPoint)
	pathSpec := p.End()

	minPoint := f32.Pt(32, 32)
	maxPoint := f32.Pt(64, 64)
	if pathSpec.bounds.Min == pathSpec.bounds.Max {
		t.Errorf("zero path")
	}
	if pathSpec.bounds.Min != minPoint.Round() {
		t.Errorf("pathSpec.bounds.Min = %v, want %v", pathSpec.bounds.Min, minPoint)
	}
	if pathSpec.bounds.Max != maxPoint.Round() {
		t.Errorf("pathSpec.bounds.Max = %v, want %v", pathSpec.bounds.Max, maxPoint)
	}
}

func TestPathBeginEmpty(t *testing.T) {
	var o op.Ops
	var p Path
	p.Begin(&o)
	spec := p.End()
	if spec.hasSegments {
		t.Errorf("empty path should have hasSegments=false")
	}
	if spec.bounds != (image.Rectangle{}) {
		t.Errorf("empty path bounds = %v, want zero", spec.bounds)
	}
}

func TestPathSinglePoint(t *testing.T) {
	var o op.Ops
	var p Path
	p.Begin(&o)
	p.MoveTo(f32.Pt(7, 8))
	spec := p.End()
	if spec.hasSegments {
		t.Errorf("MoveTo only should not set hasSegments")
	}
	if pos := p.Pos(); pos != f32.Pt(7, 8) {
		t.Errorf("Pos = %v, want (7,8)", pos)
	}
}

func TestPathLineToSamePoint(t *testing.T) {
	var o op.Ops
	var p Path
	p.Begin(&o)
	p.MoveTo(f32.Pt(10, 10))
	p.LineTo(f32.Pt(10, 10)) // degenerate line, should be skipped
	spec := p.End()
	if spec.hasSegments {
		t.Errorf("degenerate LineTo should not produce segments")
	}
}

func TestPathQuadToDegenerate(t *testing.T) {
	var o op.Ops
	var p Path
	p.Begin(&o)
	p.MoveTo(f32.Pt(5, 5))
	// QuadTo with ctrl == pen and to == pen should be skipped.
	p.QuadTo(f32.Pt(5, 5), f32.Pt(5, 5))
	spec := p.End()
	if spec.hasSegments {
		t.Errorf("degenerate QuadTo should not produce segments")
	}
}

func TestPathCubeToDegenerate(t *testing.T) {
	var o op.Ops
	var p Path
	p.Begin(&o)
	p.MoveTo(f32.Pt(2, 3))
	p.CubeTo(f32.Pt(2, 3), f32.Pt(2, 3), f32.Pt(2, 3))
	spec := p.End()
	if spec.hasSegments {
		t.Errorf("degenerate CubeTo should not produce segments")
	}
}

func TestPathBoundsExpand(t *testing.T) {
	var o op.Ops
	var p Path
	p.Begin(&o)
	p.MoveTo(f32.Pt(10, 10))
	p.LineTo(f32.Pt(50, 30))
	p.LineTo(f32.Pt(0, 100))
	spec := p.End()
	wantMin := image.Pt(0, 10)
	wantMax := image.Pt(50, 100)
	if spec.bounds.Min != wantMin || spec.bounds.Max != wantMax {
		t.Errorf("bounds = %v, want Min=%v Max=%v", spec.bounds, wantMin, wantMax)
	}
}

func TestPathClose(t *testing.T) {
	var o op.Ops
	var p Path
	p.Begin(&o)
	p.MoveTo(f32.Pt(0, 0))
	p.LineTo(f32.Pt(10, 0))
	p.LineTo(f32.Pt(10, 10))
	p.Close()
	// After Close, pen should be back at start.
	if pos := p.Pos(); pos != (f32.Pt(0, 0)) {
		t.Errorf("after Close pen = %v, want start (0,0)", pos)
	}
	spec := p.End()
	if !spec.hasSegments {
		t.Errorf("closed triangle should have segments")
	}
}

// TestPathMoveToSamePosUpdatesStart guards a regression in MoveTo where a
// no-op fast path returned without resetting start, so a subsequent Close
// would wrongly close to the previous contour's start.
func TestPathMoveToSamePosUpdatesStart(t *testing.T) {
	var o op.Ops
	var p Path
	p.Begin(&o)
	p.MoveTo(f32.Pt(0, 0))
	p.LineTo(f32.Pt(20, 0))
	// Pen is now at (20,0) but start is still (0,0).
	// Explicit MoveTo to current pen must begin a new sub-contour at (20,0).
	p.MoveTo(f32.Pt(20, 0))
	p.LineTo(f32.Pt(20, 10))
	p.Close() // should close to (20,0), not (0,0)
	if pos := p.Pos(); pos != (f32.Pt(20, 0)) {
		t.Errorf("after Close pen = %v, want (20,0)", pos)
	}
	p.End()
}

func TestPathMoveDelta(t *testing.T) {
	var o op.Ops
	var p Path
	p.Begin(&o)
	p.MoveTo(f32.Pt(5, 5))
	p.Move(f32.Pt(3, 4)) // relative move to (8, 9)
	if pos := p.Pos(); pos != f32.Pt(8, 9) {
		t.Errorf("after Move pen = %v, want (8,9)", pos)
	}
	p.Line(f32.Pt(2, 1)) // relative line to (10, 10)
	if pos := p.Pos(); pos != f32.Pt(10, 10) {
		t.Errorf("after Line pen = %v, want (10,10)", pos)
	}
	p.End()
}

func TestRectPath(t *testing.T) {
	r := Rect(image.Rect(5, 10, 100, 50))
	spec := r.Path()
	if spec.shape != ops.Rect {
		t.Errorf("Rect.Path shape = %v, want ops.Rect", spec.shape)
	}
	if spec.bounds != image.Rect(5, 10, 100, 50) {
		t.Errorf("Rect.Path bounds = %v", spec.bounds)
	}
	if spec.hasSegments {
		t.Errorf("Rect.Path should not embed explicit segments")
	}
}

func TestRectOpEncoding(t *testing.T) {
	var o op.Ops
	r := Rect(image.Rect(0, 0, 50, 50))
	stk := r.Push(&o)
	stk.Pop()
	counts := countOps(&o)
	if counts[ops.TypeClip] != 1 {
		t.Errorf("expected 1 clip op, got %d", counts[ops.TypeClip])
	}
	if counts[ops.TypePopClip] != 1 {
		t.Errorf("expected 1 pop-clip op, got %d", counts[ops.TypePopClip])
	}
	if counts[ops.TypePath] != 0 {
		t.Errorf("plain Rect should not emit a path op, got %d", counts[ops.TypePath])
	}
	if counts[ops.TypeStroke] != 0 {
		t.Errorf("plain Rect should not emit a stroke op")
	}
}

func TestRectClipDecodes(t *testing.T) {
	var o op.Ops
	r := Rect(image.Rect(2, 3, 40, 50))
	stk := r.Push(&o)
	stk.Pop()
	var r2 ops.Reader
	r2.Reset(&o.Internal)
	for {
		enc, ok := r2.Decode()
		if !ok {
			break
		}
		if ops.OpType(enc.Data[0]) == ops.TypeClip {
			var co ops.ClipOp
			co.Decode(enc.Data)
			if co.Bounds != image.Rect(2, 3, 40, 50) {
				t.Errorf("decoded bounds = %v", co.Bounds)
			}
			if co.Shape != ops.Rect {
				t.Errorf("decoded shape = %v, want Rect", co.Shape)
			}
			if !co.Outline {
				t.Errorf("Rect.Op() should set outline=true")
			}
			return
		}
	}
	t.Fatal("clip op not found in stream")
}

func TestStrokeRectExpandsBounds(t *testing.T) {
	var o op.Ops
	r := Rect(image.Rect(10, 10, 30, 30))
	st := Stroke{Path: r.Path(), Width: 4}.Op().Push(&o)
	st.Pop()
	var r2 ops.Reader
	r2.Reset(&o.Internal)
	var found bool
	for {
		enc, ok := r2.Decode()
		if !ok {
			break
		}
		if ops.OpType(enc.Data[0]) == ops.TypeClip {
			var co ops.ClipOp
			co.Decode(enc.Data)
			// width=4 -> half=2, bounds should expand by 2 on each side.
			want := image.Rect(8, 8, 32, 32)
			if co.Bounds != want {
				t.Errorf("stroke bounds = %v, want %v", co.Bounds, want)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("clip op not found")
	}
}

func TestStrokeEmitsStrokeOp(t *testing.T) {
	var o op.Ops
	var p Path
	p.Begin(&o)
	p.MoveTo(f32.Pt(0, 0))
	p.LineTo(f32.Pt(10, 0))
	st := Stroke{Path: p.End(), Width: 2}.Op().Push(&o)
	st.Pop()
	counts := countOps(&o)
	if counts[ops.TypeStroke] != 1 {
		t.Errorf("expected 1 stroke op, got %d", counts[ops.TypeStroke])
	}
	if counts[ops.TypePath] != 1 {
		t.Errorf("expected 1 path op, got %d", counts[ops.TypePath])
	}
	if counts[ops.TypeClip] != 1 {
		t.Errorf("expected 1 clip op")
	}
}

func TestStrokeZeroWidth(t *testing.T) {
	var o op.Ops
	var p Path
	p.Begin(&o)
	p.MoveTo(f32.Pt(0, 0))
	p.LineTo(f32.Pt(10, 0))
	st := Stroke{Path: p.End(), Width: 0}.Op().Push(&o)
	st.Pop()
	counts := countOps(&o)
	if counts[ops.TypeStroke] != 0 {
		t.Errorf("zero-width stroke must not emit stroke op, got %d", counts[ops.TypeStroke])
	}
}

func TestOutlineClipIsOutline(t *testing.T) {
	var o op.Ops
	var p Path
	p.Begin(&o)
	p.MoveTo(f32.Pt(0, 0))
	p.LineTo(f32.Pt(10, 0))
	p.LineTo(f32.Pt(10, 10))
	p.Close()
	st := Outline{Path: p.End()}.Op().Push(&o)
	st.Pop()
	var r ops.Reader
	r.Reset(&o.Internal)
	var found bool
	for {
		enc, ok := r.Decode()
		if !ok {
			break
		}
		if ops.OpType(enc.Data[0]) == ops.TypeClip {
			var co ops.ClipOp
			co.Decode(enc.Data)
			if !co.Outline {
				t.Errorf("Outline.Op() should set outline byte")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("clip op not emitted")
	}
}

func TestRRectZeroRadiiIsRect(t *testing.T) {
	var o op.Ops
	rr := RRect{Rect: image.Rect(0, 0, 30, 30)}
	op2 := rr.Op(&o)
	if op2.path.shape != ops.Rect {
		t.Errorf("RRect with zero radii should fall back to Rect, got shape %v", op2.path.shape)
	}
	if op2.path.hasSegments {
		t.Errorf("zero-radius RRect should not have explicit segments")
	}
}

func TestRRectPathHasSegments(t *testing.T) {
	var o op.Ops
	rr := UniformRRect(image.Rect(0, 0, 50, 50), 5)
	spec := rr.Path(&o)
	if !spec.hasSegments {
		t.Errorf("RRect path should have segments")
	}
}

func TestUniformRRectAllRadii(t *testing.T) {
	rr := UniformRRect(image.Rect(0, 0, 100, 100), 7)
	if rr.NW != 7 || rr.NE != 7 || rr.SW != 7 || rr.SE != 7 {
		t.Errorf("UniformRRect must set all four radii to 7, got %+v", rr)
	}
	if rr.Rect != image.Rect(0, 0, 100, 100) {
		t.Errorf("UniformRRect Rect = %v", rr.Rect)
	}
}

func TestEllipsePath(t *testing.T) {
	var o op.Ops
	e := Ellipse(image.Rect(0, 0, 100, 50))
	spec := e.Path(&o)
	if spec.shape != ops.Ellipse {
		t.Errorf("Ellipse.Path shape = %v, want ops.Ellipse", spec.shape)
	}
	if !spec.hasSegments {
		t.Errorf("Ellipse should emit segments")
	}
}

func TestEllipseEmptyRect(t *testing.T) {
	var o op.Ops
	// zero width
	e1 := Ellipse(image.Rect(5, 5, 5, 50))
	spec1 := e1.Path(&o)
	if spec1.shape != ops.Rect {
		t.Errorf("zero-width ellipse should be Rect shape, got %v", spec1.shape)
	}
	// zero height
	e2 := Ellipse(image.Rect(5, 5, 50, 5))
	spec2 := e2.Path(&o)
	if spec2.shape != ops.Rect {
		t.Errorf("zero-height ellipse should be Rect shape, got %v", spec2.shape)
	}
}

func TestStackPushPop(t *testing.T) {
	var o op.Ops
	r := Rect(image.Rect(0, 0, 10, 10))
	st := r.Push(&o)
	st.Pop()
	// nested
	st2 := r.Push(&o)
	st3 := r.Push(&o)
	st3.Pop()
	st2.Pop()
}

func TestStackUnbalancedPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("unbalanced clip pops should panic")
		}
	}()
	var o op.Ops
	r := Rect(image.Rect(0, 0, 10, 10))
	st1 := r.Push(&o)
	st2 := r.Push(&o)
	// pop in wrong order
	st1.Pop()
	_ = st2
}

func TestPathContoursEnd(t *testing.T) {
	var o op.Ops
	var p Path
	p.Begin(&o)
	p.MoveTo(f32.Pt(0, 0))
	p.LineTo(f32.Pt(10, 0))
	p.MoveTo(f32.Pt(20, 20))
	p.LineTo(f32.Pt(30, 20))
	if got, want := p.contour, 2; got != want {
		t.Errorf("contour = %d, want %d", got, want)
	}
	p.End()
}

func TestArcToZeroAngle(t *testing.T) {
	var o op.Ops
	var p Path
	p.Begin(&o)
	p.MoveTo(f32.Pt(10, 10))
	p.ArcTo(f32.Pt(20, 10), f32.Pt(20, 10), 0)
	spec := p.End()
	// zero-angle arc should not panic; segments may or may not exist.
	_ = spec
}

func TestArcFullCircle(t *testing.T) {
	var o op.Ops
	var p Path
	p.Begin(&o)
	p.MoveTo(f32.Pt(50, 0))
	p.ArcTo(f32.Pt(0, 0), f32.Pt(0, 0), 2*math.Pi)
	spec := p.End()
	if !spec.hasSegments {
		t.Errorf("full circle arc should produce segments")
	}
}

func TestPathSpecHashStable(t *testing.T) {
	build := func() PathSpec {
		var o op.Ops
		var p Path
		p.Begin(&o)
		p.MoveTo(f32.Pt(0, 0))
		p.LineTo(f32.Pt(10, 10))
		p.LineTo(f32.Pt(20, 0))
		return p.End()
	}
	a := build()
	b := build()
	if a.hash != b.hash {
		t.Errorf("identical paths produced different hashes %x vs %x", a.hash, b.hash)
	}
}

func TestPathSpecHashDiffers(t *testing.T) {
	var o1 op.Ops
	var p1 Path
	p1.Begin(&o1)
	p1.MoveTo(f32.Pt(0, 0))
	p1.LineTo(f32.Pt(10, 10))
	a := p1.End()

	var o2 op.Ops
	var p2 Path
	p2.Begin(&o2)
	p2.MoveTo(f32.Pt(0, 0))
	p2.LineTo(f32.Pt(20, 20))
	b := p2.End()
	if a.hash == b.hash {
		t.Errorf("different paths produced same hash %x", a.hash)
	}
}

func TestStrokeOpFromStruct(t *testing.T) {
	r := Rect(image.Rect(0, 0, 10, 10))
	op2 := Stroke{Path: r.Path(), Width: 3.5}.Op()
	if op2.width != 3.5 {
		t.Errorf("Stroke.Op width = %v", op2.width)
	}
	if op2.outline {
		t.Errorf("Stroke.Op should not set outline")
	}
}

func TestOutlineOpFromStruct(t *testing.T) {
	r := Rect(image.Rect(0, 0, 10, 10))
	op2 := Outline{Path: r.Path()}.Op()
	if !op2.outline {
		t.Errorf("Outline.Op should set outline")
	}
	if op2.width != 0 {
		t.Errorf("Outline.Op width should default to 0, got %v", op2.width)
	}
}

func TestEllipseEncodesEllipseShape(t *testing.T) {
	var o op.Ops
	e := Ellipse(image.Rect(0, 0, 40, 20))
	st := e.Push(&o)
	st.Pop()
	var r ops.Reader
	r.Reset(&o.Internal)
	for {
		enc, ok := r.Decode()
		if !ok {
			break
		}
		if ops.OpType(enc.Data[0]) == ops.TypeClip {
			var co ops.ClipOp
			co.Decode(enc.Data)
			if co.Shape != ops.Ellipse {
				t.Errorf("ellipse clip shape = %v, want Ellipse", co.Shape)
			}
			return
		}
	}
	t.Fatal("clip op not found")
}
