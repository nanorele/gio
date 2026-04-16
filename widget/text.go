package widget

import (
	"bufio"
	"image"
	"io"
	"math"
	"slices"
	"sort"
	"unicode"
	"unicode/utf8"

	"github.com/nanorele/gio/f32"
	"github.com/nanorele/gio/font"
	"github.com/nanorele/gio/layout"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/op/clip"
	"github.com/nanorele/gio/op/paint"
	"github.com/nanorele/gio/text"
	"github.com/nanorele/gio/unit"
	"golang.org/x/image/math/fixed"
)

type textSource interface {
	io.ReaderAt

	Size() int64

	Changed() bool

	ReplaceRunes(byteOffset int64, runeCount int64, replacement string)
}

type textView struct {
	Alignment text.Alignment

	LineHeight unit.Sp

	LineHeightScale float32

	SingleLine bool

	MaxLines int

	Truncator string

	WrapPolicy text.WrapPolicy

	DisableSpaceTrim bool

	Mask rune

	params     text.Parameters
	shaper     *text.Shaper
	seekCursor int64
	rr         textSource
	maskReader maskReader

	graphemes []int

	paragraphReader graphemeReader
	lastMask        rune
	viewSize        image.Point
	valid           bool
	regions         []Region
	dims            layout.Dimensions

	offIndex []offEntry

	index glyphIndex

	caret struct {
		xoff fixed.Int26_6

		start int
		end   int
	}

	scrollOff image.Point
}

func (e *textView) Changed() bool {
	return e.rr.Changed()
}

func (e *textView) Dimensions() layout.Dimensions {
	basePos := e.dims.Size.Y - e.dims.Baseline
	return layout.Dimensions{Size: e.viewSize, Baseline: e.viewSize.Y - basePos}
}

func (e *textView) FullDimensions() layout.Dimensions {
	return e.dims
}

func (e *textView) SetSource(source textSource) {
	e.rr = source
	e.invalidate()
	e.seekCursor = 0
}

func (e *textView) ReadRuneAt(off int64) (rune, int, error) {
	var buf [utf8.UTFMax]byte
	b := buf[:]
	n, err := e.rr.ReadAt(b, off)
	b = b[:n]
	r, s := utf8.DecodeRune(b)
	return r, s, err
}

func (e *textView) ReadRuneBefore(off int64) (rune, int, error) {
	var buf [utf8.UTFMax]byte
	b := buf[:]
	if off < utf8.UTFMax {
		b = b[:off]
		off = 0
	} else {
		off -= utf8.UTFMax
	}
	n, err := e.rr.ReadAt(b, off)
	b = b[:n]
	r, s := utf8.DecodeLastRune(b)
	return r, s, err
}

func (e *textView) makeValid() {
	if e.valid {
		return
	}
	e.layoutText(e.shaper)
	e.valid = true
}

func (e *textView) closestToRune(runeIdx int) combinedPos {
	e.makeValid()
	pos, _ := e.index.closestToRune(runeIdx)
	return pos
}

func (e *textView) closestToLineCol(line, col int) combinedPos {
	e.makeValid()
	return e.index.closestToLineCol(screenPos{line: line, col: col})
}

func (e *textView) closestToXY(x fixed.Int26_6, y int) (combinedPos, bool) {
	e.makeValid()
	return e.index.closestToXY(x, y)
}

func (e *textView) closestToXYGraphemes(x fixed.Int26_6, y int) (combinedPos, bool) {

	pos, atEndOfLine := e.closestToXY(x, y)
	if atEndOfLine {
		return pos, true
	}

	firstOption := e.moveByGraphemes(pos.runes, 0)
	distance := 1
	if firstOption > pos.runes {
		distance = -1
	}
	secondOption := e.moveByGraphemes(firstOption, distance)

	first := e.closestToRune(firstOption)
	firstDist := absFixed(first.x - x)
	second := e.closestToRune(secondOption)
	secondDist := absFixed(second.x - x)
	if firstDist > secondDist {
		return second, false
	} else {
		return first, false
	}
}

func absFixed(i fixed.Int26_6) fixed.Int26_6 {
	if i < 0 {
		return -i
	}
	return i
}

func (e *textView) MoveLines(distance int, selAct selectionAction) {
	caretStart := e.closestToRune(e.caret.start)
	x := caretStart.x + e.caret.xoff

	pos := e.closestToLineCol(caretStart.lineCol.line+distance, 0)
	pos, atEndOfLine := e.closestToXYGraphemes(x, pos.y)
	e.caret.start = pos.runes
	if atEndOfLine && pos.runes > 0 {
		e.caret.start = pos.runes - 1
	}
	e.caret.xoff = x - pos.x
	e.updateSelection(selAct)
}

func (e *textView) calculateViewSize(gtx layout.Context) image.Point {
	base := e.dims.Size
	if caretWidth := e.caretWidth(gtx); base.X < caretWidth {
		base.X = caretWidth
	}
	return gtx.Constraints.Constrain(base)
}

func (e *textView) Layout(gtx layout.Context, lt *text.Shaper, font font.Font, size unit.Sp) {
	if e.params.Locale != gtx.Locale {
		e.params.Locale = gtx.Locale
		e.invalidate()
	}
	textSize := fixed.I(gtx.Sp(size))
	if e.params.Font != font || e.params.PxPerEm != textSize {
		e.invalidate()
		e.params.Font = font
		e.params.PxPerEm = textSize
	}
	maxWidth := gtx.Constraints.Max.X
	if e.SingleLine {
		maxWidth = 1 << 24
	}
	minWidth := gtx.Constraints.Min.X
	if maxWidth != e.params.MaxWidth {
		e.params.MaxWidth = maxWidth
		e.invalidate()
	}
	if minWidth != e.params.MinWidth {
		e.params.MinWidth = minWidth
		e.invalidate()
	}
	if lt != e.shaper {
		e.shaper = lt
		e.invalidate()
	}
	if e.Mask != e.lastMask {
		e.lastMask = e.Mask
		e.invalidate()
	}
	if e.Alignment != e.params.Alignment {
		e.params.Alignment = e.Alignment
		e.invalidate()
	}
	if e.Truncator != e.params.Truncator {
		e.params.Truncator = e.Truncator
		e.invalidate()
	}
	if e.MaxLines != e.params.MaxLines {
		e.params.MaxLines = e.MaxLines
		e.invalidate()
	}
	if e.WrapPolicy != e.params.WrapPolicy {
		e.params.WrapPolicy = e.WrapPolicy
		e.invalidate()
	}
	if lh := fixed.I(gtx.Sp(e.LineHeight)); lh != e.params.LineHeight {
		e.params.LineHeight = lh
		e.invalidate()
	}
	if e.LineHeightScale != e.params.LineHeightScale {
		e.params.LineHeightScale = e.LineHeightScale
		e.invalidate()
	}
	if e.DisableSpaceTrim != e.params.DisableSpaceTrim {
		e.params.DisableSpaceTrim = e.DisableSpaceTrim
		e.invalidate()
	}

	e.makeValid()

	if viewSize := e.calculateViewSize(gtx); viewSize != e.viewSize {
		e.viewSize = viewSize
		e.invalidate()
	}
	e.makeValid()
}

func (e *textView) PaintSelection(gtx layout.Context, material op.CallOp) {
	localViewport := image.Rectangle{Max: e.viewSize}
	docViewport := image.Rectangle{Max: e.viewSize}.Add(e.scrollOff)
	defer clip.Rect(localViewport).Push(gtx.Ops).Pop()
	e.regions = e.index.locate(docViewport, e.caret.start, e.caret.end, e.regions)
	for _, region := range e.regions {
		area := clip.Rect(region.Bounds).Push(gtx.Ops)
		material.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		area.Pop()
	}
}

func (e *textView) PaintText(gtx layout.Context, material op.CallOp) {
	m := op.Record(gtx.Ops)
	viewport := image.Rectangle{
		Min: e.scrollOff,
		Max: e.viewSize.Add(e.scrollOff),
	}
	it := textIterator{
		viewport: viewport,
		material: material,
	}

	startGlyph := 0
	for _, line := range e.index.lines {
		if line.descent.Ceil()+line.yOff >= viewport.Min.Y {
			break
		}
		startGlyph += line.glyphs
	}
	var glyphs [512]text.Glyph
	line := glyphs[:0]
	for _, g := range e.index.glyphs[startGlyph:] {
		var ok bool
		if line, ok = it.paintGlyph(gtx, e.shaper, g, line); !ok {
			break
		}
	}

	call := m.Stop()
	viewport.Min = viewport.Min.Add(it.padding.Min)
	viewport.Max = viewport.Max.Add(it.padding.Max)
	defer clip.Rect(viewport.Sub(e.scrollOff)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
}

func (e *textView) caretWidth(gtx layout.Context) int {
	carWidth2 := max(gtx.Dp(1)/2, 1)
	return carWidth2
}

func (e *textView) PaintCaret(gtx layout.Context, material op.CallOp) {
	carWidth2 := e.caretWidth(gtx)
	caretPos, carAsc, carDesc := e.CaretInfo()

	carRect := image.Rectangle{
		Min: caretPos.Sub(image.Pt(carWidth2, carAsc)),
		Max: caretPos.Add(image.Pt(carWidth2, carDesc)),
	}
	cl := image.Rectangle{Max: e.viewSize}
	carRect = cl.Intersect(carRect)
	if !carRect.Empty() {
		defer clip.Rect(carRect).Push(gtx.Ops).Pop()
		material.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}
}

func (e *textView) CaretInfo() (pos image.Point, ascent, descent int) {
	caretStart := e.closestToRune(e.caret.start)

	ascent = caretStart.ascent.Ceil()
	descent = caretStart.descent.Ceil()

	pos = image.Point{
		X: caretStart.x.Round(),
		Y: caretStart.y,
	}
	pos = pos.Sub(e.scrollOff)
	return
}

func (e *textView) ByteOffset(runeOffset int) int64 {
	return int64(e.runeOffset(e.closestToRune(runeOffset).runes))
}

func (e *textView) Len() int {
	e.makeValid()
	return e.closestToRune(math.MaxInt).runes
}

func (e *textView) Text(buf []byte) []byte {
	size := e.rr.Size()
	if cap(buf) < int(size) {
		buf = make([]byte, size)
	}
	buf = buf[:size]
	e.Seek(0, io.SeekStart)
	n, _ := io.ReadFull(e, buf)
	buf = buf[:n]
	return buf
}

func (e *textView) ScrollBounds() image.Rectangle {
	var b image.Rectangle
	if e.SingleLine {
		if len(e.index.lines) > 0 {
			line := e.index.lines[0]
			b.Min.X = min(line.xOff.Floor(), 0)
		}
		b.Max.X = e.dims.Size.X + b.Min.X - e.viewSize.X
	} else {
		b.Max.Y = e.dims.Size.Y - e.viewSize.Y
	}
	return b
}

func (e *textView) ScrollRel(dx, dy int) {
	e.scrollAbs(e.scrollOff.X+dx, e.scrollOff.Y+dy)
}

func (e *textView) ScrollOff() image.Point {
	return e.scrollOff
}

func (e *textView) scrollAbs(x, y int) {
	e.scrollOff.X = x
	e.scrollOff.Y = y
	b := e.ScrollBounds()
	if e.scrollOff.X > b.Max.X {
		e.scrollOff.X = b.Max.X
	}
	if e.scrollOff.X < b.Min.X {
		e.scrollOff.X = b.Min.X
	}
	if e.scrollOff.Y > b.Max.Y {
		e.scrollOff.Y = b.Max.Y
	}
	if e.scrollOff.Y < b.Min.Y {
		e.scrollOff.Y = b.Min.Y
	}
}

func (e *textView) MoveCoord(pos image.Point) {
	x := fixed.I(pos.X + e.scrollOff.X)
	y := pos.Y + e.scrollOff.Y
	p, _ := e.closestToXYGraphemes(x, y)
	e.caret.start = p.runes
	e.caret.xoff = 0
}

func (e *textView) Truncated() bool {
	return e.index.truncated
}

func (e *textView) layoutText(lt *text.Shaper) {
	e.Seek(0, io.SeekStart)
	var r io.Reader = e
	if e.Mask != 0 {
		e.maskReader.Reset(e, e.Mask)
		r = &e.maskReader
	}
	e.index.reset()
	it := textIterator{viewport: image.Rectangle{Max: image.Point{X: math.MaxInt, Y: math.MaxInt}}}
	if lt != nil {
		lt.Layout(e.params, r)
		for {
			g, ok := lt.NextGlyph()
			if !it.processGlyph(g, ok) {
				break
			}
			e.index.Glyph(g)
		}
	} else {

		b := bufio.NewReader(r)
		for _, _, err := b.ReadRune(); err != io.EOF; _, _, err = b.ReadRune() {
			g := text.Glyph{Runes: 1, Flags: text.FlagClusterBreak}
			_ = it.processGlyph(g, true)
			e.index.Glyph(g)
		}
	}
	e.paragraphReader.SetSource(e.rr)
	e.graphemes = e.graphemes[:0]
	for g := e.paragraphReader.Graphemes(); len(g) > 0; g = e.paragraphReader.Graphemes() {
		if len(e.graphemes) > 0 && g[0] == e.graphemes[len(e.graphemes)-1] {
			g = g[1:]
		}
		e.graphemes = append(e.graphemes, g...)
	}
	dims := layout.Dimensions{Size: it.bounds.Size()}
	dims.Baseline = dims.Size.Y - it.baseline
	e.dims = dims
}

func (e *textView) CaretPos() (line, col int) {
	pos := e.closestToRune(e.caret.start)
	return pos.lineCol.line, pos.lineCol.col
}

func (e *textView) CaretCoords() f32.Point {
	pos := e.closestToRune(e.caret.start)
	return f32.Pt(float32(pos.x)/64-float32(e.scrollOff.X), float32(pos.y-e.scrollOff.Y))
}

func (e *textView) indexRune(r int) offEntry {

	if len(e.offIndex) == 0 {
		e.offIndex = append(e.offIndex, offEntry{})
	}
	i := sort.Search(len(e.offIndex), func(i int) bool {
		entry := e.offIndex[i]
		return entry.runes >= r
	})

	if i > 0 {
		i--
	}
	return e.offIndex[i]
}

func (e *textView) runeOffset(r int) int {
	const runesPerIndexEntry = 50
	entry := e.indexRune(r)
	lastEntry := e.offIndex[len(e.offIndex)-1].runes
	for entry.runes < r {
		if entry.runes > lastEntry && entry.runes%runesPerIndexEntry == runesPerIndexEntry-1 {
			e.offIndex = append(e.offIndex, entry)
		}
		_, s, _ := e.ReadRuneAt(int64(entry.bytes))
		entry.bytes += s
		entry.runes++
	}
	return entry.bytes
}

func (e *textView) invalidate() {
	e.offIndex = e.offIndex[:0]
	e.valid = false
}

func (e *textView) Replace(start, end int, s string) int {
	if start > end {
		start, end = end, start
	}
	startPos := e.closestToRune(start)
	endPos := e.closestToRune(end)
	startOff := e.runeOffset(startPos.runes)
	replaceSize := endPos.runes - startPos.runes
	sc := utf8.RuneCountInString(s)
	newEnd := startPos.runes + sc

	e.rr.ReplaceRunes(int64(startOff), int64(replaceSize), s)
	adjust := func(pos int) int {
		switch {
		case newEnd < pos && pos <= endPos.runes:
			pos = newEnd
		case endPos.runes < pos:
			diff := newEnd - endPos.runes
			pos = pos + diff
		}
		return pos
	}
	e.caret.start = adjust(e.caret.start)
	e.caret.end = adjust(e.caret.end)
	e.invalidate()
	return sc
}

func (e *textView) MovePages(pages int, selAct selectionAction) {
	caret := e.closestToRune(e.caret.start)
	x := caret.x + e.caret.xoff
	y := caret.y + pages*e.viewSize.Y
	pos, _ := e.closestToXYGraphemes(x, y)
	e.caret.start = pos.runes
	e.caret.xoff = x - pos.x
	e.updateSelection(selAct)
}

func (e *textView) moveByGraphemes(startRuneidx, graphemes int) int {
	if len(e.graphemes) == 0 {
		return startRuneidx
	}
	startGraphemeIdx, _ := slices.BinarySearch(e.graphemes, startRuneidx)
	startGraphemeIdx = max(startGraphemeIdx+graphemes, 0)
	startGraphemeIdx = min(startGraphemeIdx, len(e.graphemes)-1)
	startRuneIdx := e.graphemes[startGraphemeIdx]
	return e.closestToRune(startRuneIdx).runes
}

func (e *textView) clampCursorToGraphemes() {
	e.caret.start = e.moveByGraphemes(e.caret.start, 0)
	e.caret.end = e.moveByGraphemes(e.caret.end, 0)
}

func (e *textView) MoveCaret(startDelta, endDelta int) {
	e.caret.xoff = 0
	e.caret.start = e.moveByGraphemes(e.caret.start, startDelta)
	e.caret.end = e.moveByGraphemes(e.caret.end, endDelta)
}

func (e *textView) MoveTextStart(selAct selectionAction) {
	caret := e.closestToRune(e.caret.end)
	e.caret.start = 0
	e.caret.end = caret.runes
	e.caret.xoff = -caret.x
	e.updateSelection(selAct)
	e.clampCursorToGraphemes()
}

func (e *textView) MoveTextEnd(selAct selectionAction) {
	caret := e.closestToRune(math.MaxInt)
	e.caret.start = caret.runes
	e.caret.xoff = fixed.I(e.params.MaxWidth) - caret.x
	e.updateSelection(selAct)
	e.clampCursorToGraphemes()
}

func (e *textView) MoveLineStart(selAct selectionAction) {
	caret := e.closestToRune(e.caret.start)
	caret = e.closestToLineCol(caret.lineCol.line, 0)
	e.caret.start = caret.runes
	e.caret.xoff = -caret.x
	e.updateSelection(selAct)
	e.clampCursorToGraphemes()
}

func (e *textView) MoveLineEnd(selAct selectionAction) {
	caret := e.closestToRune(e.caret.start)
	caret = e.closestToLineCol(caret.lineCol.line, math.MaxInt)
	e.caret.start = caret.runes
	e.caret.xoff = fixed.I(e.params.MaxWidth) - caret.x
	e.updateSelection(selAct)
	e.clampCursorToGraphemes()
}

func (e *textView) MoveWord(distance int, selAct selectionAction) {

	words, direction := distance, 1
	if distance < 0 {
		words, direction = distance*-1, -1
	}

	caret := e.closestToRune(e.caret.start)
	atEnd := func() bool {
		return caret.runes == 0 || caret.runes == e.Len()
	}

	next := func() (r rune) {
		off := e.runeOffset(caret.runes)
		if direction < 0 {
			r, _, _ = e.ReadRuneBefore(int64(off))
		} else {
			r, _, _ = e.ReadRuneAt(int64(off))
		}
		return r
	}
	for range words {
		for r := next(); unicode.IsSpace(r) && !atEnd(); r = next() {
			e.MoveCaret(direction, 0)
			caret = e.closestToRune(e.caret.start)
		}
		e.MoveCaret(direction, 0)
		caret = e.closestToRune(e.caret.start)
		for r := next(); !unicode.IsSpace(r) && !atEnd(); r = next() {
			e.MoveCaret(direction, 0)
			caret = e.closestToRune(e.caret.start)
		}
	}
	e.updateSelection(selAct)
	e.clampCursorToGraphemes()
}

func (e *textView) ScrollToCaret() {
	caret := e.closestToRune(e.caret.start)
	if e.SingleLine {
		var dist int
		if d := caret.x.Floor() - e.scrollOff.X; d < 0 {
			dist = d
		} else if d := caret.x.Ceil() - (e.scrollOff.X + e.viewSize.X); d > 0 {
			dist = d
		}
		e.ScrollRel(dist, 0)
	} else {
		miny := caret.y - caret.ascent.Ceil()
		maxy := caret.y + caret.descent.Ceil()
		var dist int
		if d := miny - e.scrollOff.Y; d < 0 {
			dist = d
		} else if d := maxy - (e.scrollOff.Y + e.viewSize.Y); d > 0 {
			dist = d
		}
		e.ScrollRel(0, dist)
	}
}

func (e *textView) SelectionLen() int {
	return abs(e.caret.start - e.caret.end)
}

func (e *textView) Selection() (start, end int) {
	return e.caret.start, e.caret.end
}

func (e *textView) SetCaret(start, end int) {
	e.caret.start = e.closestToRune(start).runes
	e.caret.end = e.closestToRune(end).runes
	e.clampCursorToGraphemes()
}

func (e *textView) SelectedText(buf []byte) []byte {
	startOff := e.runeOffset(e.caret.start)
	endOff := e.runeOffset(e.caret.end)
	start := min(startOff, endOff)
	end := max(startOff, endOff)
	if cap(buf) < end-start {
		buf = make([]byte, end-start)
	}
	buf = buf[:end-start]
	n, _ := e.rr.ReadAt(buf, int64(start))

	return buf[:n]
}

func (e *textView) updateSelection(selAct selectionAction) {
	if selAct == selectionClear {
		e.ClearSelection()
	}
}

func (e *textView) ClearSelection() {
	e.caret.end = e.caret.start
}

func (e *textView) WriteTo(w io.Writer) (int64, error) {
	e.Seek(0, io.SeekStart)
	return io.Copy(w, struct{ io.Reader }{e})
}

func (e *textView) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		e.seekCursor = offset
	case io.SeekCurrent:
		e.seekCursor += offset
	case io.SeekEnd:
		e.seekCursor = e.rr.Size() + offset
	}
	return e.seekCursor, nil
}

func (e *textView) Read(p []byte) (int, error) {
	n, err := e.rr.ReadAt(p, e.seekCursor)
	e.seekCursor += int64(n)
	return n, err
}

func (e *textView) ReadAt(p []byte, offset int64) (int, error) {
	return e.rr.ReadAt(p, offset)
}

func (e *textView) Regions(start, end int, regions []Region) []Region {
	viewport := image.Rectangle{
		Min: e.scrollOff,
		Max: e.viewSize.Add(e.scrollOff),
	}
	return e.index.locate(viewport, start, end, regions)
}
