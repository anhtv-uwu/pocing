package render

import (
	"fmt"
	"strings"
	"testing"

	"github.com/anhtv-uwu/pocing/highlight"
)

func longResponse(n int) string {
	var b strings.Builder
	b.WriteString("HTTP/1.1 200 OK\nContent-Type: text/plain\n\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "row %d\n", i)
	}
	return b.String()
}

// --max-lines caps the panel height and the view centres the marked rows.
func TestLightViewportCapsAndCentres(t *testing.T) {
	res := longResponse(200)
	capped, err := RenderHTTP(&HTTPConfig{Request: "GET / HTTP/1.1\n", Response: res})
	if err != nil {
		t.Fatal(err)
	}
	full, err := RenderHTTP(&HTTPConfig{Request: "GET / HTTP/1.1\n", Response: res, MaxRows: -1})
	if err != nil {
		t.Fatal(err)
	}
	if capped.Bounds().Dy() >= full.Bounds().Dy() {
		t.Fatalf("capped %d should be shorter than full %d", capped.Bounds().Dy(), full.Bounds().Dy())
	}

	lines := lightLines(t, res)
	vls := buildVisualLinesMode(lines, 71, true)
	cfg := &Config{HTTPMode: true, MaxRows: 10, HighlightLines: highlight.Marks{150: nil}}
	view, sc := viewport(vls, cfg)
	if sc == nil || sc.total != len(vls) || sc.visible != 10 {
		t.Fatalf("scroll state = %+v", sc)
	}
	if len(view) != 11 { // ten rows and the peek row
		t.Fatalf("view has %d rows, want 11", len(view))
	}
	found := false
	for i, vl := range view {
		if vl.lineNum == 150 {
			found = true
			if i < 3 || i > 7 {
				t.Errorf("marked row at view index %d, want it near the middle", i)
			}
		}
	}
	if !found {
		t.Error("marked line not in the view")
	}
}

// A capped panel draws a grey thumb in the scroll lane; an uncapped one only
// the track.
func TestLightScrollThumb(t *testing.T) {
	ui := BurpLightTheme.UI
	thumbIn := func(cfg *HTTPConfig) bool {
		img, err := RenderHTTP(cfg)
		if err != nil {
			t.Fatal(err)
		}
		x := img.Bounds().Dx() - lightThumbInset - lightThumbW/2 - 1
		for y := lightHeaderH; y < img.Bounds().Dy()-lightFooterH; y++ {
			if img.NRGBAAt(x, y) == ui.ScrollThumb {
				return true
			}
		}
		return false
	}
	if !thumbIn(&HTTPConfig{Request: "GET / HTTP/1.1\n", Response: longResponse(100)}) {
		t.Error("capped panel has no scroll thumb")
	}
	if thumbIn(&HTTPConfig{Request: "GET / HTTP/1.1\n", Response: "HTTP/1.1 204 No Content\n"}) {
		t.Error("short panel should not have a thumb")
	}
}

// The search toolbar sits under the code: a field filled with FieldBg.
func TestLightFooter(t *testing.T) {
	img, err := RenderHTTP(&HTTPConfig{Request: "GET / HTTP/1.1\n", Response: "HTTP/1.1 204 No Content\n"})
	if err != nil {
		t.Fatal(err)
	}
	ui := BurpLightTheme.UI
	top := img.Bounds().Dy() - lightFooterH
	if got := img.NRGBAAt(lightFooterFieldL+120, top+lightFooterH/2); got != ui.FieldBg {
		t.Errorf("search field = %v, want %v", got, ui.FieldBg)
	}
	if got := img.NRGBAAt(5, top); got != ui.TabBorder {
		t.Errorf("footer separator = %v, want %v", got, ui.TabBorder)
	}
}

// Burp shows a match count at the bottom right of each editor; this renderer
// does not, because a screenshot has nothing to search and "0 highlights" only
// invited the reader to wonder what was missed. Nothing may be drawn to the
// right of the search field, at any panel width, in either theme.
func TestLightFooterHasNoMatchCount(t *testing.T) {
	for _, dark := range []bool{false, true} {
		for _, cols := range []int{20, 40, 68, 71, 73, 100} {
			img, err := RenderHTTP(&HTTPConfig{Request: "GET / HTTP/1.1\n", Response: "HTTP/1.1 204 No Content\n", WrapWidth: cols, Dark: dark})
			if err != nil {
				t.Fatal(err)
			}
			th := BurpLightTheme
			if dark {
				th = BurpTheme
			}
			ui := th.UI
			b := img.Bounds()
			top := b.Dy() - lightFooterH
			div := -1
			// The divider is the one column drawn from top to bottom.
			for x := b.Dx() / 4; x < 3*b.Dx()/4 && div < 0; x++ {
				c := img.NRGBAAt(x, 1)
				full := c != ui.Bg
				for y := 1; y < b.Dy()-1 && full; y++ { // inside the outline
					full = img.NRGBAAt(x, y) == c
				}
				if full {
					div = x
				}
			}
			if div < 0 {
				t.Fatalf("dark=%v cols=%d: no panel divider", dark, cols)
			}
			// Each panel is drawn into its own canvas, so both the divider and
			// the image edge are a panel's right edge.
			for _, panel := range [][2]int{{0, div}, {div + 1, b.Dx()}} {
				left, right := panel[0], panel[1]
				w := right - left
				fieldR := left + min(lightFooterFieldR, max(w-lightFooterFieldP, lightFooterFieldL+lightFooterFieldW))
				for x := fieldR + 4; x < right; x++ {
					for y := top + lightFooterSepH; y < top+lightFooterH-lightFooterSepH; y++ {
						if got := img.NRGBAAt(x, y); got != ui.Bg {
							t.Fatalf("dark=%v cols=%d: ink %v at (%d,%d), right of the search field (match count back?)", dark, cols, got, x, y)
						}
					}
				}
			}
		}
	}
}
