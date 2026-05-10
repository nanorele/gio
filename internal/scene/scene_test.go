package scene

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"
	"unsafe"

	"github.com/nanorele/gio/internal/f32"
)

func TestCommandSize(t *testing.T) {
	if got, want := CommandSize, int(unsafe.Sizeof(Command{})); got != want {
		t.Errorf("CommandSize = %d, want %d", got, want)
	}
	if CommandSize != sceneElemSize {
		t.Errorf("CommandSize = %d, want sceneElemSize %d", CommandSize, sceneElemSize)
	}
}

func TestLineRoundTrip(t *testing.T) {
	cases := []struct {
		from, to f32.Point
	}{
		{f32.Pt(0, 0), f32.Pt(0, 0)},
		{f32.Pt(1, 2), f32.Pt(3, 4)},
		{f32.Pt(-1.5, -2.25), f32.Pt(100.125, -50.5)},
		{f32.Pt(float32(math.Inf(1)), float32(math.Inf(-1))), f32.Pt(0, 0)},
	}
	for _, tc := range cases {
		cmd := Line(tc.from, tc.to)
		if got, want := cmd.Op(), OpLine; got != want {
			t.Errorf("Op = %v, want %v", got, want)
		}
		from, to := DecodeLine(cmd)
		if from != tc.from || to != tc.to {
			t.Errorf("Line(%v,%v) -> (%v,%v)", tc.from, tc.to, from, to)
		}
	}
}

func TestGapRoundTrip(t *testing.T) {
	cases := []struct {
		from, to f32.Point
	}{
		{f32.Pt(0, 0), f32.Pt(0, 0)},
		{f32.Pt(1, 2), f32.Pt(3, 4)},
		{f32.Pt(-7, 8), f32.Pt(9, -10)},
	}
	for _, tc := range cases {
		cmd := Gap(tc.from, tc.to)
		if got, want := cmd.Op(), OpGap; got != want {
			t.Errorf("Op = %v, want %v", got, want)
		}
		from, to := DecodeGap(cmd)
		if from != tc.from || to != tc.to {
			t.Errorf("Gap(%v,%v) -> (%v,%v)", tc.from, tc.to, from, to)
		}
	}
}

func TestQuadRoundTrip(t *testing.T) {
	cases := []struct {
		from, ctrl, to f32.Point
	}{
		{f32.Pt(0, 0), f32.Pt(0, 0), f32.Pt(0, 0)},
		{f32.Pt(1, 2), f32.Pt(3, 4), f32.Pt(5, 6)},
		{f32.Pt(-1, -2), f32.Pt(-3, -4), f32.Pt(-5, -6)},
	}
	for _, tc := range cases {
		cmd := Quad(tc.from, tc.ctrl, tc.to)
		if got, want := cmd.Op(), OpQuad; got != want {
			t.Errorf("Op = %v, want %v", got, want)
		}
		from, ctrl, to := DecodeQuad(cmd)
		if from != tc.from || ctrl != tc.ctrl || to != tc.to {
			t.Errorf("Quad(%v,%v,%v) -> (%v,%v,%v)", tc.from, tc.ctrl, tc.to, from, ctrl, to)
		}
	}
}

func TestCubicRoundTrip(t *testing.T) {
	cases := []struct {
		from, c0, c1, to f32.Point
	}{
		{f32.Pt(0, 0), f32.Pt(0, 0), f32.Pt(0, 0), f32.Pt(0, 0)},
		{f32.Pt(1, 2), f32.Pt(3, 4), f32.Pt(5, 6), f32.Pt(7, 8)},
		{f32.Pt(-1.5, 2.5), f32.Pt(3.25, -4.75), f32.Pt(5, 6.125), f32.Pt(-7, -8)},
	}
	for _, tc := range cases {
		cmd := Cubic(tc.from, tc.c0, tc.c1, tc.to)
		if got, want := cmd.Op(), OpCubic; got != want {
			t.Errorf("Op = %v, want %v", got, want)
		}
		from, c0, c1, to := DecodeCubic(cmd)
		if from != tc.from || c0 != tc.c0 || c1 != tc.c1 || to != tc.to {
			t.Errorf("Cubic(%v,%v,%v,%v) -> (%v,%v,%v,%v)", tc.from, tc.c0, tc.c1, tc.to, from, c0, c1, to)
		}
	}
}

func TestFillColor(t *testing.T) {
	cases := []struct {
		col  color.RGBA
		want uint32
	}{
		{color.RGBA{0, 0, 0, 0}, 0x00000000},
		{color.RGBA{255, 255, 255, 255}, 0xffffffff},
		{color.RGBA{0xab, 0xcd, 0xef, 0x12}, 0xabcdef12},
		{color.RGBA{255, 0, 0, 0}, 0xff000000},
		{color.RGBA{0, 255, 0, 0}, 0x00ff0000},
		{color.RGBA{0, 0, 255, 0}, 0x0000ff00},
		{color.RGBA{0, 0, 0, 255}, 0x000000ff},
		{color.RGBA{1, 2, 3, 4}, 0x01020304},
	}
	for _, tc := range cases {
		cmd := FillColor(tc.col)
		if got, want := cmd.Op(), OpFillColor; got != want {
			t.Errorf("Op = %v, want %v", got, want)
		}
		if got := cmd[1]; got != tc.want {
			t.Errorf("FillColor(%v) = %#08x, want %#08x", tc.col, got, tc.want)
		}
	}
}

func TestFillImage(t *testing.T) {
	cases := []struct {
		index   int
		offset  image.Point
		wantIdx uint32
		wantOff uint32
	}{
		{0, image.Pt(0, 0), 0, 0},
		{1, image.Pt(1, 1), 1, 0x00010001},
		{42, image.Pt(-1, -1), 42, 0xffffffff},
		{7, image.Pt(-32768, 32767), 7, 0x7fff8000},
		{0, image.Pt(32767, -32768), 0, 0x80007fff},
	}
	for _, tc := range cases {
		cmd := FillImage(tc.index, tc.offset)
		if got, want := cmd.Op(), OpFillImage; got != want {
			t.Errorf("Op = %v, want %v", got, want)
		}
		if cmd[1] != tc.wantIdx {
			t.Errorf("FillImage(%d,%v) idx = %#08x, want %#08x", tc.index, tc.offset, cmd[1], tc.wantIdx)
		}
		if cmd[2] != tc.wantOff {
			t.Errorf("FillImage(%d,%v) off = %#08x, want %#08x", tc.index, tc.offset, cmd[2], tc.wantOff)
		}
		gotX := int(int16(uint16(cmd[2] & 0xffff)))
		gotY := int(int16(uint16(cmd[2] >> 16)))
		wantX := int(int16(tc.offset.X))
		wantY := int(int16(tc.offset.Y))
		if gotX != wantX || gotY != wantY {
			t.Errorf("FillImage offset decode = (%d,%d), want (%d,%d)", gotX, gotY, wantX, wantY)
		}
	}
}

func TestFillImagePanicsOnNegativeIndex(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on negative index")
		}
	}()
	FillImage(-1, image.Pt(0, 0))
}

func TestTransformRoundTrip(t *testing.T) {
	cases := [][6]float32{
		{1, 0, 0, 0, 1, 0},
		{2, 3, 4, 5, 6, 7},
		{-1.5, 2.25, -3.125, 4.5, -5.625, 6.75},
		{0, 0, 0, 0, 0, 0},
	}
	for _, e := range cases {
		m := f32.NewAffine2D(e[0], e[1], e[2], e[3], e[4], e[5])
		cmd := Transform(m)
		if got, want := cmd.Op(), OpTransform; got != want {
			t.Errorf("Op = %v, want %v", got, want)
		}
		decoded := f32.NewAffine2D(
			math.Float32frombits(cmd[1]),
			math.Float32frombits(cmd[3]),
			math.Float32frombits(cmd[5]),
			math.Float32frombits(cmd[2]),
			math.Float32frombits(cmd[4]),
			math.Float32frombits(cmd[6]),
		)
		if decoded != m {
			t.Errorf("Transform round trip: got %v, want %v", decoded, m)
		}
		sx, hx, ox, hy, sy, oy := decoded.Elems()
		if sx != e[0] || hx != e[1] || ox != e[2] || hy != e[3] || sy != e[4] || oy != e[5] {
			t.Errorf("Transform elems: got (%v,%v,%v,%v,%v,%v), want (%v,%v,%v,%v,%v,%v)",
				sx, hx, ox, hy, sy, oy, e[0], e[1], e[2], e[3], e[4], e[5])
		}
	}
}

func TestBeginClipRoundTrip(t *testing.T) {
	cases := []f32.Rectangle{
		{Min: f32.Pt(0, 0), Max: f32.Pt(0, 0)},
		{Min: f32.Pt(1, 2), Max: f32.Pt(3, 4)},
		{Min: f32.Pt(-10, -20), Max: f32.Pt(30.5, 40.25)},
	}
	for _, r := range cases {
		cmd := BeginClip(r)
		if got, want := cmd.Op(), OpBeginClip; got != want {
			t.Errorf("Op = %v, want %v", got, want)
		}
		got := f32.Rectangle{
			Min: f32.Pt(math.Float32frombits(cmd[1]), math.Float32frombits(cmd[2])),
			Max: f32.Pt(math.Float32frombits(cmd[3]), math.Float32frombits(cmd[4])),
		}
		if got != r {
			t.Errorf("BeginClip(%v) -> %v", r, got)
		}
	}
}

func TestEndClipRoundTrip(t *testing.T) {
	cases := []f32.Rectangle{
		{Min: f32.Pt(0, 0), Max: f32.Pt(0, 0)},
		{Min: f32.Pt(1, 2), Max: f32.Pt(3, 4)},
		{Min: f32.Pt(-10, -20), Max: f32.Pt(30.5, 40.25)},
	}
	for _, r := range cases {
		cmd := EndClip(r)
		if got, want := cmd.Op(), OpEndClip; got != want {
			t.Errorf("Op = %v, want %v", got, want)
		}
		got := f32.Rectangle{
			Min: f32.Pt(math.Float32frombits(cmd[1]), math.Float32frombits(cmd[2])),
			Max: f32.Pt(math.Float32frombits(cmd[3]), math.Float32frombits(cmd[4])),
		}
		if got != r {
			t.Errorf("EndClip(%v) -> %v", r, got)
		}
	}
}

func TestSetLineWidth(t *testing.T) {
	cases := []float32{0, 1, 1.5, -3.25, 1e6}
	for _, w := range cases {
		cmd := SetLineWidth(w)
		if got, want := cmd.Op(), OpLineWidth; got != want {
			t.Errorf("Op = %v, want %v", got, want)
		}
		if got := math.Float32frombits(cmd[1]); got != w {
			t.Errorf("SetLineWidth(%v) -> %v", w, got)
		}
	}
}

func TestSetFillMode(t *testing.T) {
	cases := []FillMode{FillModeNonzero, FillModeStroke}
	for _, m := range cases {
		cmd := SetFillMode(m)
		if got, want := cmd.Op(), OpSetFillMode; got != want {
			t.Errorf("Op = %v, want %v", got, want)
		}
		if got := FillMode(cmd[1]); got != m {
			t.Errorf("SetFillMode(%v) -> %v", m, got)
		}
	}
}

func TestOp(t *testing.T) {
	cases := []struct {
		cmd  Command
		want Op
	}{
		{Line(f32.Pt(0, 0), f32.Pt(1, 1)), OpLine},
		{Gap(f32.Pt(0, 0), f32.Pt(1, 1)), OpGap},
		{Quad(f32.Pt(0, 0), f32.Pt(1, 1), f32.Pt(2, 2)), OpQuad},
		{Cubic(f32.Pt(0, 0), f32.Pt(1, 1), f32.Pt(2, 2), f32.Pt(3, 3)), OpCubic},
		{FillColor(color.RGBA{1, 2, 3, 4}), OpFillColor},
		{FillImage(0, image.Pt(0, 0)), OpFillImage},
		{SetLineWidth(1), OpLineWidth},
		{SetFillMode(FillModeStroke), OpSetFillMode},
		{Transform(f32.AffineId()), OpTransform},
		{BeginClip(f32.Rectangle{}), OpBeginClip},
		{EndClip(f32.Rectangle{}), OpEndClip},
		{Command{}, OpNop},
	}
	for _, tc := range cases {
		if got := tc.cmd.Op(); got != tc.want {
			t.Errorf("Op = %v, want %v", got, tc.want)
		}
	}
}

func TestString(t *testing.T) {
	cases := []struct {
		cmd     Command
		want    string
		wantPfx string
	}{
		{Command{}, "nop", ""},
		{Line(f32.Pt(1, 2), f32.Pt(3, 4)), "", "line("},
		{Gap(f32.Pt(1, 2), f32.Pt(3, 4)), "", "gap("},
		{Quad(f32.Pt(1, 2), f32.Pt(3, 4), f32.Pt(5, 6)), "", "quad("},
		{Cubic(f32.Pt(1, 2), f32.Pt(3, 4), f32.Pt(5, 6), f32.Pt(7, 8)), "", "cubic("},
		{FillColor(color.RGBA{0xab, 0xcd, 0xef, 0x12}), "fillcolor 0xabcdef12", ""},
		{SetLineWidth(2), "", "linewidth "},
		{Transform(f32.AffineId()), "", "transform "},
		{BeginClip(f32.Rect(1, 2, 3, 4)), "", "beginclip "},
		{EndClip(f32.Rect(1, 2, 3, 4)), "", "endclip "},
		{FillImage(0, image.Pt(0, 0)), "", "fillimage "},
		{SetFillMode(FillModeStroke), "", "setfillmode "},
	}
	for _, tc := range cases {
		got := tc.cmd.String()
		if tc.want != "" && got != tc.want {
			t.Errorf("String = %q, want %q", got, tc.want)
		}
		if tc.wantPfx != "" {
			if len(got) < len(tc.wantPfx) || got[:len(tc.wantPfx)] != tc.wantPfx {
				t.Errorf("String = %q, want prefix %q", got, tc.wantPfx)
			}
		}
	}
}

func TestStringPanicsOnUnknownOp(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on unknown op")
		}
		if msg, ok := r.(string); !ok || msg != "unreachable" {
			t.Errorf("got panic %v, want %q", r, "unreachable")
		}
	}()
	cmd := Command{0: 0xffffffff}
	_ = cmd.String()
}

func TestDecodePanics(t *testing.T) {
	cases := []struct {
		name string
		fn   func()
	}{
		{"DecodeLine on Quad", func() { DecodeLine(Quad(f32.Pt(0, 0), f32.Pt(1, 1), f32.Pt(2, 2))) }},
		{"DecodeLine on Gap", func() { DecodeLine(Gap(f32.Pt(0, 0), f32.Pt(1, 1))) }},
		{"DecodeQuad on Line", func() { DecodeQuad(Line(f32.Pt(0, 0), f32.Pt(1, 1))) }},
		{"DecodeCubic on Quad", func() { DecodeCubic(Quad(f32.Pt(0, 0), f32.Pt(1, 1), f32.Pt(2, 2))) }},
		{"DecodeGap on Line", func() { DecodeGap(Line(f32.Pt(0, 0), f32.Pt(1, 1))) }},
		{"DecodeGap on Nop", func() { DecodeGap(Command{}) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic")
				}
			}()
			tc.fn()
		})
	}
}

func TestStringFormatting(t *testing.T) {
	cmd := Line(f32.Pt(1, 2), f32.Pt(3, 4))
	got := cmd.String()
	want := fmt.Sprintf("line(%v, %v)", f32.Pt(1, 2), f32.Pt(3, 4))
	if got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
}
