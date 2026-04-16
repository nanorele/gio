package stroke

import (
	"math"
	"strconv"
	"testing"

	"github.com/nanorele/gio/internal/f32"
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
		{Quad: QuadSegment{From: f32.Point{0, 0}, To: f32.Point{10, 0}}},
	}
	qs.close()
	if len(qs) != 2 {
		t.Errorf("expected 2 quads after close, got %d", len(qs))
	}
	if qs[1].Quad.To != (f32.Point{0, 0}) {
		t.Errorf("expected close quad to end at (0, 0), got %v", qs[1].Quad.To)
	}

	// Already closed
	qs2 := StrokeQuads{
		{Quad: QuadSegment{From: f32.Point{0, 0}, To: f32.Point{10, 0}}},
		{Quad: QuadSegment{From: f32.Point{10, 0}, To: f32.Point{0, 0}}},
	}
	qs2.close()
	if len(qs2) != 2 {
		t.Errorf("expected still 2 quads, got %d", len(qs2))
	}
	}

	func TestStrokeQuads_Split(t *testing.T) {
	qs := StrokeQuads{
		{Contour: 0, Quad: QuadSegment{To: f32.Point{10, 0}}},
		{Contour: 0, Quad: QuadSegment{To: f32.Point{10, 10}}},
		{Contour: 1, Quad: QuadSegment{To: f32.Point{20, 20}}},
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
			{Quad: QuadSegment{From: f32.Point{0, 0}, To: f32.Point{10, 0}}},
		}
		style := StrokeStyle{Width: 2}
		stroked := qs.stroke(style)
		if len(stroked) == 0 {
			t.Error("stroke returned no quads")
		}
	}

	func TestStrokeQuads_Arc(t *testing.T) {
		qs := StrokeQuads{
			{Quad: QuadSegment{To: f32.Point{0, 0}}},
		}
		qs.arc(f32.Point{10, 0}, f32.Point{0, 10}, math.Pi/2)
		if len(qs) <= 1 {
			t.Errorf("expected more than 1 quad after arc, got %d", len(qs))
		}
	}

	func TestStrokeQuads_CCW(t *testing.T) {
		// CCW rectangle (in Y-up)
		qs := StrokeQuads{
			{Quad: QuadSegment{From: f32.Point{0, 0}, To: f32.Point{10, 0}}},
			{Quad: QuadSegment{From: f32.Point{10, 0}, To: f32.Point{10, 10}}},
			{Quad: QuadSegment{From: f32.Point{10, 10}, To: f32.Point{0, 10}}},
			{Quad: QuadSegment{From: f32.Point{0, 10}, To: f32.Point{0, 0}}},
		}
		if !qs.ccw() {
			t.Error("expected CCW rectangle to be CCW")
		}

		// CW rectangle (in Y-up)
		qsCW := StrokeQuads{
			{Quad: QuadSegment{From: f32.Point{0, 0}, To: f32.Point{0, 10}}},
			{Quad: QuadSegment{From: f32.Point{0, 10}, To: f32.Point{10, 10}}},
			{Quad: QuadSegment{From: f32.Point{10, 10}, To: f32.Point{10, 0}}},
			{Quad: QuadSegment{From: f32.Point{10, 0}, To: f32.Point{0, 0}}},
		}
		if qsCW.ccw() {
			t.Error("expected CW rectangle to NOT be CCW")
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
