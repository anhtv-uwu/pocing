package font

import (
	_ "embed"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// SymbolsTTF is exported so tests can check glyph coverage with sfnt —
// font.Face.GlyphAdvance reports ok for a missing rune (it falls back to
// .notdef), so it cannot be used to verify a codepoint exists.
//
//go:embed SymbolsNerdFontMono-Regular.ttf
var SymbolsTTF []byte

// LoadIconFace loads the Symbols Nerd Font Mono face for rendering file-type icons.
func LoadIconFace(sizePt, dpi float64) (font.Face, error) {
	f, err := opentype.Parse(SymbolsTTF)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    sizePt,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
}
