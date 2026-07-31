package text

import (
	"image"
	"sync/atomic"

	giofont "github.com/nanorele/gio/font"
	"github.com/nanorele/gio/io/system"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/op/clip"
	"github.com/nanorele/gio/op/paint"
	"golang.org/x/image/math/fixed"
)

type entry[K comparable, V any] struct {
	next, prev *entry[K, V]
	key        K
	v          V
	cost       int
}

type lru[K comparable, V any] struct {
	m          map[K]*entry[K, V]
	head, tail *entry[K, V]
	// free chains evicted entries (via next) for reuse by Put, so steady
	// eviction pressure does not allocate an entry per insertion.
	free    *entry[K, V]
	onEvict func(V)
	// capLimit overrides the package-default maxSize when > 0. Allows
	// individual cache instances (e.g. the per-glyph-batch path cache,
	// which holds clip.PathSpec values that can be tens of MB combined)
	// to be tuned tighter than the broad default.
	capLimit int
	// costOf and costLimit bound the cache by retained bytes rather than
	// by entry count. Entry counts are a poor proxy when single entries
	// vary by orders of magnitude — a shaped paragraph holds one glyph
	// record per rune, so 1000 multi-KB paragraphs retain gigabytes while
	// 1000 short labels retain almost nothing.
	costOf    func(K, V) int
	costLimit int
	cost      int
}

func (l *lru[K, V]) Get(k K) (V, bool) {
	if lt, ok := l.m[k]; ok {
		l.remove(lt)
		l.insert(lt)
		return lt.v, true
	}
	var v V
	return v, false
}

func (l *lru[K, V]) Put(k K, v V) {
	if l.m == nil {
		l.m = make(map[K]*entry[K, V])
		l.head = new(entry[K, V])
		l.tail = new(entry[K, V])
		l.head.prev = l.tail
		l.tail.next = l.head
	}
	// If the key already exists, drop the previous entry: leaving it in the
	// linked list would desync the list from the map (eviction would walk
	// the stale tail entry and delete the new map entry).
	if old, ok := l.m[k]; ok {
		l.remove(old)
		l.cost -= old.cost
		if l.onEvict != nil {
			l.onEvict(old.v)
		}
		l.release(old)
	}
	val := l.acquire(k, v)
	if l.costOf != nil {
		val.cost = l.costOf(k, v)
	}
	l.m[k] = val
	l.insert(val)
	l.cost += val.cost
	limit := maxSize
	if l.capLimit > 0 {
		limit = l.capLimit
	}
	for len(l.m) > limit || (l.costLimit > 0 && l.cost > l.costLimit && len(l.m) > 1) {
		oldest := l.tail.next
		if oldest == l.head {
			break
		}
		l.remove(oldest)
		delete(l.m, oldest.key)
		l.cost -= oldest.cost
		if l.onEvict != nil {
			l.onEvict(oldest.v)
		}
		l.release(oldest)
	}
}

func (l *lru[K, V]) acquire(k K, v V) *entry[K, V] {
	if e := l.free; e != nil {
		l.free = e.next
		*e = entry[K, V]{key: k, v: v}
		return e
	}
	return &entry[K, V]{key: k, v: v}
}

// release returns an unlinked entry to the free chain, dropping its key and
// value so they do not outlive the eviction.
func (l *lru[K, V]) release(e *entry[K, V]) {
	*e = entry[K, V]{next: l.free}
	l.free = e
}

func (l *lru[K, V]) remove(e *entry[K, V]) {
	e.next.prev = e.prev
	e.prev.next = e.next
}

func (l *lru[K, V]) Clear() {
	if l.onEvict != nil {
		for _, e := range l.m {
			l.onEvict(e.v)
		}
	}
	l.m = nil
	l.head = nil
	l.tail = nil
	l.free = nil
	l.cost = 0
}

func (l *lru[K, V]) insert(e *entry[K, V]) {
	e.next = l.head
	e.prev = l.head.prev
	e.prev.next = e
	e.next.prev = e
}

type bitmapCache = lru[GlyphID, bitmap]

type bitmap struct {
	img  paint.ImageOp
	size image.Point
}

type layoutCache = lru[layoutKey, *document]

type glyphValue[V any] struct {
	v      V
	glyphs []glyphInfo
}

type glyphLRU[V any] struct {
	seed  uint64
	cache lru[uint64, glyphValue[V]]
}

// setGlyphBudget bounds the cache by total cached glyphs instead of entry
// count. Entries hold one shaped path per glyph run, and run lengths vary
// from a single bracket to a full line, so an entry cap alone leaves the
// retained size unbounded.
func (c *glyphLRU[V]) setGlyphBudget(glyphs int) {
	c.cache.costOf = func(_ uint64, v glyphValue[V]) int { return len(v.glyphs) }
	c.cache.costLimit = glyphs
}

var seed uint32

func (c *glyphLRU[V]) hashGlyphs(gs []Glyph) uint64 {
	if c.seed == 0 {
		c.seed = uint64(atomic.AddUint32(&seed, 3900798947))
	}
	if len(gs) == 0 {
		return 0
	}

	h := c.seed
	firstX := gs[0].X
	for _, g := range gs {
		h += uint64(g.X - firstX)
		h *= 6585573582091643
		h += uint64(g.ID)
		h *= 3650802748644053
		h += uint64(g.Offset.X)<<32 | uint64(uint32(g.Offset.Y))
		h *= 7528558649875681
	}

	return h
}

func (c *glyphLRU[V]) Get(key uint64, gs []Glyph) (V, bool) {
	if v, ok := c.cache.Get(key); ok && gidsEqual(v.glyphs, gs) {
		return v.v, true
	}
	var v V
	return v, false
}

func (c *glyphLRU[V]) Put(key uint64, glyphs []Glyph, v V) {
	gids := make([]glyphInfo, len(glyphs))
	firstX := fixed.I(0)
	for i, glyph := range glyphs {
		if i == 0 {
			firstX = glyph.X
		}

		gids[i] = glyphInfo{ID: glyph.ID, X: glyph.X - firstX, Offset: glyph.Offset}
	}
	val := glyphValue[V]{
		glyphs: gids,
		v:      v,
	}
	c.cache.Put(key, val)
}

type pathCache = glyphLRU[clip.PathSpec]

type bitmapShapeCache = glyphLRU[op.CallOp]

type glyphInfo struct {
	ID     GlyphID
	X      fixed.Int26_6
	Offset fixed.Point26_6
}

type layoutKey struct {
	ppem               fixed.Int26_6
	maxWidth, minWidth int
	maxLines           int
	str                string
	truncator          string
	locale             system.Locale
	font               giofont.Font
	forceTruncate      bool
	wrapPolicy         WrapPolicy
	lineHeight         fixed.Int26_6
	lineHeightScale    float32
	disableSpaceTrim   bool
}

const maxSize = 1000

func gidsEqual(a []glyphInfo, glyphs []Glyph) bool {
	if len(a) != len(glyphs) {
		return false
	}
	firstX := fixed.Int26_6(0)
	for i := range a {
		if i == 0 {
			firstX = glyphs[i].X
		}

		if a[i].ID != glyphs[i].ID || a[i].X != (glyphs[i].X-firstX) || a[i].Offset != glyphs[i].Offset {
			return false
		}
	}
	return true
}
