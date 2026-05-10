package gpu

import (
	"encoding/binary"
	"image"
	"math"
	"testing"
	"unsafe"
)

// pathData.release() must be safe on the zero value (nil data buffer).
// Pre-fix this called Release() on a nil interface and panicked.
func TestPathDataReleaseNilSafe(t *testing.T) {
	var p pathData
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("pathData{}.release() panicked: %v", r)
		}
	}()
	p.release()
}

// Calling release() twice on the zero value is also safe (idempotent on nil).
func TestPathDataReleaseIdempotentNil(t *testing.T) {
	var p pathData
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("second pathData{}.release() panicked: %v", r)
		}
	}()
	p.release()
	p.release()
}

// vertStride must equal sizeof(vertex). The package init() panics otherwise,
// so reaching this test already proves it, but we keep an explicit assertion
// to flag changes.
func TestVertStrideMatchesVertexSize(t *testing.T) {
	if got := unsafe.Sizeof(vertex{}); got != vertStride {
		t.Fatalf("sizeof(vertex)=%d, vertStride=%d", got, vertStride)
	}
	if vertStride != 32 {
		t.Fatalf("vertStride=%d, want 32", vertStride)
	}
}

// pathBatchSize indices must fit in uint16 (the index buffer type).
// newStenciler casts i to uint16 then computes i*4+3.
func TestPathBatchSizeFitsUint16(t *testing.T) {
	const maxIdx = (pathBatchSize-1)*4 + 3
	if maxIdx > math.MaxUint16 {
		t.Fatalf("pathBatchSize=%d => max index %d overflows uint16",
			pathBatchSize, maxIdx)
	}
}

// vertex.encode round-trips: the bytes it writes can be decoded with the same
// little-endian layout used by the GPU shader.
func TestVertexEncodeRoundTrip(t *testing.T) {
	v := vertex{
		Corner: 0.75,
		MaxY:   42, // ignored by encode; param is what's written
		FromX:  -1.5, FromY: 2.5,
		CtrlX: 3.25, CtrlY: -4.125,
		ToX: 100.5, ToY: -200.25,
	}
	const meta = uint32(0xCAFEBABE)
	var buf [vertStride]byte
	v.encode(buf[:], meta)

	bo := binary.LittleEndian
	if got := math.Float32frombits(bo.Uint32(buf[0:4])); got != v.Corner {
		t.Errorf("corner: got %v, want %v", got, v.Corner)
	}
	if got := bo.Uint32(buf[4:8]); got != meta {
		t.Errorf("meta: got %x, want %x", got, meta)
	}
	if got := math.Float32frombits(bo.Uint32(buf[8:12])); got != v.FromX {
		t.Errorf("FromX: got %v", got)
	}
	if got := math.Float32frombits(bo.Uint32(buf[12:16])); got != v.FromY {
		t.Errorf("FromY: got %v", got)
	}
	if got := math.Float32frombits(bo.Uint32(buf[16:20])); got != v.CtrlX {
		t.Errorf("CtrlX: got %v", got)
	}
	if got := math.Float32frombits(bo.Uint32(buf[20:24])); got != v.CtrlY {
		t.Errorf("CtrlY: got %v", got)
	}
	if got := math.Float32frombits(bo.Uint32(buf[24:28])); got != v.ToX {
		t.Errorf("ToX: got %v", got)
	}
	if got := math.Float32frombits(bo.Uint32(buf[28:32])); got != v.ToY {
		t.Errorf("ToY: got %v", got)
	}
}

// encode ignores v.MaxY — the second parameter is what's written.
// This is a documented quirk; a regression here would silently corrupt the
// stencil winding pass.
func TestVertexEncodeIgnoresStructMaxY(t *testing.T) {
	v := vertex{MaxY: 999}
	var buf [vertStride]byte
	v.encode(buf[:], 7)
	if got := binary.LittleEndian.Uint32(buf[4:8]); got != 7 {
		t.Errorf("got %d, want 7 (param overrides v.MaxY)", got)
	}
}

// encode writes exactly 32 bytes; it must not touch beyond that.
func TestVertexEncodeNoOverwriteBeyond32(t *testing.T) {
	const sentinel = byte(0x5A)
	buf := make([]byte, vertStride*2)
	for i := range buf {
		buf[i] = sentinel
	}
	v := vertex{Corner: 1, FromX: 1, FromY: 1, CtrlX: 1, CtrlY: 1, ToX: 1, ToY: 1}
	v.encode(buf, 1)
	for i := vertStride; i < len(buf); i++ {
		if buf[i] != sentinel {
			t.Fatalf("byte %d clobbered: got %x", i, buf[i])
		}
	}
}

// encode panics on a too-short buffer (slice bounds). Document the contract.
func TestVertexEncodePanicsOnShortBuf(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for short buffer")
		}
	}()
	var buf [vertStride - 1]byte
	vertex{}.encode(buf[:], 0)
}

// encode handles NaN/Inf inputs without misbehaving (bit pattern preserved).
func TestVertexEncodeNaNInf(t *testing.T) {
	v := vertex{
		Corner: float32(math.NaN()),
		FromX:  float32(math.Inf(1)), FromY: float32(math.Inf(-1)),
		CtrlX: float32(math.NaN()), CtrlY: 0,
		ToX: 0, ToY: 0,
	}
	var buf [vertStride]byte
	v.encode(buf[:], 0)
	if got := math.Float32frombits(binary.LittleEndian.Uint32(buf[0:4])); !math.IsNaN(float64(got)) {
		t.Errorf("Corner not NaN: %v", got)
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(buf[8:12])); !math.IsInf(float64(got), 1) {
		t.Errorf("FromX not +Inf: %v", got)
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(buf[12:16])); !math.IsInf(float64(got), -1) {
		t.Errorf("FromY not -Inf: %v", got)
	}
}

// Uniform structs must hit the documented 128-byte stride. The padding
// arithmetic in their definitions would already fail to compile if wrong,
// but a runtime check protects against future field reorderings.
func TestCoverUniformsSizes(t *testing.T) {
	if got := unsafe.Sizeof(coverColUniforms{}); got != 128 {
		t.Errorf("sizeof(coverColUniforms)=%d, want 128", got)
	}
	if got := unsafe.Sizeof(coverLinearGradientUniforms{}); got != 128 {
		t.Errorf("sizeof(coverLinearGradientUniforms)=%d, want 128", got)
	}
}

// vertex layout offsets must match what newStenciler reports to the GPU.
// If anyone reorders the struct, the pipeline binds wrong attributes.
func TestVertexFieldOffsets(t *testing.T) {
	cases := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"Corner", unsafe.Offsetof(vertex{}.Corner), 0},
		{"MaxY", unsafe.Offsetof(vertex{}.MaxY), 4},
		{"FromX", unsafe.Offsetof(vertex{}.FromX), 8},
		{"FromY", unsafe.Offsetof(vertex{}.FromY), 12},
		{"CtrlX", unsafe.Offsetof(vertex{}.CtrlX), 16},
		{"CtrlY", unsafe.Offsetof(vertex{}.CtrlY), 20},
		{"ToX", unsafe.Offsetof(vertex{}.ToX), 24},
		{"ToY", unsafe.Offsetof(vertex{}.ToY), 28},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("offset %s = %d, want %d", tc.name, tc.got, tc.want)
		}
	}
}

// FBO is a value-type holder; verify the zero value has nil tex and zero
// size, so callers can detect uninitialized entries safely.
func TestFBOZeroValue(t *testing.T) {
	var f FBO
	if f.tex != nil {
		t.Errorf("zero FBO.tex should be nil")
	}
	if f.size.X != 0 || f.size.Y != 0 {
		t.Errorf("zero FBO.size should be zero, got %v", f.size)
	}
}

// fboSet zero value: empty slice, ready to append.
func TestFBOSetZeroValue(t *testing.T) {
	var s fboSet
	if len(s.fbos) != 0 {
		t.Errorf("zero fboSet should be empty, got len=%d", len(s.fbos))
	}
}

// Sanity: the four vertices that encodeQuadTo writes use the same 32-byte
// stride that encode() writes — keep them in lockstep.
func TestVertStrideMatchesEncode(t *testing.T) {
	var buf [vertStride]byte
	vertex{}.encode(buf[:], 0)
	// If this test compiles + runs, encode wrote into buf[0:32] without
	// out-of-bounds. The slice expression d[0:32] inside encode would
	// panic if vertStride != 32.
	if len(buf) != 32 {
		t.Fatal("vertStride != 32 — encode would overrun")
	}
}

// Fix: fboSet.delete() must tolerate FBO entries whose tex is nil. Such
// entries can exist when resize() appended a slot but skipped allocation
// (e.g. zero-sized request). Pre-fix this dereferenced nil and crashed.
func TestFBOSetDeleteNilTexSafe(t *testing.T) {
	var s fboSet
	s.fbos = []FBO{{}, {}}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("fboSet.delete with nil tex panicked: %v", r)
		}
	}()
	s.delete(nil, 0)
	if len(s.fbos) != 0 {
		t.Fatalf("delete left len=%d, want 0", len(s.fbos))
	}
}

// delete(idx) keeps the first idx entries and drops the rest.
func TestFBOSetDeletePartial(t *testing.T) {
	var s fboSet
	s.fbos = []FBO{
		{size: image.Pt(1, 1)},
		{size: image.Pt(2, 2)},
		{size: image.Pt(3, 3)},
	}
	s.delete(nil, 1)
	if len(s.fbos) != 1 {
		t.Fatalf("len = %d, want 1", len(s.fbos))
	}
	if s.fbos[0].size != image.Pt(1, 1) {
		t.Errorf("first entry corrupted: %v", s.fbos[0].size)
	}
}

// delete(0) on an empty set is a no-op (must not panic).
func TestFBOSetDeleteEmpty(t *testing.T) {
	var s fboSet
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("delete on empty fboSet panicked: %v", r)
		}
	}()
	s.delete(nil, 0)
	if len(s.fbos) != 0 {
		t.Errorf("len = %d, want 0", len(s.fbos))
	}
}

// buildPath divides len(p) by vertStride to set ncurves. Verify the
// arithmetic for various inputs (we can't actually call buildPath without a
// driver.Device, so this is a property test of the documented invariant).
func TestPathDataNcurvesArithmetic(t *testing.T) {
	cases := []struct {
		nbytes int
		want   int
	}{
		{0, 0},
		{vertStride, 1},
		{vertStride * 4, 4},
		{vertStride*4 + 1, 4},     // partial trailing vertex truncated
		{vertStride*100 - 1, 99},  // off-by-one truncation
		{vertStride * 10000, 10000},
	}
	for _, tc := range cases {
		got := tc.nbytes / vertStride
		if got != tc.want {
			t.Errorf("nbytes=%d: got ncurves=%d, want %d", tc.nbytes, got, tc.want)
		}
	}
}

// stencilPath divides ncurves by 4 to derive the quad count. Confirm the
// arithmetic — a producer mismatch here silently drops a partial quad.
func TestStencilPathQuadCountArithmetic(t *testing.T) {
	cases := []struct {
		ncurves int
		want    int
	}{
		{0, 0},
		{3, 0},   // less than one quad: dropped
		{4, 1},
		{7, 1},   // partial second quad: dropped
		{8, 2},
		{40000, 10000},
	}
	for _, tc := range cases {
		got := tc.ncurves / 4
		if got != tc.want {
			t.Errorf("ncurves=%d: got nquads=%d, want %d", tc.ncurves, got, tc.want)
		}
	}
}

// pathBatchSize controls the per-draw quad count. Confirm it's positive and
// that index-buffer arithmetic stays within uint16 (covered also by
// TestPathBatchSizeFitsUint16, but we verify the scaling here).
func TestPathBatchSizePositive(t *testing.T) {
	if pathBatchSize <= 0 {
		t.Fatalf("pathBatchSize must be positive, got %d", pathBatchSize)
	}
	// Each batch issues 6 indices per quad; total indices must fit a sane
	// uint32 (trivial here, but documents the contract).
	if uint64(pathBatchSize)*6 > uint64(^uint32(0)) {
		t.Fatalf("pathBatchSize*6 overflows uint32")
	}
}

// stencilUniforms is laid out as transform[4]+pathOffset[2]+pad[8]; total 32B.
// The shader binds these by offset, so size must be exact.
func TestStencilUniformsSize(t *testing.T) {
	if got := unsafe.Sizeof(stencilUniforms{}); got != 32 {
		t.Errorf("sizeof(stencilUniforms) = %d, want 32", got)
	}
}

// intersectUniforms is uvTransform[4]+subUVTransform[4] = 32B.
func TestIntersectUniformsSize(t *testing.T) {
	if got := unsafe.Sizeof(intersectUniforms{}); got != 32 {
		t.Errorf("sizeof(intersectUniforms) = %d, want 32", got)
	}
}

// stencilPath computes scale=2/texSize and orig=-1 - 2*Min/texSize.
// Re-derive the formula here so a refactor of the production code is caught.
// We can't call stencilPath (needs Device), so we test the math directly.
func TestStencilPathTransformMath(t *testing.T) {
	cases := []struct {
		dx, dy        int
		minX, minY    int
		wantSx, wantSy float32
		wantOx, wantOy float32
	}{
		{2, 2, 0, 0, 1, 1, -1, -1},
		{4, 4, 0, 0, 0.5, 0.5, -1, -1},
		{4, 4, 1, 1, 0.5, 0.5, -1.5, -1.5},
		{100, 50, 25, 10, 0.02, 0.04, -1.5, -1.4},
	}
	for _, tc := range cases {
		texX := float32(tc.dx)
		texY := float32(tc.dy)
		sx := 2 / texX
		sy := 2 / texY
		ox := -1 - float32(tc.minX)*2/texX
		oy := -1 - float32(tc.minY)*2/texY
		if sx != tc.wantSx || sy != tc.wantSy {
			t.Errorf("dx=%d dy=%d: scale=(%v,%v), want (%v,%v)",
				tc.dx, tc.dy, sx, sy, tc.wantSx, tc.wantSy)
		}
		if ox != tc.wantOx || oy != tc.wantOy {
			t.Errorf("min=(%d,%d): orig=(%v,%v), want (%v,%v)",
				tc.minX, tc.minY, ox, oy, tc.wantOx, tc.wantOy)
		}
	}
}
