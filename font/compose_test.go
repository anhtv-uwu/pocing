package font

import (
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// vietnamese is every precomposed letter Vietnamese writes with: the tone-mark
// block at U+1EA0 plus the letters it shares with Western European Latin.
func vietnamese() []rune {
	rs := make([]rune, 0, 160)
	for r := rune(0x1EA0); r <= 0x1EF9; r++ {
		rs = append(rs, r)
	}
	for _, r := range "ăâđêôơưĂÂĐÊÔƠƯáàãạéèẽíìĩóòõúùũýỳỹÁÀÃẠÉÈẼÍÌĨÓÒÕÚÙŨÝỲỸ" {
		rs = append(rs, r)
	}
	return rs
}

func lightFace(t *testing.T, ttf []byte) *zoneFace {
	t.Helper()
	f, err := ZoneFace(ttf, 13, 144, 2)
	if err != nil {
		t.Fatalf("ZoneFace: %v", err)
	}
	z, ok := f.(*zoneFace)
	if !ok {
		t.Fatalf("ZoneFace returned %T, want *zoneFace", f)
	}
	return z
}

// DejaVu Sans Mono ships only part of Latin Extended Additional: 46 Vietnamese
// letters have no glyph of their own and used to render as .notdef boxes. Every
// one of them must now resolve, in both weights — this is the regression the
// whole of compose.go exists for.
func TestVietnameseResolves(t *testing.T) {
	for _, w := range []struct {
		name string
		ttf  []byte
	}{{"regular", DejaVuSansMonoTTF}, {"bold", DejaVuSansMonoBoldTTF}} {
		t.Run(w.name, func(t *testing.T) {
			z := lightFace(t, w.ttf)
			for _, r := range vietnamese() {
				if _, ok := z.GlyphAdvance(r); !ok {
					t.Errorf("GlyphAdvance(%q U+%04X) not ok", r, r)
					continue
				}
				b, _, ok := z.GlyphBounds(r)
				if !ok {
					t.Errorf("GlyphBounds(%q U+%04X) not ok", r, r)
					continue
				}
				if b.Empty() {
					t.Errorf("%q U+%04X has no ink", r, r)
				}
			}
		})
	}
}

// A composed glyph is still one monospace cell. If an assembled letter advanced
// differently from the rest of the font, every column after it on the row —
// and so every highlight box — would slide.
func TestVietnameseKeepsTheCell(t *testing.T) {
	z := lightFace(t, DejaVuSansMonoTTF)
	want, ok := z.GlyphAdvance('M')
	if !ok {
		t.Fatal("no advance for M")
	}
	for _, r := range vietnamese() {
		got, ok := z.GlyphAdvance(r)
		if !ok {
			continue // reported by TestVietnameseResolves
		}
		if got != want {
			t.Errorf("advance(%q U+%04X) = %v, want %v", r, r, got, want)
		}
	}
}

// The mark has to land on top of the base, not inside it: a composed letter is
// taller than the base it was built from. sfnt Y grows downward, so ink further
// above the baseline is a smaller (more negative) Min.Y.
func TestComposedMarksSitAboveTheBase(t *testing.T) {
	z := lightFace(t, DejaVuSansMonoTTF)
	for _, c := range []struct{ composed, base rune }{
		{'ả', 'a'}, {'ế', 'ê'}, {'ồ', 'ô'}, {'ắ', 'ă'}, {'ẩ', 'â'},
		{'ử', 'ư'}, {'ỉ', 'i'}, {'Ẳ', 'Ă'}, {'Ố', 'Ô'}, {'Ỷ', 'Y'},
	} {
		cb, _, ok := z.GlyphBounds(c.composed)
		if !ok {
			t.Errorf("%q did not resolve", c.composed)
			continue
		}
		bb, _, ok := z.GlyphBounds(c.base)
		if !ok {
			t.Fatalf("base %q missing from the font", c.base)
		}
		if cb.Min.Y >= bb.Min.Y {
			t.Errorf("%q ink top %v is not above base %q top %v", c.composed, cb.Min.Y, c.base, bb.Min.Y)
		}
	}
	// And it must actually be drawn, not merely measured.
	dr, mask, _, _, ok := z.Glyph(fixed.P(0, 20), 'ế')
	if !ok || mask == nil || dr.Empty() {
		t.Errorf("Glyph('ế') = %v, mask %v, ok %v; want a non-empty raster", dr, mask != nil, ok)
	}
}

// stackRise is a guess at how a type designer raises a second accent. Check it
// against the designer's own answer, on the letters this font does carry: an
// accent over a plain base (á), over a capital (Á) and over another accent (ṍ).
// Within a 2x pixel of the real glyph is as close as a rule of thumb gets.
func TestStackRiseMatchesTheFontsOwnGlyphs(t *testing.T) {
	z := lightFace(t, DejaVuSansMonoTTF)
	const tol = fixed.Int26_6(64) // one pixel at 2x
	for _, r := range []rune{'á', 'à', 'ã', 'é', 'ó', 'ú', 'Á', 'Ã', 'ṍ', 'ṹ'} {
		x := z.index(r)
		if x == 0 {
			t.Fatalf("%q is missing from the font, so it cannot be a reference", r)
		}
		real, _, err := z.f.GlyphBounds(&z.buf, x, z.ppem, font.HintingNone)
		if err != nil {
			t.Fatal(err)
		}
		c := z.compose(r)
		if c == nil {
			t.Errorf("compose(%q) = nil; the rule cannot build a glyph the font has", r)
			continue
		}
		if d := c.b.Min.Y - real.Min.Y; d < -tol || d > tol {
			t.Errorf("%q composed ink top is %v off the font's own (%v vs %v)", r, d, c.b.Min.Y, real.Min.Y)
		}
	}
}

// A rune no font in the binary can draw still has to hold its column. A zero
// advance would pull the rest of the row left while the layout kept counting
// the character, silently moving every highlight box on that line.
func TestUnknownRuneKeepsItsAdvance(t *testing.T) {
	z := lightFace(t, DejaVuSansMonoTTF)
	for _, r := range []rune{0x2E9F, 0xE000, 0x10FFFD} {
		adv, ok := z.GlyphAdvance(r)
		if ok {
			t.Errorf("GlyphAdvance(U+%04X) reported a glyph", r)
		}
		if adv == 0 {
			t.Errorf("GlyphAdvance(U+%04X) = 0; an unknown rune must still take a cell", r)
		}
	}
}
