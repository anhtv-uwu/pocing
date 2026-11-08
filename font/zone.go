package font

import (
	"fmt"
	"image"
	"image/draw"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

// zoneFace rasterizes an unhinted outline whose vertical alignment zones
// (baseline, x-height, cap height, ascender, descender) are snapped to a pixel
// grid. Burp's text, as a Mac draws it, has crisp top and bottom edges on
// capitals and lowercase while stems keep their natural, unhinted widths.
// x/image/font/opentype ignores TrueType hints entirely, so without this the
// cap line of DejaVu at 13px falls mid-pixel and every row of text gets a
// soft grey fringe that Burp does not have. Only Y is touched: advances, side
// bearings and stem widths stay exactly as designed.
type zoneFace struct {
	f     *sfnt.Font
	buf   sfnt.Buffer
	ppem  fixed.Int26_6
	knots []knot // ascending in design height (pixels, y up)
	m     font.Metrics
	rast  vector.Rasterizer
	mask  image.Alpha
	// refTop is the ink top of 'a': the base height a combining mark is
	// drawn for, and so the zero point for stacking one (see compose.go).
	// stackTop is the ceiling the font's own stacked accents stop at. Both
	// are zero when the font does not supply the glyphs to measure them, and
	// composition is then switched off rather than guessed at.
	refTop   fixed.Int26_6
	stackTop fixed.Int26_6
	comp     map[rune]*composed
}

// knot maps a natural height (px above the baseline) to its snapped height.
type knot struct{ from, to float64 }

// ZoneFace loads data at sizePt/dpi with its alignment zones rounded to
// multiples of gridPx. At fontDPI 144 (2x output) a grid of 2 lands every
// zone on a whole 1x pixel, the way Burp's 1x text is drawn.
func ZoneFace(data []byte, sizePt, dpi, gridPx float64) (font.Face, error) {
	f, err := sfnt.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse font: %w", err)
	}
	z := &zoneFace{f: f, ppem: fixed.Int26_6(math.Round(sizePt * dpi / 72 * 64))}
	if z.m, err = f.Metrics(&z.buf, z.ppem, font.HintingNone); err != nil {
		return nil, fmt.Errorf("font metrics: %w", err)
	}
	snap := func(v float64) float64 { return math.Round(v/gridPx) * gridPx }
	// Reference glyphs for each zone, as an autohinter would pick them.
	zones := []struct {
		r   rune
		top bool
	}{{'p', false}, {'x', true}, {'H', true}, {'d', true}}
	z.knots = []knot{{0, 0}}
	for _, zr := range zones {
		h, ok := z.edge(zr.r, zr.top)
		if !ok {
			continue
		}
		k := knot{h, snap(h)}
		last := z.knots[len(z.knots)-1]
		if h < 0 {
			z.knots = append([]knot{k}, z.knots...)
			continue
		}
		if h > last.from && k.to >= last.to {
			z.knots = append(z.knots, k)
		}
	}
	if h, ok := z.edge('a', true); ok {
		z.refTop = fixed.Int26_6(math.Round(h * 64))
	}
	// A type designer does not keep raising a second accent with the height of
	// its base: past a point they all park on one line. These are the font's
	// own doubly accented glyphs; the highest of them is that line.
	for _, r := range []rune{'ṍ', 'Ằ', 'Ǚ', 'Ǘ', 'Ṹ', 'ǘ', 'ằ'} {
		if h, ok := z.edge(r, true); ok {
			if t := fixed.Int26_6(math.Round(h * 64)); t > z.stackTop {
				z.stackTop = t
			}
		}
	}
	return z, nil
}

// index is r's glyph in the font, or 0 when the font has none.
func (z *zoneFace) index(r rune) sfnt.GlyphIndex {
	x, err := z.f.GlyphIndex(&z.buf, r)
	if err != nil {
		return 0
	}
	return x
}

// outline is r's contours at the face's ppem, assembled from a base glyph and
// a combining mark when the font has no glyph of its own for r.
func (z *zoneFace) outline(r rune) (segs []sfnt.Segment, b fixed.Rectangle26_6, adv fixed.Int26_6, ok bool) {
	if x := z.index(r); x != 0 {
		b, adv, err := z.f.GlyphBounds(&z.buf, x, z.ppem, font.HintingNone)
		if err != nil {
			return nil, b, 0, false
		}
		segs, err := z.f.LoadGlyph(&z.buf, x, z.ppem, nil)
		if err != nil {
			return nil, b, 0, false
		}
		return segs, b, adv, true
	}
	if c := z.composition(r); c != nil {
		return c.segs, c.b, c.adv, true
	}
	return nil, fixed.Rectangle26_6{}, 0, false
}

// edge is the top (or bottom) of r's flat ink, in px above the baseline.
func (z *zoneFace) edge(r rune, top bool) (float64, bool) {
	x, err := z.f.GlyphIndex(&z.buf, r)
	if err != nil || x == 0 {
		return 0, false
	}
	b, _, err := z.f.GlyphBounds(&z.buf, x, z.ppem, font.HintingNone)
	if err != nil {
		return 0, false
	}
	if top {
		return -float64(b.Min.Y) / 64, true
	}
	return -float64(b.Max.Y) / 64, true
}

// warp maps a natural height to its snapped height: piecewise linear between
// zone knots, and the outer segments shifted with their nearest knot.
func (z *zoneFace) warp(h float64) float64 {
	k := z.knots
	if h <= k[0].from {
		return h + k[0].to - k[0].from
	}
	for i := 1; i < len(k); i++ {
		if h <= k[i].from {
			t := (h - k[i-1].from) / (k[i].from - k[i-1].from)
			return k[i-1].to + t*(k[i].to-k[i-1].to)
		}
	}
	last := k[len(k)-1]
	return h + last.to - last.from
}

func (z *zoneFace) warpY(y fixed.Int26_6) float32 {
	return float32(-z.warp(-float64(y) / 64))
}

func (z *zoneFace) Close() error { return nil }

func (z *zoneFace) Metrics() font.Metrics { return z.m }

func (z *zoneFace) Kern(r0, r1 rune) fixed.Int26_6 {
	x0, _ := z.f.GlyphIndex(&z.buf, r0)
	x1, _ := z.f.GlyphIndex(&z.buf, r1)
	k, err := z.f.Kern(&z.buf, x0, x1, z.ppem, font.HintingNone)
	if err != nil {
		return 0
	}
	return k
}

func (z *zoneFace) GlyphAdvance(r rune) (fixed.Int26_6, bool) {
	if x := z.index(r); x != 0 {
		adv, err := z.f.GlyphAdvance(&z.buf, x, z.ppem, font.HintingNone)
		return adv, err == nil
	}
	if c := z.composition(r); c != nil {
		return c.adv, true
	}
	// No glyph anywhere: report .notdef's advance so the cell still holds its
	// width. A zero advance would collapse the column and slide the rest of
	// the row — and every highlight box on it — to the left.
	adv, err := z.f.GlyphAdvance(&z.buf, 0, z.ppem, font.HintingNone)
	if err != nil {
		return 0, false
	}
	return adv, false
}

func (z *zoneFace) GlyphBounds(r rune) (fixed.Rectangle26_6, fixed.Int26_6, bool) {
	_, b, adv, ok := z.outline(r)
	if !ok {
		x := sfnt.GlyphIndex(0)
		var err error
		if b, adv, err = z.f.GlyphBounds(&z.buf, x, z.ppem, font.HintingNone); err != nil {
			return fixed.Rectangle26_6{}, 0, false
		}
	}
	b.Min.Y = fixed.Int26_6(math.Floor(float64(z.warpY(b.Min.Y)) * 64))
	b.Max.Y = fixed.Int26_6(math.Ceil(float64(z.warpY(b.Max.Y)) * 64))
	return b, adv, ok
}

func (z *zoneFace) Glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	segs, _, adv, ok := z.outline(r)
	if !ok {
		// Draw .notdef, as any other face would, but keep its advance.
		var err error
		if segs, err = z.f.LoadGlyph(&z.buf, 0, z.ppem, nil); err != nil {
			return image.Rectangle{}, nil, image.Point{}, 0, false
		}
		if adv, err = z.f.GlyphAdvance(&z.buf, 0, z.ppem, font.HintingNone); err != nil {
			return image.Rectangle{}, nil, image.Point{}, 0, false
		}
	}

	type pt struct{ x, y float32 }
	ox, oy := float32(dot.X)/64, float32(dot.Y)/64
	conv := func(p fixed.Point26_6) pt { return pt{float32(p.X)/64 + ox, z.warpY(p.Y) + oy} }
	minX, minY := float32(math.Inf(1)), float32(math.Inf(1))
	maxX, maxY := float32(math.Inf(-1)), float32(math.Inf(-1))
	n := []int{1, 1, 2, 3}
	for _, s := range segs {
		for i := 0; i < n[s.Op]; i++ {
			p := conv(s.Args[i])
			minX, maxX = min(minX, p.x), max(maxX, p.x)
			minY, maxY = min(minY, p.y), max(maxY, p.y)
		}
	}
	if len(segs) == 0 {
		return image.Rectangle{}, &z.mask, image.Point{}, adv, ok
	}
	dr := image.Rect(int(math.Floor(float64(minX))), int(math.Floor(float64(minY))),
		int(math.Ceil(float64(maxX))), int(math.Ceil(float64(maxY))))
	w, h := dr.Dx(), dr.Dy()
	if w <= 0 || h <= 0 {
		return image.Rectangle{}, &z.mask, image.Point{}, adv, ok
	}
	if cap(z.mask.Pix) < w*h {
		z.mask.Pix = make([]uint8, 2*w*h)
	}
	z.mask.Pix = z.mask.Pix[:w*h]
	z.mask.Stride, z.mask.Rect = w, image.Rect(0, 0, w, h)

	bx, by := float32(dr.Min.X), float32(dr.Min.Y)
	z.rast.Reset(w, h)
	z.rast.DrawOp = draw.Src
	for _, s := range segs {
		var p [3]pt
		for i := 0; i < n[s.Op]; i++ {
			q := conv(s.Args[i])
			p[i] = pt{q.x - bx, q.y - by}
		}
		switch s.Op {
		case sfnt.SegmentOpMoveTo:
			z.rast.MoveTo(p[0].x, p[0].y)
		case sfnt.SegmentOpLineTo:
			z.rast.LineTo(p[0].x, p[0].y)
		case sfnt.SegmentOpQuadTo:
			z.rast.QuadTo(p[0].x, p[0].y, p[1].x, p[1].y)
		case sfnt.SegmentOpCubeTo:
			z.rast.CubeTo(p[0].x, p[0].y, p[1].x, p[1].y, p[2].x, p[2].y)
		}
	}
	z.rast.Draw(&z.mask, z.mask.Bounds(), image.Opaque, image.Point{})
	return dr, &z.mask, image.Point{}, adv, ok
}
