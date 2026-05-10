package gpu

import (
	"image"
	"testing"
)

func BenchmarkPacker(b *testing.B) {
	var p packer
	p.maxDims = image.Point{X: 4096, Y: 4096}
	for i := 0; b.Loop(); i++ {
		p.clear()
		p.newPage()
		for k := range 500 {
			_, ok := p.tryAdd(xy(k))
			if !ok {
				b.Fatal("add failed", i, k, xy(k))
			}
		}
	}
}

func xy(v int) image.Point {
	return image.Point{
		X: ((v / 16) % 16) + 8,
		Y: (v % 16) + 8,
	}
}

func newPacker(maxX, maxY int) *packer {
	p := &packer{maxDims: image.Point{X: maxX, Y: maxY}}
	p.newPage()
	return p
}

func TestPackerEmptyAtlasOrigin(t *testing.T) {
	p := newPacker(1024, 1024)
	got, ok := p.tryAdd(image.Point{X: 10, Y: 10})
	if !ok {
		t.Fatal("first add to fresh atlas failed")
	}
	if got.Pos != (image.Point{}) {
		t.Errorf("first item Pos = %v, want (0,0)", got.Pos)
	}
	if got.Idx != 0 {
		t.Errorf("first item Idx = %d, want 0", got.Idx)
	}
}

func TestPackerTryAddBeforeNewPage(t *testing.T) {
	p := &packer{maxDims: image.Point{X: 100, Y: 100}}
	if _, ok := p.tryAdd(image.Point{X: 1, Y: 1}); ok {
		t.Errorf("tryAdd succeeded with no page")
	}
}

func TestPackerItemTooLarge(t *testing.T) {
	p := newPacker(64, 64)
	if _, ok := p.tryAdd(image.Point{X: 100, Y: 10}); ok {
		t.Errorf("tryAdd with X > maxDims succeeded")
	}
	if _, ok := p.tryAdd(image.Point{X: 10, Y: 100}); ok {
		t.Errorf("tryAdd with Y > maxDims succeeded")
	}
	if _, ok := p.tryAdd(image.Point{X: 100, Y: 100}); ok {
		t.Errorf("tryAdd with both > maxDims succeeded")
	}
}

func TestPackerExactFit(t *testing.T) {
	p := newPacker(64, 64)
	got, ok := p.tryAdd(image.Point{X: 64, Y: 64})
	if !ok {
		t.Fatal("tryAdd of exact-fit item failed")
	}
	if got.Pos != (image.Point{}) {
		t.Errorf("exact-fit Pos = %v, want (0,0)", got.Pos)
	}
}

func TestPackerExactFitOneAxis(t *testing.T) {
	p := newPacker(64, 64)
	if _, ok := p.tryAdd(image.Point{X: 64, Y: 32}); !ok {
		t.Fatal("tryAdd with X exactly at max failed")
	}
	p2 := newPacker(64, 64)
	if _, ok := p2.tryAdd(image.Point{X: 32, Y: 64}); !ok {
		t.Fatal("tryAdd with Y exactly at max failed")
	}
}

func TestPackerFillRow(t *testing.T) {
	// Tall narrow atlas forces the packer to stack horizontally to minimize area growth.
	p := newPacker(1000, 10)
	for i := range 10 {
		pl, ok := p.tryAdd(image.Point{X: 10, Y: 10})
		if !ok {
			t.Fatalf("item %d failed", i)
		}
		if pl.Pos.Y != 0 {
			t.Errorf("item %d Y = %d, want 0 (height-constrained atlas)", i, pl.Pos.Y)
		}
	}
}

func TestPackerFillAtlas(t *testing.T) {
	p := newPacker(100, 100)
	// 100 items 10x10 should exactly fill 100x100
	for i := range 100 {
		if _, ok := p.tryAdd(image.Point{X: 10, Y: 10}); !ok {
			t.Fatalf("item %d failed to pack into 100x100 atlas", i)
		}
	}
	// Next 10x10 must fail.
	if _, ok := p.tryAdd(image.Point{X: 10, Y: 10}); ok {
		t.Errorf("packer accepted item beyond atlas capacity")
	}
}

func TestPackerNoOverlap(t *testing.T) {
	p := newPacker(256, 256)
	type rect struct{ x0, y0, x1, y1 int }
	var placed []rect
	sizes := []image.Point{
		{X: 32, Y: 32}, {X: 16, Y: 64}, {X: 64, Y: 16},
		{X: 8, Y: 8}, {X: 100, Y: 50}, {X: 50, Y: 100},
		{X: 10, Y: 10}, {X: 20, Y: 30}, {X: 30, Y: 20},
	}
	for i, s := range sizes {
		pl, ok := p.tryAdd(s)
		if !ok {
			t.Fatalf("size[%d]=%v failed", i, s)
		}
		r := rect{pl.Pos.X, pl.Pos.Y, pl.Pos.X + s.X, pl.Pos.Y + s.Y}
		for j, q := range placed {
			if r.x0 < q.x1 && q.x0 < r.x1 && r.y0 < q.y1 && q.y0 < r.y1 {
				t.Errorf("placement %d %v overlaps placement %d %v", i, r, j, q)
			}
		}
		placed = append(placed, r)
	}
}

func TestPackerDeterminism(t *testing.T) {
	sizes := []image.Point{
		{X: 17, Y: 13}, {X: 5, Y: 9}, {X: 24, Y: 8}, {X: 11, Y: 11},
		{X: 33, Y: 7}, {X: 4, Y: 12}, {X: 19, Y: 19}, {X: 7, Y: 22},
	}
	run := func() []placement {
		p := newPacker(256, 256)
		out := make([]placement, len(sizes))
		for i, s := range sizes {
			pl, ok := p.tryAdd(s)
			if !ok {
				t.Fatalf("size[%d]=%v failed", i, s)
			}
			out[i] = pl
		}
		return out
	}
	a := run()
	b := run()
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("non-deterministic at %d: %v vs %v", i, a[i], b[i])
		}
	}
}

func TestPackerAddCreatesNewPage(t *testing.T) {
	p := &packer{maxDims: image.Point{X: 64, Y: 64}}
	// add (not tryAdd) starts a page on first call
	pl, ok := p.add(image.Point{X: 32, Y: 32})
	if !ok {
		t.Fatal("first add failed")
	}
	if pl.Idx != 0 {
		t.Errorf("first add Idx = %d, want 0", pl.Idx)
	}
	if len(p.sizes) != 1 {
		t.Errorf("len(sizes) = %d, want 1", len(p.sizes))
	}
}

func TestPackerAddOverflowToNewPage(t *testing.T) {
	p := newPacker(64, 64)
	// Fill page with one big item.
	if _, ok := p.add(image.Point{X: 60, Y: 60}); !ok {
		t.Fatal("first add failed")
	}
	// Second item that won't fit alongside: needs full row of height 60.
	pl, ok := p.add(image.Point{X: 60, Y: 60})
	if !ok {
		t.Fatal("second add failed")
	}
	if pl.Idx != 1 {
		t.Errorf("second add Idx = %d, want 1 (new page)", pl.Idx)
	}
	if len(p.sizes) != 2 {
		t.Errorf("len(sizes) = %d, want 2", len(p.sizes))
	}
}

func TestPackerClear(t *testing.T) {
	p := newPacker(64, 64)
	if _, ok := p.tryAdd(image.Point{X: 10, Y: 10}); !ok {
		t.Fatal("add failed")
	}
	p.clear()
	if len(p.sizes) != 0 || len(p.spaces) != 0 {
		t.Errorf("after clear: sizes=%d spaces=%d", len(p.sizes), len(p.spaces))
	}
	// Should require a new page now.
	if _, ok := p.tryAdd(image.Point{X: 1, Y: 1}); ok {
		t.Errorf("tryAdd succeeded after clear without newPage")
	}
}

func TestPackerAtlasGrowth(t *testing.T) {
	p := newPacker(1024, 1024)
	p.tryAdd(image.Point{X: 10, Y: 10})
	if p.sizes[0] != (image.Point{X: 10, Y: 10}) {
		t.Errorf("size after one item = %v", p.sizes[0])
	}
	p.tryAdd(image.Point{X: 50, Y: 5})
	// Atlas should grow to fit both. X=50 is wider, so size.X = 50.
	if p.sizes[0].X < 50 {
		t.Errorf("atlas X = %d, want >= 50", p.sizes[0].X)
	}
}

func TestPackerZeroSizedItem(t *testing.T) {
	p := newPacker(64, 64)
	pl, ok := p.tryAdd(image.Point{})
	if !ok {
		t.Fatal("zero-sized add failed")
	}
	if pl.Pos != (image.Point{}) {
		t.Errorf("zero-sized Pos = %v, want (0,0)", pl.Pos)
	}
	// Atlas must NOT have grown to consume any space.
	if p.sizes[0].X != 0 || p.sizes[0].Y != 0 {
		t.Errorf("atlas grew on zero-sized item: %v", p.sizes[0])
	}
	// Another real item should still pack at origin.
	pl2, ok := p.tryAdd(image.Point{X: 10, Y: 10})
	if !ok {
		t.Fatal("subsequent real add failed")
	}
	if pl2.Pos != (image.Point{}) {
		t.Errorf("real-item Pos after zero-item = %v, want (0,0)", pl2.Pos)
	}
}

func TestPackerManyItemsFit(t *testing.T) {
	p := newPacker(4096, 4096)
	for k := range 500 {
		if _, ok := p.tryAdd(xy(k)); !ok {
			t.Fatalf("item %d (%v) failed", k, xy(k))
		}
	}
	if p.sizes[0].X > 4096 || p.sizes[0].Y > 4096 {
		t.Errorf("atlas exceeded maxDims: %v", p.sizes[0])
	}
}

func TestPackerOnePixelItem(t *testing.T) {
	p := newPacker(8, 8)
	for k := range 64 {
		pl, ok := p.tryAdd(image.Point{X: 1, Y: 1})
		if !ok {
			t.Fatalf("1x1 item %d failed in 8x8 atlas", k)
		}
		_ = pl
	}
	if _, ok := p.tryAdd(image.Point{X: 1, Y: 1}); ok {
		t.Errorf("65th 1x1 item in 8x8 atlas accepted")
	}
}
