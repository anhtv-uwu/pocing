package render

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/font"
)

// Burp light's scroll lane and bottom search toolbar, in 2x pixels measured
// from a Burp Suite 2026 light-theme capture (1x values doubled).
const (
	DefaultHTTPMaxRows = 32 // rows a default-height Burp editor shows

	lightScrollW      = 20 // track: the panel's right 10px at 1x
	lightThumbW       = 12
	lightThumbInset   = 4 // thumb sits 2px (1x) in from the panel edge
	lightThumbPadY    = 4
	lightPeekH        = 14 // a capped view shows the top of the next row
	lightFooterSepH   = 2
	lightFooterBodyH  = 92
	lightFooterH      = lightFooterSepH + lightFooterBodyH + lightFooterSepH
	lightFooterFieldL = 244
	lightFooterFieldR = 1020 // widest the search field gets (the default 71-column panel)
	lightFooterFieldP = 23   // search field to the panel's right edge
	lightFooterFieldW = 120  // narrowest search field, still holding the magnifier
)

const lightUIInkGamma = 0.7 // macOS draws UI labels heavier than FreeType-style AA

// scrollState is where a capped panel's view sits in the whole message.
type scrollState struct {
	total, start, visible int
}

// viewport caps an HTTP panel at cfg.MaxRows visual rows. A Burp screenshot of
// a long message is a scrolled editor, not a 9000px strip: the view keeps the
// top of the message or, when something is marked, centres the marked rows.
// One extra row is returned when available; the light layout shows its top
// edge the way a real editor clips its last visible row.
func viewport(vlines []visualLine, cfg *Config) ([]visualLine, *scrollState) {
	n := cfg.MaxRows
	if !cfg.HTTPMode || n <= 0 || len(vlines) <= n {
		return vlines, nil
	}
	first, last := -1, -1
	for i, vl := range vlines {
		if vl.isGap {
			continue
		}
		if _, ok := cfg.HighlightLines[vl.lineNum]; ok {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	start := 0
	switch {
	case first < 0, last < n:
		// Nothing marked, or every mark is already in the top view.
	case last-first+1 <= n:
		start = first - (n-(last-first+1))/2 // centre the marked block
	default:
		start = first - 2 // block taller than the view: lead with its start
	}
	start = max(0, min(start, len(vlines)-n))
	end := min(start+n+1, len(vlines))
	return vlines[start:end], &scrollState{total: len(vlines), start: start, visible: n}
}

// codeBottom is the last pixel row of the light code area (exclusive).
func (l *layout) codeBottom() int { return l.totalHeight - lightFooterH }

// drawLightScrollbar paints the scroll lane Burp always shows, with a thumb
// when the view is capped.
func drawLightScrollbar(img *image.NRGBA, l *layout, ui UIColors) {
	w := l.totalWidth
	top, bottom := l.codeAreaY, l.codeBottom()
	fillRect(img, image.Rect(w-lightScrollW, top, w, bottom), ui.ScrollTrack)
	sc := l.scroll
	if sc == nil || sc.total <= sc.visible {
		return
	}
	total := float64(sc.total)
	trackTop := float64(top + lightThumbPadY)
	trackH := float64(bottom-lightThumbPadY) - trackTop
	visible := float64(sc.visible) + float64(lightPeekH)/float64(l.lineHeight)
	thumbH := math.Max(trackH*visible/total, 48)
	thumbY := trackTop + (trackH-thumbH)*float64(sc.start)/math.Max(total-visible, 1)
	x1 := float64(w - lightThumbInset - lightThumbW)
	fillRoundRectF(img, x1, thumbY, x1+lightThumbW, thumbY+thumbH, 3, ui.ScrollThumb)
}

// drawLightFooter draws the editor's search toolbar: help, settings, previous
// and next match, and the search field.
func drawLightFooter(img *image.NRGBA, l *layout, ui UIColors) {
	w := l.totalWidth
	top := l.codeBottom()
	fillRect(img, image.Rect(0, top, w, top+lightFooterSepH), ui.TabBorder)
	fillRect(img, image.Rect(0, top+lightFooterSepH, w, top+lightFooterH-lightFooterSepH), ui.Bg)
	fillRect(img, image.Rect(0, top+lightFooterH-lightFooterSepH, w, top+lightFooterH), ui.TabBorder)

	fg := ui.IconFg
	cy := float64(top) + 43

	// Help: a ring with a question mark.
	strokeCircle(img, 32.5, cy+3, 18.3, 3.2, fg)
	drawTextCentered(img, l.helpFace, "?", 32, int(cy)+4, fg)

	// Settings gear.
	drawGearIcon(img, 86, cy+3, fg)

	// Previous / next match buttons.
	for i, dir := range []float64{-1, 1} {
		x0 := float64(118 + i*58)
		b := image.Rect(int(x0), top+22, int(x0)+48, top+70)
		fillRoundRect(img, b, 7, ui.FooterBtnBorder)
		fillRoundRect(img, b.Inset(2), 5, ui.FooterBtnBg)
		drawArrow(img, x0+24, float64(top)+46, dir, fg)
	}

	// The search field runs to the panel's right edge, keeping a margin, and
	// never narrower than the magnifier needs.
	fieldR := min(lightFooterFieldR, w-lightFooterFieldP)
	fieldR = max(fieldR, lightFooterFieldL+lightFooterFieldW)

	// Search field.
	f := image.Rect(lightFooterFieldL, top+24, fieldR, top+70)
	fillRoundRect(img, f, 8, ui.FieldBorder)
	fillRoundRect(img, f.Inset(2), 6, ui.FieldBg)
	inkLeft(img, l.uiFace, "Search", f.Min.X+16, top+56, ui.FieldPlaceholder)
	// Magnifier, handle down and to the left as Burp draws it.
	mx, my := float64(fieldR-26), float64(top)+44
	strokeCircle(img, mx, my, 9, 2.6, fg)
	strokeLine(img, mx-6.5, my+6.5, mx-14, my+14, 2.8, fg)
}

// inkLeft draws s with its ink starting exactly at x.
func inkLeft(img *image.NRGBA, face font.Face, s string, x, baseline int, c color.NRGBA) {
	drawString(img, face, s, x-inkX(face, s), baseline, c)
}

// inkLeftInk is inkLeft with drawStringInk's coverage gamma.
func inkLeftInk(img *image.NRGBA, face font.Face, s string, x, baseline int, c color.NRGBA, gamma float64) {
	drawStringInk(img, face, s, x-inkX(face, s), baseline, c, gamma)
}

// inkX is how far right of the pen origin the first ink of s lands.
func inkX(face font.Face, s string) int {
	b, _ := font.BoundString(face, s)
	return b.Min.X.Round()
}

// inkW is the width of the ink of s.
func inkW(face font.Face, s string) int {
	b, _ := font.BoundString(face, s)
	return (b.Max.X - b.Min.X).Round()
}

// drawArrow is Burp's previous/next chevron arrow: a shaft and two barbs.
func drawArrow(img *image.NRGBA, cx, cy, dir float64, c color.NRGBA) {
	const half, barb, w = 10.0, 8.0, 3.0
	tip := cx + dir*half
	strokeLine(img, cx-dir*half, cy, tip, cy, w, c)
	strokeLine(img, tip, cy, tip-dir*barb, cy-barb, w, c)
	strokeLine(img, tip, cy, tip-dir*barb, cy+barb, w, c)
}

// drawGearIcon is an outlined eight-tooth gear with a round hub.
func drawGearIcon(img *image.NRGBA, cx, cy float64, c color.NRGBA) {
	const teeth = 8
	outer, inner, hub := 20.5, 16.0, 6.5
	b := image.Rect(int(cx-outer-2), int(cy-outer-2), int(cx+outer+3), int(cy+outer+3))
	// Signed distance to the gear outline, sampled 4x4 per pixel.
	inside := func(x, y float64) bool {
		dx, dy := x-cx, y-cy
		r := math.Hypot(dx, dy)
		a := math.Atan2(dy, dx)
		ph := math.Mod(a*teeth/(2*math.Pi)+100.25, 1) // tooth phase
		edge := inner
		// Trapezoid teeth with rounded flanks.
		t := math.Abs(ph-0.5) * 2 // 0 at tooth centre, 1 at the gap centre
		switch {
		case t < 0.4:
			edge = outer
		case t < 0.75:
			// Rounded flank: a quarter-cosine from tip to root.
			edge = inner + (outer-inner)*(0.5+0.5*math.Cos(math.Pi*(t-0.4)/0.35))
		}
		return r <= edge
	}
	ring := func(x, y float64) bool {
		r := math.Hypot(x-cx, y-cy)
		if r <= hub+1.8 && r >= hub-1.8 {
			return true
		}
		if !inside(x, y) || r < hub+1.5 {
			return false
		}
		// Outline only: inside the gear but within 3px of its boundary.
		for _, d := range [][2]float64{{3.5, 0}, {-3.5, 0}, {0, 3.5}, {0, -3.5}, {2.5, 2.5}, {-2.5, 2.5}, {2.5, -2.5}, {-2.5, -2.5}} {
			if !inside(x+d[0], y+d[1]) {
				return true
			}
		}
		return false
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			hits := 0
			for sy := 0; sy < 4; sy++ {
				for sx := 0; sx < 4; sx++ {
					if ring(float64(x)+(float64(sx)+0.5)/4, float64(y)+(float64(sy)+0.5)/4) {
						hits++
					}
				}
			}
			if hits > 0 {
				img.SetNRGBA(x, y, mix(img.NRGBAAt(x, y), c, float64(hits)/16))
			}
		}
	}
}

// drawEyeOffIcon is Burp's "hide headers" eye: a thin almond outline, a
// pupil ring, and a slash from top-left to bottom-right.
func drawEyeOffIcon(img *image.NRGBA, cx, cy float64, c, bg color.NRGBA) {
	const a, b, w = 14.2, 9.6, 2.3
	for y := int(cy - b - w - 1); y <= int(cy+b+w)+1; y++ {
		for x := int(cx - a - w - 1); x <= int(cx+a+w)+1; x++ {
			hits := 0
			for sy := 0; sy < 4; sy++ {
				for sx := 0; sx < 4; sx++ {
					px := float64(x) + (float64(sx)+0.5)/4 - cx
					py := float64(y) + (float64(sy)+0.5)/4 - cy
					f := math.Hypot(px/a, py/b) - 1
					gx, gy := px/(a*a), py/(b*b)
					g := math.Hypot(gx, gy) / math.Max(math.Hypot(px/a, py/b), 1e-6)
					if math.Abs(f/math.Max(g, 1e-6)) <= w/2 {
						hits++
					}
				}
			}
			if hits > 0 && image.Pt(x, y).In(img.Rect) {
				img.SetNRGBA(x, y, mix(img.NRGBAAt(x, y), c, float64(hits)/16))
			}
		}
	}
	strokeCircle(img, cx, cy+1.5, 4, 2.2, c)
	// The slash is cut out of the eye with a background-coloured halo.
	strokeLine(img, cx-12.5, cy-13.4, cx+10.5, cy+12.4, 6, bg)
	strokeLine(img, cx-14, cy-15, cx+12, cy+14, 2.3, c)
}

// strokeLine draws an antialiased segment of width w with round caps.
func strokeLine(img *image.NRGBA, x0, y0, x1, y1, w float64, c color.NRGBA) {
	r := w / 2
	minX, maxX := int(math.Floor(math.Min(x0, x1)-r-1)), int(math.Ceil(math.Max(x0, x1)+r+1))
	minY, maxY := int(math.Floor(math.Min(y0, y1)-r-1)), int(math.Ceil(math.Max(y0, y1)+r+1))
	dx, dy := x1-x0, y1-y0
	l2 := dx*dx + dy*dy
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			t := 0.0
			if l2 > 0 {
				t = math.Max(0, math.Min(1, ((px-x0)*dx+(py-y0)*dy)/l2))
			}
			d := math.Hypot(px-(x0+t*dx), py-(y0+t*dy))
			if cov := clamp01(r - d + 0.5); cov > 0 && image.Pt(x, y).In(img.Rect) {
				img.SetNRGBA(x, y, mix(img.NRGBAAt(x, y), c, cov))
			}
		}
	}
}

// strokeCircle draws an antialiased ring of radius r (to the stroke centre).
func strokeCircle(img *image.NRGBA, cx, cy, r, w float64, c color.NRGBA) {
	for y := int(cy - r - w); y <= int(cy+r+w)+1; y++ {
		for x := int(cx - r - w); x <= int(cx+r+w)+1; x++ {
			d := math.Abs(math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy) - r)
			if cov := clamp01(w/2 - d + 0.5); cov > 0 && image.Pt(x, y).In(img.Rect) {
				img.SetNRGBA(x, y, mix(img.NRGBAAt(x, y), c, cov))
			}
		}
	}
}

// fillRoundRectF is fillRoundRect with fractional edges.
func fillRoundRectF(img *image.NRGBA, x0, y0, x1, y1, rad float64, c color.NRGBA) {
	for y := int(math.Floor(y0)); y < int(math.Ceil(y1)); y++ {
		for x := int(math.Floor(x0)); x < int(math.Ceil(x1)); x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			dx := math.Max(math.Max(x0+rad-px, px-(x1-rad)), 0)
			dy := math.Max(math.Max(y0+rad-py, py-(y1-rad)), 0)
			cov := clamp01(rad - math.Hypot(dx, dy) + 0.5)
			// Fractional straight edges.
			cov = math.Min(cov, clamp01(px-x0+0.5)*clamp01(x1-px+0.5))
			cov = math.Min(cov, clamp01(py-y0+0.5)*clamp01(y1-py+0.5))
			if cov > 0 && image.Pt(x, y).In(img.Rect) {
				img.SetNRGBA(x, y, mix(img.NRGBAAt(x, y), c, cov))
			}
		}
	}
}
