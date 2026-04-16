package text

import (
	"bufio"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/go-text/typesetting/font"
	giofont "github.com/nanorele/gio/font"
	"github.com/nanorele/gio/io/system"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/op/clip"
	"golang.org/x/image/math/fixed"
)

type WrapPolicy uint8

const (
	WrapHeuristically WrapPolicy = iota

	WrapWords

	WrapGraphemes
)

type Parameters struct {
	Font giofont.Font

	Alignment Alignment

	PxPerEm fixed.Int26_6

	MaxLines int

	Truncator string

	WrapPolicy WrapPolicy

	MinWidth, MaxWidth int

	Locale system.Locale

	LineHeightScale float32

	LineHeight fixed.Int26_6

	forceTruncate bool

	DisableSpaceTrim bool
}

type FontFace = giofont.FontFace

type Glyph struct {
	ID GlyphID

	X fixed.Int26_6

	Y int32

	Advance fixed.Int26_6

	Ascent fixed.Int26_6

	Descent fixed.Int26_6

	Offset fixed.Point26_6

	Bounds fixed.Rectangle26_6

	Runes uint16

	Flags Flags
}

type Flags uint16

const (
	FlagTowardOrigin Flags = 1 << iota

	FlagLineBreak

	FlagRunBreak

	FlagClusterBreak

	FlagParagraphBreak

	FlagParagraphStart

	FlagTruncator
)

func (f Flags) String() string {
	var b strings.Builder
	if f&FlagParagraphStart != 0 {
		b.WriteString("S")
	} else {
		b.WriteString("_")
	}
	if f&FlagParagraphBreak != 0 {
		b.WriteString("P")
	} else {
		b.WriteString("_")
	}
	if f&FlagTowardOrigin != 0 {
		b.WriteString("T")
	} else {
		b.WriteString("_")
	}
	if f&FlagLineBreak != 0 {
		b.WriteString("L")
	} else {
		b.WriteString("_")
	}
	if f&FlagRunBreak != 0 {
		b.WriteString("R")
	} else {
		b.WriteString("_")
	}
	if f&FlagClusterBreak != 0 {
		b.WriteString("C")
	} else {
		b.WriteString("_")
	}
	if f&FlagTruncator != 0 {
		b.WriteString("…")
	} else {
		b.WriteString("_")
	}
	return b.String()
}

type GlyphID uint64

type Shaper struct {
	config struct {
		disableSystemFonts bool
		collection         []FontFace
	}
	initialized      bool
	shaper           shaperImpl
	pathCache        pathCache
	bitmapShapeCache bitmapShapeCache
	layoutCache      layoutCache

	reader    *bufio.Reader
	paragraph []byte

	brokeParagraph   bool
	pararagraphStart Glyph
	txt              document
	line             int
	run              int
	glyph            int

	advance fixed.Int26_6

	done bool
	err  error
}

type ShaperOption func(*Shaper)

func NoSystemFonts() ShaperOption {
	return func(s *Shaper) {
		s.config.disableSystemFonts = true
	}
}

func WithCollection(collection []FontFace) ShaperOption {
	return func(s *Shaper) {
		s.config.collection = collection
	}
}

func NewShaper(options ...ShaperOption) *Shaper {
	l := &Shaper{}
	for _, opt := range options {
		opt(l)
	}
	l.init()
	return l
}

func (l *Shaper) init() {
	if l.initialized {
		return
	}
	l.initialized = true
	l.reader = bufio.NewReader(nil)
	l.shaper = *newShaperImpl(!l.config.disableSystemFonts, l.config.collection)
}

func (l *Shaper) Layout(params Parameters, txt io.Reader) {
	l.init()
	l.layoutText(params, txt, "")
}

func (l *Shaper) LayoutString(params Parameters, str string) {
	l.init()
	l.layoutText(params, nil, str)
}

func (l *Shaper) reset(align Alignment) {
	l.line, l.run, l.glyph, l.advance = 0, 0, 0, 0
	l.done = false
	l.txt.reset()
	l.txt.alignment = align
}

func (l *Shaper) layoutText(params Parameters, txt io.Reader, str string) {
	l.reset(params.Alignment)
	if txt == nil && len(str) == 0 {
		l.txt.append(l.layoutParagraph(params, "", nil))
		return
	}
	l.reader.Reset(txt)
	truncating := params.MaxLines > 0
	var done bool
	var endByte int
	for !done {
		l.paragraph = l.paragraph[:0]
		if txt != nil {
			for {
				b, err := l.reader.ReadByte()
				if err != nil {

					done = true
					break
				}
				l.paragraph = append(l.paragraph, b)
				if b == '\n' {
					break
				}
			}
			if !done {
				_, re := l.reader.ReadByte()
				done = re != nil
				if !done {
					_ = l.reader.UnreadByte()
				}
			}
		} else {
			idx := strings.IndexByte(str, '\n')
			if idx == -1 {
				done = true
				endByte = len(str)
			} else {
				endByte = idx + 1
				done = endByte == len(str)
			}
		}
		if len(str[:endByte]) > 0 || (len(l.paragraph) > 0 || len(l.txt.lines) == 0) {
			params.forceTruncate = truncating && !done
			lines := l.layoutParagraph(params, str[:endByte], l.paragraph)
			if truncating {
				params.MaxLines -= len(lines.lines)
				if params.MaxLines == 0 {
					done = true

					var unreadRunes int
					if txt == nil {
						unreadRunes = utf8.RuneCountInString(str[endByte:])
					} else {
						for {
							_, _, e := l.reader.ReadRune()
							if e != nil {
								break
							}
							unreadRunes++
						}
					}
					l.txt.unreadRuneCount = unreadRunes
				}
			}
			l.txt.append(lines)
		}
		if done {
			return
		}
		str = str[endByte:]
	}
}

func (l *Shaper) layoutParagraph(params Parameters, asStr string, asBytes []byte) document {
	if l == nil {
		return document{}
	}
	if len(asStr) == 0 && len(asBytes) > 0 {
		asStr = string(asBytes)
	}

	lk := layoutKey{
		ppem:            params.PxPerEm,
		maxWidth:        params.MaxWidth,
		minWidth:        params.MinWidth,
		maxLines:        params.MaxLines,
		truncator:       params.Truncator,
		locale:          params.Locale,
		font:            params.Font,
		forceTruncate:   params.forceTruncate,
		wrapPolicy:      params.WrapPolicy,
		str:             asStr,
		lineHeight:      params.LineHeight,
		lineHeightScale: params.LineHeightScale,
	}
	if l, ok := l.layoutCache.Get(lk); ok {
		return l
	}
	lines := l.shaper.LayoutRunes(params, []rune(asStr))
	l.layoutCache.Put(lk, lines)
	return lines
}

func (l *Shaper) NextGlyph() (_ Glyph, ok bool) {
	l.init()
	if l.done {
		return Glyph{}, false
	}
	for {
		if l.line == len(l.txt.lines) {
			if l.brokeParagraph {
				l.brokeParagraph = false
				return l.pararagraphStart, true
			}
			if l.err == nil {
				l.err = io.EOF
			}
			return Glyph{}, false
		}
		line := l.txt.lines[l.line]
		if l.run == len(line.runs) {
			l.line++
			l.run = 0
			continue
		}
		run := line.runs[l.run]
		align := l.txt.alignment.Align(line.direction, line.width, l.txt.alignWidth)
		if l.line == 0 && l.run == 0 && len(run.Glyphs) == 0 {

			l.done = true
			return Glyph{
				X:       align,
				Y:       int32(line.yOffset),
				Runes:   0,
				Flags:   FlagLineBreak | FlagClusterBreak | FlagRunBreak,
				Ascent:  line.ascent,
				Descent: line.descent,
			}, true
		}
		if l.glyph == len(run.Glyphs) {
			l.run++
			l.glyph = 0
			l.advance = 0
			continue
		}
		glyphIdx := l.glyph
		rtl := run.Direction.Progression() == system.TowardOrigin
		if rtl {

			glyphIdx = len(run.Glyphs) - 1 - glyphIdx
		}
		g := run.Glyphs[glyphIdx]
		if rtl {

			l.advance += g.xAdvance
		}

		runOffset := l.advance
		if rtl {
			runOffset = run.Advance - l.advance
		}
		glyph := Glyph{
			ID:      g.id,
			X:       align + run.X + runOffset,
			Y:       int32(line.yOffset),
			Ascent:  line.ascent,
			Descent: line.descent,
			Advance: g.xAdvance,
			Runes:   uint16(g.runeCount),
			Offset: fixed.Point26_6{
				X: g.xOffset,
				Y: g.yOffset,
			},
			Bounds: g.bounds,
		}
		if run.truncator {
			glyph.Flags |= FlagTruncator
		}
		l.glyph++
		if !rtl {
			l.advance += g.xAdvance
		}

		endOfRun := l.glyph == len(run.Glyphs)
		if endOfRun {
			glyph.Flags |= FlagRunBreak
		}
		endOfLine := endOfRun && l.run == len(line.runs)-1
		if endOfLine {
			glyph.Flags |= FlagLineBreak
		}
		endOfText := endOfLine && l.line == len(l.txt.lines)-1
		nextGlyph := l.glyph
		if rtl {
			nextGlyph = len(run.Glyphs) - 1 - nextGlyph
		}
		endOfCluster := endOfRun || run.Glyphs[nextGlyph].clusterIndex != g.clusterIndex
		if run.truncator {

			endOfCluster = endOfRun
		}
		if endOfCluster {
			glyph.Flags |= FlagClusterBreak
			if run.truncator {
				glyph.Runes += uint16(l.txt.unreadRuneCount)
			}
		} else {
			glyph.Runes = 0
		}
		if run.Direction.Progression() == system.TowardOrigin {
			glyph.Flags |= FlagTowardOrigin
		}
		if l.brokeParagraph {
			glyph.Flags |= FlagParagraphStart
			l.brokeParagraph = false
		}
		if g.glyphCount == 0 {
			glyph.Flags |= FlagParagraphBreak
			l.brokeParagraph = true
			if endOfText {
				l.pararagraphStart = Glyph{
					Ascent:  glyph.Ascent,
					Descent: glyph.Descent,
					Flags:   FlagParagraphStart | FlagLineBreak | FlagRunBreak | FlagClusterBreak,
				}

				l.pararagraphStart.X = l.txt.alignment.Align(line.direction, 0, l.txt.alignWidth)
				l.pararagraphStart.Y = glyph.Y + int32(line.lineHeight.Round())
			}
		}
		return glyph, true
	}
}

const (
	facebits = 16
	sizebits = 16
	gidbits  = 64 - facebits - sizebits
)

func newGlyphID(ppem fixed.Int26_6, faceIdx int, gid font.GID) GlyphID {
	if gid&^((1<<gidbits)-1) != 0 {
		panic("glyph id out of bounds")
	}
	if faceIdx&^((1<<facebits)-1) != 0 {
		panic("face index out of bounds")
	}
	if ppem&^((1<<sizebits)-1) != 0 {
		panic("ppem out of bounds")
	}

	ppem &= ((1 << sizebits) - 1)
	return GlyphID(faceIdx)<<(gidbits+sizebits) | GlyphID(ppem)<<(gidbits) | GlyphID(gid)
}

func splitGlyphID(g GlyphID) (fixed.Int26_6, int, font.GID) {
	faceIdx := int(uint64(g) >> (gidbits + sizebits))
	ppem := fixed.Int26_6((g & ((1<<sizebits - 1) << gidbits)) >> gidbits)
	gid := font.GID(g) & (1<<gidbits - 1)
	return ppem, faceIdx, gid
}

func (l *Shaper) Shape(gs []Glyph) clip.PathSpec {
	l.init()
	key := l.pathCache.hashGlyphs(gs)
	shape, ok := l.pathCache.Get(key, gs)
	if ok {
		return shape
	}
	pathOps := new(op.Ops)
	shape = l.shaper.Shape(pathOps, gs)
	l.pathCache.Put(key, gs, shape)
	return shape
}

func (l *Shaper) Bitmaps(gs []Glyph) op.CallOp {
	l.init()
	key := l.bitmapShapeCache.hashGlyphs(gs)
	call, ok := l.bitmapShapeCache.Get(key, gs)
	if ok {
		return call
	}
	callOps := new(op.Ops)
	call = l.shaper.Bitmaps(callOps, gs)
	l.bitmapShapeCache.Put(key, gs, call)
	return call
}
