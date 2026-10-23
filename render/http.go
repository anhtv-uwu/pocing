package render

import (
	"fmt"
	"image"
	"image/draw"
	"os"
	"regexp"

	"github.com/anhtv-uwu/pocing/highlight"
)

const (
	httpDividerW    = 2
	httpDefaultCols = 90
)

// HTTPConfig holds parameters for HTTP request/response evidence rendering.
type HTTPConfig struct {
	Request    string
	Response   string
	RequestHL  highlight.Marks
	ResponseHL highlight.Marks
	// Literals to find and mark in whichever panel contains them.
	Mark []string
	// Caption states what the pane proves; empty leaves the pane bare.
	Caption string

	// Line specs (with gap sentinels) limiting each panel; nil renders all.
	RequestLines  []int
	ResponseLines []int
	// Context lines to keep either side of each mark. Resolved here rather than
	// by the caller because --mark's matches are only known after tokenizing.
	Context int

	WrapWidth int // columns per panel (0 = httpDefaultCols)

	// Dark selects Burp's dark theme; the default is Burp's light theme.
	// There is deliberately no free theme field. HTTP evidence always renders
	// in a Burp palette: a report reader recognises a proxy capture at a
	// glance, and a Dracula-coloured pane labelled Request/Response/Pretty/
	// Raw/Hex is a screenshot of an application that does not exist.
	Dark bool
	// Bold draws the light theme's message text in DejaVu Sans Mono Bold;
	// Burp itself uses the regular weight.
	Bold bool
	// RawHTML keeps markup bodies as sent instead of Burp's re-indented
	// Pretty view.
	RawHTML bool
	// MaxRows caps each panel's visible rows; longer panels show a Burp-style
	// scrollbar. 0 = DefaultHTTPMaxRows, negative = no cap.
	MaxRows int
	// Caret lines for the current-line band in the request and response
	// (0 = line 1, negative = none).
	RequestCaret, ResponseCaret int
}

// themeFor is the Burp palette an HTTP render uses.
func (cfg *HTTPConfig) themeFor() *Theme {
	if cfg.Dark {
		return BurpTheme
	}
	return BurpLightTheme
}

func tokenizeHTTP(src, filename string, theme *Theme, rawHTML bool) ([]highlight.LineTokens, error) {
	if theme.HTTPSyntax != nil {
		return highlight.TokenizeBurpHTTPWith(src, *theme.HTTPSyntax, highlight.BurpOptions{RawHTML: rawHTML, KeepFinalLine: theme.BurpLayout}), nil
	}
	return highlight.Tokenize(src, "http", filename, theme.Chroma)
}

// RenderHTTP renders a side-by-side HTTP request/response evidence image.
func RenderHTTP(cfg *HTTPConfig) (*image.NRGBA, error) {
	theme := cfg.themeFor()

	wrapCols := cfg.WrapWidth
	if wrapCols <= 0 {
		wrapCols = httpDefaultCols
		if theme.BurpLayout {
			wrapCols = lightWrapCols
		}
	}
	maxRows := cfg.MaxRows
	switch {
	case maxRows == 0:
		maxRows = DefaultHTTPMaxRows
	case maxRows < 0:
		maxRows = 0
	}

	reqLines, err := tokenizeHTTP(cfg.Request, "request.http", theme, cfg.RawHTML)
	if err != nil {
		return nil, fmt.Errorf("tokenize request: %w", err)
	}
	resLines, err := tokenizeHTTP(cfg.Response, "response.http", theme, cfg.RawHTML)
	if err != nil {
		return nil, fmt.Errorf("tokenize response: %w", err)
	}

	// Literals are located in the full capture, before any trimming: --context
	// has to be able to center the view on a match.
	reqHL, resHL := cfg.RequestHL, cfg.ResponseHL
	for _, needle := range cfg.Mark {
		inReq := highlight.FindLiteral(reqLines, needle)
		inRes := highlight.FindLiteral(resLines, needle)
		if inReq == nil && inRes == nil {
			fmt.Fprintf(os.Stderr, "taker: warning: --mark %q is not in the capture\n", needle)
			continue
		}
		reqHL = reqHL.Merge(inReq)
		resHL = resHL.Merge(inRes)
	}

	// A 3000-line body is taller than any viewer can open, so trim to what
	// matters. Gap bands report what was left out.
	reqSpec, resSpec := cfg.RequestLines, cfg.ResponseLines
	if s := highlight.ContextSpec(reqHL, cfg.Context); s != nil {
		reqSpec = s
	}
	if s := highlight.ContextSpec(resHL, cfg.Context); s != nil {
		resSpec = s
	}
	reqLines = highlight.FilterLines(reqLines, reqSpec)
	resLines = highlight.FilterLines(resLines, resSpec)
	if len(reqLines) == 0 {
		return nil, fmt.Errorf("--nr matched no request line")
	}
	if len(resLines) == 0 {
		return nil, fmt.Errorf("--ns matched no response line")
	}

	// Tabs read the whole capture, not the trimmed view: in Burp an editor tab
	// reflects the message it holds, not how far you scrolled.
	req := &Config{
		Lines: reqLines, Title: "Request", HighlightLines: reqHL,
		Theme: theme, Wrap: true, WrapWidth: wrapCols, HTTPMode: true,
		Bold: cfg.Bold, MaxRows: maxRows, CaretLine: cfg.RequestCaret,
		httpTabs: burpTabs("Request", cfg.Request),
	}
	res := &Config{
		Lines: resLines, Title: "Response", HighlightLines: resHL,
		Theme: theme, Wrap: true, WrapWidth: wrapCols, HTTPMode: true,
		Bold: cfg.Bold, MaxRows: maxRows, CaretLine: cfg.ResponseCaret,
		httpTabs: burpTabs("Response", cfg.Response),
	}

	reqImg, resImg, err := renderPair(req, res)
	if err != nil {
		return nil, err
	}
	// No window chrome here: a Burp screenshot is a flush pane, and evidence
	// that looks like it came straight out of Burp reads as more credible.
	var pane *image.NRGBA
	if theme.BurpLayout {
		pane = composeLightHTTP(reqImg, resImg, theme.UI)
	} else {
		pane = composeHTTP(reqImg, resImg, theme.UI)
	}
	if cfg.Caption == "" {
		return pane, nil
	}
	m, err := loadMetricsFor(theme, cfg.Bold)
	if err != nil {
		return nil, err
	}
	return withCaption(pane, cfg.Caption, theme.UI, m), nil
}

// renderPair renders both panels at the same height, so the shorter one's
// gutter and separator run the full length instead of stopping mid-image.
func renderPair(a, b *Config) (*image.NRGBA, *image.NRGBA, error) {
	imgA, err := Render(a)
	if err != nil {
		return nil, nil, fmt.Errorf("render %s: %w", a.Title, err)
	}
	imgB, err := Render(b)
	if err != nil {
		return nil, nil, fmt.Errorf("render %s: %w", b.Title, err)
	}

	h := max(imgA.Bounds().Dy(), imgB.Bounds().Dy())
	if imgA.Bounds().Dy() < h {
		a.MinHeight, a.noWarn = h, true
		if imgA, err = Render(a); err != nil {
			return nil, nil, fmt.Errorf("render %s: %w", a.Title, err)
		}
	}
	if imgB.Bounds().Dy() < h {
		b.MinHeight, b.noWarn = h, true
		if imgB, err = Render(b); err != nil {
			return nil, nil, fmt.Errorf("render %s: %w", b.Title, err)
		}
	}
	return imgA, imgB, nil
}

// composeHTTP places two equal-height panels side by side with a divider.
func composeHTTP(left, right *image.NRGBA, ui UIColors) *image.NRGBA {
	lw, rw := left.Bounds().Dx(), right.Bounds().Dx()
	h := max(left.Bounds().Dy(), right.Bounds().Dy())

	out := image.NewNRGBA(image.Rect(0, 0, lw+httpDividerW+rw, h))
	fillRect(out, out.Bounds(), ui.Bg)

	draw.Draw(out, image.Rect(0, 0, lw, h), left, left.Bounds().Min, draw.Src)
	fillRect(out, image.Rect(lw, 0, lw+httpDividerW, h), ui.GutterSep)
	draw.Draw(out, image.Rect(lw+httpDividerW, 0, lw+httpDividerW+rw, h), right, right.Bounds().Min, draw.Src)

	// Hairline around the pane, like Burp's split container.
	drawRectOutline(out, out.Bounds(), ui.TabBorder, 1)

	return out
}

// Burp light's split is a single 1px (2x: 2px) rule in the border grey.
const lightDividerW = 2

// composeLightHTTP places two equal-height light panels side by side.
func composeLightHTTP(left, right *image.NRGBA, ui UIColors) *image.NRGBA {
	lw, rw := left.Bounds().Dx(), right.Bounds().Dx()
	h := max(left.Bounds().Dy(), right.Bounds().Dy())

	out := image.NewNRGBA(image.Rect(0, 0, lw+lightDividerW+rw, h))
	fillRect(out, out.Bounds(), ui.Bg)
	draw.Draw(out, image.Rect(0, 0, lw, h), left, left.Bounds().Min, draw.Src)
	fillRect(out, image.Rect(lw, 0, lw+lightDividerW, h), ui.TabBorder)
	draw.Draw(out, image.Rect(lw+lightDividerW, 0, lw+lightDividerW+rw, h), right, right.Bounds().Min, draw.Src)
	return out
}

const burpTabGap = 15

type burpTab struct {
	label    string
	disabled bool // Burp dims a view that cannot render the current body
}

// jwtRe matches a compact JWS: two base64url segments that both decode to a
// JSON object, so both start with the "eyJ" that encodes `{"`.
var jwtRe = regexp.MustCompile(`eyJ[A-Za-z0-9_-]{8,}\.eyJ[A-Za-z0-9_-]{8,}`)

// htmlRe recognises markup Burp's Render view can display. The opening tag
// needs a delimiter after it, or prose like "not <html-ish" counts as markup.
var htmlRe = regexp.MustCompile(`(?i)content-type:[ \t]*text/html|<!doctype[ \t]+html|<html[ \t\r\n>]`)

// isHTML reports whether a response body is markup Burp could render.
func isHTML(message string) bool { return htmlRe.MatchString(message) }

// burpTabs is the view strip Burp shows above a message editor. Burp adds an
// editor tab per detected payload type, so a request bearing a JWT gets one.
func burpTabs(panel, message string) []burpTab {
	tabs := []burpTab{{label: "Pretty"}, {label: "Raw"}, {label: "Hex"}}
	if panel == "Response" {
		// Render only works on markup, so Burp greys it out for anything else.
		return append(tabs, burpTab{label: "Render", disabled: !isHTML(message)})
	}
	if jwtRe.MatchString(message) {
		tabs = append(tabs, burpTab{label: "JSON Web Token"})
	}
	return tabs
}

// drawHTTPPanelHeader draws Burp Suite's message-editor header: a title strip
// over a row of view tabs, with the active tab underlined in the accent color.
func drawHTTPPanelHeader(img *image.NRGBA, l *layout, cfg *Config, ui UIColors) {
	// Title strip, closed by a hairline so it reads as its own band. The title
	// takes the larger code face: in Burp it outranks the tab labels.
	fillRect(img, image.Rect(0, 0, l.totalWidth, burpTitleH), ui.TabBarBg)
	fillRect(img, image.Rect(0, burpTitleH-1, l.totalWidth, burpTitleH), ui.TabBorder)
	drawString(img, l.face, cfg.Title, burpTitleX, l.baselineIn(0, burpTitleH-1), ui.TabText)

	// Tab row
	tabTop := burpTitleH
	tabBottom := tabTop + burpTabRowH
	fillRect(img, image.Rect(0, tabTop, l.totalWidth, tabBottom), ui.ActiveTab)
	fillRect(img, image.Rect(0, tabBottom-1, l.totalWidth, tabBottom), ui.TabBorder)

	// Sit the labels above the baseline of the row so the active tab's
	// underline is clearly a tab indicator and not a text decoration.
	baseline := l.baselineUI(tabTop, burpTabRowH-burpTabUnderlineH-7)
	x := burpTitleX
	for i, tab := range cfg.httpTabs {
		w := textWidth(l.uiFace, tab.label)
		fg := ui.LineNumFg
		if tab.disabled {
			fg = ui.GapFg
		}
		if i == 0 { // Pretty is the active view
			fg = ui.TabText
			fillRect(img, image.Rect(x, tabBottom-burpTabUnderlineH, x+w+2*burpTabGap, tabBottom),
				ui.TabAccent)
		}
		drawString(img, l.uiFace, tab.label, x+burpTabGap, baseline, fg)
		x += w + 2*burpTabGap
	}

	// Burp's right-hand controls, right-aligned: hide-headers, pretty-print
	// (lit when active), show-non-printing, and the editor menu.
	ix := l.totalWidth - burpTitleX - burpControlsWidth(l.metrics)
	iconBaseline := tabTop + (burpTabRowH+l.iconAscent)/2 - 1
	textBaseline := l.baselineUI(tabTop, burpTabRowH)
	for _, c := range burpControls {
		if c.text != "" {
			drawString(img, l.uiFace, c.text, ix, textBaseline, ui.LineNumFg)
			ix += textWidth(l.uiFace, c.text) + burpTabGap
			continue
		}
		fg := ui.LineNumFg
		if c.active {
			// Bright, not accent-colored: in evidence the only loud thing
			// should be the highlight box marking the finding.
			fg = ui.TabText
		}
		drawString(img, l.iconFace, string(c.glyph), ix, iconBaseline, fg)
		ix += l.iconW + burpTabGap
	}
}

// burpControl is one item in the editor's right-hand control strip. Burp mixes
// glyphs with a literal "\n" label, so an item is either a glyph or text.
type burpControl struct {
	glyph  rune
	text   string
	active bool
}

var burpControls = []burpControl{
	{glyph: '\ue668'},               // hide headers (eye with slash)
	{glyph: '\uf039', active: true}, // pretty-print, shown as the active toggle
	{text: `\n`},                    // show non-printing characters
	{glyph: '\uf0c9'},               // editor menu
}

func burpControlsWidth(m *metrics) int {
	w := 0
	for _, c := range burpControls {
		if c.text != "" {
			w += textWidth(m.uiFace, c.text) + burpTabGap
			continue
		}
		w += m.iconW + burpTabGap
	}
	return w
}
