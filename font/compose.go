package font

import (
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/text/unicode/norm"
)

// DejaVu Sans Mono — the face Burp's message editor is drawn in — ships only
// part of Latin Extended Additional, so 46 of the 134 precomposed Vietnamese
// letters (ả ấ ầ ẩ ẫ ắ ẳ ẵ ẻ ế ề ể ễ ỉ ỏ ố ồ ổ ỗ ở ủ ử ỷ and their capitals)
// have no glyph and come out as .notdef boxes. Every one of them decomposes
// into a precomposed base the font does have plus one combining mark the font
// also has, so they are assembled from those two outlines rather than borrowed
// from a second typeface: a borrowed glyph would put a different design in the
// middle of a word, and the point of this renderer is to look like Burp.
//
// composed is the assembled outline of one such rune at the face's ppem.
type composed struct {
	segs []sfnt.Segment
	adv  fixed.Int26_6
	b    fixed.Rectangle26_6
}

// stackRise is how much of a tall base's extra height a second mark climbs.
// A mark is drawn for a base of x-height, so over â or Ă it has to rise; the
// font's own double-accented glyphs (ṍ, ǘ, Á) rise by 0.75–0.91 of the extra
// height rather than all of it, keeping stacked accents from drifting off the
// line. 0.85 lands within half a 2x pixel of each of them.
const stackRise = 0.85

// dotless maps a base whose tittle a mark above would collide with onto its
// dotless form, which is what a font's own í and ỉ are built from.
var dotless = map[rune]rune{'i': 'ı', 'j': 'ȷ'}

// composition builds r from a base glyph and one combining mark, or reports
// nil if the font cannot supply both. Results are cached, including failures.
func (z *zoneFace) composition(r rune) *composed {
	if c, ok := z.comp[r]; ok {
		return c
	}
	c := z.compose(r)
	if z.comp == nil {
		z.comp = map[rune]*composed{}
	}
	z.comp[r] = c
	return c
}

func (z *zoneFace) compose(r rune) *composed {
	d := []rune(norm.NFD.String(string(r)))
	if len(d) < 2 {
		return nil
	}
	// Peel off the outermost mark and recompose the rest: ấ is â plus acute,
	// not a plus circumflex plus acute, so the base keeps the font's own
	// design for the inner accent.
	mark := d[len(d)-1]
	base := []rune(norm.NFC.String(string(d[:len(d)-1])))
	if len(base) != 1 {
		return nil
	}

	mi := z.index(mark)
	if mi == 0 {
		return nil
	}
	mb, _, err := z.f.GlyphBounds(&z.buf, mi, z.ppem, font.HintingNone)
	if err != nil {
		return nil
	}
	// A mark whose ink sits entirely above the baseline is an accent; one
	// below (dot below, cedilla) stays where the font put it.
	above := mb.Max.Y < 0

	b := base[0]
	if above {
		if dl, ok := dotless[b]; ok {
			b = dl
		}
	}
	bi := z.index(b)
	if bi == 0 {
		return nil
	}
	bb, adv, err := z.f.GlyphBounds(&z.buf, bi, z.ppem, font.HintingNone)
	if err != nil {
		return nil
	}

	var rise fixed.Int26_6
	if above {
		if extra := z.refTop - (-bb.Min.Y); extra < 0 {
			rise = fixed.Int26_6(float64(-extra) * stackRise)
		}
	}

	// LoadGlyph hands back a slice into the shared buffer, so the base has to
	// be copied before the mark is loaded over it.
	bsegs, err := z.f.LoadGlyph(&z.buf, bi, z.ppem, nil)
	if err != nil {
		return nil
	}
	segs := append([]sfnt.Segment(nil), bsegs...)
	msegs, err := z.f.LoadGlyph(&z.buf, mi, z.ppem, nil)
	if err != nil {
		return nil
	}
	for _, s := range msegs {
		for i := range s.Args {
			s.Args[i].Y -= rise // sfnt Y grows downward
		}
		segs = append(segs, s)
	}

	mb.Min.Y -= rise
	mb.Max.Y -= rise
	return &composed{segs: segs, adv: adv, b: union(bb, mb)}
}

func union(a, b fixed.Rectangle26_6) fixed.Rectangle26_6 {
	if b.Empty() {
		return a
	}
	if a.Empty() {
		return b
	}
	return fixed.Rectangle26_6{
		Min: fixed.Point26_6{X: min(a.Min.X, b.Min.X), Y: min(a.Min.Y, b.Min.Y)},
		Max: fixed.Point26_6{X: max(a.Max.X, b.Max.X), Y: max(a.Max.Y, b.Max.Y)},
	}
}
