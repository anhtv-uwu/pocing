package render

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	customfont "github.com/anhtv-uwu/pocing/font"
	"github.com/anhtv-uwu/pocing/highlight"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

const (
	fontSizePt    = 13.0
	fontDPI       = 144.0
	lineHeightMul = 1.4
	paddingX      = 32
	paddingY      = 20
	tabBarHeight  = 38
	tabPad        = 14
	tabIconGap    = 7
	gutterPadR    = 20
	gutterPadL    = 16
	codeGutterGap = 16

	// Burp's panels are denser than an editor's and its gutter is narrower.
	burpLineHeightMul = 1.25
	burpTitleH        = 32
	burpTabRowH       = 34
	burpTabUnderlineH = 2
	burpPadY          = 8
	burpGutterPadL    = 10
	burpGutterPadR    = 12
	burpCodeGutterGap = 10
	burpTitleX        = 12
	burpControlsGap   = 40
	gapRowHeight      = 30
	highlightBorderW  = 2
	highlightPadR     = 29 // keeps the box border off the window edge
	markPadX          = 3  // breathing room around a column mark
	maxImageWidth     = 2400
	// Browsers stop decoding past 32767px, so anything taller is not evidence
	// anyone can open. Warn well before that: past a few thousand pixels the
	// image stops being readable as a screenshot even though it still loads.
	maxImageHeight  = 32000
	warnImageHeight = 8000
	iconSizePt      = 9.0
	uiSizePt        = 10.0 // panel titles and tabs: UI text is smaller than code
	defaultWrapCols = 100
	minContentChars = 20
	chromeMargin    = 28
	chromeRadius    = 10
	chromeBorderW   = 1
)

// Config holds all parameters for a single render pass.
type Config struct {
	Lines          []highlight.LineTokens
	Title          string
	HighlightLines highlight.Marks
	Theme          *Theme
	Wrap           bool   // enable word wrap
	WrapWidth      int    // columns per visual row (0 = defaultWrapCols)
	HTTPMode       bool   // panel header instead of tab bar; caller adds chrome
	MinHeight      int    // pad the panel to at least this height
	Caption        string // optional strip stating what the image proves

	// Bold draws Burp light message text in DejaVu Sans Mono Bold instead of
	// Burp's regular weight. Ignored by the other themes.
	Bold bool
	// MaxRows caps an HTTP panel's visible rows (0 = no cap); a capped light
	// panel shows a scrollbar thumb.
	MaxRows int
	// CaretLine is the line Burp's current-line band sits on in the light
	// theme: 0 means line 1, negative draws no band.
	CaretLine int

	scroll *scrollState // set by viewport

	httpTabs []burpTab // view strip for an HTTP panel

	noWarn bool // set on a re-render, so warnings are not printed twice
}

// visualLine is one rendered row — may be a logical line, a continuation row
// from word wrap, or a gap sentinel.
type visualLine struct {
	lineNum        int
	isGap          bool
	skipped        int  // lines hidden by this gap (0 = unknown)
	isContinuation bool // second+ row of a wrapped logical line
	rowIndex       int  // which row of the wrapped logical line this is
	startCol       int  // 1-based column of the logical line this row starts at
	indent         int  // columns of hanging indent before the row's text
	tokens         []highlight.Token
}

// runeLen is how many characters this row actually shows.
func (v visualLine) runeLen() int {
	n := 0
	for _, tok := range v.tokens {
		n += utf8.RuneCountInString(tok.Text)
	}
	return n
}

// buildVisualLines converts logical lines into visual rows, splitting long
// lines when wrapCols > 0.
func buildVisualLines(lines []highlight.LineTokens, wrapCols int) []visualLine {
	return buildVisualLinesMode(lines, wrapCols, false)
}

// buildVisualLinesMode is buildVisualLines with a choice of wrapping: words
// breaks after spaces and parameter separators the way Burp's editor does,
// otherwise rows are cut at exactly wrapCols characters.
func buildVisualLinesMode(lines []highlight.LineTokens, wrapCols int, words bool) []visualLine {
	var result []visualLine
	for _, lt := range lines {
		if lt.IsGap {
			result = append(result, visualLine{isGap: true, skipped: lt.Skipped})
			continue
		}
		if wrapCols <= 0 {
			col := 1
			for i, seg := range splitBreaks(lt.Tokens) {
				vl := visualLine{lineNum: lt.LineNum, isContinuation: i > 0, rowIndex: i, startCol: col, tokens: seg}
				col += vl.runeLen()
				result = append(result, vl)
			}
			continue
		}
		col, i := 1, 0
		for _, seg := range splitBreaks(lt.Tokens) {
			var rows [][]highlight.Token
			indent := 0
			if words {
				// A wrapped row hangs at the segment's own indentation, the
				// way Burp lines up the continuation of an indented tag.
				indent = leadingSpaces(seg)
				if indent > wrapCols/2 {
					indent = 0
				}
				rows = splitTokensWordsIndent(seg, wrapCols, indent)
			} else {
				rows = splitTokensAt(seg, wrapCols)
			}
			for j, row := range rows {
				vl := visualLine{
					lineNum:        lt.LineNum,
					isContinuation: i > 0,
					rowIndex:       i,
					startCol:       col,
					tokens:         row,
				}
				if j > 0 {
					vl.indent = indent
				}
				col += vl.runeLen()
				result = append(result, vl)
				i++
			}
		}
	}
	return result
}

// splitTokensAt splits a token slice into rows of at most cols characters each.
func splitTokensAt(tokens []highlight.Token, cols int) [][]highlight.Token {
	var rows [][]highlight.Token
	var current []highlight.Token
	used := 0

	for _, tok := range tokens {
		runes := []rune(tok.Text)
		start := 0
		for start < len(runes) {
			avail := cols - used
			if avail <= 0 {
				rows = append(rows, current)
				current = nil
				used = 0
				avail = cols
			}
			end := start + avail
			if end > len(runes) {
				end = len(runes)
			}
			current = append(current, highlight.Token{
				Text: string(runes[start:end]),
				R:    tok.R,
				G:    tok.G,
				B:    tok.B,
			})
			used += end - start
			start = end
		}
	}
	// Always append the last (possibly empty) row so blank lines render.
	rows = append(rows, current)
	return rows
}

// splitBreaks cuts a line's tokens at Break tokens into row segments.
func splitBreaks(tokens []highlight.Token) [][]highlight.Token {
	segs := [][]highlight.Token{nil}
	for _, t := range tokens {
		if t.Break {
			segs = append(segs, nil)
			continue
		}
		segs[len(segs)-1] = append(segs[len(segs)-1], t)
	}
	return segs
}

// leadingSpaces counts the spaces a row segment starts with.
func leadingSpaces(tokens []highlight.Token) int {
	n := 0
	for _, t := range tokens {
		for _, r := range t.Text {
			if r != ' ' {
				return n
			}
			n++
		}
	}
	return 0 // all blank: nothing to hang from
}

// wrapBreakAt reports whether Burp's editor may break a row after cells[i].
// It breaks after whitespace, and inside markup after the opening quote of
// an attribute value (content="|text/html"); anything else is hard-cut.
func wrapBreakAt(prev, r rune) bool {
	return r == ' ' || r == '\t' || (r == '"' && prev == '=')
}

// splitTokensWords splits a token slice into rows of at most cols characters,
// preferring to break after a wrapBreakAfter character. A run with no break
// opportunity (a long token or base64 blob) is cut at cols, as Burp does.
func splitTokensWords(tokens []highlight.Token, cols int) [][]highlight.Token {
	return splitTokensWordsIndent(tokens, cols, 0)
}

// splitTokensWordsIndent is splitTokensWords where every row after the first
// has indent fewer columns to fill.
func splitTokensWordsIndent(tokens []highlight.Token, cols, indent int) [][]highlight.Token {
	type cell struct {
		r   rune
		tok int
	}
	var cells []cell
	for i, tok := range tokens {
		for _, r := range tok.Text {
			cells = append(cells, cell{r, i})
		}
	}
	emit := func(part []cell) []highlight.Token {
		var row []highlight.Token
		for j := 0; j < len(part); {
			k := j
			for k < len(part) && part[k].tok == part[j].tok {
				k++
			}
			var b strings.Builder
			for _, c := range part[j:k] {
				b.WriteRune(c.r)
			}
			t := tokens[part[j].tok]
			row = append(row, highlight.Token{Text: b.String(), R: t.R, G: t.G, B: t.B})
			j = k
		}
		return row
	}

	var rows [][]highlight.Token
	pos := 0
	width := cols
	for len(cells)-pos > width {
		end := pos + width
		// A space that lands just past the edge still ends the row: Burp
		// lets trailing whitespace hang into the margin.
		if cells[end].r == ' ' {
			end++
		} else {
			for b := end; b > pos+1; b-- {
				prev := rune(0)
				if b >= 2 {
					prev = cells[b-2].r
				}
				if wrapBreakAt(prev, cells[b-1].r) {
					end = b
					break
				}
			}
		}
		rows = append(rows, emit(cells[pos:end]))
		pos = end
		width = max(cols-indent, 1)
	}
	// Always append the last (possibly empty) row so blank lines render.
	return append(rows, emit(cells[pos:]))
}

// metrics holds the font faces and derived pixel sizes. Independent of content.
type metrics struct {
	face       font.Face
	codeFace   font.Face // the message/code text: bold in Burp light
	numFace    font.Face // line numbers
	titleFace  font.Face // HTTP panel title
	uiFace     font.Face // panel chrome (titles, tabs) — smaller than the code face
	iconFace   font.Face
	eyeFace    font.Face // Burp light's larger hide-headers eye
	helpFace   font.Face // the "?" inside the footer's help ring
	charWidth  int
	ascent     int
	descent    int
	uiAscent   int
	uiDescent  int
	iconAscent int
	iconW      int
	// inkGamma (<1) thickens anti-aliased code glyphs toward the heavier
	// look of macOS text rendering; 0 draws coverage as-is.
	inkGamma float64
}

func loadMetrics() (*metrics, error) {
	return loadMetricsFor(nil, false)
}

// Burp layout faces, in points at fontDPI (pixels are twice these values).
// Burp on macOS draws Menlo 13px and Java rounds its advance to an 8px cell;
// we use DejaVu Sans Mono (Menlo's Bitstream Vera ancestor, freely
// licensed) at the same 13px, forced onto 16px cells at 2x, with its
// vertical zones (baseline, x-height, cap, ascender, descender) snapped to
// whole 1x pixels so horizontal strokes come out crisp as on macOS.
// Inter Bold 29.5px matches the "Request" title ink, Inter 24.5px the tabs.
const (
	lightCellW    = 16  // 8px at 1x, Burp's Menlo cell
	lightZoneGrid = 2.0 // 1 px at 1x
	lightCodePt   = 13.0
	lightNumPt    = 13.0
	lightTitlePt  = 14.75
	lightTabPt    = 12.25
	lightIconPt   = 12.0
	lightEyePt    = 16.0
	lightHelpPt   = 12.0
)

// loadMetricsFor loads the faces a theme draws with. Only Burp light differs
// from the upstream JetBrains Mono set.
func loadMetricsFor(t *Theme, bold bool) (*metrics, error) {
	if t != nil && t.BurpLayout {
		return loadLightMetrics(bold)
	}
	face, err := customfont.LoadFace(fontSizePt, fontDPI)
	if err != nil {
		return nil, fmt.Errorf("load font: %w", err)
	}
	uiFace, err := customfont.LoadFace(uiSizePt, fontDPI)
	if err != nil {
		return nil, fmt.Errorf("load ui font: %w", err)
	}
	iconFace, err := customfont.LoadIconFace(iconSizePt, fontDPI)
	if err != nil {
		return nil, fmt.Errorf("load icon font: %w", err)
	}

	iconMetrics := iconFace.Metrics()
	iconAdv, ok := iconFace.GlyphAdvance('\ue627')
	if !ok {
		iconAdv = fixed.I(14)
	}

	fm := face.Metrics()
	ascent := int(fm.Ascent >> 6)
	descent := int(fm.Descent >> 6)

	adv, ok := face.GlyphAdvance('M')
	if !ok {
		adv = fixed.I(8)
	}

	uiM := uiFace.Metrics()

	return &metrics{
		face:       face,
		codeFace:   face,
		numFace:    face,
		titleFace:  face,
		uiFace:     uiFace,
		iconFace:   iconFace,
		charWidth:  int(adv >> 6),
		ascent:     ascent,
		descent:    descent,
		uiAscent:   int(uiM.Ascent >> 6),
		uiDescent:  int(uiM.Descent >> 6),
		iconAscent: int(iconMetrics.Ascent >> 6),
		iconW:      int(iconAdv >> 6),
	}, nil
}

func loadLightMetrics(bold bool) (*metrics, error) {
	load := func(data []byte, pt float64) (font.Face, error) {
		return customfont.LoadTTF(data, pt, fontDPI, lightHinting)
	}
	mono := func(data []byte, pt float64) (font.Face, error) {
		f, err := customfont.ZoneFace(data, pt, fontDPI, lightZoneGrid)
		if err != nil {
			return nil, err
		}
		return customfont.CellFace(f, lightCellW), nil
	}
	face, err := mono(customfont.DejaVuSansMonoTTF, lightCodePt)
	if err != nil {
		return nil, fmt.Errorf("load font: %w", err)
	}
	codeFace := face
	if bold {
		if codeFace, err = mono(customfont.DejaVuSansMonoBoldTTF, lightCodePt); err != nil {
			return nil, fmt.Errorf("load bold font: %w", err)
		}
	}
	numFace, err := mono(customfont.DejaVuSansMonoTTF, lightNumPt)
	if err != nil {
		return nil, fmt.Errorf("load ui font: %w", err)
	}
	// UI labels keep their natural outlines: zone snapping measured no gain.
	titleFace, err := load(customfont.InterBoldTTF, lightTitlePt)
	if err != nil {
		return nil, fmt.Errorf("load ui font: %w", err)
	}
	uiFace, err := load(customfont.InterRegularTTF, lightTabPt)
	if err != nil {
		return nil, fmt.Errorf("load ui font: %w", err)
	}
	iconFace, err := customfont.LoadIconFace(lightIconPt, fontDPI)
	if err != nil {
		return nil, fmt.Errorf("load icon font: %w", err)
	}
	eyeFace, err := customfont.LoadIconFace(lightEyePt, fontDPI)
	if err != nil {
		return nil, fmt.Errorf("load icon font: %w", err)
	}
	helpFace, err := load(customfont.InterBoldTTF, lightHelpPt)
	if err != nil {
		return nil, fmt.Errorf("load ui font: %w", err)
	}
	adv, ok := codeFace.GlyphAdvance('M')
	if !ok {
		adv = fixed.I(18)
	}
	iconAdv, ok := iconFace.GlyphAdvance('\ue627')
	if !ok {
		iconAdv = fixed.I(22)
	}
	fm, uiM, im := face.Metrics(), uiFace.Metrics(), iconFace.Metrics()
	return &metrics{
		face:       face,
		codeFace:   codeFace,
		numFace:    numFace,
		titleFace:  titleFace,
		uiFace:     uiFace,
		iconFace:   iconFace,
		eyeFace:    eyeFace,
		helpFace:   helpFace,
		charWidth:  int(adv >> 6),
		ascent:     int(fm.Ascent >> 6),
		descent:    int(fm.Descent >> 6),
		uiAscent:   int(uiM.Ascent >> 6),
		uiDescent:  int(uiM.Descent >> 6),
		iconAscent: int(im.Ascent >> 6),
		iconW:      int(iconAdv >> 6),
		inkGamma:   lightInkGamma,
	}, nil
}

// baselineIn returns the baseline that vertically centers one text line in a
// band of the given height starting at y.
func (m *metrics) baselineIn(y, height int) int {
	return y + (height-(m.ascent+m.descent))/2 + m.ascent
}

// baselineUI is baselineIn for the smaller chrome face.
func (m *metrics) baselineUI(y, height int) int {
	return y + (height-(m.uiAscent+m.uiDescent))/2 + m.uiAscent
}

type layout struct {
	*metrics
	digits      int
	lineHeight  int
	gutterW     int
	contentX    int
	codeX       int
	highlightR  int // right edge of the highlight band
	totalWidth  int
	neededWidth int // width before clamping to maxImageWidth
	totalHeight int
	codeAreaY   int // first pixel below the header
	padY        int
	gutterR     int
	g           geom
	scroll      *scrollState // non-nil when the view is capped
}

// geom is the mode-dependent spacing. Named fields on purpose: as a tuple of
// six ints, reordering it broke both call sites silently and still compiled.
type geom struct {
	lineMul float64
	// Burp light pins these to measured pixels; zero means derive them.
	lineH     int // row height
	codeBase  int // code baseline below the row top
	numBase   int // line-number baseline below the row top
	minDigits int // gutter is at least this many digits wide
	padR      int // right padding after the code (0 = paddingX)
	headerH   int
	padY      int
	gutterL   int
	gutterR   int
	codeGap   int
}

// geometryFor returns the spacing for a mode. HTTP panels imitate Burp Suite's
// message editor; snap imitates a VS Code tab.
func geometryFor(httpMode bool) geom { return geometryForTheme(httpMode, nil) }

// Burp light code-area geometry, in output pixels (2x a Burp 1x capture):
// 17px rows, baseline 13px down, the gutter rule 22px in, text from 27.5px.
const (
	lightPadR     = 28 // code end to the panel edge (holds the scroll lane); keeps the panel an even width so the response text lands on whole 1x pixels
	lightGutterSW = 2  // gutter rule width
	lightWrapCols = 71 // Burp's default editor width in the reference layout
)

const (
	lightHinting  = font.HintingNone // Burp/Java on macOS renders unhinted
	lightInkGamma = 0.8              // <1 darkens coverage toward Core Text's weight
)

func geometryForTheme(httpMode bool, t *Theme) geom {
	if httpMode && t != nil && t.BurpLayout {
		return geom{
			lineMul:   1,
			lineH:     34,
			codeBase:  26,
			numBase:   26,
			minDigits: 2,
			padR:      lightPadR,
			headerH:   lightHeaderH,
			padY:      4,
			gutterL:   4,
			gutterR:   8,
			codeGap:   10,
		}
	}
	if httpMode {
		return geom{
			lineMul: burpLineHeightMul,
			headerH: burpTitleH + burpTabRowH,
			padY:    burpPadY,
			gutterL: burpGutterPadL,
			gutterR: burpGutterPadR,
			codeGap: burpCodeGutterGap,
		}
	}
	return geom{
		lineMul: lineHeightMul,
		headerH: tabBarHeight,
		padY:    paddingY,
		gutterL: gutterPadL,
		gutterR: gutterPadR,
		codeGap: codeGutterGap,
	}
}

// gutterMetrics sizes the line-number gutter from the highest line number.
func gutterMetrics(lines []highlight.LineTokens, charW int, padL, padR int) (digits, gutterW int) {
	return gutterMetricsMin(lines, charW, padL, padR, 1)
}

func gutterMetricsMin(lines []highlight.LineTokens, charW int, padL, padR, minDigits int) (digits, gutterW int) {
	maxNum := 1
	for _, lt := range lines {
		if !lt.IsGap && lt.LineNum > maxNum {
			maxNum = lt.LineNum
		}
	}
	digits = max(len(strconv.Itoa(maxNum)), minDigits)
	return digits, padL + digits*charW + padR
}

func buildLayout(cfg *Config, m *metrics, vlines []visualLine, digits, gutterW int) *layout {
	g := geometryForTheme(cfg.HTTPMode, cfg.Theme)
	lineH := int(float64(m.ascent+m.descent) * g.lineMul)
	if g.lineH > 0 {
		lineH = g.lineH
	}

	light := cfg.HTTPMode && cfg.Theme != nil && cfg.Theme.BurpLayout
	contentX := gutterW + 1
	if light {
		contentX = gutterW + lightGutterSW
	}
	codeX := contentX + g.codeGap

	// Canvas width from the widest rendered row, measured with real glyph
	// advances so non-ASCII text is not under-counted.
	contentW := minContentChars * m.charWidth
	for _, vl := range vlines {
		if vl.isGap {
			continue
		}
		if w := vl.indent*m.charWidth + rowWidth(m.codeFace, vl.tokens); w > contentW {
			contentW = w
		}
	}

	// neededW is content only: it drives the auto-wrap decision, and wrapping
	// cannot make an over-long title fit. The header widens the canvas but must
	// not make short code look like it overflowed.
	padR := paddingX
	if g.padR > 0 {
		padR = g.padR
	}
	// Burp light panels are as wide as the wrap width, like an editor pane,
	// rather than shrink-wrapping the longest row.
	if cfg.Theme != nil && cfg.Theme.BurpLayout && cfg.Wrap && cfg.WrapWidth > 0 {
		contentW = max(contentW, cfg.WrapWidth*m.charWidth)
	}
	neededW := codeX + contentW + padR
	totalW := neededW
	if hw := headerWidth(m, cfg); hw > totalW {
		totalW = hw
	}
	if totalW > maxImageWidth {
		totalW = maxImageWidth
	}

	totalH := g.headerH + g.padY
	for _, vl := range vlines {
		if vl.isGap {
			totalH += gapRowHeight
		} else {
			totalH += lineH
		}
	}
	totalH += g.padY
	if light {
		totalH += lightFooterH
		if sc := cfg.scroll; sc != nil {
			totalH = g.headerH + g.padY + sc.visible*lineH + lightPeekH + lightFooterH
		}
	}
	if cfg.MinHeight > totalH {
		totalH = cfg.MinHeight
	}

	return &layout{
		metrics:     m,
		digits:      digits,
		lineHeight:  lineH,
		gutterW:     gutterW,
		contentX:    contentX,
		codeX:       codeX,
		highlightR:  totalW - highlightPadR,
		totalWidth:  totalW,
		neededWidth: neededW,
		totalHeight: totalH,
		codeAreaY:   g.headerH,
		padY:        g.padY,
		gutterR:     g.gutterR,
		g:           g,
		scroll:      cfg.scroll,
	}
}

// headerWidth is the minimum width needed to show the header without clipping —
// for an HTTP panel that is the tab row, which is wider than the title.
func headerWidth(m *metrics, cfg *Config) int {
	if cfg.HTTPMode && cfg.Theme != nil && cfg.Theme.BurpLayout {
		return lightHeaderWidth(m, cfg)
	}
	if cfg.HTTPMode {
		w := burpTitleX
		for _, tab := range cfg.httpTabs {
			w += textWidth(m.uiFace, tab.label) + 2*burpTabGap
		}
		// Keep the tabs and the control strip from touching on a narrow panel.
		return w + burpControlsGap + burpControlsWidth(m) + burpTitleX
	}
	return tabPad + m.iconW + tabIconGap + textWidth(m.face, cfg.Title) + tabPad
}

// MaxWrapCols is the widest wrap that still fits the canvas, assuming the
// narrowest possible gutter. Content with more line numbers needs a wider
// gutter and so wraps a little sooner; this is the bound a caller can check
// without having tokenized anything.
func MaxWrapCols(httpMode bool) int { return MaxWrapColsFor(httpMode, nil) }

// MaxWrapColsFor is MaxWrapCols for a given theme's fonts and spacing.
func MaxWrapColsFor(httpMode bool, t *Theme) int {
	m, err := loadMetricsFor(t, false)
	if err != nil {
		return defaultWrapCols
	}
	g := geometryForTheme(httpMode, t)
	gutterW := g.gutterL + m.charWidth + g.gutterR // one digit
	cols := (maxImageWidth - (gutterW + 1 + g.codeGap) - paddingX) / m.charWidth
	if cols < minContentChars {
		return minContentChars
	}
	return cols
}

// Render produces the final PNG image.
func Render(cfg *Config) (*image.NRGBA, error) {
	if cfg.Theme == nil {
		cfg.Theme = DefaultTheme
	}
	ui := cfg.Theme.UI

	m, err := loadMetricsFor(cfg.Theme, cfg.Bold)
	if err != nil {
		return nil, err
	}
	g := geometryForTheme(cfg.HTTPMode, cfg.Theme)
	digits, gutterW := gutterMetricsMin(cfg.Lines, m.charWidth, g.gutterL, g.gutterR, max(g.minDigits, 1))

	// Columns that still fit inside the canvas cap. Anything past this would be
	// cropped off the right edge, which is exactly the silent evidence loss the
	// auto-wrap below exists to prevent.
	fitCols := (maxImageWidth - (gutterW + 1 + g.codeGap) - paddingX) / m.charWidth
	if fitCols < minContentChars {
		fitCols = minContentChars
	}

	wrapCols := 0
	if cfg.Wrap {
		wrapCols = cfg.WrapWidth
		switch {
		case wrapCols <= 0:
			wrapCols = defaultWrapCols
		case wrapCols < minContentChars:
			// Defensive floor for library callers; the CLI reports it once.
			// One character per row turns a long body into a 38000px column.
			wrapCols = minContentChars
		}
		// Silent, like the minimum-column floor: this is content-aware and would
		// fire once per panel with a different number each time. MaxWrapCols
		// gives the CLI a content-independent bound to report once.
		if wrapCols > fitCols {
			wrapCols = fitCols
		}
	}

	wordWrap := cfg.HTTPMode && cfg.Theme.BurpLayout
	vlines := buildVisualLinesMode(cfg.Lines, wrapCols, wordWrap)
	vlines, cfg.scroll = viewport(vlines, cfg)
	l := buildLayout(cfg, m, vlines, digits, gutterW)

	// Content wider than the canvas would be silently cropped — that loses the
	// evidence. Wrap instead, and say so.
	// ponytail: one retry is enough; wrapped rows are bounded by wrapCols.
	if wrapCols == 0 && l.neededWidth > maxImageWidth {
		wrapCols = fitCols
		if !cfg.noWarn {
			fmt.Fprintf(os.Stderr,
				"pocing: content exceeds %dpx; wrapping at %d columns (use --wrap-width to override)\n",
				maxImageWidth, wrapCols)
		}
		vlines = buildVisualLinesMode(cfg.Lines, wrapCols, wordWrap)
		vlines, cfg.scroll = viewport(vlines, cfg)
		l = buildLayout(cfg, m, vlines, digits, gutterW)
	}

	// Refuse before allocating: a 130000px canvas is ~580MB and unviewable.
	if l.totalHeight > maxImageHeight {
		return nil, fmt.Errorf(
			"%q needs %dpx of height, past the %dpx browsers can decode — "+
				"trim it with --context or a line range",
			cfg.Title, l.totalHeight, maxImageHeight)
	}
	if l.totalHeight > warnImageHeight && !cfg.noWarn {
		fmt.Fprintf(os.Stderr,
			"pocing: warning: %q is %dpx tall; --context or a line range makes it readable\n",
			cfg.Title, l.totalHeight)
	}

	img := image.NewNRGBA(image.Rect(0, 0, l.totalWidth, l.totalHeight))

	// Pass 1: backgrounds. Burp draws no rule between gutter and text.
	fillRect(img, image.Rect(0, 0, l.totalWidth, l.totalHeight), ui.Bg)
	light := cfg.HTTPMode && cfg.Theme.BurpLayout
	if !cfg.HTTPMode {
		fillRect(img, image.Rect(l.gutterW, l.codeAreaY, l.gutterW+1, l.totalHeight), ui.GutterSep)
	}
	caretLine := cfg.CaretLine
	if caretLine == 0 {
		caretLine = 1
	}

	// Pass 2: header
	switch {
	case cfg.HTTPMode && cfg.Theme.BurpLayout:
		drawLightPanelHeader(img, l, cfg, ui)
	case cfg.HTTPMode:
		drawHTTPPanelHeader(img, l, cfg, ui)
	default:
		drawTabBar(img, l, cfg.Title, ui)
	}

	// Pass 3: visual lines
	var hlRuns []markRun
	y := l.codeAreaY + l.padY

	for _, vl := range vlines {
		if vl.isGap {
			fillRect(img, image.Rect(0, y, l.totalWidth, y+gapRowHeight), ui.GapBg)
			// LineNumFg, not GapFg: the count has to be readable in a report.
			drawString(img, l.face, gapLabel(vl.skipped), l.codeX,
				l.baselineIn(y, gapRowHeight), ui.LineNumFg)
			y += gapRowHeight
			continue
		}

		rowY := y
		rowBottom := y + l.lineHeight

		// Burp bands the caret's row; a fresh message has the caret on line 1.
		if vl.lineNum == caretLine && !vl.isContinuation && ui.CaretLineBg != (color.NRGBA{}) {
			right := l.totalWidth
			if light {
				right -= lightScrollW
			}
			fillRect(img, image.Rect(l.contentX, rowY, right, rowBottom), ui.CaretLineBg)
		}

		// A whole-line band plus a column box draws a border around the line and
		// another inside it, which reads as a rendering fault. When a line names
		// columns, the band keeps its wash for context and the precise marks own
		// every border on that line.
		precise := namesColumns(cfg.HighlightLines[vl.lineNum])
		for _, box := range l.markBoxes(cfg.HighlightLines, vl.lineNum, vl, wrapCols) {
			fillRect(img, image.Rect(box.x1, rowY, box.x2, rowBottom), ui.HighlightBg)
			if box.wholeLine && precise {
				continue
			}
			hlRuns = append(hlRuns, markRun{box.x1, box.x2, rowY, rowBottom})
		}

		// Line number on the first visual row only; continuation rows get a
		// subtle wrap arrow in the gutter instead.
		baseline := l.baselineIn(rowY, l.lineHeight)
		if l.g.codeBase > 0 {
			baseline = rowY + l.g.codeBase
		}
		switch {
		case !vl.isContinuation && l.g.numBase > 0:
			// Right-aligned against the gutter rule.
			s := strconv.Itoa(vl.lineNum)
			numX := l.gutterW - l.gutterR - textWidth(l.numFace, s)
			drawStringInk(img, l.numFace, s, numX, rowY+l.g.numBase, ui.LineNumFg, l.inkGamma)
		case !vl.isContinuation:
			lineNumStr := fmt.Sprintf("%*d", l.digits, vl.lineNum)
			numX := l.gutterW - l.gutterR - l.digits*l.charWidth
			drawString(img, l.face, lineNumStr, numX, baseline, ui.LineNumFg)
		case !cfg.HTTPMode:
			// Burp leaves the gutter blank on wrapped rows; the missing number
			// is the marker. Snap gets an explicit arrow.
			arrowX := l.gutterW - l.gutterR - l.charWidth
			drawString(img, l.face, "↩", arrowX, baseline, ui.GapFg)
		}

		// Code tokens
		tokenX := l.codeX + vl.indent*l.charWidth
		for _, tok := range vl.tokens {
			c := color.NRGBA{tok.R, tok.G, tok.B, 0xff}
			drawStringInk(img, l.codeFace, tok.Text, tokenX, baseline, c, l.inkGamma)
			tokenX += textWidth(l.codeFace, tok.Text)
		}

		y += l.lineHeight
	}

	// Asking for marks and getting none is how a useless screenshot reaches a
	// report. Judge it on the boxes actually drawn, not on whether the line
	// numbers exist: "7:900-999" names a real line and no real columns.
	if !cfg.noWarn && len(cfg.HighlightLines) > 0 && len(hlRuns) == 0 {
		fmt.Fprintf(os.Stderr,
			"pocing: warning: nothing was marked in %q; the requested lines or columns are not in the output\n",
			cfg.Title)
	}

	// Pass 4: one border per contiguous run of equally-wide marked rows
	for _, run := range mergeRuns(hlRuns) {
		drawRectOutline(img, image.Rect(run.x1, run.y1, run.x2, run.y2),
			ui.HighlightBorder, highlightBorderW)
	}

	if light {
		// The gutter rule, scroll lane and toolbar sit over the rows: a capped
		// view's last row is clipped by them exactly as in Burp.
		fillRect(img, image.Rect(l.gutterW, l.codeAreaY, l.gutterW+lightGutterSW, l.codeBottom()), ui.GutterSep)
		drawLightScrollbar(img, l, ui)
		drawLightFooter(img, l, ui, 0)
	}
	if cfg.HTTPMode {
		return img, nil // caller composes the HTTP panels
	}
	// Caption goes outside the window: it annotates the screenshot, it is not
	// part of the editor being screenshotted.
	return withCaption(applyChrome(img, ui), cfg.Caption, ui, m), nil
}

// markRun is one marked rectangle. Column marks make the horizontal extent
// vary between rows, so it travels with the run.
type markRun struct{ x1, x2, y1, y2 int }

// mergeRuns collapses vertically adjacent rows into single boxes, so that
// non-contiguous highlights get one box each instead of one box swallowing
// everything between them. Rows only merge when they span the same columns —
// otherwise a narrow column mark would be stretched to a neighbour's width.
func mergeRuns(rows []markRun) []markRun {
	if len(rows) == 0 {
		return nil
	}
	runs := []markRun{rows[0]}
	for _, r := range rows[1:] {
		last := &runs[len(runs)-1]
		if r.y1 == last.y2 && r.x1 == last.x1 && r.x2 == last.x2 {
			last.y2 = r.y2
			continue
		}
		runs = append(runs, r)
	}
	return runs
}

// markBox is the horizontal extent of one mark on one visual row.
type markBox struct {
	x1, x2    int
	wholeLine bool
}

// markBoxes maps a line's marks onto the visual row in vl. A wrapped line
// splits into rows of wrapCols characters, so a span may land on one row,
// straddle several, or miss this row entirely.
//
// It takes the whole map rather than one line's spans so that "absent" and
// "marked with no spans" stay distinguishable: with a bare slice the two look
// alike, and Marks.Has would call a line marked while nothing was drawn.
func (l *layout) markBoxes(marks highlight.Marks, line int, vl visualLine, wrapCols int) []markBox {
	spans, marked := marks[line]
	if vl.isGap || !marked {
		return nil
	}
	if len(spans) == 0 {
		// Marked, but no region named.
		return []markBox{{x1: l.contentX, x2: l.highlightR, wholeLine: true}}
	}

	rowStart, rowEnd := 1, 0 // 1-based columns covered by this visual row
	if wrapCols > 0 {
		// Rows carry their own start: word wrap makes them uneven.
		rowStart = vl.startCol
		if rowStart == 0 {
			rowStart = vl.rowIndex*wrapCols + 1
		}
		rowEnd = rowStart + vl.runeLen() - 1
	}

	var boxes []markBox
	for _, sp := range spans {
		if sp.WholeLine() {
			// Full-bleed band, like an editor's current-line highlight.
			boxes = append(boxes, markBox{x1: l.contentX, x2: l.highlightR, wholeLine: true})
			continue
		}
		start, end := sp.Start, sp.End
		if rowEnd > 0 { // wrapped: clip to the columns this row shows
			if end < rowStart || start > rowEnd {
				continue
			}
			start = max(start, rowStart) - rowStart + 1
			end = min(end, rowEnd) - rowStart + 1
		} else if vl.isContinuation {
			continue
		}
		// Never mark past the text: a box floating in blank space reads as a
		// rendering fault, not as evidence.
		if width := vl.runeLen(); end > width {
			end = width
		}
		if end < start {
			continue
		}
		// Breathe a little around the glyph cells: a border flush against the
		// first and last character reads as clipping, not as a marker.
		x0 := l.codeX + vl.indent*l.charWidth
		boxes = append(boxes, markBox{
			x1: x0 + (start-1)*l.charWidth - markPadX,
			x2: x0 + end*l.charWidth + markPadX,
		})
	}
	return boxes
}

// namesColumns reports whether any span on a line points at specific columns.
func namesColumns(spans []highlight.Span) bool {
	for _, sp := range spans {
		if !sp.WholeLine() {
			return true
		}
	}
	return false
}

func gapLabel(skipped int) string {
	switch {
	case skipped == 1:
		return "··· 1 line skipped"
	case skipped > 1:
		return fmt.Sprintf("··· %d lines skipped", skipped)
	}
	return "···"
}

func drawTabBar(img *image.NRGBA, l *layout, title string, ui UIColors) {
	// A title wider than the canvas would run off the edge mid-word, which
	// looks like a broken render rather than a long path.
	fixed := tabPad + l.iconW + tabIconGap + tabPad
	title = ellipsizeLeft(l.face, title, l.totalWidth-fixed)

	tabW := fixed + textWidth(l.face, title)
	if tabW > l.totalWidth {
		tabW = l.totalWidth
	}

	fillRect(img, image.Rect(0, 0, l.totalWidth, tabBarHeight), ui.TabBarBg)
	fillRect(img, image.Rect(0, 0, tabW, tabBarHeight), ui.ActiveTab)
	fillRect(img, image.Rect(0, 0, tabW, 2), ui.TabAccent)
	fillRect(img, image.Rect(tabW, 0, tabW+1, tabBarHeight), ui.TabBorder)
	fillRect(img, image.Rect(0, tabBarHeight-1, l.totalWidth, tabBarHeight), ui.TabBorder)

	ic := iconFor(title)
	drawString(img, l.iconFace, string(ic.glyph), tabPad,
		(tabBarHeight+l.iconAscent)/2, ic.color)
	drawString(img, l.face, title, tabPad+l.iconW+tabIconGap,
		l.baselineIn(0, tabBarHeight), ui.TabText)
}

// ellipsizeLeft shortens s from the left to fit maxWidth. The tail is kept
// because for a path the file name identifies the evidence; the directories
// above it do not.
func ellipsizeLeft(face font.Face, s string, maxWidth int) string {
	if maxWidth <= 0 || textWidth(face, s) <= maxWidth {
		return s
	}
	const ellipsis = "…"
	ellW := textWidth(face, ellipsis)
	r := []rune(s)
	for i := 1; i < len(r); i++ {
		if ellW+textWidth(face, string(r[i:])) <= maxWidth {
			return ellipsis + string(r[i:])
		}
	}
	return ellipsis
}

// applyChrome wraps a panel in a margin with rounded corners and a hairline
// border, so the output reads as a window instead of a raw crop.
func applyChrome(panel *image.NRGBA, ui UIColors) *image.NRGBA {
	pw, ph := panel.Bounds().Dx(), panel.Bounds().Dy()
	out := image.NewNRGBA(image.Rect(0, 0, pw+2*chromeMargin, ph+2*chromeMargin))
	fillRect(out, out.Bounds(), ui.ChromeBg)

	inner := image.Rect(chromeMargin, chromeMargin, chromeMargin+pw, chromeMargin+ph)
	draw.Draw(out, inner, panel, panel.Bounds().Min, draw.Src)
	roundCorners(out, inner, chromeRadius, ui.ChromeBg, ui.TabBorder)
	return out
}

// roundCorners masks the corners of r back to bg and strokes a rounded border.
func roundCorners(img *image.NRGBA, r image.Rectangle, radius int, bg, border color.NRGBA) {
	fillRect(img, image.Rect(r.Min.X+radius, r.Min.Y, r.Max.X-radius, r.Min.Y+chromeBorderW), border)
	fillRect(img, image.Rect(r.Min.X+radius, r.Max.Y-chromeBorderW, r.Max.X-radius, r.Max.Y), border)
	fillRect(img, image.Rect(r.Min.X, r.Min.Y+radius, r.Min.X+chromeBorderW, r.Max.Y-radius), border)
	fillRect(img, image.Rect(r.Max.X-chromeBorderW, r.Min.Y+radius, r.Max.X, r.Max.Y-radius), border)

	// Each corner box is radius×radius; its arc center sits at the box's inner
	// diagonal, on pixel centers (hence the 0.5 offsets).
	type corner struct {
		cx, cy float64
		x0, y0 int
	}
	left, right := float64(r.Min.X+radius)-0.5, float64(r.Max.X-radius)-0.5
	top, bottom := float64(r.Min.Y+radius)-0.5, float64(r.Max.Y-radius)-0.5
	for _, c := range []corner{
		{left, top, r.Min.X, r.Min.Y},
		{right, top, r.Max.X - radius, r.Min.Y},
		{left, bottom, r.Min.X, r.Max.Y - radius},
		{right, bottom, r.Max.X - radius, r.Max.Y - radius},
	} {
		for y := c.y0; y < c.y0+radius; y++ {
			for x := c.x0; x < c.x0+radius; x++ {
				dist := math.Hypot(float64(x)-c.cx, float64(y)-c.cy)
				outer := clamp01(float64(radius) - dist + 0.5)
				ring := outer - clamp01(float64(radius-chromeBorderW)-dist+0.5)
				px := mix(bg, img.NRGBAAt(x, y), outer)
				img.SetNRGBA(x, y, mix(px, border, ring))
			}
		}
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// mix blends b over a with weight t (t=0 → a, t=1 → b).
func mix(a, b color.NRGBA, t float64) color.NRGBA {
	lerp := func(x, y uint8) uint8 {
		return uint8(math.Round(float64(x)*(1-t) + float64(y)*t))
	}
	return color.NRGBA{lerp(a.R, b.R), lerp(a.G, b.G), lerp(a.B, b.B), lerp(a.A, b.A)}
}

// textWidth is the drawn advance of s, matching what drawString will emit.
func textWidth(face font.Face, s string) int {
	return int(font.MeasureString(face, s) >> 6)
}

func rowWidth(face font.Face, toks []highlight.Token) int {
	w := 0
	for _, tok := range toks {
		w += textWidth(face, tok.Text)
	}
	return w
}

func drawString(img *image.NRGBA, face font.Face, s string, x, baseline int, c color.NRGBA) {
	d := font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, baseline),
	}
	d.DrawString(s)
}

// drawStringInk is drawString with the glyph coverage raised to gamma, which
// darkens thin anti-aliased stems without changing advances or shapes.
func drawStringInk(img *image.NRGBA, face font.Face, s string, x, baseline int, c color.NRGBA, gamma float64) {
	if gamma <= 0 || gamma == 1 {
		drawString(img, face, s, x, baseline, c)
		return
	}
	b, adv := font.BoundString(face, s)
	r := image.Rect(x+b.Min.X.Floor()-1, baseline+b.Min.Y.Floor()-1, x+b.Max.X.Ceil()+1, baseline+b.Max.Y.Ceil()+1)
	if r.Empty() || adv == 0 {
		return
	}
	mask := image.NewAlpha(r)
	d := font.Drawer{Dst: mask, Src: image.Opaque, Face: face, Dot: fixed.P(x, baseline)}
	d.DrawString(s)
	var lut [256]uint8
	for i := range lut {
		lut[i] = uint8(math.Round(255 * math.Pow(float64(i)/255, gamma)))
	}
	for i, v := range mask.Pix {
		mask.Pix[i] = lut[v]
	}
	draw.DrawMask(img, r, image.NewUniform(c), image.Point{}, mask, r.Min, draw.Over)
}

func fillRect(img *image.NRGBA, r image.Rectangle, c color.NRGBA) {
	draw.Draw(img, r, image.NewUniform(c), image.Point{}, draw.Src)
}

func drawRectOutline(img *image.NRGBA, r image.Rectangle, c color.NRGBA, thickness int) {
	fillRect(img, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+thickness), c)
	fillRect(img, image.Rect(r.Min.X, r.Max.Y-thickness, r.Max.X, r.Max.Y), c)
	fillRect(img, image.Rect(r.Min.X, r.Min.Y, r.Min.X+thickness, r.Max.Y), c)
	fillRect(img, image.Rect(r.Max.X-thickness, r.Min.Y, r.Max.X, r.Max.Y), c)
}

const (
	captionGap     = 10           // separates the annotation from the screenshot
	captionPadX    = chromeMargin // lines the annotation up with the window edge
	captionPadY    = 11
	captionBarW    = 4
	captionMinPad  = 28 // space kept to the right of the text
	captionLineMul = 1.45
)

// withCaption appends a strip stating what the image proves. Evidence that
// travels into a report is read by someone who was not there: a red box says
// "look here", the caption says why. Drawn only when asked, so an unlabelled
// render still passes for a plain screenshot.
func withCaption(shot *image.NRGBA, text string, ui UIColors, m *metrics) *image.NRGBA {
	if text == "" {
		return shot
	}
	sw, sh := shot.Bounds().Dx(), shot.Bounds().Dy()
	textX := captionPadX + captionBarW + captionPadX

	// Wrap to the screenshot's width rather than widening the image: a long
	// finding description would otherwise stretch the PNG past the pane.
	avail := sw - textX - captionMinPad
	lines := wrapWords(m.face, text, avail)
	if len(lines) == 0 {
		return shot // whitespace only: no strip, no empty band
	}
	if len(lines) == 1 {
		if w := textX + textWidth(m.face, lines[0]) + captionMinPad; w > sw {
			sw = w
		}
	}
	stripH := captionPadY*2 + len(lines)*m.lineHeightFor(captionLineMul)

	out := image.NewNRGBA(image.Rect(0, 0, sw, sh+captionGap+stripH))
	// ChromeBg, never the app's own palette: this line is the analyst talking,
	// and evidence must not imply the tool under test printed it.
	fillRect(out, out.Bounds(), ui.ChromeBg)
	draw.Draw(out, shot.Bounds(), shot, shot.Bounds().Min, draw.Src)

	top := sh + captionGap
	fillRect(out, image.Rect(captionPadX, top+captionPadY, captionPadX+captionBarW, top+stripH-captionPadY),
		ui.HighlightBorder)

	lineH := m.lineHeightFor(captionLineMul)
	for i, line := range lines {
		y := top + captionPadY + i*lineH
		drawString(out, m.face, line, textX, m.baselineIn(y, lineH), ui.TabText)
	}
	return out
}

// lineHeightFor is the row height this face needs at the given multiplier.
func (m *metrics) lineHeightFor(mul float64) int {
	return int(float64(m.ascent+m.descent) * mul)
}

// wrapWords breaks prose at spaces to fit maxWidth. Unlike the code wrapper it
// never splits mid-word: a caption is read, not inspected character by character.
func wrapWords(face font.Face, text string, maxWidth int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	if maxWidth <= 0 {
		return []string{strings.Join(words, " ")}
	}

	var lines []string
	current := words[0]
	for _, w := range words[1:] {
		if textWidth(face, current+" "+w) <= maxWidth {
			current += " " + w
			continue
		}
		lines = append(lines, current)
		current = w
	}
	return append(lines, current)
}
