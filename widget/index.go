package widget

import (
	"bufio"
	"image"
	"io"
	"math"
	"sort"

	"github.com/go-text/typesetting/segmenter"
	"github.com/nanorele/gio/text"
	"golang.org/x/image/math/fixed"
)

type lineInfo struct {
	xOff            fixed.Int26_6
	yOff            int
	width           fixed.Int26_6
	ascent, descent fixed.Int26_6
	glyphs          int
}

func (l lineInfo) getLineEnd() fixed.Int26_6 {
	return l.xOff + l.width
}

type glyphIndex struct {
	glyphs []text.Glyph

	positions []combinedPos

	lines []lineInfo

	currentLineMin, currentLineMax fixed.Int26_6

	currentLineGlyphs int

	pos combinedPos

	prog text.Flags

	clusterAdvance fixed.Int26_6

	truncated bool

	midCluster bool
}

func (g *glyphIndex) reset() {
	g.glyphs = g.glyphs[:0]
	g.positions = g.positions[:0]
	g.lines = g.lines[:0]
	g.currentLineMin = 0
	g.currentLineMax = 0
	g.currentLineGlyphs = 0
	g.pos = combinedPos{}
	g.prog = 0
	g.clusterAdvance = 0
	g.truncated = false
	g.midCluster = false
}

type screenPos struct {
	col  int
	line int
}

type combinedPos struct {
	runes int

	lineCol screenPos

	x fixed.Int26_6
	y int

	ascent, descent fixed.Int26_6

	runIndex int

	towardOrigin bool
}

func (g *glyphIndex) incrementPosition(pos combinedPos) (next combinedPos, eof bool) {
	candidate, index := g.closestToRune(pos.runes)
	for candidate != pos && index+1 < len(g.positions) {
		index++
		candidate = g.positions[index]
	}
	if index+1 < len(g.positions) {
		return g.positions[index+1], false
	}
	return candidate, true
}

func (g *glyphIndex) insertPosition(pos combinedPos) {
	lastIdx := len(g.positions) - 1
	if lastIdx >= 0 {
		lastPos := g.positions[lastIdx]
		if lastPos.runes == pos.runes && (lastPos.y != pos.y || (lastPos.x == pos.x)) {

			g.positions[lastIdx] = pos
			return
		}
	}
	g.positions = append(g.positions, pos)
}

func (g *glyphIndex) Glyph(gl text.Glyph) {
	g.glyphs = append(g.glyphs, gl)
	g.currentLineGlyphs++
	if len(g.positions) == 0 {

		g.currentLineMin = math.MaxInt32
		g.currentLineMax = 0
	}
	if gl.X < g.currentLineMin {
		g.currentLineMin = gl.X
	}
	if end := gl.X + gl.Advance; end > g.currentLineMax {
		g.currentLineMax = end
	}

	needsNewLine := gl.Flags&text.FlagLineBreak != 0
	needsNewRun := gl.Flags&text.FlagRunBreak != 0
	breaksParagraph := gl.Flags&text.FlagParagraphBreak != 0
	breaksCluster := gl.Flags&text.FlagClusterBreak != 0

	insertPositionsWithin := breaksCluster && !breaksParagraph && gl.Runes > 0

	g.prog = gl.Flags & text.FlagTowardOrigin
	g.pos.towardOrigin = g.prog == text.FlagTowardOrigin
	if !g.midCluster {

		g.pos.x = gl.X
		g.pos.y = int(gl.Y)
		g.pos.ascent = gl.Ascent
		g.pos.descent = gl.Descent
		if g.pos.towardOrigin {
			g.pos.x += gl.Advance
		}
		g.insertPosition(g.pos)
	}

	g.midCluster = !breaksCluster

	if breaksParagraph {

		g.clusterAdvance = 0
		g.pos.runes += int(gl.Runes)
	}

	g.clusterAdvance += gl.Advance
	if insertPositionsWithin {

		g.pos.y = int(gl.Y)
		g.pos.ascent = gl.Ascent
		g.pos.descent = gl.Descent
		width := g.clusterAdvance
		positionCount := int(gl.Runes)
		runesPerPosition := 1
		if gl.Flags&text.FlagTruncator != 0 {

			positionCount = 1
			runesPerPosition = int(gl.Runes)
			g.truncated = true
		}
		perRune := width / fixed.Int26_6(positionCount)
		adjust := fixed.Int26_6(0)
		if g.pos.towardOrigin {

			adjust = width
			perRune = -perRune
		}
		for i := 1; i <= positionCount; i++ {
			g.pos.x = gl.X + adjust + perRune*fixed.Int26_6(i)
			g.pos.runes += runesPerPosition
			g.pos.lineCol.col += runesPerPosition
			g.insertPosition(g.pos)
		}
		g.clusterAdvance = 0
	}
	if needsNewRun {
		g.pos.runIndex++
	}
	if needsNewLine {
		g.lines = append(g.lines, lineInfo{
			xOff:    g.currentLineMin,
			yOff:    int(gl.Y),
			width:   g.currentLineMax - g.currentLineMin,
			ascent:  g.positions[len(g.positions)-1].ascent,
			descent: g.positions[len(g.positions)-1].descent,
			glyphs:  g.currentLineGlyphs,
		})
		g.pos.lineCol.line++
		g.pos.lineCol.col = 0
		g.pos.runIndex = 0
		g.currentLineMin = math.MaxInt32
		g.currentLineMax = 0
		g.currentLineGlyphs = 0
	}
}

func (g *glyphIndex) closestToRune(runeIdx int) (combinedPos, int) {
	n := len(g.positions)
	if n == 0 {
		return combinedPos{}, 0
	}
	i := sort.Search(n, func(i int) bool {
		pos := g.positions[i]
		return pos.runes >= runeIdx
	})

	notFound := i == n
	if notFound {
		return g.positions[n-1], n - 1
	}
	return g.positions[i], i
}

func (g *glyphIndex) closestToLineCol(lineCol screenPos) combinedPos {
	n := len(g.positions)
	if n == 0 {
		return combinedPos{}
	}
	i := sort.Search(n, func(i int) bool {
		pos := g.positions[i]
		return pos.lineCol.line > lineCol.line || (pos.lineCol.line == lineCol.line && pos.lineCol.col >= lineCol.col)
	})
	notFound := i == n
	if notFound {
		return g.positions[n-1]
	}
	pos := g.positions[i]
	foundInNextLine := pos.lineCol.line > lineCol.line
	if foundInNextLine && i > 0 {
		prior := g.positions[i-1]
		prior.x = g.lines[lineCol.line].getLineEnd()
		return prior
	}
	return pos
}

func (g *glyphIndex) atStartOfLine(pos combinedPos) bool {
	if pos.runes == 0 {
		return true
	}
	prevRuneIndex := pos.runes - 1
	lineOfPrevRune := g.positions[prevRuneIndex].lineCol.line
	return lineOfPrevRune < pos.lineCol.line
}

func (g *glyphIndex) atEndOfLine(pos combinedPos) bool {
	if pos.runes == g.positions[len(g.positions)-1].runes {
		return true
	}
	next := pos.runes + 1
	hasNext := next < len(g.positions)
	return hasNext && g.positions[next].lineCol.line > pos.lineCol.line
}

func dist(a, b fixed.Int26_6) fixed.Int26_6 {
	if a > b {
		return a - b
	}
	return b - a
}

func (g *glyphIndex) closestToXY(x fixed.Int26_6, y int) (pos combinedPos, atEndOfLine bool) {
	if len(g.positions) == 0 {
		return combinedPos{}, false
	}
	i := sort.Search(len(g.positions), func(i int) bool {
		pos := g.positions[i]
		return pos.y+pos.descent.Round() >= y
	})

	if i == len(g.positions) {
		return g.positions[i-1], false
	}
	first := g.positions[i]

	closest := i
	closestDist := dist(first.x, x)
	line := first.lineCol.line

	for i := i + 1; i < len(g.positions) && g.positions[i].lineCol.line == line; i++ {
		candidate := g.positions[i]
		distance := dist(candidate.x, x)

		if distance.Round() == 0 {
			return g.positions[i], false
		}
		if distance < closestDist {
			closestDist = distance
			closest = i
		}
	}
	next := closest + 1
	hasNext := next < len(g.positions)
	if hasNext && g.atEndOfLine(g.positions[closest]) {
		distance := dist(g.lines[line].getLineEnd(), x)
		if distance < closestDist {
			return g.positions[next], true
		}
	}
	return g.positions[closest], false
}

func makeRegion(line lineInfo, y int, start, end fixed.Int26_6) Region {
	if start > end {
		start, end = end, start
	}
	dotStart := image.Pt(start.Round(), y)
	dotEnd := image.Pt(end.Round(), y)
	return Region{
		Bounds: image.Rectangle{
			Min: dotStart.Sub(image.Point{Y: line.ascent.Ceil()}),
			Max: dotEnd.Add(image.Point{Y: line.descent.Floor()}),
		},
		Baseline: line.descent.Floor(),
	}
}

type Region struct {
	Bounds image.Rectangle

	Baseline int
}

func (g *glyphIndex) locate(viewport image.Rectangle, startRune, endRune int, rects []Region) []Region {
	if startRune > endRune {
		startRune, endRune = endRune, startRune
	}
	rects = rects[:0]
	caretStart, _ := g.closestToRune(startRune)
	caretEnd, _ := g.closestToRune(endRune)

	for lineIdx := caretStart.lineCol.line; lineIdx < len(g.lines); lineIdx++ {
		if lineIdx > caretEnd.lineCol.line {
			break
		}
		pos := g.closestToLineCol(screenPos{line: lineIdx})
		if int(pos.y)+pos.descent.Ceil() < viewport.Min.Y {
			continue
		}
		if int(pos.y)-pos.ascent.Ceil() > viewport.Max.Y {
			break
		}
		line := g.lines[lineIdx]
		if lineIdx > caretStart.lineCol.line && lineIdx < caretEnd.lineCol.line {
			startX := line.xOff
			endX := startX + line.width

			rects = append(rects, makeRegion(line, pos.y, startX, endX))
			continue
		}
		selectionStart := caretStart
		selectionEnd := caretEnd
		if lineIdx != caretStart.lineCol.line {

			selectionStart = g.closestToLineCol(screenPos{line: lineIdx})
		}
		if lineIdx != caretEnd.lineCol.line {

			selectionEnd = g.closestToLineCol(screenPos{line: lineIdx, col: math.MaxInt})
		}

		var (
			startX, endX fixed.Int26_6
			eof          bool
		)
	lineLoop:
		for !eof {
			startX = selectionStart.x
			if selectionStart.runIndex == selectionEnd.runIndex {

				endX = selectionEnd.x
				rects = append(rects, makeRegion(line, pos.y, startX, endX))
				break
			} else {
				currentDirection := selectionStart.towardOrigin
				previous := selectionStart
			runLoop:
				for !eof {

					for startRun := selectionStart.runIndex; selectionStart.runIndex == startRun; {
						previous = selectionStart
						selectionStart, eof = g.incrementPosition(selectionStart)
						if eof {
							endX = selectionStart.x
							rects = append(rects, makeRegion(line, pos.y, startX, endX))
							break runLoop
						}
					}
					if selectionStart.towardOrigin != currentDirection {
						endX = previous.x
						rects = append(rects, makeRegion(line, pos.y, startX, endX))
						break
					}
					if selectionStart.runIndex == selectionEnd.runIndex {

						endX = selectionEnd.x
						rects = append(rects, makeRegion(line, pos.y, startX, endX))
						break lineLoop
					}
				}
			}
		}
	}
	for i := range rects {
		rects[i].Bounds = rects[i].Bounds.Sub(viewport.Min)
	}
	return rects
}

type graphemeReader struct {
	segmenter.Segmenter
	graphemes  []int
	paragraph  []rune
	source     io.ReaderAt
	cursor     int64
	reader     *bufio.Reader
	runeOffset int
}

func (p *graphemeReader) SetSource(source io.ReaderAt) {
	p.source = source
	p.cursor = 0
	p.reader = bufio.NewReader(p)
	p.runeOffset = 0
}

func (p *graphemeReader) Read(b []byte) (int, error) {
	n, err := p.source.ReadAt(b, p.cursor)
	p.cursor += int64(n)
	return n, err
}

func (p *graphemeReader) next() ([]rune, bool) {
	p.paragraph = p.paragraph[:0]
	var err error
	var r rune
	for err == nil {
		r, _, err = p.reader.ReadRune()
		if err != nil {
			break
		}
		p.paragraph = append(p.paragraph, r)
		if r == '\n' {
			break
		}
	}
	return p.paragraph, err == nil
}

func (p *graphemeReader) Graphemes() []int {
	var more bool
	p.graphemes = p.graphemes[:0]
	p.paragraph, more = p.next()
	if len(p.paragraph) == 0 && !more {
		return nil
	}
	p.Segmenter.Init(p.paragraph)
	iter := p.Segmenter.GraphemeIterator()
	if iter.Next() {
		graph := iter.Grapheme()
		p.graphemes = append(p.graphemes,
			p.runeOffset+graph.Offset,
			p.runeOffset+graph.Offset+len(graph.Text),
		)
	}
	for iter.Next() {
		graph := iter.Grapheme()
		p.graphemes = append(p.graphemes, p.runeOffset+graph.Offset+len(graph.Text))
	}
	p.runeOffset += len(p.paragraph)
	return p.graphemes
}
