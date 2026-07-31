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

	"github.com/nanorele/typesetting/di"
	"github.com/nanorele/typesetting/font"
	gotextot "github.com/nanorele/typesetting/font/opentype"
	"github.com/nanorele/typesetting/fontscan"
	"github.com/nanorele/typesetting/language"
	"github.com/nanorele/typesetting/shaping"
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
	runs            []runLayout
	glyphs          []glyph
	visual          []int
	alignment       Alignment
	alignWidth      int
	unreadRuneCount int
	// gen is the layout generation (see shaperImpl.docGen) during which this
	// document was last produced or served from the layout cache. Shaper.txt
	// aliases the document's slices for one generation, so a document may
	// only be recycled once its generation has passed.
	gen uint64
}

func (l *document) append(other *document) {
	startIdx := len(l.lines)
	l.lines = append(l.lines, other.lines...)
	l.runs = append(l.runs, other.runs...)
	l.glyphs = append(l.glyphs, other.glyphs...)
	l.visual = append(l.visual, other.visual...)
	l.alignWidth = max(l.alignWidth, other.alignWidth)
	calculateYOffsetsFrom(l.lines, startIdx)
}

func (l *document) reset() {
	l.lines = l.lines[:0]
	l.runs = l.runs[:0]
	l.glyphs = l.glyphs[:0]
	l.visual = l.visual[:0]
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
			l.runs[finalRunIdx].Glyphs[i].runeCount = 0
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
	bidiScratch      []shaping.Input
	outScratchBuf    []shaping.Output
	scratchRunes     []rune
	bitmapGlyphCache bitmapCache
	glyphDataCache   map[glyphDataKey]glyphDataEntry

	// truncCache holds shaped truncator symbols. Every truncated label
	// otherwise pays a full shaping pass for the ellipsis on each layout
	// cache miss. Cleared whenever a new face is registered, since face
	// resolution for the truncator may change.
	truncCache map[truncKey]shaping.Output

	// docGen counts layout passes of the owning Shaper; docFree pools
	// documents evicted from the layout cache for reuse by LayoutRunes.
	// Only documents whose gen is older than docGen may be pooled — see
	// document.gen.
	docGen  uint64
	docFree []*document

	// disableSingleLineTrim turns off the MaxLines==1 shaping cutoff.
	// Used by tests to compare trimmed output against full shaping.
	disableSingleLineTrim bool

	lazyFaces   []lazyFaceEntry
	lazyPending int
	// lazyLow counts, per rune below lazyLowLimit, how many unloaded lazy
	// faces claim it, and lazyBlocks does the same per 256-rune block above
	// that. Latin text must not pay for the isolated low codepoints an emoji
	// face claims, so the busy range is tracked exactly.
	lazyLow    []uint8
	lazyBlocks []uint16
}

type lazyFaceEntry struct {
	LazyFace
	loaded bool
}

const runeBlockCount = (unicodeMax >> runeBlockShift) + 1

const (
	runeBlockShift = 8
	unicodeMax     = 0x10FFFF
	lazyLowLimit   = 0x1000
)

type glyphDataKey struct {
	face *font.Face
	gid  font.GID
}

// truncKey identifies a shaped truncator symbol. The shaped result depends on
// the text itself, the size, the queried font and the locale (language and
// direction).
type truncKey struct {
	truncator string
	ppem      fixed.Int26_6
	font      giofont.Font
	locale    system.Locale
}

// maxCachedTruncators bounds truncCache. Truncator/size/font combinations are
// few in practice; the bound only guards degenerate callers.
const maxCachedTruncators = 64

// growSlice returns a slice of length n, reusing s's backing array when it is
// large enough. The caller is expected to overwrite every element.
func growSlice[T any](s []T, n int) []T {
	if cap(s) >= n {
		return s[:n]
	}
	return make([]T, n)
}

// getDoc returns a pooled document (emptied, capacity preserved) or a new one.
func (s *shaperImpl) getDoc() *document {
	if n := len(s.docFree); n > 0 {
		d := s.docFree[n-1]
		s.docFree = s.docFree[:n-1]
		return d
	}
	return new(document)
}

// recycleDoc accepts a document evicted from the layout cache for reuse.
// Documents produced or served during the current generation are still
// aliased by Shaper.txt and must not be reused; oversized documents are not
// worth retaining.
func (s *shaperImpl) recycleDoc(d *document) {
	const (
		maxPooledDocs      = 8
		maxPooledDocGlyphs = 8192
	)
	if d == nil || d.gen == s.docGen ||
		len(s.docFree) >= maxPooledDocs || cap(d.glyphs) > maxPooledDocGlyphs {
		return
	}
	d.reset()
	s.docFree = append(s.docFree, d)
}

type glyphDataEntry struct {
	outline font.GlyphOutline
	bitmap  font.GlyphBitmap
	isBmp   bool
}

// maxCachedGlyphData bounds the per-face glyph data cache. Entries hold
// unscaled outlines, so one entry serves every text size; a few hundred
// covers the Latin/Cyrillic working set of a code view.
const maxCachedGlyphData = 4096

// glyphData returns the parsed data for gid, caching it. Without the cache
// every repaint re-parses the glyf table for every visible glyph — both to
// build outlines and, in Bitmaps, merely to discover the glyph is not a
// bitmap. That parse dominates allocation in text-heavy views.
func (s *shaperImpl) glyphData(face *font.Face, gid font.GID) glyphDataEntry {
	k := glyphDataKey{face: face, gid: gid}
	if e, ok := s.glyphDataCache[k]; ok {
		return e
	}
	var e glyphDataEntry
	switch data := face.GlyphData(gid).(type) {
	case font.GlyphOutline:
		e.outline = data
	case font.GlyphSVG:
		e.outline = data.Outline
	case font.GlyphBitmap:
		e.bitmap = data
		e.isBmp = true
	}
	if s.glyphDataCache == nil {
		s.glyphDataCache = make(map[glyphDataKey]glyphDataEntry)
	}
	if len(s.glyphDataCache) >= maxCachedGlyphData {
		clear(s.glyphDataCache)
	}
	s.glyphDataCache[k] = e
	return e
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

func newShaperImpl(systemFonts bool, collection []FontFace, lazy []LazyFace) *shaperImpl {
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
	shaper.initLazy(lazy)
	shaper.shaper.SetFontCacheSize(32)
	return &shaper
}

func (s *shaperImpl) initLazy(lazy []LazyFace) {
	if len(lazy) == 0 {
		return
	}
	s.lazyFaces = make([]lazyFaceEntry, len(lazy))
	s.lazyLow = make([]uint8, lazyLowLimit)
	s.lazyBlocks = make([]uint16, runeBlockCount)
	for i, lf := range lazy {
		s.lazyFaces[i] = lazyFaceEntry{LazyFace: lf}
		s.defaultFaces = append(s.defaultFaces, string(lf.Typeface))
		s.markClaims(lf.Ranges, 1)
	}
	s.lazyPending = len(lazy)
}

func (s *shaperImpl) markClaims(ranges []RuneRange, delta int) {
	for _, rr := range ranges {
		lo, hi := rr.Lo, rr.Hi
		if lo < 0 {
			lo = 0
		}
		if hi > unicodeMax {
			hi = unicodeMax
		}
		for r := lo; r <= hi && r < lazyLowLimit; r++ {
			s.lazyLow[r] = uint8(int(s.lazyLow[r]) + delta)
		}
		if hi < lazyLowLimit {
			continue
		}
		if lo < lazyLowLimit {
			lo = lazyLowLimit
		}
		for b := lo >> runeBlockShift; b <= hi>>runeBlockShift; b++ {
			s.lazyBlocks[b] = uint16(int(s.lazyBlocks[b]) + delta)
		}
	}
}

func (s *shaperImpl) lazyClaimed(r rune) bool {
	if r < 0 || r > unicodeMax {
		return false
	}
	if r < lazyLowLimit {
		return s.lazyLow[r] != 0
	}
	return s.lazyBlocks[r>>runeBlockShift] != 0
}

func (s *shaperImpl) loadLazyFace(i int) {
	e := &s.lazyFaces[i]
	if e.loaded {
		return
	}
	e.loaded = true
	s.lazyPending--
	s.markClaims(e.Ranges, -1)
	ff, err := e.Load()
	if err != nil {
		s.logger.Printf("failed loading deferred face %q: %v", e.Typeface, err)
		return
	}
	s.Load(ff)
}

// claimLazyFaces loads any unloaded lazy face declaring r.
func (s *shaperImpl) claimLazyFaces(r rune) {
	if s.lazyPending == 0 || !s.lazyClaimed(r) {
		return
	}
	for i := range s.lazyFaces {
		e := &s.lazyFaces[i]
		if e.loaded || !runeInRanges(e.Ranges, r) {
			continue
		}
		s.loadLazyFace(i)
	}
}

// claimLazyFacesByFont loads any unloaded lazy face whose Match predicate
// accepts the font requested by the current layout. Runs before the font
// query is set so the freshly loaded face participates in resolution.
func (s *shaperImpl) claimLazyFacesByFont(f giofont.Font) {
	if s.lazyPending == 0 {
		return
	}
	for i := range s.lazyFaces {
		e := &s.lazyFaces[i]
		if e.loaded || e.Match == nil || !e.Match(f) {
			continue
		}
		s.loadLazyFace(i)
	}
}

// loadAllLazyFaces materializes every remaining lazy face. It runs when a rune
// is covered by no loaded face, so that deferring a face can never lose
// coverage an eager collection would have had.
func (s *shaperImpl) loadAllLazyFaces() {
	for i := range s.lazyFaces {
		s.loadLazyFace(i)
	}
}

func faceHasRune(f *font.Face, r rune) bool {
	if f == nil {
		return false
	}
	_, ok := f.NominalGlyph(r)
	return ok
}

func runeInRanges(ranges []RuneRange, r rune) bool {
	for _, rr := range ranges {
		if r >= rr.Lo && r <= rr.Hi {
			return true
		}
	}
	return false
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
	// A new face can change which face the truncator resolves to.
	clear(s.truncCache)
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

func isASCII(rs []rune) bool {
	for _, r := range rs {
		if r >= 0x80 {
			return false
		}
	}
	return true
}

func (s *shaperImpl) splitBidi(input shaping.Input) []shaping.Input {
	// The returned slice is only read (and copied from) before the next
	// splitBidi call, so a single scratch buffer serves every call.
	splitInputs := s.bidiScratch[:0]
	defer func() { s.bidiScratch = splitInputs }()
	if input.Direction.Axis() != di.Horizontal || input.RunStart == input.RunEnd {
		splitInputs = append(splitInputs, input)
		return splitInputs
	}
	// Fast path: in an LTR paragraph, pure ASCII text has no bidi
	// reordering; skip the bidi.Paragraph machinery, which otherwise
	// copies the input into []byte and allocates per-rune Class/bracket
	// tables (~10x the input size for large ASCII paragraphs). In an RTL
	// paragraph the fast path would keep the locale's RTL direction on
	// strong-LTR characters and render them reversed, so bidi resolution
	// must still run there.
	if input.Direction.Progression() != di.TowardTopLeft && isASCII(input.Text[input.RunStart:input.RunEnd]) {
		splitInputs = append(splitInputs, input)
		return splitInputs
	}
	def := bidi.LeftToRight
	if input.Direction.Progression() == di.TowardTopLeft {
		def = bidi.RightToLeft
	}
	s.bidiParagraph.SetString(string(input.Text), bidi.DefaultDirection(def)) //nolint:errcheck // Subsequent Order() call surfaces any error.
	out, err := s.bidiParagraph.Order()
	if err != nil {
		splitInputs = append(splitInputs, input)
		return splitInputs
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
	s.claimLazyFaces(r)
	face := s.fontMap.ResolveFace(r)
	if s.lazyPending > 0 && !faceHasRune(face, r) {
		s.loadAllLazyFaces()
		face = s.fontMap.ResolveFace(r)
	}
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
	// Store grown buffers back so capacity is retained across calls.
	s.splitScratch1 = inputs
	inputs = splitByScript(inputs, lcfg.Direction, s.splitScratch2[:0])
	s.splitScratch2 = inputs
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
	s.claimLazyFacesByFont(params.Font)
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
		wc.Truncator = s.shapedTruncator(params)
	}
	txt, trimmed, outs := s.trimForSingleLine(params, wc, txt)
	if outs == nil {
		outs = s.shapeText(params.PxPerEm, params.Locale, txt)
	}
	lines, truncated := s.wrapper.WrapParagraph(wc, params.MaxWidth, txt, shaping.NewSliceIterator(outs))
	// The trimmed tail could never be visible, so it is truncated by
	// construction (trimming only happens when the retained prefix alone
	// overflows MaxWidth).
	truncated += trimmed
	return lines, truncated
}

// shapedTruncator returns the shaped truncator symbol for params, cached.
// Without the cache every truncated label pays a full shaping pass for the
// ellipsis on each layout-cache miss.
func (s *shaperImpl) shapedTruncator(params Parameters) shaping.Output {
	k := truncKey{
		truncator: params.Truncator,
		ppem:      params.PxPerEm,
		font:      params.Font,
		locale:    params.Locale,
	}
	if out, ok := s.truncCache[k]; ok {
		return out
	}
	// The Output struct in outScratchBuf is overwritten by the next
	// shapeText call, but its Glyphs array is freshly allocated per Shape,
	// so the copy stored in the cache owns its glyph data. The wrapper
	// never mutates the truncator's Glyphs (it copies the Output struct
	// and only adjusts the copy's Runes), so sharing it across wraps is
	// safe.
	out := s.shapeText(params.PxPerEm, params.Locale, []rune(params.Truncator))[0]
	if s.truncCache == nil {
		s.truncCache = make(map[truncKey]shaping.Output)
	}
	if len(s.truncCache) >= maxCachedTruncators {
		clear(s.truncCache)
	}
	s.truncCache[k] = out
	return out
}

// trimForSingleLine bounds the shaping cost of a single truncated line: for
// MaxLines==1 the glyphs past the visible width can never be displayed, yet
// shaping them makes an 80-character cell cost several times a short one.
// It returns a prefix of txt whose shaped advance comfortably exceeds the
// space a truncated line can use, the number of runes cut off, and the
// prefix's shaped outputs (nil when no trim applied — the caller shapes
// then).
//
// The cut must not change the visible glyphs, so it only applies to ASCII
// text in a left-to-right locale: no bidi reordering (bracket pairing is
// paragraph-global), no cross-boundary cluster or joining effects, and any
// kerning/ligature difference at the cut sits beyond the visible width by at
// least the safety margin.
func (s *shaperImpl) trimForSingleLine(params Parameters, wc shaping.WrapConfig, txt []rune) (_ []rune, trimmed int, outs []shaping.Output) {
	if wc.TruncateAfterLines != 1 || s.disableSingleLineTrim ||
		params.PxPerEm < fixed.I(1) || params.MaxWidth <= 0 ||
		params.Locale.Direction.Progression() != system.FromOrigin ||
		!isASCII(txt) {
		return txt, 0, nil
	}
	// Everything that fits must be shaped, plus a margin for the truncator
	// and boundary effects (kerning, ligatures) at the cut.
	needed := fixed.I(params.MaxWidth) + wc.Truncator.Advance + 2*params.PxPerEm
	// No printable ASCII glyph is narrower than ~1/8 em, which bounds how
	// many runes can contribute to the visible width.
	maxVisibleRunes := 8 * needed.Ceil() / params.PxPerEm.Floor()
	if len(txt) <= maxVisibleRunes {
		return txt, 0, nil
	}
	n := maxVisibleRunes
	for {
		outs = s.shapeText(params.PxPerEm, params.Locale, txt[:n])
		var advance fixed.Int26_6
		for _, o := range outs {
			advance += o.Advance
		}
		// The estimate is conservative, so a single pass suffices unless
		// the font is far narrower than any plausible ASCII face; the
		// loop guards correctness in that case.
		if advance >= needed || n == len(txt) {
			return txt[:n], len(txt) - n, outs
		}
		n = min(2*n, len(txt))
	}
}

func replaceControlCharacters(in []rune) []rune {
	for i, r := range in {
		switch r {
		case '\t':
			// Tabs render as an em space instead of the single-space
			// width harfbuzz would give them.
			in[i] = '\u2003'
		// ASCII file/group/record separators, VT, FF, CR, LF, "next line" and
		// "paragraph separator": paragraphs are split before shaping, so
		// any stragglers render as plain spaces instead of injecting
		// mandatory breaks mid-paragraph. U+2028 LINE SEPARATOR is
		// deliberately NOT in this list: it exists to break lines and
		// the wrapper honors it. Everything else must
		// reach the shaper untouched: zero-width characters (ZWSP, ZWNJ,
		// ZWJ, WJ, BOM) drive emoji sequences and cursive joining, and
		// typographic spaces (NBSP, en/em/thin) carry their own width
		// and line-break semantics.
		case '\u001C', '\u001D', '\u001E', '\v', '\f', '\r', '\n', '\u0085', '\u2029':
			in[i] = ' '
		}
	}
	return in
}

func (s *shaperImpl) LayoutString(params Parameters, txt string) *document {
	// Decode straight into the scratch buffer: []rune(txt) would allocate
	// a throwaway intermediate slice on every call.
	s.scratchRunes = s.scratchRunes[:0]
	for _, r := range txt {
		s.scratchRunes = append(s.scratchRunes, r)
	}
	return s.LayoutRunes(params, s.scratchRunes)
}

func (s *shaperImpl) Layout(params Parameters, txt io.RuneReader) *document {
	s.scratchRunes = s.scratchRunes[:0]
	for {
		r, _, err := txt.ReadRune()
		if err != nil {
			break
		}
		s.scratchRunes = append(s.scratchRunes, r)
	}
	return s.LayoutRunes(params, s.scratchRunes)
}

func calculateYOffsets(lines []line) {
	calculateYOffsetsFrom(lines, 0)
}

// calculateYOffsetsFrom fills in yOffset for lines[startIdx:] only, reusing
// lines[startIdx-1].yOffset as the carry. For a full recompute pass startIdx=0.
func calculateYOffsetsFrom(lines []line, startIdx int) {
	if len(lines) < 1 {
		return
	}
	if startIdx <= 0 {
		lines[0].yOffset = lines[0].ascent.Ceil()
		startIdx = 1
	}
	currentY := lines[startIdx-1].yOffset
	for i := startIdx; i < len(lines); i++ {
		currentY += lines[i].lineHeight.Round()
		lines[i].yOffset = currentY
	}
}

func (s *shaperImpl) LayoutRunes(params Parameters, txt []rune) *document {
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

	totalRuns := 0
	totalGlyphs := 0
	for _, l := range ls {
		totalRuns += len(l)
		for _, r := range l {
			totalGlyphs += len(r.Glyphs)
		}
	}

	// Reuse a document evicted from the layout cache when one is available:
	// on a cache miss these four slices are the layout's dominant
	// allocation, and eviction means a same-sized buffer just became free.
	doc := s.getDoc()
	resLines := growSlice(doc.lines, len(ls))
	resRuns := growSlice(doc.runs, totalRuns)
	resGlyphs := growSlice(doc.glyphs, totalGlyphs)
	resVisual := growSlice(doc.visual, totalRuns)

	runIdx := 0
	glyphIdx := 0
	maxHeight := fixed.Int26_6(0)
	for i := range ls {
		l := ls[i]
		lineRuns := resRuns[runIdx : runIdx+len(l)]
		lineVisual := resVisual[runIdx : runIdx+len(l)]
		runIdx += len(l)

		otLine := line{
			runs:        lineRuns,
			visualOrder: lineVisual,
			direction:   params.Locale.Direction,
		}

		for j := range l {
			run := l[j]
			if run.Size > maxHeight {
				maxHeight = run.Size
			}
			var font *font.Font
			if run.Face != nil {
				font = run.Face.Font
			}

			runGlyphs := resGlyphs[glyphIdx : glyphIdx+len(run.Glyphs)]
			glyphIdx += len(run.Glyphs)

			for k, g := range run.Glyphs {
				var bounds fixed.Rectangle26_6
				bounds.Min.X = g.XBearing
				bounds.Min.Y = -g.YBearing
				bounds.Max = bounds.Min.Add(fixed.Point26_6{X: g.Width, Y: -g.Height})
				runGlyphs[k] = glyph{
					id:           newGlyphID(run.Size, s.faceToIndex[font], g.GlyphID),
					clusterIndex: g.TextIndex(),
					runeCount:    g.RunesCount(),
					glyphCount:   g.GlyphsCount(),
					xAdvance:     g.Advance,
					xOffset:      g.XOffset,
					yOffset:      g.YOffset,
					bounds:       bounds,
				}
			}

			lineRuns[j] = runLayout{
				Glyphs:         runGlyphs,
				Runes:          Range{Count: run.Runes.Count, Offset: otLine.runeCount},
				Direction:      unmapDirection(run.Direction),
				face:           run.Face,
				Advance:        run.Advance,
				PPEM:           run.Size,
				VisualPosition: int(run.VisualIndex),
			}
			lineVisual[run.VisualIndex] = j
			otLine.runeCount += run.Runes.Count
			otLine.width += run.Advance
			if otLine.ascent < run.LineBounds.Ascent {
				otLine.ascent = run.LineBounds.Ascent
			}
			if otLine.descent < -run.LineBounds.Descent+run.LineBounds.Gap {
				otLine.descent = -run.LineBounds.Descent + run.LineBounds.Gap
			}
		}

		otLine.lineHeight = maxHeight
		x := fixed.Int26_6(0)
		for _, idx := range lineVisual {
			lineRuns[idx].X = x
			x += lineRuns[idx].Advance
		}

		if isFinalLine := i == len(ls)-1; isFinalLine {
			if hasNewline {
				otLine.insertTrailingSyntheticNewline(len(txt))
			}
			if hasTruncator {
				otLine.setTruncatedCount(truncated)
			}
		}
		resLines[i] = otLine
	}

	if params.LineHeight != 0 {
		maxHeight = params.LineHeight
	}
	if params.LineHeightScale == 0 {
		params.LineHeightScale = 1.2
	}

	maxHeight = floatToFixed(fixedToFloat(maxHeight) * params.LineHeightScale)
	for i := range resLines {
		resLines[i].lineHeight = maxHeight
	}
	calculateYOffsets(resLines)
	*doc = document{
		lines:      resLines,
		runs:       resRuns,
		glyphs:     resGlyphs,
		visual:     resVisual,
		alignment:  params.Alignment,
		alignWidth: alignWidth(params.MinWidth, resLines),
		gen:        s.docGen,
	}
	// Drop references to per-paragraph shaping outputs now that the data
	// has been copied into the document; otherwise the underlying
	// []shaping.Glyph (potentially hundreds of MB for large texts) stays
	// alive until the next shape call. The wrapper's internal buffers
	// also hold Output copies that alias the same glyph arrays; clear
	// them too.
	for i := range s.outScratchBuf {
		s.outScratchBuf[i] = shaping.Output{}
	}
	s.wrapper.ResetBuffers()
	return doc
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
	var cachedPpem fixed.Int26_6
	cachedFaceIdx := -1
	var cachedScaleFactor float32
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
		var scaleFactor float32
		if ppem == cachedPpem && faceIdx == cachedFaceIdx {
			scaleFactor = cachedScaleFactor
		} else {
			scaleFactor = fixedToFloat(ppem) / float32(face.Upem())
			cachedPpem = ppem
			cachedFaceIdx = faceIdx
			cachedScaleFactor = scaleFactor
		}
		outline := s.glyphData(face, gid).outline
		if outline.Segments == nil {
			continue
		}

		pos := f32.Point{
			X: fixedToFloat((g.X - x) + g.Offset.X),
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

func (s *shaperImpl) hasBitmapGlyphs(gs []Glyph) bool {
	for _, g := range gs {
		_, faceIdx, gid := splitGlyphID(g.ID)
		if faceIdx >= len(s.faces) {
			continue
		}
		face := s.faces[faceIdx]
		if face == nil {
			continue
		}
		if s.glyphData(face, gid).isBmp {
			return true
		}
	}
	return false
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
		entry := s.glyphData(face, gid)
		switch {
		case entry.isBmp:
			glyphData := entry.bitmap
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

func toGioGlyphs(in []shaping.Glyph, ppem fixed.Int26_6, faceIdx int) []glyph {
	out := make([]glyph, 0, len(in))
	for _, g := range in {
		var bounds fixed.Rectangle26_6
		bounds.Min.X = g.XBearing
		bounds.Min.Y = -g.YBearing
		bounds.Max = bounds.Min.Add(fixed.Point26_6{X: g.Width, Y: -g.Height})
		out = append(out, glyph{
			id:           newGlyphID(ppem, faceIdx, g.GlyphID),
			clusterIndex: g.TextIndex(),
			runeCount:    g.RunesCount(),
			glyphCount:   g.GlyphsCount(),
			xAdvance:     g.Advance,
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

func unmapDirection(d di.Direction) system.TextDirection {
	switch d {
	case di.DirectionLTR:
		return system.LTR
	case di.DirectionRTL:
		return system.RTL
	}
	return system.LTR
}
