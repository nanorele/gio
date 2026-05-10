package stroke

import (
	"encoding/binary"
	"math"
	"strconv"
	"testing"

	"github.com/nanorele/gio/internal/f32"
	"github.com/nanorele/gio/internal/scene"
)

func TestNormPt(t *testing.T) {
	type scenario struct {
		l           float32
		ptIn, ptOut f32.Point
	}

	scenarios := []scenario{
		{l: 10, ptIn: f32.Point{X: 0, Y: 0}, ptOut: f32.Point{X: 0, Y: 0}},
		{l: -10, ptIn: f32.Point{X: 0, Y: 0}, ptOut: f32.Point{X: 0, Y: 0}},
		{l: +20, ptIn: f32.Point{X: +30, Y: 0}, ptOut: f32.Point{X: +20, Y: 0}},
		{l: +20, ptIn: f32.Point{X: +20, Y: 0}, ptOut: f32.Point{X: +20, Y: 0}},
		{l: +20, ptIn: f32.Point{X: +10, Y: 0}, ptOut: f32.Point{X: +20, Y: 0}},
		{l: +20, ptIn: f32.Point{X: -10, Y: 0}, ptOut: f32.Point{X: -20, Y: 0}},
		{l: +20, ptIn: f32.Point{X: -20, Y: 0}, ptOut: f32.Point{X: -20, Y: 0}},
		{l: +20, ptIn: f32.Point{X: -30, Y: 0}, ptOut: f32.Point{X: -20, Y: 0}},
		{l: -20, ptIn: f32.Point{X: +30, Y: 0}, ptOut: f32.Point{X: -20, Y: 0}},
		{l: -20, ptIn: f32.Point{X: +20, Y: 0}, ptOut: f32.Point{X: -20, Y: 0}},
		{l: -20, ptIn: f32.Point{X: +10, Y: 0}, ptOut: f32.Point{X: -20, Y: 0}},
		{l: -20, ptIn: f32.Point{X: -10, Y: 0}, ptOut: f32.Point{X: +20, Y: 0}},
		{l: -20, ptIn: f32.Point{X: -20, Y: 0}, ptOut: f32.Point{X: +20, Y: 0}},
		{l: -20, ptIn: f32.Point{X: -30, Y: 0}, ptOut: f32.Point{X: +20, Y: 0}},
		{l: +20, ptIn: f32.Point{X: 0, Y: +30}, ptOut: f32.Point{X: 0, Y: +20}},
		{l: +20, ptIn: f32.Point{X: 0, Y: +20}, ptOut: f32.Point{X: 0, Y: +20}},
		{l: +20, ptIn: f32.Point{X: 0, Y: +10}, ptOut: f32.Point{X: 0, Y: +20}},
		{l: +20, ptIn: f32.Point{X: 0, Y: -10}, ptOut: f32.Point{X: 0, Y: -20}},
		{l: +20, ptIn: f32.Point{X: 0, Y: -20}, ptOut: f32.Point{X: 0, Y: -20}},
		{l: +20, ptIn: f32.Point{X: 0, Y: -30}, ptOut: f32.Point{X: 0, Y: -20}},
		{l: -20, ptIn: f32.Point{X: 0, Y: +30}, ptOut: f32.Point{X: 0, Y: -20}},
		{l: -20, ptIn: f32.Point{X: 0, Y: +20}, ptOut: f32.Point{X: 0, Y: -20}},
		{l: -20, ptIn: f32.Point{X: 0, Y: +10}, ptOut: f32.Point{X: 0, Y: -20}},
		{l: -20, ptIn: f32.Point{X: 0, Y: -10}, ptOut: f32.Point{X: 0, Y: +20}},
		{l: -20, ptIn: f32.Point{X: 0, Y: -20}, ptOut: f32.Point{X: 0, Y: +20}},
		{l: -20, ptIn: f32.Point{X: 0, Y: -30}, ptOut: f32.Point{X: 0, Y: +20}},
		{l: +20, ptIn: f32.Point{X: +90, Y: +90}, ptOut: f32.Point{X: +14.142137, Y: +14.142137}},
		{l: +20, ptIn: f32.Point{X: +30, Y: +30}, ptOut: f32.Point{X: +14.142136, Y: +14.142136}},
		{l: +20, ptIn: f32.Point{X: +20, Y: +20}, ptOut: f32.Point{X: +14.142136, Y: +14.142136}},
		{l: +20, ptIn: f32.Point{X: +10, Y: +10}, ptOut: f32.Point{X: +14.142136, Y: +14.142136}},
		{l: +20, ptIn: f32.Point{X: -10, Y: -10}, ptOut: f32.Point{X: -14.142136, Y: -14.142136}},
		{l: +20, ptIn: f32.Point{X: -20, Y: -20}, ptOut: f32.Point{X: -14.142136, Y: -14.142136}},
		{l: +20, ptIn: f32.Point{X: -30, Y: -30}, ptOut: f32.Point{X: -14.142136, Y: -14.142136}},
		{l: +20, ptIn: f32.Point{X: -90, Y: -90}, ptOut: f32.Point{X: -14.142137, Y: -14.142137}},
		{l: +20, ptIn: f32.Point{X: +90, Y: -90}, ptOut: f32.Point{X: +14.142137, Y: -14.142137}},
		{l: +20, ptIn: f32.Point{X: +30, Y: -30}, ptOut: f32.Point{X: +14.142136, Y: -14.142136}},
		{l: +20, ptIn: f32.Point{X: +20, Y: -20}, ptOut: f32.Point{X: +14.142136, Y: -14.142136}},
		{l: +20, ptIn: f32.Point{X: +10, Y: -10}, ptOut: f32.Point{X: +14.142136, Y: -14.142136}},
		{l: +20, ptIn: f32.Point{X: -10, Y: +10}, ptOut: f32.Point{X: -14.142136, Y: +14.142136}},
		{l: +20, ptIn: f32.Point{X: -20, Y: +20}, ptOut: f32.Point{X: -14.142136, Y: +14.142136}},
		{l: +20, ptIn: f32.Point{X: -30, Y: +30}, ptOut: f32.Point{X: -14.142136, Y: +14.142136}},
		{l: +20, ptIn: f32.Point{X: -90, Y: +90}, ptOut: f32.Point{X: -14.142137, Y: +14.142137}},
		{l: -20, ptIn: f32.Point{X: +90, Y: +90}, ptOut: f32.Point{X: -14.142137, Y: -14.142137}},
		{l: -20, ptIn: f32.Point{X: +30, Y: +30}, ptOut: f32.Point{X: -14.142136, Y: -14.142136}},
		{l: -20, ptIn: f32.Point{X: +20, Y: +20}, ptOut: f32.Point{X: -14.142136, Y: -14.142136}},
		{l: -20, ptIn: f32.Point{X: +10, Y: +10}, ptOut: f32.Point{X: -14.142136, Y: -14.142136}},
		{l: -20, ptIn: f32.Point{X: -10, Y: -10}, ptOut: f32.Point{X: +14.142136, Y: +14.142136}},
		{l: -20, ptIn: f32.Point{X: -20, Y: -20}, ptOut: f32.Point{X: +14.142136, Y: +14.142136}},
		{l: -20, ptIn: f32.Point{X: -30, Y: -30}, ptOut: f32.Point{X: +14.142136, Y: +14.142136}},
		{l: -20, ptIn: f32.Point{X: -90, Y: -90}, ptOut: f32.Point{X: +14.142137, Y: +14.142137}},
		{l: -20, ptIn: f32.Point{X: +90, Y: -90}, ptOut: f32.Point{X: -14.142137, Y: +14.142137}},
		{l: -20, ptIn: f32.Point{X: +30, Y: -30}, ptOut: f32.Point{X: -14.142136, Y: +14.142136}},
		{l: -20, ptIn: f32.Point{X: +20, Y: -20}, ptOut: f32.Point{X: -14.142136, Y: +14.142136}},
		{l: -20, ptIn: f32.Point{X: +10, Y: -10}, ptOut: f32.Point{X: -14.142136, Y: +14.142136}},
		{l: -20, ptIn: f32.Point{X: -10, Y: +10}, ptOut: f32.Point{X: +14.142136, Y: -14.142136}},
		{l: -20, ptIn: f32.Point{X: -20, Y: +20}, ptOut: f32.Point{X: +14.142136, Y: -14.142136}},
		{l: -20, ptIn: f32.Point{X: -30, Y: +30}, ptOut: f32.Point{X: +14.142136, Y: -14.142136}},
		{l: -20, ptIn: f32.Point{X: -90, Y: +90}, ptOut: f32.Point{X: +14.142137, Y: -14.142137}},
		{l: 5, ptIn: f32.Point{X: 3, Y: 4}, ptOut: f32.Point{X: 3, Y: 4}},
		{l: 5, ptIn: f32.Point{X: 3, Y: -4}, ptOut: f32.Point{X: 3, Y: -4}},
		{l: 5, ptIn: f32.Point{X: -3, Y: -4}, ptOut: f32.Point{X: -3, Y: -4}},
		{l: 5, ptIn: f32.Point{X: -3, Y: 4}, ptOut: f32.Point{X: -3, Y: 4}},
		{l: -5, ptIn: f32.Point{X: 3, Y: 4}, ptOut: f32.Point{X: -3, Y: -4}},
		{l: -5, ptIn: f32.Point{X: 3, Y: -4}, ptOut: f32.Point{X: -3, Y: 4}},
		{l: -5, ptIn: f32.Point{X: -3, Y: -4}, ptOut: f32.Point{X: 3, Y: 4}},
		{l: -5, ptIn: f32.Point{X: -3, Y: 4}, ptOut: f32.Point{X: 3, Y: -4}},
	}

	for i, s := range scenarios {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			actual := normPt(s.ptIn, s.l)
			if actual != s.ptOut {
				t.Errorf("in: %v*%v, expected: %v, actual: %v", s.l, s.ptIn, s.ptOut, actual)
			}
		})
	}
}

func TestStrokeQuads_LineTo(t *testing.T) {
	qs := StrokeQuads{
		{Quad: QuadSegment{To: f32.Point{X: 0, Y: 0}}},
	}
	qs.lineTo(f32.Point{X: 10, Y: 0})
	if len(qs) != 2 {
		t.Errorf("expected 2 quads, got %d", len(qs))
	}
	if qs[1].Quad.To != (f32.Point{X: 10, Y: 0}) {
		t.Errorf("expected end point (10, 0), got %v", qs[1].Quad.To)
	}
}

func TestStrokeQuads_Close(t *testing.T) {
	qs := StrokeQuads{
		{Quad: QuadSegment{From: f32.Point{X: 0, Y: 0}, To: f32.Point{X: 10, Y: 0}}},
	}
	qs.close()
	if len(qs) != 2 {
		t.Errorf("expected 2 quads after close, got %d", len(qs))
	}
	if qs[1].Quad.To != (f32.Point{X: 0, Y: 0}) {
		t.Errorf("expected close quad to end at (0, 0), got %v", qs[1].Quad.To)
	}

	// Already closed
	qs2 := StrokeQuads{
		{Quad: QuadSegment{From: f32.Point{X: 0, Y: 0}, To: f32.Point{X: 10, Y: 0}}},
		{Quad: QuadSegment{From: f32.Point{X: 10, Y: 0}, To: f32.Point{X: 0, Y: 0}}},
	}
	qs2.close()
	if len(qs2) != 2 {
		t.Errorf("expected still 2 quads, got %d", len(qs2))
	}
}

func TestStrokeQuads_Split(t *testing.T) {
	qs := StrokeQuads{
		{Contour: 0, Quad: QuadSegment{To: f32.Point{X: 10, Y: 0}}},
		{Contour: 0, Quad: QuadSegment{To: f32.Point{X: 10, Y: 10}}},
		{Contour: 1, Quad: QuadSegment{To: f32.Point{X: 20, Y: 20}}},
	}
	parts := qs.split()
	if len(parts) != 2 {
		t.Errorf("expected 2 parts, got %d", len(parts))
	}
	if len(parts[0]) != 2 || len(parts[1]) != 1 {
		t.Errorf("expected parts lengths 2 and 1, got %d and %d", len(parts[0]), len(parts[1]))
	}
}

func TestStrokeQuads_Stroke(t *testing.T) {
	qs := StrokeQuads{
		{Quad: QuadSegment{From: f32.Point{X: 0, Y: 0}, To: f32.Point{X: 10, Y: 0}}},
	}
	style := StrokeStyle{Width: 2}
	stroked := qs.stroke(style)
	if len(stroked) == 0 {
		t.Error("stroke returned no quads")
	}
}

func TestStrokeQuads_Arc(t *testing.T) {
	qs := StrokeQuads{
		{Quad: QuadSegment{To: f32.Point{X: 0, Y: 0}}},
	}
	qs.arc(f32.Point{X: 10, Y: 0}, f32.Point{X: 0, Y: 10}, math.Pi/2)
	if len(qs) <= 1 {
		t.Errorf("expected more than 1 quad after arc, got %d", len(qs))
	}
}

func TestStrokeQuads_CCW(t *testing.T) {
	// CCW rectangle (in Y-up)
	qs := StrokeQuads{
		{Quad: QuadSegment{From: f32.Point{X: 0, Y: 0}, To: f32.Point{X: 10, Y: 0}}},
		{Quad: QuadSegment{From: f32.Point{X: 10, Y: 0}, To: f32.Point{X: 10, Y: 10}}},
		{Quad: QuadSegment{From: f32.Point{X: 10, Y: 10}, To: f32.Point{X: 0, Y: 10}}},
		{Quad: QuadSegment{From: f32.Point{X: 0, Y: 10}, To: f32.Point{X: 0, Y: 0}}},
	}
	if !qs.ccw() {
		t.Error("expected CCW rectangle to be CCW")
	}

	// CW rectangle (in Y-up)
	qsCW := StrokeQuads{
		{Quad: QuadSegment{From: f32.Point{X: 0, Y: 0}, To: f32.Point{X: 0, Y: 10}}},
		{Quad: QuadSegment{From: f32.Point{X: 0, Y: 10}, To: f32.Point{X: 10, Y: 10}}},
		{Quad: QuadSegment{From: f32.Point{X: 10, Y: 10}, To: f32.Point{X: 10, Y: 0}}},
		{Quad: QuadSegment{From: f32.Point{X: 10, Y: 0}, To: f32.Point{X: 0, Y: 0}}},
	}
	if qsCW.ccw() {
		t.Error("expected CW rectangle to NOT be CCW")
	}
}

func TestStrokeQuads_ReverseAppend(t *testing.T) {
	qs := StrokeQuads{
		{Quad: QuadSegment{From: f32.Point{X: 0, Y: 0}, To: f32.Point{X: 10, Y: 0}}},
	}
	rev := qs.reverse()
	if rev[0].Quad.From != (f32.Point{X: 10, Y: 0}) || rev[0].Quad.To != (f32.Point{X: 0, Y: 0}) {
		t.Error("reverse failed")
	}

	appended := qs.append(rev)
	if len(appended) != 2 {
		t.Errorf("expected 2 quads (1+1 and no glue since endpoints match), got %d", len(appended))
	}
}

func TestQuadSegment_Transform(t *testing.T) {
	q := QuadSegment{
		From: f32.Point{X: 0, Y: 0},
		Ctrl: f32.Point{X: 5, Y: 5},
		To:   f32.Point{X: 10, Y: 0},
	}
	trans := f32.AffineId().Offset(f32.Point{X: 10, Y: 10})
	q2 := q.Transform(trans)
	if q2.From != (f32.Point{X: 10, Y: 10}) || q2.To != (f32.Point{X: 20, Y: 10}) {
		t.Error("transform failed")
	}
}

func TestStrokePathCommands(t *testing.T) {
	style := StrokeStyle{Width: 2}

	encode := func(contour uint32, cmd scene.Command) []byte {
		data := make([]byte, 4+scene.CommandSize)
		binary.LittleEndian.PutUint32(data, contour)
		for i, v := range cmd {
			binary.LittleEndian.PutUint32(data[4+i*4:], v)
		}
		return data
	}

	var data []byte
	data = append(data, encode(1, scene.Line(f32.Pt(0, 0), f32.Pt(10, 0)))...)
	data = append(data, encode(1, scene.Quad(f32.Pt(10, 0), f32.Pt(15, 5), f32.Pt(20, 0)))...)
	data = append(data, encode(1, scene.Cubic(f32.Pt(20, 0), f32.Pt(25, -5), f32.Pt(30, -5), f32.Pt(35, 0)))...)
	data = append(data, encode(1, scene.Gap(f32.Pt(35, 0), f32.Pt(0, 0)))...)

	stroked := StrokePathCommands(style, data)
	if len(stroked) == 0 {
		t.Error("expected stroked quads")
	}
}

func TestCurvature(t *testing.T) {
	c := strokePathCurv(f32.Point{X: 0, Y: 0}, f32.Point{X: 5, Y: 5}, f32.Point{X: 10, Y: 0}, 0.5)
	if math.IsNaN(float64(c)) {
		t.Error("curvature is NaN")
	}

	flat := strokePathCurv(f32.Point{X: 0, Y: 0}, f32.Point{X: 5, Y: 0}, f32.Point{X: 10, Y: 0}, 0.5)
	if !math.IsNaN(float64(flat)) {
		t.Error("expected NaN curvature for flat line")
	}
}

func BenchmarkSplitCubic(b *testing.B) {
	type scenario struct {
		segments               int
		from, ctrl0, ctrl1, to f32.Point
	}

	scenarios := []scenario{
		{
			segments: 4,
			from:     f32.Pt(0, 0),
			ctrl0:    f32.Pt(10, 10),
			ctrl1:    f32.Pt(10, 10),
			to:       f32.Pt(20, 0),
		},
		{
			segments: 8,
			from:     f32.Pt(-145.90305, 703.21277),
			ctrl0:    f32.Pt(-940.20215, 606.05994),
			ctrl1:    f32.Pt(74.58341, 405.815),
			to:       f32.Pt(104.35474, -241.543),
		},
		{
			segments: 16,
			from:     f32.Pt(770.35626, 639.77765),
			ctrl0:    f32.Pt(735.57135, 545.07837),
			ctrl1:    f32.Pt(286.7138, 853.7052),
			to:       f32.Pt(286.7138, 890.5413),
		},
		{
			segments: 33,
			from:     f32.Pt(0, 0),
			ctrl0:    f32.Pt(0, 0),
			ctrl1:    f32.Pt(100, 100),
			to:       f32.Pt(100, 100),
		},
	}

	for _, s := range scenarios {
		b.Run(strconv.Itoa(s.segments), func(b *testing.B) {
			from, ctrl0, ctrl1, to := s.from, s.ctrl0, s.ctrl1, s.to
			quads := make([]QuadSegment, s.segments)
			b.ResetTimer()
			for b.Loop() {
				quads = SplitCubic(from, ctrl0, ctrl1, to, quads[:0])
			}
			if len(quads) != s.segments {
				b.Fatalf("expected %d but got %d", s.segments, len(quads))
			}
		})
	}
}

func nearlyEqualPt(a, b f32.Point, eps float32) bool {
	dx := a.X - b.X
	dy := a.Y - b.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx <= eps && dy <= eps
}

func nearlyEqual(a, b, eps float32) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
}

func TestQuadBezierSample_Endpoints(t *testing.T) {
	p0 := f32.Pt(0, 0)
	p1 := f32.Pt(5, 10)
	p2 := f32.Pt(10, 0)
	if got := quadBezierSample(p0, p1, p2, 0); got != p0 {
		t.Errorf("t=0 want %v got %v", p0, got)
	}
	if got := quadBezierSample(p0, p1, p2, 1); got != p2 {
		t.Errorf("t=1 want %v got %v", p2, got)
	}
	mid := quadBezierSample(p0, p1, p2, 0.5)
	want := f32.Pt(5, 5)
	if !nearlyEqualPt(mid, want, 1e-5) {
		t.Errorf("midpoint want %v got %v", want, mid)
	}
}

func TestQuadBezierSample_Linear(t *testing.T) {
	// When ctrl is on the line, all samples lie on the line.
	p0 := f32.Pt(0, 0)
	p1 := f32.Pt(5, 0)
	p2 := f32.Pt(10, 0)
	for _, ti := range []float32{0, 0.1, 0.25, 0.5, 0.75, 0.9, 1} {
		got := quadBezierSample(p0, p1, p2, ti)
		if got.Y != 0 {
			t.Errorf("t=%v expected Y=0, got %v", ti, got)
		}
		if got.X < -1e-6 || got.X > 10+1e-6 {
			t.Errorf("t=%v X out of range: %v", ti, got.X)
		}
	}
}

func TestQuadBezierD1_Endpoints(t *testing.T) {
	p0 := f32.Pt(0, 0)
	p1 := f32.Pt(5, 10)
	p2 := f32.Pt(10, 0)
	d0 := quadBezierD1(p0, p1, p2, 0)
	want0 := f32.Pt(10, 20)
	if !nearlyEqualPt(d0, want0, 1e-5) {
		t.Errorf("D1(0) want %v got %v", want0, d0)
	}
	d1 := quadBezierD1(p0, p1, p2, 1)
	want1 := f32.Pt(10, -20)
	if !nearlyEqualPt(d1, want1, 1e-5) {
		t.Errorf("D1(1) want %v got %v", want1, d1)
	}
}

func TestQuadBezierD2_Constant(t *testing.T) {
	p0 := f32.Pt(0, 0)
	p1 := f32.Pt(5, 10)
	p2 := f32.Pt(10, 0)
	a := quadBezierD2(p0, p1, p2, 0)
	b := quadBezierD2(p0, p1, p2, 0.5)
	c := quadBezierD2(p0, p1, p2, 1)
	if a != b || b != c {
		t.Errorf("d2 should be constant, got %v %v %v", a, b, c)
	}
	want := f32.Pt(0, -40)
	if !nearlyEqualPt(a, want, 1e-5) {
		t.Errorf("d2 want %v got %v", want, a)
	}
}

func TestQuadInterp(t *testing.T) {
	p := f32.Pt(0, 0)
	q := f32.Pt(10, 20)
	if got := quadInterp(p, q, 0); got != p {
		t.Errorf("t=0 got %v", got)
	}
	if got := quadInterp(p, q, 1); got != q {
		t.Errorf("t=1 got %v", got)
	}
	if got := quadInterp(p, q, 0.5); !nearlyEqualPt(got, f32.Pt(5, 10), 1e-6) {
		t.Errorf("t=0.5 got %v", got)
	}
}

func TestQuadBezierSplit_Endpoints(t *testing.T) {
	p0 := f32.Pt(0, 0)
	p1 := f32.Pt(5, 10)
	p2 := f32.Pt(10, 0)
	for _, ti := range []float32{0.1, 0.5, 0.9} {
		b0, _, b2, a0, _, a2 := quadBezierSplit(p0, p1, p2, ti)
		if b0 != p0 {
			t.Errorf("t=%v: b0=%v want %v", ti, b0, p0)
		}
		if a2 != p2 {
			t.Errorf("t=%v: a2=%v want %v", ti, a2, p2)
		}
		if !nearlyEqualPt(b2, a0, 1e-5) {
			t.Errorf("t=%v: split point mismatch %v vs %v", ti, b2, a0)
		}
		// b2 must equal sample at ti
		samp := quadBezierSample(p0, p1, p2, ti)
		if !nearlyEqualPt(b2, samp, 1e-5) {
			t.Errorf("t=%v: split mid %v != sample %v", ti, b2, samp)
		}
	}
}

func TestPerpDot(t *testing.T) {
	p := f32.Pt(1, 0)
	q := f32.Pt(0, 1)
	if got := perpDot(p, q); got != 1 {
		t.Errorf("perpDot(x,y)=1 want, got %v", got)
	}
	if got := perpDot(q, p); got != -1 {
		t.Errorf("perpDot(y,x)=-1 want, got %v", got)
	}
	// parallel vectors should give 0
	if got := perpDot(p, p); got != 0 {
		t.Errorf("parallel got %v", got)
	}
}

func TestRot90CW(t *testing.T) {
	if got := rot90CW(f32.Pt(1, 0)); got != (f32.Pt(0, -1)) {
		t.Errorf("rot90CW(1,0) got %v", got)
	}
	if got := rot90CW(f32.Pt(0, 1)); got != (f32.Pt(1, 0)) {
		t.Errorf("rot90CW(0,1) got %v", got)
	}
	// Two rotations produce negation
	p := f32.Pt(3, 4)
	if got := rot90CW(rot90CW(p)); got != (f32.Pt(-3, -4)) {
		t.Errorf("double rotate got %v", got)
	}
}

func TestLenPt(t *testing.T) {
	if got := lenPt(f32.Pt(3, 4)); !nearlyEqual(got, 5, 1e-5) {
		t.Errorf("lenPt(3,4) got %v", got)
	}
	if got := lenPt(f32.Pt(0, 0)); got != 0 {
		t.Errorf("lenPt(0,0) got %v", got)
	}
	// large value
	if got := lenPt(f32.Pt(3e10, 4e10)); !nearlyEqual(got, 5e10, 1e6) {
		t.Errorf("large lenPt got %v", got)
	}
}

func TestAngleBetween(t *testing.T) {
	// 90° CCW rotation in math coords
	a := angleBetween(f32.Pt(1, 0), f32.Pt(0, 1))
	if !nearlyEqual(float32(a), float32(math.Pi/2), 1e-5) {
		t.Errorf("expected pi/2 got %v", a)
	}
	a = angleBetween(f32.Pt(1, 0), f32.Pt(1, 0))
	if a != 0 {
		t.Errorf("expected 0 got %v", a)
	}
}

func TestStrokePathNorm_ZeroSegment(t *testing.T) {
	// p0==p1 should produce zero normal at t=0
	p := f32.Pt(5, 5)
	n := strokePathNorm(p, p, f32.Pt(10, 10), 0, 1)
	if n != (f32.Point{}) {
		t.Errorf("expected zero, got %v", n)
	}
	// p1==p2 should produce zero at t=1
	n = strokePathNorm(f32.Pt(0, 0), p, p, 1, 1)
	if n != (f32.Point{}) {
		t.Errorf("expected zero, got %v", n)
	}
}

func TestStrokePathNorm_Magnitude(t *testing.T) {
	// horizontal segment, half-width=2 should give normal of magnitude 2
	n := strokePathNorm(f32.Pt(0, 0), f32.Pt(10, 0), f32.Pt(20, 0), 0, 2)
	if !nearlyEqual(lenPt(n), 2, 1e-5) {
		t.Errorf("expected magnitude 2, got %v (%v)", lenPt(n), n)
	}
	// at t=1
	n = strokePathNorm(f32.Pt(0, 0), f32.Pt(10, 0), f32.Pt(20, 0), 1, 2)
	if !nearlyEqual(lenPt(n), 2, 1e-5) {
		t.Errorf("expected magnitude 2, got %v (%v)", lenPt(n), n)
	}
}

func TestStrokePathNorm_PanicOnInvalidT(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for t out of {0,1}")
		}
	}()
	strokePathNorm(f32.Pt(0, 0), f32.Pt(1, 1), f32.Pt(2, 0), 0.5, 1)
}

func TestStrokePathCurv_Linear(t *testing.T) {
	// a true straight line (collinear) should produce NaN
	c := strokePathCurv(f32.Pt(0, 0), f32.Pt(5, 0), f32.Pt(10, 0), 0.0)
	if !math.IsNaN(float64(c)) {
		t.Errorf("expected NaN for straight line at t=0, got %v", c)
	}
	c = strokePathCurv(f32.Pt(0, 0), f32.Pt(5, 0), f32.Pt(10, 0), 1.0)
	if !math.IsNaN(float64(c)) {
		t.Errorf("expected NaN at t=1, got %v", c)
	}
}

func TestStrokePathCurv_Symmetric(t *testing.T) {
	// symmetric quadratic about x=5 — curvature at t=0 and t=1 should match in magnitude.
	c0 := strokePathCurv(f32.Pt(0, 0), f32.Pt(5, 5), f32.Pt(10, 0), 0)
	c1 := strokePathCurv(f32.Pt(0, 0), f32.Pt(5, 5), f32.Pt(10, 0), 1)
	if math.IsNaN(float64(c0)) || math.IsNaN(float64(c1)) {
		t.Fatalf("unexpected NaN c0=%v c1=%v", c0, c1)
	}
	if !nearlyEqual(c0, c1, 1e-3) {
		t.Errorf("expected symmetric curvature, c0=%v c1=%v", c0, c1)
	}
}

func TestNormPt_VerySmall(t *testing.T) {
	// Small input, normal length 1
	got := normPt(f32.Pt(1e-20, 0), 1)
	if !nearlyEqual(lenPt(got), 1, 1e-5) {
		t.Errorf("expected magnitude 1, got %v (%v)", lenPt(got), got)
	}
}

func TestNormPt_VeryLarge(t *testing.T) {
	got := normPt(f32.Pt(1e15, 1e15), 1)
	// Should still produce a unit vector
	if !nearlyEqual(lenPt(got), 1, 1e-5) {
		t.Errorf("expected magnitude 1, got %v (%v)", lenPt(got), got)
	}
}

func TestArcTransform_ZeroAngle(t *testing.T) {
	// Zero angle should still yield at least 1 segment (no crash)
	_, segs := ArcTransform(f32.Pt(10, 0), f32.Pt(0, 0), f32.Pt(0, 0), 0)
	if segs < 1 {
		t.Errorf("expected segments >=1, got %d", segs)
	}
}

func TestArcTransform_NegativeAngle(t *testing.T) {
	// Negative angle should still produce positive segment count
	_, segs := ArcTransform(f32.Pt(10, 0), f32.Pt(0, 0), f32.Pt(0, 0), -math.Pi)
	if segs < 1 {
		t.Errorf("expected positive segments, got %d", segs)
	}
}

func TestArcTransform_FullCircle(t *testing.T) {
	// 2π should give 16 segments (segmentsPerCircle const)
	_, segs := ArcTransform(f32.Pt(10, 0), f32.Pt(0, 0), f32.Pt(0, 0), 2*math.Pi)
	if segs != 16 {
		t.Errorf("expected 16 segments for full circle, got %d", segs)
	}
}

func TestArcTransform_CircleProperty(t *testing.T) {
	// Apply transform repeatedly to a starting point: each step should preserve
	// distance from center for a true circle (f1==f2).
	p := f32.Pt(10, 0)
	center := f32.Pt(0, 0)
	m, segs := ArcTransform(p, center, center, math.Pi/2)
	r0 := lenPt(p.Sub(center))
	cur := p
	for i := 0; i < segs*2; i++ {
		cur = m.Transform(cur)
		ri := lenPt(cur.Sub(center))
		if !nearlyEqual(ri, r0, 1e-3) {
			t.Errorf("step %d: radius drift %v -> %v", i, r0, ri)
			break
		}
	}
}

func TestSplitCubic_StraightLine(t *testing.T) {
	// degenerate cubic: all collinear
	from := f32.Pt(0, 0)
	c0 := f32.Pt(10, 0)
	c1 := f32.Pt(20, 0)
	to := f32.Pt(30, 0)
	out := SplitCubic(from, c0, c1, to, nil)
	if len(out) == 0 {
		t.Error("expected at least one quad")
	}
	if out[0].From != from || out[len(out)-1].To != to {
		t.Errorf("endpoints mismatch: from=%v to=%v", out[0].From, out[len(out)-1].To)
	}
}

func TestSplitCubic_Coincident(t *testing.T) {
	// All 4 points coincident — degenerate point
	p := f32.Pt(5, 5)
	out := SplitCubic(p, p, p, p, nil)
	if len(out) == 0 {
		t.Error("expected at least one quad")
	}
	for _, q := range out {
		if q.From != p || q.To != p {
			t.Errorf("expected all-p quad, got %v", q)
		}
	}
}

func TestSplitCubic_Continuity(t *testing.T) {
	// quads from SplitCubic should chain end-to-start
	out := SplitCubic(
		f32.Pt(0, 0), f32.Pt(10, 30), f32.Pt(20, -30), f32.Pt(30, 0), nil,
	)
	if len(out) < 2 {
		t.Skip("only one quad, skipping continuity check")
	}
	for i := 1; i < len(out); i++ {
		if !nearlyEqualPt(out[i-1].To, out[i].From, 1e-4) {
			t.Errorf("discontinuity at %d: %v -> %v", i, out[i-1].To, out[i].From)
		}
	}
}

func TestSplitCubic_BoundsContainEndpoints(t *testing.T) {
	from := f32.Pt(-5, -10)
	to := f32.Pt(15, 20)
	out := SplitCubic(from, f32.Pt(0, 0), f32.Pt(10, 10), to, nil)
	if out[0].From != from {
		t.Errorf("first From=%v want %v", out[0].From, from)
	}
	if out[len(out)-1].To != to {
		t.Errorf("last To=%v want %v", out[len(out)-1].To, to)
	}
}

func TestStrokeQuads_PenMultiple(t *testing.T) {
	qs := StrokeQuads{
		{Quad: QuadSegment{From: f32.Pt(0, 0), To: f32.Pt(10, 0)}},
		{Quad: QuadSegment{From: f32.Pt(10, 0), To: f32.Pt(20, 5)}},
	}
	if got := qs.pen(); got != (f32.Pt(20, 5)) {
		t.Errorf("pen got %v", got)
	}
}

func TestStrokeQuads_AppendEmpty(t *testing.T) {
	a := StrokeQuads{
		{Quad: QuadSegment{From: f32.Pt(0, 0), To: f32.Pt(10, 0)}},
	}
	if got := a.append(nil); len(got) != 1 {
		t.Errorf("append nil to len-1: got len %d", len(got))
	}
	var empty StrokeQuads
	if got := empty.append(a); len(got) != 1 {
		t.Errorf("append a to nil: got len %d", len(got))
	}
}

func TestStrokeQuads_AppendBridgesGap(t *testing.T) {
	// Endpoints differ by less than strokeTolerance — should bridge with a quad.
	a := StrokeQuads{
		{Quad: QuadSegment{From: f32.Pt(0, 0), To: f32.Pt(10, 0)}},
	}
	b := StrokeQuads{
		{Quad: QuadSegment{From: f32.Pt(10.001, 0), To: f32.Pt(20, 0)}},
	}
	got := a.append(b)
	if len(got) != 3 {
		t.Errorf("expected bridge insertion (len 3), got %d", len(got))
	}
}

func TestStrokeQuads_AppendNoBridgeWhenFar(t *testing.T) {
	a := StrokeQuads{
		{Quad: QuadSegment{From: f32.Pt(0, 0), To: f32.Pt(10, 0)}},
	}
	b := StrokeQuads{
		{Quad: QuadSegment{From: f32.Pt(20, 0), To: f32.Pt(30, 0)}},
	}
	got := a.append(b)
	if len(got) != 2 {
		t.Errorf("expected no bridge (len 2), got %d", len(got))
	}
}

func TestStrokeQuads_ReverseEmpty(t *testing.T) {
	var empty StrokeQuads
	if got := empty.reverse(); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestStrokeQuads_ReverseChain(t *testing.T) {
	// Reverse twice should return original sequence.
	qs := StrokeQuads{
		{Quad: QuadSegment{From: f32.Pt(0, 0), Ctrl: f32.Pt(5, 5), To: f32.Pt(10, 0)}},
		{Quad: QuadSegment{From: f32.Pt(10, 0), Ctrl: f32.Pt(15, -5), To: f32.Pt(20, 0)}},
	}
	rev2 := qs.reverse().reverse()
	if len(rev2) != len(qs) {
		t.Fatalf("len mismatch: %d vs %d", len(rev2), len(qs))
	}
	for i := range qs {
		if rev2[i].Quad.From != qs[i].Quad.From || rev2[i].Quad.To != qs[i].Quad.To {
			t.Errorf("at %d: got %+v want %+v", i, rev2[i].Quad, qs[i].Quad)
		}
	}
}

func TestStrokeQuads_SplitEmpty(t *testing.T) {
	var qs StrokeQuads
	if parts := qs.split(); parts != nil {
		t.Errorf("expected nil split, got %v", parts)
	}
}

func TestStrokeQuads_SplitMultipleContours(t *testing.T) {
	qs := StrokeQuads{
		{Contour: 0, Quad: QuadSegment{To: f32.Pt(1, 0)}},
		{Contour: 1, Quad: QuadSegment{To: f32.Pt(2, 0)}},
		{Contour: 2, Quad: QuadSegment{To: f32.Pt(3, 0)}},
		{Contour: 2, Quad: QuadSegment{To: f32.Pt(4, 0)}},
	}
	parts := qs.split()
	if len(parts) != 3 {
		t.Errorf("expected 3 parts, got %d", len(parts))
	}
	if len(parts[2]) != 2 {
		t.Errorf("expected last part len 2, got %d", len(parts[2]))
	}
}

func TestStrokeQuads_StrokeClosed(t *testing.T) {
	// A closed triangle.
	qs := StrokeQuads{
		{Quad: QuadSegment{From: f32.Pt(0, 0), Ctrl: f32.Pt(5, 0), To: f32.Pt(10, 0)}},
		{Quad: QuadSegment{From: f32.Pt(10, 0), Ctrl: f32.Pt(10, 5), To: f32.Pt(10, 10)}},
		{Quad: QuadSegment{From: f32.Pt(10, 10), Ctrl: f32.Pt(5, 5), To: f32.Pt(0, 0)}},
	}
	out := qs.stroke(StrokeStyle{Width: 1})
	if len(out) == 0 {
		t.Error("closed stroke produced no quads")
	}
}

func TestStrokeQuads_StrokeVerySmallWidth(t *testing.T) {
	qs := StrokeQuads{
		{Quad: QuadSegment{From: f32.Pt(0, 0), Ctrl: f32.Pt(5, 0), To: f32.Pt(10, 0)}},
	}
	out := qs.stroke(StrokeStyle{Width: 1e-6})
	// With tiny width, output may be empty or tiny — should not panic.
	_ = out
}

func TestStrokeQuads_StrokeLargeWidth(t *testing.T) {
	qs := StrokeQuads{
		{Quad: QuadSegment{From: f32.Pt(0, 0), Ctrl: f32.Pt(50, 50), To: f32.Pt(100, 0)}},
	}
	out := qs.stroke(StrokeStyle{Width: 1000})
	if len(out) == 0 {
		t.Error("expected stroke output for large width")
	}
}

func TestStrokeQuads_StrokeZeroWidth(t *testing.T) {
	qs := StrokeQuads{
		{Quad: QuadSegment{From: f32.Pt(0, 0), Ctrl: f32.Pt(5, 5), To: f32.Pt(10, 0)}},
	}
	out := qs.stroke(StrokeStyle{Width: 0})
	// With zero width, the stroke should not panic; output may be degenerate.
	_ = out
}

func TestStrokeQuads_LineToContinuity(t *testing.T) {
	// chained lineTo calls should leave a connected path
	qs := StrokeQuads{
		{Quad: QuadSegment{To: f32.Pt(0, 0)}},
	}
	pts := []f32.Point{
		f32.Pt(10, 0), f32.Pt(10, 10), f32.Pt(0, 10), f32.Pt(0, 0),
	}
	for _, p := range pts {
		qs.lineTo(p)
	}
	for i := 1; i < len(qs); i++ {
		if qs[i].Quad.From != qs[i-1].Quad.To {
			t.Errorf("discontinuity at %d", i)
		}
	}
}

func TestQuadSegment_TransformIdentity(t *testing.T) {
	q := QuadSegment{From: f32.Pt(1, 2), Ctrl: f32.Pt(3, 4), To: f32.Pt(5, 6)}
	got := q.Transform(f32.AffineId())
	if got != q {
		t.Errorf("identity changed segment: %v -> %v", q, got)
	}
}

func TestStrokeQuads_CCWEmpty(t *testing.T) {
	var qs StrokeQuads
	// area is 0 — function returns true (area<=0)
	if !qs.ccw() {
		t.Errorf("empty should be considered ccw (area==0)")
	}
}

func TestStrokeQuads_ArcZeroAngle(t *testing.T) {
	qs := StrokeQuads{
		{Quad: QuadSegment{To: f32.Pt(0, 0)}},
	}
	qs.arc(f32.Pt(10, 0), f32.Pt(10, 0), 0)
	// Should produce at least one quad via segments=1 fallback
	if len(qs) < 2 {
		t.Errorf("expected at least one extra quad, got len %d", len(qs))
	}
}

func TestArcTransform_Determinism(t *testing.T) {
	// Same input must produce same output.
	m1, s1 := ArcTransform(f32.Pt(10, 0), f32.Pt(0, 0), f32.Pt(0, 0), math.Pi)
	m2, s2 := ArcTransform(f32.Pt(10, 0), f32.Pt(0, 0), f32.Pt(0, 0), math.Pi)
	if s1 != s2 {
		t.Errorf("segments differ %d vs %d", s1, s2)
	}
	a1sx, a1hx, a1ox, a1hy, a1sy, a1oy := m1.Elems()
	a2sx, a2hx, a2ox, a2hy, a2sy, a2oy := m2.Elems()
	if a1sx != a2sx || a1hx != a2hx || a1ox != a2ox ||
		a1hy != a2hy || a1sy != a2sy || a1oy != a2oy {
		t.Error("affine transforms differ between identical calls")
	}
}

func TestNormPt_LengthZero(t *testing.T) {
	// length 0 must produce zero point (not div-by-zero NaN)
	got := normPt(f32.Pt(3, 4), 0)
	if got != (f32.Point{}) {
		t.Errorf("expected zero, got %v", got)
	}
}
