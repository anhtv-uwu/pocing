package font

import (
	_ "embed"
	"fmt"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// Faces for the Burp Suite light theme. Burp's message editor draws the
// message in a DejaVu-derived monospace (Menlo on macOS, DejaVu Sans Mono on
// Linux) and its chrome in the platform sans; Inter stands in for the latter.
//
// Sources (see LICENSE-DejaVu.txt and LICENSE-Inter.txt):
//   - DejaVuSansMono*.ttf: DejaVu Fonts 2.37, Debian package fonts-dejavu-mono.
//   - Inter-*.ttf: Inter 4.1, github.com/rsms/inter release Inter-4.1.zip,
//     extras/ttf/.

//go:embed DejaVuSansMono.ttf
var DejaVuSansMonoTTF []byte

//go:embed DejaVuSansMono-Bold.ttf
var DejaVuSansMonoBoldTTF []byte

//go:embed Inter-Regular.ttf
var InterRegularTTF []byte

//go:embed Inter-Bold.ttf
var InterBoldTTF []byte

// LoadTTF parses an embedded TrueType font at the given size. Unlike LoadFace
// it does not fall back: a broken embedded font is a build defect, and a
// silently substituted face would shift every measured coordinate.
//
// hinting is font.HintingNone for the Burp faces: macOS CoreText ignores
// TrueType hints and places outlines at their exact scaled positions, so
// grid-fitted stems read as heavier and blockier than Burp on a Mac.
func LoadTTF(data []byte, sizePt, dpi float64, hinting font.Hinting) (font.Face, error) {
	f, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse font: %w", err)
	}
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    sizePt,
		DPI:     dpi,
		Hinting: hinting,
	})
}
