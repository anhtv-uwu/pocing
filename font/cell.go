package font

import (
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// cellFace lays a monospace face out on a fixed pixel cell. Burp is a Java
// app: with fractional metrics off, Java rounds each glyph advance to whole
// pixels, so Menlo at 13px (7.83px advance) sits on 8px cells. The glyphs are
// therefore a little smaller than the cell, which is what makes Burp's text
// read tidier than a font simply scaled until its advance is 8px.
type cellFace struct {
	font.Face
	adv fixed.Int26_6
}

// CellFace returns face with every non-zero advance replaced by cellPx.
func CellFace(face font.Face, cellPx int) font.Face {
	return &cellFace{Face: face, adv: fixed.I(cellPx)}
}

func (f *cellFace) cell(adv fixed.Int26_6) fixed.Int26_6 {
	if adv == 0 {
		return 0
	}
	return f.adv
}

func (f *cellFace) Glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	dr, mask, mp, adv, ok := f.Face.Glyph(dot, r)
	return dr, mask, mp, f.cell(adv), ok
}

func (f *cellFace) GlyphBounds(r rune) (fixed.Rectangle26_6, fixed.Int26_6, bool) {
	b, adv, ok := f.Face.GlyphBounds(r)
	return b, f.cell(adv), ok
}

func (f *cellFace) GlyphAdvance(r rune) (fixed.Int26_6, bool) {
	adv, ok := f.Face.GlyphAdvance(r)
	return f.cell(adv), ok
}

// Kern is zero: a monospace grid has no pair adjustments.
func (f *cellFace) Kern(r0, r1 rune) fixed.Int26_6 { return 0 }
