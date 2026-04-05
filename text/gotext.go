package text

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
	"slices"
	"unicode"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	gotextot "github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/fontscan"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
	"golang.org/x/text/unicode/bidi"

	"github.com/nanorele/gio/f32"
	giofont "github.com/nanorele/gio/font"
	"github.com/nanorele/gio/font/opentype"
	"github.com/nanorele/gio/internal/debug"
	"github.com/nanorele/gio/io/system"
	"github.com/nanorele/gio/op"
	"github.com/nanorele/gio/op/clip"
	"github.com/nanorele/gio/op/paint"
)

type document struct {
	lines           []line
	alignment       Alignment
	alignWidth      int
	unreadRuneCount int
}

func (l *document) append(other document) {
	l.lines = append(l.lines, other.lines...)
	l.alignWidth = max(l.alignWidth, other.alignWidth)
	calculateYOffsets(l.lines)
}

func (l *document) reset() {
	l.lines = l.lines[:0]
	l.alignment = Start
	l.alignWidth = 0
	l.unreadRuneCount = 0
}

type line struct {
	runs        []runLayout
	visualOrder []int
	width       fixed.Int26_6
	ascent      fixed.Int26_6
	descent     fixed.Int26_6
	lineHeight  fixed.Int26_6
	direction   system.TextDirection
	runeCount   int
	yOffset     int
}

func (l *line) insertTrailingSyntheticNewline(newLineClusterIdx int) {
	finalContentRun := len(l.runs) - 1
	l.runeCount += 1
	l.runs[finalContentRun].Runes.Count += 1

	syntheticGlyph := glyph{
		id:           0,
		clusterIndex: newLineClusterIdx,
		glyphCount:   0,
		runeCount:    1,
		xAdvance:     0,
		yAdvance:     0,
		xOffset:      0,
		yOffset:      0,
	}

	if l.runs[finalContentRun].Direction.Progression() == system.FromOrigin {
		l.runs[finalContentRun].Glyphs = append(l.runs[finalContentRun].Glyphs, syntheticGlyph)
	} else {
		l.runs[finalContentRun].Glyphs = append(l.runs[finalContentRun].Glyphs, glyph{})
		copy(l.runs[finalContentRun].Glyphs[1:], l.runs[finalContentRun].Glyphs)
		l.runs[finalContentRun].Glyphs[0] = syntheticGlyph
	}
}

func (l *line) setTruncatedCount(truncatedCount int) {
	finalRunIdx := len(l.runs) - 1
	l.runs[finalRunIdx].truncator = true
	finalGlyphIdx := len(l.runs[finalRunIdx].Glyphs) - 1
	l.runs[finalRunIdx].Runes.Count = truncatedCount
	for i := range l.runs[finalRunIdx].Glyphs {
		if i == finalGlyphIdx {
			l.runs[finalRunIdx].Glyphs[finalGlyphIdx].runeCount = truncatedCount
		} else {
			l.runs[finalRunIdx].Glyphs[finalGlyphIdx].runeCount = 0
		}
	}
}

type Range struct {
	Count  int
	Offset int
}

type glyph struct {
	id           GlyphID
	clusterIndex int
	glyphCount   int
	runeCount    int
	xAdvance     fixed.Int26_6
	yAdvance     fixed.Int26_6
	xOffset      fixed.Int26_6
	yOffset      fixed.Int26_6
	bounds       fixed.Rectangle26_6
}

type runLayout struct {
	VisualPosition int
	X              fixed.Int26_6
	Glyphs         []glyph
	Runes          Range
	Advance        fixed.Int26_6
	PPEM           fixed.Int26_6
	Direction      system.TextDirection
	face           *font.Face
	truncator      bool
}

type shaperImpl struct {
	fontMap      *fontscan.FontMap
	faces        []*font.Face
	faceToIndex  map[*font.Font]int
	faceMeta     []giofont.Font
	defaultFaces []string
	logger       interface {
		Printf(format string, args ...any)
	}
	parser           parser
	shaper           shaping.HarfbuzzShaper
	wrapper          shaping.LineWrapper
	bidiParagraph    bidi.Paragraph
	splitScratch1    []shaping.Input
	splitScratch2    []shaping.Input
	outScratchBuf    []shaping.Output
	scratchRunes     []rune
	bitmapGlyphCache bitmapCache
}

type debugLogger struct {
	*log.Logger
}

func newDebugLogger() debugLogger {
	return debugLogger{Logger: log.New(log.Writer(), "[text] ", log.Default().Flags())}
}

func (d debugLogger) Printf(format string, args ...any) {
	if debug.Text.Load() {
		d.Logger.Printf(format, args...)
	}
}

func newShaperImpl(systemFonts bool, collection []FontFace) *shaperImpl {
	var shaper shaperImpl
	shaper.logger = newDebugLogger()
	shaper.fontMap = fontscan.NewFontMap(shaper.logger)
	shaper.faceToIndex = make(map[*font.Font]int)
	if systemFonts {
		str, err := os.UserCacheDir()
		if err != nil {
			shaper.logger.Printf("failed resolving font cache dir: %v", err)
			shaper.logger.Printf("skipping system font load")
		}
		if err := shaper.fontMap.UseSystemFonts(str); err != nil {
			shaper.logger.Printf("failed loading system fonts: %v", err)
		}
	}
	for _, f := range collection {
		shaper.Load(f)
		shaper.defaultFaces = append(shaper.defaultFaces, string(f.Font.Typeface))
	}
	shaper.shaper.SetFontCacheSize(32)
	return &shaper
}

func (s *shaperImpl) Load(f FontFace) {
	desc := opentype.FontToDescription(f.Font)
	s.fontMap.AddFace(f.Face.Face(), fontscan.Location{File: fmt.Sprint(desc)}, desc)
	s.addFace(f.Face.Face(), f.Font)
}

func (s *shaperImpl) addFace(f *font.Face, md giofont.Font) {
	if _, ok := s.faceToIndex[f.Font]; ok {
		return
	}
	s.logger.Printf("loaded face %s(style:%s, weight:%d)", md.Typeface, md.Style, md.Weight)
	idx := len(s.faces)
	s.faceToIndex[f.Font] = idx
	s.faces = append(s.faces, f)
	s.faceMeta = append(s.faceMeta, md)
}

func splitByScript(inputs []shaping.Input, documentDir di.Direction, buf []shaping.Input) []shaping.Input {
	var splitInputs []shaping.Input
	if buf == nil {
		splitInputs = make([]shaping.Input, 0, len(inputs))
	} else {
		splitInputs = buf
	}
	for _, input := range inputs {
		currentInput := input
		if input.RunStart == input.RunEnd {
			return []shaping.Input{input}
		}
		firstNonCommonRune := input.RunStart
		for i := firstNonCommonRune; i < input.RunEnd; i++ {
			if language.LookupScript(input.Text[i]) != language.Common {
				firstNonCommonRune = i
				break
			}
		}
		currentInput.Script = language.LookupScript(input.Text[firstNonCommonRune])
		for i := firstNonCommonRune + 1; i < input.RunEnd; i++ {
			r := input.Text[i]
			runeScript := language.LookupScript(r)

			if runeScript == language.Common || runeScript == language.Inherited || runeScript == currentInput.Script {
				continue
			}

			if i != input.RunStart {
				currentInput.RunEnd = i
				splitInputs = append(splitInputs, currentInput)
			}

			currentInput = input
			currentInput.RunStart = i
			currentInput.Script = runeScript
		}
		currentInput.RunEnd = input.RunEnd
		splitInputs = append(splitInputs, currentInput)
	}

	return splitInputs
}

func (s *shaperImpl) splitBidi(input shaping.Input) []shaping.Input {
	var splitInputs []shaping.Input
	if input.Direction.Axis() != di.Horizontal || input.RunStart == input.RunEnd {
		return []shaping.Input{input}
	}
	def := bidi.LeftToRight
	if input.Direction.Progression() == di.TowardTopLeft {
		def = bidi.RightToLeft
	}
	s.bidiParagraph.SetString(string(input.Text), bidi.DefaultDirection(def))
	out, err := s.bidiParagraph.Order()
	if err != nil {
		return []shaping.Input{input}
	}
	for i := range out.NumRuns() {
		currentInput := input
		run := out.Run(i)
		dir := run.Direction()
		_, endRune := run.Pos()
		currentInput.RunEnd = endRune + 1
		if dir == bidi.RightToLeft {
			currentInput.Direction = di.DirectionRTL
		} else {
			currentInput.Direction = di.DirectionLTR
		}
		splitInputs = append(splitInputs, currentInput)
		input.RunStart = currentInput.RunEnd
	}
	return splitInputs
}

func (s *shaperImpl) ResolveFace(r rune) *font.Face {
	face := s.fontMap.ResolveFace(r)
	if face != nil {
		family, aspect := s.fontMap.FontMetadata(face.Font)
		md := opentype.DescriptionToFont(font.Description{
			Family: family,
			Aspect: aspect,
		})
		s.addFace(face, md)
		return face
	}
	return nil
}

func (s *shaperImpl) splitByFaces(inputs []shaping.Input, buf []shaping.Input) []shaping.Input {
	var split []shaping.Input
	if buf == nil {
		split = make([]shaping.Input, 0, len(inputs))
	} else {
		split = buf
	}
	for _, input := range inputs {
		split = append(split, shaping.SplitByFace(input, s)...)
	}
	return split
}

func (s *shaperImpl) shapeText(ppem fixed.Int26_6, lc system.Locale, txt []rune) []shaping.Output {
	lcfg := langConfig{
		Language:  language.NewLanguage(lc.Language),
		Direction: mapDirection(lc.Direction),
	}
	input := toInput(nil, ppem, lcfg, txt)
	if input.RunStart == input.RunEnd && len(s.faces) > 0 {
		input.Face = s.faces[0]
	}
	inputs := s.splitBidi(input)
	inputs = s.splitByFaces(inputs, s.splitScratch1[:0])
	inputs = splitByScript(inputs, lcfg.Direction, s.splitScratch2[:0])
	if needed := len(inputs) - len(s.outScratchBuf); needed > 0 {
		s.outScratchBuf = slices.Grow(s.outScratchBuf, needed)
	}
	s.outScratchBuf = s.outScratchBuf[:0]
	for _, input := range inputs {
		if input.Face != nil {
			s.outScratchBuf = append(s.outScratchBuf, s.shaper.Shape(input))
		} else {
			s.outScratchBuf = append(s.outScratchBuf, shaping.Output{
				Advance: input.Size,
				Size:    input.Size,
				Glyphs: []shaping.Glyph{
					{
						Width:        input.Size,
						Height:       input.Size,
						XBearing:     0,
						YBearing:     0,
						XAdvance:     input.Size,
						YAdvance:     input.Size,
						XOffset:      0,
						YOffset:      0,
						ClusterIndex: input.RunStart,
						RuneCount:    input.RunEnd - input.RunStart,
						GlyphCount:   1,
						GlyphID:      0,
						Mask:         0,
					},
				},
				LineBounds: shaping.Bounds{
					Ascent:  input.Size,
					Descent: 0,
					Gap:     0,
				},
				GlyphBounds: shaping.Bounds{
					Ascent:  input.Size,
					Descent: 0,
					Gap:     0,
				},
				Direction: input.Direction,
				Runes: shaping.Range{
					Offset: input.RunStart,
					Count:  input.RunEnd - input.RunStart,
				},
			})
		}
	}
	return s.outScratchBuf
}

func wrapPolicyToGoText(p WrapPolicy) shaping.LineBreakPolicy {
	switch p {
	case WrapGraphemes:
		return shaping.Always
	case WrapWords:
		return shaping.Never
	default:
		return shaping.WhenNecessary
	}
}

func (s *shaperImpl) shapeAndWrapText(params Parameters, txt []rune) (_ []shaping.Line, truncated int) {
	wc := shaping.WrapConfig{
		Direction:                     mapDirection(params.Locale.Direction),
		TruncateAfterLines:            params.MaxLines,
		TextContinues:                 params.forceTruncate,
		BreakPolicy:                   wrapPolicyToGoText(params.WrapPolicy),
		DisableTrailingWhitespaceTrim: params.DisableSpaceTrim,
	}
	families := s.defaultFaces
	if params.Font.Typeface != "" {
		parsed, err := s.parser.parse(string(params.Font.Typeface))
		if err != nil {
			s.logger.Printf("Unable to parse typeface %q: %v", params.Font.Typeface, err)
		} else {
			families = parsed
		}
	}
	s.fontMap.SetQuery(fontscan.Query{
		Families: families,
		Aspect:   opentype.FontToDescription(params.Font).Aspect,
	})
	if wc.TruncateAfterLines > 0 {
		if len(params.Truncator) == 0 {
			params.Truncator = "…"
		}
		wc.Truncator = s.shapeText(params.PxPerEm, params.Locale, []rune(params.Truncator))[0]
	}
	return s.wrapper.WrapParagraph(wc, params.MaxWidth, txt, shaping.NewSliceIterator(s.shapeText(params.PxPerEm, params.Locale, txt)))
}

func replaceControlCharacters(in []rune) []rune {
	for i, r := range in {
		if r == '\t' {
			in[i] = '\u2003'
			continue
		}

		if unicode.IsSpace(r) {
			in[i] = ' '
			continue
		}

		switch r {
		case '\u001C', '\u001D', '\u001E', '\u200B', '\u200C', '\u200D', '\u2060', '\uFEFF':
			in[i] = ' '
		}
	}
	return in
}

func (s *shaperImpl) LayoutString(params Parameters, txt string) document {
	return s.LayoutRunes(params, []rune(txt))
}

func (s *shaperImpl) Layout(params Parameters, txt io.RuneReader) document {
	s.scratchRunes = s.scratchRunes[:0]
	for r, _, err := txt.ReadRune(); err != nil; r, _, err = txt.ReadRune() {
		s.scratchRunes = append(s.scratchRunes, r)
	}
	return s.LayoutRunes(params, s.scratchRunes)
}

func calculateYOffsets(lines []line) {
	if len(lines) < 1 {
		return
	}
	currentY := lines[0].ascent.Ceil()
	for i := range lines {
		if i > 0 {
			currentY += lines[i].lineHeight.Round()
		}
		lines[i].yOffset = currentY
	}
}

func (s *shaperImpl) LayoutRunes(params Parameters, txt []rune) document {
	hasNewline := len(txt) > 0 && txt[len(txt)-1] == '\n'
	var ls []shaping.Line
	var truncated int
	if hasNewline {
		txt = txt[:len(txt)-1]
	}
	if params.MaxLines != 0 && hasNewline {
		params.forceTruncate = true
	}
	ls, truncated = s.shapeAndWrapText(params, replaceControlCharacters(txt))

	hasTruncator := truncated > 0 || (params.forceTruncate && params.MaxLines == len(ls))
	if hasTruncator && hasNewline {
		truncated++
		hasNewline = false
	}

	textLines := make([]line, len(ls))
	maxHeight := fixed.Int26_6(0)
	for i := range ls {
		otLine := toLine(s.faceToIndex, ls[i], params.Locale.Direction)
		if otLine.lineHeight > maxHeight {
			maxHeight = otLine.lineHeight
		}
		if isFinalLine := i == len(ls)-1; isFinalLine {
			if hasNewline {
				otLine.insertTrailingSyntheticNewline(len(txt))
			}
			if hasTruncator {
				otLine.setTruncatedCount(truncated)
			}
		}
		textLines[i] = otLine
	}
	if params.LineHeight != 0 {
		maxHeight = params.LineHeight
	}
	if params.LineHeightScale == 0 {
		params.LineHeightScale = 1.2
	}

	maxHeight = floatToFixed(fixedToFloat(maxHeight) * params.LineHeightScale)
	for i := range textLines {
		textLines[i].lineHeight = maxHeight
	}
	calculateYOffsets(textLines)
	return document{
		lines:      textLines,
		alignment:  params.Alignment,
		alignWidth: alignWidth(params.MinWidth, textLines),
	}
}

func alignWidth(minWidth int, lines []line) int {
	for _, l := range lines {
		minWidth = max(minWidth, l.width.Ceil())
	}
	return minWidth
}

func (s *shaperImpl) Shape(pathOps *op.Ops, gs []Glyph) clip.PathSpec {
	var lastPos f32.Point
	var x fixed.Int26_6
	var builder clip.Path
	builder.Begin(pathOps)
	for i, g := range gs {
		if i == 0 {
			x = g.X
		}
		ppem, faceIdx, gid := splitGlyphID(g.ID)
		if faceIdx >= len(s.faces) {
			continue
		}
		face := s.faces[faceIdx]
		if face == nil {
			continue
		}
		scaleFactor := fixedToFloat(ppem) / float32(face.Upem())
		glyphData := face.GlyphData(gid)

		var outline font.GlyphOutline
		switch glyphData := glyphData.(type) {
		case font.GlyphOutline:
			outline = glyphData
		case font.GlyphSVG:
			outline = glyphData.Outline
		default:
			continue
		}

		pos := f32.Point{
			X: fixedToFloat((g.X - x) - g.Offset.X),
			Y: -fixedToFloat(g.Offset.Y),
		}
		builder.Move(pos.Sub(lastPos))
		lastPos = pos
		var lastArg f32.Point

		for _, fseg := range outline.Segments {
			nargs := 1
			switch fseg.Op {
			case gotextot.SegmentOpQuadTo:
				nargs = 2
			case gotextot.SegmentOpCubeTo:
				nargs = 3
			}
			var args [3]f32.Point
			for i := range nargs {
				a := f32.Point{
					X: fseg.Args[i].X * scaleFactor,
					Y: -fseg.Args[i].Y * scaleFactor,
				}
				args[i] = a.Sub(lastArg)
				if i == nargs-1 {
					lastArg = a
				}
			}
			switch fseg.Op {
			case gotextot.SegmentOpMoveTo:
				builder.Move(args[0])
			case gotextot.SegmentOpLineTo:
				builder.Line(args[0])
			case gotextot.SegmentOpQuadTo:
				builder.Quad(args[0], args[1])
			case gotextot.SegmentOpCubeTo:
				builder.Cube(args[0], args[1], args[2])
			default:
				panic("unsupported segment op")
			}
		}
		lastPos = lastPos.Add(lastArg)
	}
	return builder.End()
}

func fixedToFloat(i fixed.Int26_6) float32 {
	return float32(i) / 64.0
}

func floatToFixed(f float32) fixed.Int26_6 {
	return fixed.Int26_6(f * 64)
}

func (s *shaperImpl) Bitmaps(ops *op.Ops, gs []Glyph) op.CallOp {
	var x fixed.Int26_6
	bitmapMacro := op.Record(ops)
	for i, g := range gs {
		if i == 0 {
			x = g.X
		}
		_, faceIdx, gid := splitGlyphID(g.ID)
		if faceIdx >= len(s.faces) {
			continue
		}
		face := s.faces[faceIdx]
		if face == nil {
			continue
		}
		glyphData := face.GlyphData(gid)
		switch glyphData := glyphData.(type) {
		case font.GlyphBitmap:
			var imgOp paint.ImageOp
			var imgSize image.Point
			bitmapData, ok := s.bitmapGlyphCache.Get(g.ID)
			if !ok {
				var img image.Image
				var err error
				switch glyphData.Format {
				case font.PNG, font.JPG, font.TIFF:
					img, _, err = image.Decode(bytes.NewReader(glyphData.Data))
				case font.BlackAndWhite:
					continue
				default:
					continue
				}

				if err != nil || img == nil {
					continue
				}

				imgOp = paint.NewImageOp(img)
				imgSize = img.Bounds().Size()
				s.bitmapGlyphCache.Put(g.ID, bitmap{img: imgOp, size: imgSize})
			} else {
				imgOp = bitmapData.img
				imgSize = bitmapData.size
			}
			off := op.Affine(f32.AffineId().Offset(f32.Point{
				X: fixedToFloat((g.X - x) + g.Offset.X),
				Y: fixedToFloat(g.Offset.Y + g.Bounds.Min.Y),
			})).Push(ops)
			cl := clip.Rect{Max: imgSize}.Push(ops)

			glyphSize := image.Rectangle{
				Min: image.Point{
					X: g.Bounds.Min.X.Round(),
					Y: g.Bounds.Min.Y.Round(),
				},
				Max: image.Point{
					X: g.Bounds.Max.X.Round(),
					Y: g.Bounds.Max.Y.Round(),
				},
			}.Size()
			aff := op.Affine(f32.AffineId().Scale(f32.Point{}, f32.Point{
				X: float32(glyphSize.X) / float32(imgSize.X),
				Y: float32(glyphSize.Y) / float32(imgSize.Y),
			})).Push(ops)
			imgOp.Add(ops)
			paint.PaintOp{}.Add(ops)
			aff.Pop()
			cl.Pop()
			off.Pop()
		}
	}
	return bitmapMacro.Stop()
}

type langConfig struct {
	language.Language
	language.Script
	di.Direction
}

func toInput(face *font.Face, ppem fixed.Int26_6, lc langConfig, runes []rune) shaping.Input {
	var input shaping.Input
	input.Direction = lc.Direction
	input.Text = runes
	input.Size = ppem
	input.Face = face
	input.Language = lc.Language
	input.Script = lc.Script
	input.RunStart = 0
	input.RunEnd = len(runes)
	return input
}

func mapDirection(d system.TextDirection) di.Direction {
	switch d {
	case system.LTR:
		return di.DirectionLTR
	case system.RTL:
		return di.DirectionRTL
	}
	return di.DirectionLTR
}

func unmapDirection(d di.Direction) system.TextDirection {
	switch d {
	case di.DirectionLTR:
		return system.LTR
	case di.DirectionRTL:
		return system.RTL
	}
	return system.LTR
}

func toGioGlyphs(in []shaping.Glyph, ppem fixed.Int26_6, faceIdx int) []glyph {
	out := make([]glyph, 0, len(in))
	for _, g := range in {
		var bounds fixed.Rectangle26_6
		bounds.Min.X = g.XBearing
		bounds.Min.Y = -g.YBearing
		bounds.Max = bounds.Min.Add(fixed.Point26_6{X: g.Width, Y: -g.Height})
		out = append(out, glyph{
			id:           newGlyphID(ppem, faceIdx, g.GlyphID),
			clusterIndex: g.ClusterIndex,
			runeCount:    g.RuneCount,
			glyphCount:   g.GlyphCount,
			xAdvance:     g.XAdvance,
			yAdvance:     g.YAdvance,
			xOffset:      g.XOffset,
			yOffset:      g.YOffset,
			bounds:       bounds,
		})
	}
	return out
}

func toLine(faceToIndex map[*font.Font]int, o shaping.Line, dir system.TextDirection) line {
	if len(o) < 1 {
		return line{}
	}
	line := line{
		runs:        make([]runLayout, len(o)),
		direction:   dir,
		visualOrder: make([]int, len(o)),
	}
	maxSize := fixed.Int26_6(0)
	for i := range o {
		run := o[i]
		if run.Size > maxSize {
			maxSize = run.Size
		}
		var font *font.Font
		if run.Face != nil {
			font = run.Face.Font
		}
		line.runs[i] = runLayout{
			Glyphs: toGioGlyphs(run.Glyphs, run.Size, faceToIndex[font]),
			Runes: Range{
				Count:  run.Runes.Count,
				Offset: line.runeCount,
			},
			Direction:      unmapDirection(run.Direction),
			face:           run.Face,
			Advance:        run.Advance,
			PPEM:           run.Size,
			VisualPosition: int(run.VisualIndex),
		}
		line.visualOrder[run.VisualIndex] = i
		line.runeCount += run.Runes.Count
		line.width += run.Advance
		if line.ascent < run.LineBounds.Ascent {
			line.ascent = run.LineBounds.Ascent
		}
		if line.descent < -run.LineBounds.Descent+run.LineBounds.Gap {
			line.descent = -run.LineBounds.Descent + run.LineBounds.Gap
		}
	}
	line.lineHeight = maxSize
	x := fixed.Int26_6(0)
	for _, runIdx := range line.visualOrder {
		line.runs[runIdx].X = x
		x += line.runs[runIdx].Advance
	}
	return line
}
