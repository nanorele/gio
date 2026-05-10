package gpu

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/nanorele/gio/internal/f32"
	"github.com/nanorele/gio/internal/stroke"
)

func BenchmarkEncodeQuadTo(b *testing.B) {
	var data [vertStride * 4]byte
	for i := 0; b.Loop(); i++ {
		v := float32(i)
		encodeQuadTo(data[:], 123,
			f32.Point{X: v, Y: v},
			f32.Point{X: v, Y: v},
			f32.Point{X: v, Y: v},
		)
	}
}

// readVertex decodes one 32-byte vertex written by encodeQuadTo.
func readVertex(data []byte) (corner float32, meta uint32, from, ctrl, to f32.Point) {
	bo := binary.LittleEndian
	corner = math.Float32frombits(bo.Uint32(data[0:4]))
	meta = bo.Uint32(data[4:8])
	from.X = math.Float32frombits(bo.Uint32(data[8:12]))
	from.Y = math.Float32frombits(bo.Uint32(data[12:16]))
	ctrl.X = math.Float32frombits(bo.Uint32(data[16:20]))
	ctrl.Y = math.Float32frombits(bo.Uint32(data[20:24]))
	to.X = math.Float32frombits(bo.Uint32(data[24:28]))
	to.Y = math.Float32frombits(bo.Uint32(data[28:32]))
	return
}

func TestEncodeQuadToLayout(t *testing.T) {
	var data [vertStride * 4]byte
	from := f32.Point{X: 1, Y: 2}
	ctrl := f32.Point{X: 3, Y: 4}
	to := f32.Point{X: 5, Y: 6}
	const meta = uint32(0xDEADBEEF)
	encodeQuadTo(data[:], meta, from, ctrl, to)

	wantCorners := [4]float32{nwCorner, neCorner, swCorner, seCorner}
	for i := range 4 {
		off := i * vertStride
		c, m, f, ct, tp := readVertex(data[off : off+vertStride])
		if c != wantCorners[i] {
			t.Errorf("vertex %d corner = %v, want %v", i, c, wantCorners[i])
		}
		if m != meta {
			t.Errorf("vertex %d meta = %x, want %x", i, m, meta)
		}
		if f != from || ct != ctrl || tp != to {
			t.Errorf("vertex %d coords = %v %v %v, want %v %v %v", i, f, ct, tp, from, ctrl, to)
		}
	}
}

func TestEncodeQuadToZero(t *testing.T) {
	var data [vertStride * 4]byte
	encodeQuadTo(data[:], 0, f32.Point{}, f32.Point{}, f32.Point{})
	for i := range 4 {
		off := i * vertStride
		_, m, f, c, tp := readVertex(data[off : off+vertStride])
		if m != 0 {
			t.Errorf("vertex %d meta = %x, want 0", i, m)
		}
		zero := f32.Point{}
		if f != zero || c != zero || tp != zero {
			t.Errorf("vertex %d non-zero coords: %v %v %v", i, f, c, tp)
		}
	}
}

func TestEncodeQuadToExtremeValues(t *testing.T) {
	cases := []struct {
		name           string
		from, ctrl, to f32.Point
	}{
		{"tiny", f32.Point{X: 1e-30, Y: -1e-30}, f32.Point{X: 1e-30, Y: 1e-30}, f32.Point{X: -1e-30, Y: -1e-30}},
		{"huge", f32.Point{X: 1e30, Y: -1e30}, f32.Point{X: 1e30, Y: 1e30}, f32.Point{X: -1e30, Y: -1e30}},
		{"negative", f32.Point{X: -100, Y: -200}, f32.Point{X: -300, Y: -400}, f32.Point{X: -500, Y: -600}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var data [vertStride * 4]byte
			encodeQuadTo(data[:], 7, tc.from, tc.ctrl, tc.to)
			for i := range 4 {
				off := i * vertStride
				c, _, f, ct, tp := readVertex(data[off : off+vertStride])
				if math.IsNaN(float64(c)) || math.IsInf(float64(c), 0) {
					t.Errorf("vertex %d corner is NaN/Inf: %v", i, c)
				}
				for _, p := range []f32.Point{f, ct, tp} {
					if math.IsNaN(float64(p.X)) || math.IsNaN(float64(p.Y)) ||
						math.IsInf(float64(p.X), 0) || math.IsInf(float64(p.Y), 0) {
						t.Errorf("vertex %d point NaN/Inf: %v", i, p)
					}
				}
			}
		})
	}
}

func TestEncodeQuadToOverwritesExistingBytes(t *testing.T) {
	var data [vertStride * 4]byte
	for i := range data {
		data[i] = 0xAA
	}
	from := f32.Point{X: 10, Y: 20}
	ctrl := f32.Point{X: 30, Y: 40}
	to := f32.Point{X: 50, Y: 60}
	encodeQuadTo(data[:], 1, from, ctrl, to)
	// Every byte in the 128-byte region must have been written; none should remain 0xAA
	// for the per-vertex header bytes [0:8]. Coord bytes are also fully overwritten.
	for i := range 4 {
		off := i * vertStride
		_, m, f, c, tp := readVertex(data[off : off+vertStride])
		if m != 1 {
			t.Errorf("vertex %d meta not written: %x", i, m)
		}
		if f != from || c != ctrl || tp != to {
			t.Errorf("vertex %d garbage in coords: %v %v %v", i, f, c, tp)
		}
	}
}

func TestEncodeVertexCorners(t *testing.T) {
	from := f32.Point{X: 1, Y: 2}
	ctrl := f32.Point{X: 3, Y: 4}
	to := f32.Point{X: 5, Y: 6}
	cases := []struct {
		cx, cy int16
		want   float32
	}{
		{0, 0, 0},
		{1, 0, 0.5},
		{0, 1, 0.25},
		{1, 1, 0.75},
	}
	for _, tc := range cases {
		var data [vertStride]byte
		encodeVertex(data[:], 9, tc.cx, tc.cy, from, ctrl, to)
		c, m, f, ct, tp := readVertex(data[:])
		if c != tc.want {
			t.Errorf("corner(%d,%d) = %v, want %v", tc.cx, tc.cy, c, tc.want)
		}
		if m != 9 {
			t.Errorf("meta = %x", m)
		}
		if f != from || ct != ctrl || tp != to {
			t.Errorf("coords mismatch: %v %v %v", f, ct, tp)
		}
	}
}

func TestUnionRectEmpty(t *testing.T) {
	a := f32.Rectangle{Min: f32.Point{X: 1, Y: 2}, Max: f32.Point{X: 3, Y: 4}}
	b := f32.Rectangle{}
	got := unionRect(a, b)
	// unionRect doesn't special-case empty rects; it just min/max each side.
	want := f32.Rectangle{Min: f32.Point{X: 0, Y: 0}, Max: f32.Point{X: 3, Y: 4}}
	if got != want {
		t.Errorf("unionRect(a, empty) = %v, want %v", got, want)
	}
}

func TestUnionRectIdempotent(t *testing.T) {
	a := f32.Rectangle{Min: f32.Point{X: -5, Y: -10}, Max: f32.Point{X: 5, Y: 10}}
	got := unionRect(a, a)
	if got != a {
		t.Errorf("unionRect(a,a) = %v, want %v", got, a)
	}
}

func TestUnionRectContaining(t *testing.T) {
	big := f32.Rectangle{Min: f32.Point{X: -100, Y: -100}, Max: f32.Point{X: 100, Y: 100}}
	small := f32.Rectangle{Min: f32.Point{X: -1, Y: -1}, Max: f32.Point{X: 1, Y: 1}}
	if got := unionRect(big, small); got != big {
		t.Errorf("union(big,small) = %v, want %v", got, big)
	}
	if got := unionRect(small, big); got != big {
		t.Errorf("union(small,big) = %v, want %v", got, big)
	}
}

func TestUnionRectDisjoint(t *testing.T) {
	a := f32.Rectangle{Min: f32.Point{X: 0, Y: 0}, Max: f32.Point{X: 1, Y: 1}}
	b := f32.Rectangle{Min: f32.Point{X: 10, Y: 10}, Max: f32.Point{X: 11, Y: 11}}
	want := f32.Rectangle{Min: f32.Point{X: 0, Y: 0}, Max: f32.Point{X: 11, Y: 11}}
	if got := unionRect(a, b); got != want {
		t.Errorf("union = %v, want %v", got, want)
	}
}

// newQuadSplitter creates a quadSplitter with a real drawOps so we can exercise
// splitAndEncode end-to-end.
func newQuadSplitter() *quadSplitter {
	return &quadSplitter{
		bounds: f32.Rectangle{
			Min: f32.Point{X: math.MaxFloat32, Y: math.MaxFloat32},
			Max: f32.Point{X: -math.MaxFloat32, Y: -math.MaxFloat32},
		},
		d: &drawOps{},
	}
}

func TestSplitAndEncodeSimple(t *testing.T) {
	qs := newQuadSplitter()
	q := stroke.QuadSegment{
		From: f32.Point{X: 0, Y: 0},
		Ctrl: f32.Point{X: 1, Y: 1},
		To:   f32.Point{X: 2, Y: 0},
	}
	qs.splitAndEncode(q)
	// Single quad => one segment of 4 vertices = 128 bytes.
	if got := len(qs.d.vertCache); got != vertStride*4 {
		t.Errorf("vertCache len = %d, want %d", got, vertStride*4)
	}
}

func TestSplitAndEncodeMonotonicNoSplit(t *testing.T) {
	// A monotonic quad (control same as from/to in X) should not split.
	qs := newQuadSplitter()
	q := stroke.QuadSegment{
		From: f32.Point{X: 0, Y: 0},
		Ctrl: f32.Point{X: 0, Y: 1},
		To:   f32.Point{X: 0, Y: 2},
	}
	qs.splitAndEncode(q)
	if got := len(qs.d.vertCache); got != vertStride*4 {
		t.Errorf("monotonic quad split: vertCache len = %d", got)
	}
}

func TestSplitAndEncodeXExtremum(t *testing.T) {
	// Control's v0 and v1 differ in sign on X => extremum exists, must split.
	qs := newQuadSplitter()
	q := stroke.QuadSegment{
		From: f32.Point{X: 0, Y: 0},
		Ctrl: f32.Point{X: 5, Y: 1},
		To:   f32.Point{X: 0, Y: 2},
	}
	qs.splitAndEncode(q)
	// Should split into two quads = 8 vertices
	if got := len(qs.d.vertCache); got != vertStride*8 {
		t.Errorf("X-extremum quad: vertCache len = %d, want %d", got, vertStride*8)
	}
	if qs.bounds.Max.X < 2.5 {
		t.Errorf("bounds Max.X should account for X extremum, got %v", qs.bounds.Max.X)
	}
}

func TestSplitAndEncodeYExtremum(t *testing.T) {
	qs := newQuadSplitter()
	// Y-only extremum: from→ctrl positive Y, ctrl→to negative Y.
	q := stroke.QuadSegment{
		From: f32.Point{X: 0, Y: 0},
		Ctrl: f32.Point{X: 1, Y: 5},
		To:   f32.Point{X: 2, Y: 0},
	}
	qs.splitAndEncode(q)
	if qs.bounds.Max.Y < 2.0 {
		t.Errorf("bounds Max.Y should track Y extremum, got %v", qs.bounds.Max.Y)
	}
}

func TestSplitAndEncodeEmptyQuad(t *testing.T) {
	qs := newQuadSplitter()
	q := stroke.QuadSegment{} // all zero points
	qs.splitAndEncode(q)
	// A degenerate zero quad still encodes 1 segment (4 vertices).
	if got := len(qs.d.vertCache); got != vertStride*4 {
		t.Errorf("vertCache len = %d, want %d", got, vertStride*4)
	}
}

func TestSplitAndEncodeExtremeCoords(t *testing.T) {
	qs := newQuadSplitter()
	q := stroke.QuadSegment{
		From: f32.Point{X: 1e10, Y: 1e10},
		Ctrl: f32.Point{X: -1e10, Y: -1e10},
		To:   f32.Point{X: 1e10, Y: 1e10},
	}
	// Should not crash; verify produced bytes contain no NaN.
	qs.splitAndEncode(q)
	bo := binary.LittleEndian
	for i := 0; i+4 <= len(qs.d.vertCache); i += 4 {
		v := math.Float32frombits(bo.Uint32(qs.d.vertCache[i : i+4]))
		if math.IsNaN(float64(v)) {
			t.Fatalf("NaN at offset %d", i)
		}
	}
}

func TestCornerConstants(t *testing.T) {
	// Sanity: corners must lie in [0,1] and be all distinct.
	corners := []float32{nwCorner, neCorner, swCorner, seCorner}
	for i, c := range corners {
		if c < 0 || c > 1 {
			t.Errorf("corner %d out of [0,1]: %v", i, c)
		}
	}
	for i := range corners {
		for j := i + 1; j < len(corners); j++ {
			if corners[i] == corners[j] {
				t.Errorf("corners %d and %d are equal: %v", i, j, corners[i])
			}
		}
	}
}
