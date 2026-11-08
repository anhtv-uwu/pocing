package render

import (
	"image"
	"image/color"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/image/font"
)

const (
	vietReq = "POST /api/don-hang HTTP/1.1\nHost: app.example.com\n" +
		"X-Nguoi-Dung: Nguyễn Thị Ổn Ứng\nContent-Type: application/json\n\n" +
		`{"trạng_thái":"ĐẶT KIỂM","ghi_chú":"kết nối ồn ẩm ửng ỷ Ổn Ứng"}` + "\n"
	vietRes = "HTTP/1.1 200 OK\nContent-Type: application/json\n" +
		"X-Thong-Bao: Đã KIỂM tra, kết quả ổn\n\n" +
		`{"kết_quả":"ĐẶT KIỂM xong","lý_do":"ế ồ ẩ ử ỷ Ổ Ứ"}` + "\n"
)

// inkOver counts the pixels in r that are not the panel background. Unlike
// inkPixels it works in either theme, because Burp dark draws light text on a
// dark panel and "ink" there is the brighter pixel, not the darker one.
func inkOver(img *image.NRGBA, r image.Rectangle, bg color.NRGBA) int {
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if img.NRGBAAt(x, y) != bg {
				n++
			}
		}
	}
	return n
}

// Vietnamese has to survive the whole render, not only the font layer: the
// tokenizer, the word wrapper and the glyph cache each sit between a capture
// and its pixels, and a composed letter that resolves in isolation is still
// worthless if the pane comes out blank. Both themes are checked because the
// light and dark panels load their faces separately.
func TestVietnameseRendersInBothThemes(t *testing.T) {
	for _, dark := range []bool{false, true} {
		img, err := RenderHTTP(&HTTPConfig{Request: vietReq, Response: vietRes, Dark: dark})
		if err != nil {
			t.Fatalf("dark=%v: %v", dark, err)
		}
		b := img.Bounds()
		if b.Dx() <= 0 || b.Dy() <= 0 {
			t.Fatalf("dark=%v: empty image %v", dark, b)
		}
		ui := BurpLightTheme.UI
		if dark {
			ui = BurpTheme.UI
		}
		// The request panel's header lines, below line 1's caret band and
		// right of the gutter.
		code := image.Rect(60, lightHeaderH+40, 900, lightHeaderH+160)
		if ink := inkOver(img, code, ui.Bg); ink < 500 {
			t.Errorf("dark=%v: code area has %d ink pixels, want a drawn line", dark, ink)
		}
	}
}

// The Burp face is forced onto fixed cells, so a Vietnamese line occupies
// exactly as many columns as an ASCII line of the same rune count. If a
// composed letter advanced differently the pane would widen with the text, and
// every highlight box past it would slide off its glyphs.
func TestVietnameseKeepsTheFixedCell(t *testing.T) {
	const viet = "Nguyễn Thị Ổn Ứng: ế ồ ẩ ử ỷ Ổ Ứ"
	ascii := strings.Repeat("x", utf8.RuneCountInString(viet))

	m, err := loadMetricsFor(BurpLightTheme, false)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := font.MeasureString(m.codeFace, viet), font.MeasureString(m.codeFace, ascii); got != want {
		t.Errorf("%d runes of Vietnamese advance %v, the same count of ASCII %v",
			utf8.RuneCountInString(viet), got, want)
	}

	// And the same through the public door: equal-length header values must
	// produce panes of equal width.
	width := func(value string) int {
		img, err := RenderHTTP(&HTTPConfig{
			Request:  "GET / HTTP/1.1\nX-Note: " + value + "\n",
			Response: "HTTP/1.1 204 No Content\n",
		})
		if err != nil {
			t.Fatal(err)
		}
		return img.Bounds().Dx()
	}
	if got, want := width(viet), width(ascii); got != want {
		t.Errorf("Vietnamese pane is %dpx wide, ASCII pane %dpx", got, want)
	}
}
