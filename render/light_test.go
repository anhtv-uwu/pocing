package render

import (
	"image"
	"strings"
	"testing"

	"github.com/tachote/taker/highlight"
)

func lightLines(t *testing.T, src string) []highlight.LineTokens {
	t.Helper()
	return highlight.TokenizeBurpHTTP(src, *BurpLightTheme.HTTPSyntax)
}

// Burp wraps after a space (or an attribute's opening quote) rather than mid-token, and every row records
// the column it starts at so marks land on the right glyphs.
func TestLightWordWrap(t *testing.T) {
	lines := lightLines(t, "Cookie: session=abcdefghij; theme=dark\n")
	vls := buildVisualLinesMode(lines, 20, true)
	var rows []string
	col := 1
	for _, vl := range vls {
		var b strings.Builder
		for _, tok := range vl.tokens {
			b.WriteString(tok.Text)
		}
		rows = append(rows, b.String())
		if vl.startCol != col {
			t.Errorf("row %q startCol = %d, want %d", b.String(), vl.startCol, col)
		}
		col += vl.runeLen()
	}
	want := []string{"Cookie: ", "session=abcdefghij; ", "theme=dark"}
	if strings.Join(rows, "|") != strings.Join(want, "|") {
		t.Errorf("rows = %q, want %q", rows, want)
	}

	// A token with no break point is still hard-cut at the width.
	vls = buildVisualLinesMode(lightLines(t, strings.Repeat("x", 45)+"\n"), 20, true)
	if len(vls) != 3 || vls[2].startCol != 41 {
		t.Errorf("unbroken run: %d rows, last startCol %d; want 3 rows from col 41", len(vls), vls[len(vls)-1].startCol)
	}
}

// A mark on a word-wrapped line is drawn on the row that holds it, at the
// columns it occupies there.
func TestLightMarkOnWrappedRow(t *testing.T) {
	m, err := loadMetricsFor(BurpLightTheme, false)
	if err != nil {
		t.Fatal(err)
	}
	lines := lightLines(t, "Cookie: session=abcdefghij; theme=dark\n")
	cfg := &Config{Lines: lines, Theme: BurpLightTheme, Wrap: true, WrapWidth: 20, HTTPMode: true}
	vls := buildVisualLinesMode(lines, 20, true)
	digits, gw := gutterMetricsMin(lines, m.charWidth, 0, 0, 2)
	l := buildLayout(cfg, m, vls, digits, gw)
	// "dark" is columns 35-38 of the logical line: the whole third row.
	marks := highlight.Marks{1: {{Start: 35, End: 38}}}
	if got := l.markBoxes(marks, 1, vls[0], 20); len(got) != 0 {
		t.Errorf("row 1 got %d boxes, want none", len(got))
	}
	if got := l.markBoxes(marks, 1, vls[2], 20); len(got) != 1 {
		t.Fatalf("row 3 got %d boxes, want 1", len(got))
	}
}

func inkPixels(img *image.NRGBA, r image.Rectangle) int {
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			c := img.NRGBAAt(x, y)
			if int(c.R)+int(c.G)+int(c.B) < 3*128 {
				n++
			}
		}
	}
	return n
}

// Message text is regular weight by default, like Burp's editor; --bold opts in.
func TestLightBoldOptIn(t *testing.T) {
	req := "GET /index.html HTTP/1.1\nHost: example.com\n"
	res := "HTTP/1.1 200 OK\nContent-Length: 0\n"
	bold, err := RenderHTTP(&HTTPConfig{Request: req, Response: res, Bold: true})
	if err != nil {
		t.Fatal(err)
	}
	regular, err := RenderHTTP(&HTTPConfig{Request: req, Response: res})
	if err != nil {
		t.Fatal(err)
	}
	if bold.Bounds() != regular.Bounds() {
		t.Fatalf("weight changed the geometry: %v vs %v", bold.Bounds(), regular.Bounds())
	}
	// The code area of the request panel, below the header.
	code := image.Rect(60, lightHeaderH+4, 600, lightHeaderH+80)
	b, r := inkPixels(bold, code), inkPixels(regular, code)
	if b <= r*11/10 {
		t.Errorf("bold ink %d not clearly heavier than regular %d", b, r)
	}
}

// The light header carries Burp's orange underline under the active tab and
// the blue pretty-print toggle in the control strip.
func TestLightHeaderChrome(t *testing.T) {
	img, err := RenderHTTP(&HTTPConfig{Request: "GET / HTTP/1.1\n", Response: "HTTP/1.1 204 No Content\n"})
	if err != nil {
		t.Fatal(err)
	}
	ui := BurpLightTheme.UI
	if got := img.NRGBAAt(10, lightUnderlineY+1); got != ui.TabAccent {
		t.Errorf("tab underline = %v, want %v", got, ui.TabAccent)
	}
	found := false
	for x := 0; x < img.Bounds().Dx() && !found; x++ {
		found = img.NRGBAAt(x, lightIconCY) == ui.ControlBg
	}
	if !found {
		t.Error("no pretty-print toggle in the header")
	}
}
