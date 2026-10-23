package render

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/font"
)

// Burp light's message-editor header, in 2x pixels measured from a Burp Suite
// 2026 light-theme capture (every 1x value doubled). Horizontal control
// positions are offsets from the panel's right edge to the control's centre.
const (
	lightTitleInkX     = 34 // "Request" ink starts 16px (1x) in
	lightTitleBaseline = 48
	lightTabInkPadL    = 32 // tab box left edge to its label ink
	lightTabInkPadR    = 30 // label ink to the end of the tab box
	lightTabStep       = 1  // extra space between tab boxes
	lightTabBaseline   = 122
	lightUnderlineY    = 144
	lightUnderlineH    = 6
	lightSepH          = 2
	lightHeaderH       = lightUnderlineY + lightUnderlineH + lightSepH // 152

	lightIconCY    = 113
	lightEyeCX     = 204
	lightToggleCX  = 147
	lightNewlineCX = 90
	lightMenuCX    = 35
	lightPillSize  = 38
	lightPillR     = 7
	lightControlsW = lightEyeCX + 20 // leftmost ink to the right edge
	lightTabsGap   = 40              // keeps tabs and controls apart

	// The menu is three hairline bars, not a font glyph.
	lightMenuW     = 22.0
	lightMenuBarH  = 2.4
	lightMenuPitch = 8.3

	// Response-only layout switch: three segments, the first active.
	lightLayoutL   = 170 // left edge, from the panel's right edge
	lightLayoutR   = 20
	lightLayoutTop = 18
	lightLayoutBot = 56
	lightLayoutSeg = 48
	lightLayoutRad = 8
)

const (
	glyphEyeOff = '\U000F06D1' // nf-md-eye_off_outline
	glyphWrap   = '\U000F05B6' // nf-md-wrap: Burp's pretty-print toggle
	glyphGear   = '\ueb51'     // codicon settings-gear (unused: drawn as vector)
)

func lightHeaderWidth(m *metrics, cfg *Config) int {
	w := 0
	for _, tab := range cfg.httpTabs {
		w += lightTabInkPadL + inkW(m.uiFace, tab.label) + lightTabInkPadR + lightTabStep
	}
	return w + lightTabsGap + lightControlsW
}

// drawLightPanelHeader draws Burp's light message-editor header: a bold title
// on a white strip, the view tabs with an orange underline on the active one,
// a hairline, and the control strip at the right.
func drawLightPanelHeader(img *image.NRGBA, l *layout, cfg *Config, ui UIColors) {
	w := l.totalWidth
	inkLeft(img, l.titleFace, cfg.Title, lightTitleInkX, lightTitleBaseline, ui.TabText)

	x := 0
	for i, tab := range cfg.httpTabs {
		boxW := lightTabInkPadL + inkW(l.uiFace, tab.label) + lightTabInkPadR
		fg := ui.TabMutedFg
		if tab.disabled {
			fg = ui.GapFg
		}
		if i == 0 { // Pretty is the active view
			fg = ui.TabText
			fillRect(img, image.Rect(x, lightUnderlineY, x+boxW, lightUnderlineY+lightUnderlineH), ui.TabAccent)
		}
		inkLeftInk(img, l.uiFace, tab.label, x+lightTabInkPadL, lightTabBaseline, fg, lightUIInkGamma)
		x += boxW + lightTabStep
	}
	sepY := lightUnderlineY + lightUnderlineH
	fillRect(img, image.Rect(0, sepY, w, sepY+lightSepH), ui.TabBorder)

	cy := lightIconCY
	drawEyeOffIcon(img, float64(w-lightEyeCX), float64(cy)+0.5, ui.IconFg, ui.Bg)
	half := lightPillSize / 2
	box := image.Rect(w-lightToggleCX-half, cy-half, w-lightToggleCX+half, cy+half)
	fillRoundRect(img, box, lightPillR, ui.ControlBg)
	drawGlyphCentered(img, l.iconFace, glyphWrap, w-lightToggleCX, cy, ui.ControlFg)
	drawTextCentered(img, l.uiFace, `\n`, w-lightNewlineCX, cy-3, ui.IconFg)
	mx := float64(w-lightMenuCX) - lightMenuW/2
	for i := -1; i <= 1; i++ {
		y := float64(cy) + float64(i)*lightMenuPitch - lightMenuBarH/2
		fillRectF(img, mx, y, mx+lightMenuW, y+lightMenuBarH, ui.IconFg)
	}

	if cfg.Title == "Response" {
		drawLayoutSwitch(img, w, ui)
	}
}

// drawLayoutSwitch is the side-by-side / stacked / single-pane control at the
// top right of Burp's response editor, with side-by-side selected.
func drawLayoutSwitch(img *image.NRGBA, w int, ui UIColors) {
	l, r := float64(w-lightLayoutL), float64(w-lightLayoutR)
	t, b := float64(lightLayoutTop), float64(lightLayoutBot)
	fillRoundRectF(img, l, t, r, b, lightLayoutRad, ui.TabBorder)
	seg := float64(lightLayoutSeg)
	// Active first segment: round its left corners only.
	fillRoundRectF(img, l, t, l+seg, b, lightLayoutRad, ui.ControlBg)
	fillRectF(img, l+seg-lightLayoutRad, t, l+seg, b, ui.ControlBg)

	iy0, iy1 := t+10, b-10
	// Side by side: two columns.
	ix := l + 14
	fillRectF(img, ix, iy0, ix+10, iy1, ui.ControlFg)
	fillRectF(img, ix+12, iy0, ix+22, iy1, ui.ControlFg)
	// Stacked: two rows.
	ix = float64(w) - 106
	mid := (iy0 + iy1) / 2
	fillRectF(img, ix, iy0, ix+22, mid-1, ui.IconFg)
	fillRectF(img, ix, mid+1, ix+22, iy1, ui.IconFg)
	// Single pane.
	ix = float64(w) - 56
	fillRectF(img, ix, iy0, ix+22, iy1, ui.IconFg)
}

// fillRectF paints a rectangle with fractional edges, antialiased by area
// coverage.
func fillRectF(img *image.NRGBA, x0, y0, x1, y1 float64, c color.NRGBA) {
	for y := int(math.Floor(y0)); y < int(math.Ceil(y1)); y++ {
		cy := math.Min(y1, float64(y+1)) - math.Max(y0, float64(y))
		for x := int(math.Floor(x0)); x < int(math.Ceil(x1)); x++ {
			cx := math.Min(x1, float64(x+1)) - math.Max(x0, float64(x))
			if cov := cx * cy; cov > 0 && image.Pt(x, y).In(img.Rect) {
				img.SetNRGBA(x, y, mix(img.NRGBAAt(x, y), c, cov))
			}
		}
	}
}

// drawGlyphCentered centers the ink of one glyph on (cx, cy).
func drawGlyphCentered(img *image.NRGBA, face font.Face, r rune, cx, cy int, c color.NRGBA) {
	drawTextCentered(img, face, string(r), cx, cy, c)
}

// drawTextCentered centers the ink box of s on (cx, cy).
func drawTextCentered(img *image.NRGBA, face font.Face, s string, cx, cy int, c color.NRGBA) {
	b, _ := font.BoundString(face, s)
	inkW := (b.Max.X - b.Min.X).Round()
	inkH := (b.Max.Y - b.Min.Y).Round()
	x := cx - inkW/2 - b.Min.X.Round()
	baseline := cy - inkH/2 - b.Min.Y.Round()
	drawString(img, face, s, x, baseline, c)
}

// fillRoundRect paints r with antialiased rounded corners over what is there.
func fillRoundRect(img *image.NRGBA, r image.Rectangle, radius int, c color.NRGBA) {
	fillRoundRectF(img, float64(r.Min.X), float64(r.Min.Y), float64(r.Max.X), float64(r.Max.Y), float64(radius), c)
}
