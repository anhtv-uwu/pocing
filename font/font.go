package font

import (
	_ "embed"
	"fmt"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
)

//go:embed JetBrainsMono-Regular.ttf
var JetBrainsMonoTTF []byte

// LoadFace parses the embedded JetBrains Mono TTF and returns a font.Face.
// DPI should be 144 for 2x (retina-quality) output.
func LoadFace(sizePt, dpi float64) (font.Face, error) {
	f, err := opentype.Parse(JetBrainsMonoTTF)
	if err != nil {
		return fallbackFace(sizePt, dpi)
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    sizePt,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return fallbackFace(sizePt, dpi)
	}
	return face, nil
}

func fallbackFace(sizePt, dpi float64) (font.Face, error) {
	f, err := opentype.Parse(gomono.TTF)
	if err != nil {
		return nil, fmt.Errorf("load fallback font: %w", err)
	}
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    sizePt,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
}
