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
}

type lru[K comparable, V any] struct {
	m          map[K]*entry[K, V]
	head, tail *entry[K, V]
	onEvict    func(V)
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
	val := &entry[K, V]{key: k, v: v}
	l.m[k] = val
	l.insert(val)
	if len(l.m) > maxSize {
		oldest := l.tail.next
		l.remove(oldest)
		delete(l.m, oldest.key)
		if l.onEvict != nil {
			l.onEvict(oldest.v)
		}
	}
}

func (l *lru[K, V]) remove(e *entry[K, V]) {
	e.next.prev = e.prev
	e.prev.next = e.next
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

type layoutCache = lru[layoutKey, document]

type glyphValue[V any] struct {
	v      V
	glyphs []glyphInfo
}

type glyphLRU[V any] struct {
	seed  uint64
	cache lru[uint64, glyphValue[V]]
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

		gids[i] = glyphInfo{ID: glyph.ID, X: glyph.X - firstX}
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
	ID GlyphID
	X  fixed.Int26_6
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

		if a[i].ID != glyphs[i].ID || a[i].X != (glyphs[i].X-firstX) {
			return false
		}
	}
	return true
}
