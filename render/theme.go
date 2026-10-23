package render

import (
	"image/color"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/tachote/taker/highlight"
)

// UIColors holds all non-syntax UI element colors for a theme.
type UIColors struct {
	Bg              color.NRGBA
	ChromeBg        color.NRGBA // margin around the window; derived from Bg if unset
	TabBarBg        color.NRGBA
	ActiveTab       color.NRGBA
	TabBorder       color.NRGBA
	TabAccent       color.NRGBA
	TabText         color.NRGBA
	LineNumFg       color.NRGBA
	GutterSep       color.NRGBA
	GapFg           color.NRGBA
	GapBg           color.NRGBA
	HighlightBg     color.NRGBA
	HighlightBorder color.NRGBA

	// Burp layout only; the other themes leave these zero.
	CaretLineBg color.NRGBA // Burp's current-line band on line 1
	TabMutedFg  color.NRGBA // inactive (but enabled) tab labels
	ControlBg   color.NRGBA // rounded square behind the active toggle
	ControlFg   color.NRGBA // glyph on that square

	IconFg           color.NRGBA // toolbar glyphs
	ScrollTrack      color.NRGBA
	ScrollThumb      color.NRGBA
	ScrollThumbEdge  color.NRGBA
	FooterBtnBg      color.NRGBA
	FooterBtnBorder  color.NRGBA
	FieldBg          color.NRGBA
	FieldBorder      color.NRGBA
	FieldPlaceholder color.NRGBA
}

// Theme bundles a chroma syntax style with UI element colors.
type Theme struct {
	Name   string
	Chroma *chroma.Style
	UI     UIColors
	// HTTPOnly marks a palette built for one lexer. Offering it for code is a
	// trap: the style defines only the token types that lexer emits, so every
	// other language falls back to the plain text color and renders flat.
	HTTPOnly bool
	// BurpLayout selects the Burp Suite (macOS) panel layout used by both Burp
	// palettes: DejaVu Sans Mono text, Inter chrome, Burp's own spacing,
	// footer and scrollbar, and the HTTPSyntax colorizer instead of the
	// chroma style.
	BurpLayout bool
	HTTPSyntax *highlight.BurpPalette
}

func rgb(r, g, b uint8) color.NRGBA { return color.NRGBA{r, g, b, 0xff} }

// Standard red highlights work across all dark themes.
var redHighlight = struct{ bg, border color.NRGBA }{
	bg:     rgb(0x3d, 0x15, 0x15),
	border: rgb(0xf1, 0x4c, 0x4c),
}

// ── Chroma styles ────────────────────────────────────────────────────────────

// VSCodeDarkStyle is a custom chroma style matching VS Code Dark+.
var VSCodeDarkStyle = chroma.MustNewStyle("vscode-dark", chroma.StyleEntries{
	chroma.Background:            "bg:#1e1e1e #d4d4d4",
	chroma.Comment:               "#6a9955",
	chroma.CommentSingle:         "#6a9955",
	chroma.CommentMultiline:      "#6a9955",
	chroma.CommentHashbang:       "#6a9955",
	chroma.Keyword:               "#569cd6",
	chroma.KeywordDeclaration:    "#569cd6",
	chroma.KeywordType:           "#4ec9b0",
	chroma.KeywordNamespace:      "#c586c0",
	chroma.KeywordPseudo:         "#569cd6",
	chroma.KeywordReserved:       "#569cd6",
	chroma.KeywordConstant:       "#569cd6",
	chroma.NameFunction:          "#dcdcaa",
	chroma.NameFunctionMagic:     "#dcdcaa",
	chroma.NameClass:             "#4ec9b0",
	chroma.NameBuiltin:           "#4ec9b0",
	chroma.NameBuiltinPseudo:     "#9cdcfe",
	chroma.NameDecorator:         "#dcdcaa",
	chroma.NameAttribute:         "#9cdcfe",
	chroma.NameTag:               "#569cd6",
	chroma.NameVariable:          "#9cdcfe",
	chroma.NameVariableClass:     "#9cdcfe",
	chroma.NameVariableGlobal:    "#9cdcfe",
	chroma.NameVariableInstance:  "#9cdcfe",
	chroma.NameConstant:          "#4fc1ff",
	chroma.LiteralString:         "#ce9178",
	chroma.LiteralStringDouble:   "#ce9178",
	chroma.LiteralStringSingle:   "#ce9178",
	chroma.LiteralStringBacktick: "#ce9178",
	chroma.LiteralStringDoc:      "#ce9178",
	chroma.LiteralStringHeredoc:  "#ce9178",
	chroma.LiteralStringInterpol: "#ce9178",
	chroma.LiteralStringOther:    "#ce9178",
	chroma.LiteralStringRegex:    "#d16969",
	chroma.LiteralStringSymbol:   "#ce9178",
	chroma.LiteralNumber:         "#b5cea8",
	chroma.LiteralNumberFloat:    "#b5cea8",
	chroma.LiteralNumberHex:      "#b5cea8",
	chroma.LiteralNumberInteger:  "#b5cea8",
	chroma.LiteralNumberOct:      "#b5cea8",
	chroma.Operator:              "#d4d4d4",
	chroma.OperatorWord:          "#569cd6",
	chroma.Punctuation:           "#d4d4d4",
	chroma.GenericHeading:        "bold #569cd6",
	chroma.GenericSubheading:     "bold #4ec9b0",
	chroma.GenericEmph:           "italic",
	chroma.GenericStrong:         "bold",
})

// BurpStyle mirrors Burp Suite's Pretty view: the header block is almost
// monochrome (name barely brighter than value) and only the JSON body carries
// color. Token types are the ones chroma's http lexer actually emits —
// NameFunction for the method, Name/Literal for header name/value, and the
// delegated json lexer's NameTag for keys.
var BurpStyle = chroma.MustNewStyle("burp", chroma.StyleEntries{
	chroma.Background:           "bg:#1b1b1b #e4e4e4",
	chroma.Text:                 "#e4e4e4",
	chroma.NameFunction:         "#e8e8e8", // request method
	chroma.NameNamespace:        "#e4e4e4", // request path
	chroma.KeywordReserved:      "#e4e4e4", // "HTTP"
	chroma.NameException:        "#e4e4e4", // status text
	chroma.Operator:             "#e4e4e4",
	chroma.Punctuation:          "#e4e4e4",
	chroma.LiteralNumber:        "#e4e4e4", // version and status code
	chroma.Name:                 "#e4e4e4", // header name
	chroma.Literal:              "#c6c6c6", // header value
	chroma.NameTag:              "#cc8b3c", // json key
	chroma.LiteralStringDouble:  "#dcdcdc", // json string value
	chroma.LiteralString:        "#dcdcdc",
	chroma.LiteralNumberInteger: "#6897bb", // json number
	chroma.LiteralNumberFloat:   "#6897bb",
	chroma.KeywordConstant:      "#6897bb", // true / false / null
	chroma.Comment:              "#808080",
})

func builtinStyle(name string) *chroma.Style {
	s := styles.Get(name)
	if s == nil {
		s = styles.Fallback
	}
	return s
}

// ── Theme definitions ────────────────────────────────────────────────────────

var Themes = map[string]*Theme{
	// Default for `taker http`: evidence that reads as a Burp screenshot.
	// The UI fields are reused by the Burp panel header — TabBarBg is the title
	// strip, ActiveTab the tab row, TabAccent the active tab's underline.
	// Burp Suite's dark theme on macOS, measured pixel by pixel from a real
	// capture: same layout as the light theme, every colour re-measured.
	"burp": {
		Name:       "Burp Suite (dark)",
		HTTPOnly:   true,
		BurpLayout: true,
		Chroma:     BurpStyle, // unused for HTTP: HTTPSyntax colours the text
		HTTPSyntax: &BurpDarkSyntax,
		UI: UIColors{
			Bg:               rgb(0x26, 0x26, 0x26),
			ChromeBg:         rgb(0x2c, 0x2d, 0x2e),
			TabBarBg:         rgb(0x26, 0x26, 0x26),
			ActiveTab:        rgb(0x26, 0x26, 0x26),
			TabBorder:        rgb(0x48, 0x48, 0x4b),
			TabAccent:        rgb(0xd6, 0x56, 0x2a),
			TabText:          rgb(0xff, 0xff, 0xff),
			LineNumFg:        rgb(0x96, 0x96, 0x96),
			GutterSep:        rgb(0x4b, 0x4b, 0x4b),
			GapFg:            rgb(0x76, 0x76, 0x76),
			GapBg:            rgb(0x2c, 0x2d, 0x2e),
			HighlightBg:      rgb(0x5a, 0x24, 0x24),
			HighlightBorder:  rgb(0xf1, 0x4c, 0x4c),
			CaretLineBg:      rgb(0x2f, 0x31, 0x33),
			TabMutedFg:       rgb(0xbd, 0xc0, 0xc1),
			ControlBg:        rgb(0x22, 0x59, 0x92),
			ControlFg:        rgb(0xff, 0xff, 0xff),
			IconFg:           rgb(0xbd, 0xc0, 0xc1),
			ScrollTrack:      rgb(0x2c, 0x2d, 0x2e),
			ScrollThumb:      rgb(0x48, 0x48, 0x4b),
			ScrollThumbEdge:  rgb(0x46, 0x46, 0x48),
			FooterBtnBg:      rgb(0x3a, 0x3a, 0x3c),
			FooterBtnBorder:  rgb(0x48, 0x48, 0x4b),
			FieldBg:          rgb(0x2c, 0x2d, 0x2e),
			FieldBorder:      rgb(0x5f, 0x5f, 0x61),
			FieldPlaceholder: rgb(0x76, 0x76, 0x76),
		},
	},
	// Burp Suite's light theme, measured from PortSwigger's own documentation
	// screenshots (proxy history, Request/Response panels).
	"burp-light": {
		Name:       "Burp Suite (light)",
		HTTPOnly:   true,
		BurpLayout: true,
		Chroma:     BurpStyle, // unused for HTTP: HTTPSyntax colours the text
		HTTPSyntax: &BurpLightSyntax,
		UI: UIColors{
			Bg:               rgb(0xff, 0xff, 0xff),
			ChromeBg:         rgb(0xf2, 0xf2, 0xf2),
			TabBarBg:         rgb(0xff, 0xff, 0xff),
			ActiveTab:        rgb(0xff, 0xff, 0xff),
			TabBorder:        rgb(0xda, 0xda, 0xda),
			TabAccent:        rgb(0xd6, 0x56, 0x2a),
			TabText:          rgb(0x00, 0x00, 0x00),
			LineNumFg:        rgb(0x6d, 0x6d, 0x6d),
			GutterSep:        rgb(0xd8, 0xd8, 0xd8),
			GapFg:            rgb(0x96, 0x96, 0x98),
			GapBg:            rgb(0xf5, 0xf5, 0xf5),
			HighlightBg:      rgb(0xff, 0xe4, 0xe1),
			HighlightBorder:  rgb(0xe0, 0x1b, 0x1b),
			CaretLineBg:      rgb(0xee, 0xee, 0xee),
			TabMutedFg:       rgb(0x26, 0x26, 0x26),
			ControlBg:        rgb(0x22, 0x59, 0x92),
			ControlFg:        rgb(0xff, 0xff, 0xff),
			IconFg:           rgb(0x26, 0x26, 0x26),
			ScrollTrack:      rgb(0xfa, 0xfa, 0xfa),
			ScrollThumb:      rgb(0xda, 0xda, 0xda),
			ScrollThumbEdge:  rgb(0xe9, 0xe9, 0xe9),
			FooterBtnBg:      rgb(0xeb, 0xeb, 0xeb),
			FooterBtnBorder:  rgb(0xda, 0xda, 0xda),
			FieldBg:          rgb(0xfa, 0xfa, 0xfa),
			FieldBorder:      rgb(0xc2, 0xc2, 0xc2),
			FieldPlaceholder: rgb(0x33, 0x33, 0x33),
		},
	},
	"vscode-dark": {
		Name:   "VS Code Dark+",
		Chroma: VSCodeDarkStyle,
		UI: UIColors{
			Bg:              rgb(0x1e, 0x1e, 0x1e),
			TabBarBg:        rgb(0x25, 0x25, 0x26),
			ActiveTab:       rgb(0x1e, 0x1e, 0x1e),
			TabBorder:       rgb(0x3c, 0x3c, 0x3c),
			TabAccent:       rgb(0x00, 0x78, 0xd4),
			TabText:         rgb(0xcc, 0xcc, 0xcc),
			LineNumFg:       rgb(0x85, 0x85, 0x85),
			GutterSep:       rgb(0x3c, 0x3c, 0x3c),
			GapFg:           rgb(0x4e, 0x4e, 0x4e),
			GapBg:           rgb(0x25, 0x25, 0x26),
			HighlightBg:     redHighlight.bg,
			HighlightBorder: redHighlight.border,
		},
	},
	"monokai": {
		Name:   "Monokai",
		Chroma: builtinStyle("monokai"),
		UI: UIColors{
			Bg:              rgb(0x27, 0x28, 0x22),
			TabBarBg:        rgb(0x1e, 0x1f, 0x1c),
			ActiveTab:       rgb(0x27, 0x28, 0x22),
			TabBorder:       rgb(0x3e, 0x3d, 0x32),
			TabAccent:       rgb(0xf9, 0x26, 0x72),
			TabText:         rgb(0xf8, 0xf8, 0xf2),
			LineNumFg:       rgb(0x75, 0x71, 0x5e),
			GutterSep:       rgb(0x3e, 0x3d, 0x32),
			GapFg:           rgb(0x49, 0x48, 0x3e),
			GapBg:           rgb(0x1e, 0x1f, 0x1c),
			HighlightBg:     redHighlight.bg,
			HighlightBorder: redHighlight.border,
		},
	},
	"dracula": {
		Name:   "Dracula",
		Chroma: builtinStyle("dracula"),
		UI: UIColors{
			Bg:              rgb(0x28, 0x2a, 0x36),
			TabBarBg:        rgb(0x21, 0x22, 0x2c),
			ActiveTab:       rgb(0x28, 0x2a, 0x36),
			TabBorder:       rgb(0x44, 0x47, 0x5a),
			TabAccent:       rgb(0xbd, 0x93, 0xf9),
			TabText:         rgb(0xf8, 0xf8, 0xf2),
			LineNumFg:       rgb(0x62, 0x72, 0xa4),
			GutterSep:       rgb(0x44, 0x47, 0x5a),
			GapFg:           rgb(0x44, 0x47, 0x5a),
			GapBg:           rgb(0x21, 0x22, 0x2c),
			HighlightBg:     redHighlight.bg,
			HighlightBorder: redHighlight.border,
		},
	},
	"github-dark": {
		Name:   "GitHub Dark",
		Chroma: builtinStyle("github-dark"),
		UI: UIColors{
			Bg:              rgb(0x0d, 0x11, 0x17),
			TabBarBg:        rgb(0x16, 0x1b, 0x22),
			ActiveTab:       rgb(0x0d, 0x11, 0x17),
			TabBorder:       rgb(0x30, 0x36, 0x3d),
			TabAccent:       rgb(0xf7, 0x81, 0x66),
			TabText:         rgb(0xc9, 0xd1, 0xd9),
			LineNumFg:       rgb(0x6e, 0x76, 0x81),
			GutterSep:       rgb(0x30, 0x36, 0x3d),
			GapFg:           rgb(0x30, 0x36, 0x3d),
			GapBg:           rgb(0x16, 0x1b, 0x22),
			HighlightBg:     redHighlight.bg,
			HighlightBorder: redHighlight.border,
		},
	},
	"nord": {
		Name:   "Nord",
		Chroma: builtinStyle("nord"),
		UI: UIColors{
			Bg:              rgb(0x2e, 0x34, 0x40),
			TabBarBg:        rgb(0x24, 0x29, 0x33),
			ActiveTab:       rgb(0x2e, 0x34, 0x40),
			TabBorder:       rgb(0x3b, 0x42, 0x52),
			TabAccent:       rgb(0x88, 0xc0, 0xd0),
			TabText:         rgb(0xec, 0xef, 0xf4),
			LineNumFg:       rgb(0x4c, 0x56, 0x6a),
			GutterSep:       rgb(0x3b, 0x42, 0x52),
			GapFg:           rgb(0x4c, 0x56, 0x6a),
			GapBg:           rgb(0x24, 0x29, 0x33),
			HighlightBg:     redHighlight.bg,
			HighlightBorder: redHighlight.border,
		},
	},
	"catppuccin-mocha": {
		Name:   "Catppuccin Mocha",
		Chroma: builtinStyle("catppuccin-mocha"),
		UI: UIColors{
			Bg:              rgb(0x1e, 0x1e, 0x2e),
			TabBarBg:        rgb(0x18, 0x18, 0x25),
			ActiveTab:       rgb(0x1e, 0x1e, 0x2e),
			TabBorder:       rgb(0x31, 0x32, 0x44),
			TabAccent:       rgb(0xcb, 0xa6, 0xf7),
			TabText:         rgb(0xcd, 0xd6, 0xf4),
			LineNumFg:       rgb(0x6c, 0x70, 0x86),
			GutterSep:       rgb(0x31, 0x32, 0x44),
			GapFg:           rgb(0x45, 0x47, 0x5a),
			GapBg:           rgb(0x18, 0x18, 0x25),
			HighlightBg:     redHighlight.bg,
			HighlightBorder: redHighlight.border,
		},
	},
	"gruvbox": {
		Name:   "Gruvbox",
		Chroma: builtinStyle("gruvbox"),
		UI: UIColors{
			Bg:              rgb(0x28, 0x28, 0x28),
			TabBarBg:        rgb(0x1d, 0x20, 0x21),
			ActiveTab:       rgb(0x28, 0x28, 0x28),
			TabBorder:       rgb(0x3c, 0x38, 0x36),
			TabAccent:       rgb(0xd7, 0x99, 0x21),
			TabText:         rgb(0xeb, 0xdb, 0xb2),
			LineNumFg:       rgb(0x92, 0x83, 0x74),
			GutterSep:       rgb(0x3c, 0x38, 0x36),
			GapFg:           rgb(0x50, 0x49, 0x45),
			GapBg:           rgb(0x1d, 0x20, 0x21),
			HighlightBg:     redHighlight.bg,
			HighlightBorder: redHighlight.border,
		},
	},
	"one-dark": {
		Name:   "One Dark",
		Chroma: builtinStyle("onedark"),
		UI: UIColors{
			Bg:              rgb(0x28, 0x2c, 0x34),
			TabBarBg:        rgb(0x21, 0x25, 0x2b),
			ActiveTab:       rgb(0x28, 0x2c, 0x34),
			TabBorder:       rgb(0x3e, 0x44, 0x51),
			TabAccent:       rgb(0x61, 0xaf, 0xef),
			TabText:         rgb(0xab, 0xb2, 0xbf),
			LineNumFg:       rgb(0x5c, 0x63, 0x70),
			GutterSep:       rgb(0x3e, 0x44, 0x51),
			GapFg:           rgb(0x3e, 0x44, 0x51),
			GapBg:           rgb(0x21, 0x25, 0x2b),
			HighlightBg:     redHighlight.bg,
			HighlightBorder: redHighlight.border,
		},
	},
	"solarized-dark": {
		Name:   "Solarized Dark",
		Chroma: builtinStyle("solarized-dark"),
		UI: UIColors{
			Bg:              rgb(0x00, 0x2b, 0x36),
			TabBarBg:        rgb(0x07, 0x36, 0x42),
			ActiveTab:       rgb(0x00, 0x2b, 0x36),
			TabBorder:       rgb(0x09, 0x4e, 0x5b),
			TabAccent:       rgb(0x26, 0x8b, 0xd2),
			TabText:         rgb(0x83, 0x94, 0x96),
			LineNumFg:       rgb(0x58, 0x6e, 0x75),
			GutterSep:       rgb(0x09, 0x4e, 0x5b),
			GapFg:           rgb(0x58, 0x6e, 0x75),
			GapBg:           rgb(0x07, 0x36, 0x42),
			HighlightBg:     redHighlight.bg,
			HighlightBorder: redHighlight.border,
		},
	},
	"rose-pine": {
		Name:   "Rosé Pine",
		Chroma: builtinStyle("rose-pine"),
		UI: UIColors{
			Bg:              rgb(0x19, 0x17, 0x24),
			TabBarBg:        rgb(0x1f, 0x1d, 0x2e),
			ActiveTab:       rgb(0x19, 0x17, 0x24),
			TabBorder:       rgb(0x26, 0x23, 0x3a),
			TabAccent:       rgb(0xc4, 0xa7, 0xe7),
			TabText:         rgb(0xe0, 0xde, 0xf4),
			LineNumFg:       rgb(0x6e, 0x6a, 0x86),
			GutterSep:       rgb(0x26, 0x23, 0x3a),
			GapFg:           rgb(0x40, 0x3d, 0x52),
			GapBg:           rgb(0x1f, 0x1d, 0x2e),
			HighlightBg:     redHighlight.bg,
			HighlightBorder: redHighlight.border,
		},
	},
}

// DefaultTheme is used when no --theme flag is provided.
var DefaultTheme = Themes["vscode-dark"]

// BurpTheme is Burp's dark palette (`--theme dark`). HTTP captures always render in one of
// Burp's two palettes: evidence that claims to be a proxy capture has to look
// like one.
var BurpTheme = Themes["burp"]

// BurpLightTheme is the default for `taker http`.
var BurpLightTheme = Themes["burp-light"]

// BurpDarkSyntax is Burp's dark-theme message colouring, measured from a
// real macOS capture: request/status line and header names in pale blue,
// header values grey, HTML element names gold, attribute names periwinkle,
// values green with grey quotes. Parameter and JSON colours (absent from the
// capture) follow the same mapping the light palette uses.
var BurpDarkSyntax = highlight.BurpPalette{
	Text:        highlight.RGB{R: 0xc7, G: 0xc7, B: 0xc7},
	StartLine:   highlight.RGB{R: 0xcb, G: 0xe5, B: 0xf8},
	HeaderValue: highlight.RGB{R: 0xb1, G: 0xb1, B: 0xb1},
	HeaderName:  highlight.RGB{R: 0xcb, G: 0xe5, B: 0xf8},
	ParamName:   highlight.RGB{R: 0xb3, G: 0xc6, B: 0xff},
	ParamValue:  highlight.RGB{R: 0x9b, G: 0xbb, B: 0x51},
	JSONKey:     highlight.RGB{R: 0xcb, G: 0xe5, B: 0xf8},
	JSONString:  highlight.RGB{R: 0x9b, G: 0xbb, B: 0x51},
	JSONNumber:  highlight.RGB{R: 0xb3, G: 0xc6, B: 0xff},
	HTMLTag:     highlight.RGB{R: 0xe6, G: 0xb8, B: 0x58},
	HTMLAttr:    highlight.RGB{R: 0xb3, G: 0xc6, B: 0xff},
	HTMLEquals:  highlight.RGB{R: 0xcb, G: 0xe5, B: 0xf8},
	HTMLValue:   highlight.RGB{R: 0x9b, G: 0xbb, B: 0x51},
	HTMLQuote:   highlight.RGB{R: 0xc7, G: 0xc7, B: 0xc7},
}

// BurpLightSyntax is Burp's light-theme message colouring. Text, header,
// parameter and HTML colours are measured from a real Burp Suite light
// capture (compare/compare_v2.py); the JSON colours reuse the header and
// parameter colours (no reference capture with a JSON body).
var BurpLightSyntax = highlight.BurpPalette{
	Text:        highlight.RGB{R: 0x14, G: 0x14, B: 0x14},
	HeaderValue: highlight.RGB{R: 0x1d, G: 0x1d, B: 0x1d},
	HeaderName:  highlight.RGB{R: 0x00, G: 0x00, B: 0x6a},
	ParamName:   highlight.RGB{R: 0x00, G: 0x00, B: 0xb8},
	ParamValue:  highlight.RGB{R: 0x96, G: 0x11, B: 0x11},
	JSONKey:     highlight.RGB{R: 0x00, G: 0x00, B: 0x6a},
	JSONString:  highlight.RGB{R: 0x96, G: 0x11, B: 0x11},
	JSONNumber:  highlight.RGB{R: 0x00, G: 0x00, B: 0xb8},
	HTMLTag:     highlight.RGB{R: 0xa7, G: 0x00, B: 0xb8},
	HTMLAttr:    highlight.RGB{R: 0x00, G: 0x00, B: 0xb8},
	HTMLValue:   highlight.RGB{R: 0x96, G: 0x11, B: 0x11},
}

// Derive the chrome margin from each theme's background instead of hand-picking
// ten more colors. The hairline border does the separating work.
func init() {
	for _, t := range Themes {
		if t.UI.ChromeBg == (color.NRGBA{}) {
			t.UI.ChromeBg = mix(t.UI.Bg, rgb(0x00, 0x00, 0x00), 0.45)
		}
	}
}
