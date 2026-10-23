package render

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/anhtv-uwu/pocing/highlight"
)

const (
	testReq = "POST /login HTTP/1.1\nHost: app.example.com\nContent-Type: application/json\n\n" +
		`{"user":"admin","pass":"hunter2"}` + "\n"
	testRes = "HTTP/1.1 200 OK\nContent-Type: application/json\nSet-Cookie: sid=abc; HttpOnly\n\n" +
		`{"token":"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.sig","admin":true}` + "\n"
)

func TestRenderHTTPComposition(t *testing.T) {
	img, err := RenderHTTP(&HTTPConfig{
		Request:    testReq,
		Response:   testRes,
		RequestHL:  highlight.Marks{5: {{}}},
		ResponseHL: highlight.Marks{5: {{}}},
	})
	if err != nil {
		t.Fatal(err)
	}

	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		t.Fatalf("empty image %v", b)
	}
	// The pane is flush: no chrome margin, so the top-left pixel belongs to the
	// panel's title strip, not to a surrounding background. Light is the
	// default; its title strip is the editor's white.
	if got := img.NRGBAAt(1, 1); got != BurpLightTheme.UI.Bg {
		t.Errorf("top-left pixel = %v, want light title strip %v", got, BurpLightTheme.UI.Bg)
	}

	dark, err := RenderHTTP(&HTTPConfig{Request: testReq, Response: testRes, Dark: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := dark.NRGBAAt(1, 1); got != BurpTheme.UI.TabBarBg {
		t.Errorf("dark top-left pixel = %v, want title strip %v", got, BurpTheme.UI.TabBarBg)
	}
	// Dark is the same macOS panel as light, only recoloured: same geometry,
	// footer included.
	light := img.Bounds()
	if dark.Bounds() != light {
		t.Errorf("dark bounds %v, want the light layout's %v", dark.Bounds(), light)
	}
	if got := dark.NRGBAAt(1, dark.Bounds().Dy()-2); got != BurpTheme.UI.TabBorder {
		t.Errorf("dark bottom edge = %v, want footer hairline %v", got, BurpTheme.UI.TabBorder)
	}
}

// A short panel is re-rendered to the taller one's height, so the gutter and
// background run the full length instead of stopping mid-image.
func TestRenderPairEqualisesHeight(t *testing.T) {
	short := &Config{
		Lines: tokenizeFor(t, "GET / HTTP/1.1\n"), Title: "Request",
		Theme: BurpTheme, Wrap: true, WrapWidth: 80, HTTPMode: true,
	}
	tall := &Config{
		Lines: tokenizeFor(t, strings.Repeat("X-Pad: y\n", 30)), Title: "Response",
		Theme: BurpTheme, Wrap: true, WrapWidth: 80, HTTPMode: true,
	}

	a, b, err := renderPair(short, tall)
	if err != nil {
		t.Fatal(err)
	}
	if a.Bounds().Dy() != b.Bounds().Dy() {
		t.Fatalf("panel heights differ: %d vs %d", a.Bounds().Dy(), b.Bounds().Dy())
	}

	out := composeHTTP(a, b, BurpTheme.UI)
	wantW := a.Bounds().Dx() + httpDividerW + b.Bounds().Dx()
	if out.Bounds().Dx() != wantW {
		t.Errorf("composed width %d, want %d", out.Bounds().Dx(), wantW)
	}
	if out.Bounds().Dy() != a.Bounds().Dy() {
		t.Errorf("composed height %d, want %d", out.Bounds().Dy(), a.Bounds().Dy())
	}

	// The short panel's gutter must reach the bottom of the composed pane.
	deep := out.Bounds().Dy() - 4
	if got := out.NRGBAAt(2, deep); got != BurpTheme.UI.Bg {
		t.Errorf("short panel at y=%d is %v, want panel background %v", deep, got, BurpTheme.UI.Bg)
	}
}

// Line specs must reach the HTTP panels, or a long body has no way to be
// trimmed and produces an image no viewer can open.
func TestRenderHTTPLineSpecs(t *testing.T) {
	long := "HTTP/1.1 200 OK\n\n" + strings.Repeat("padding line\n", 400)

	full, err := RenderHTTP(&HTTPConfig{Request: testReq, Response: long})
	if err != nil {
		t.Fatal(err)
	}
	trimmed, err := RenderHTTP(&HTTPConfig{
		Request: testReq, Response: long,
		ResponseLines: []int{1, 2, 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	if trimmed.Bounds().Dy() >= full.Bounds().Dy() {
		t.Errorf("trimmed pane (%d) should be shorter than the full one (%d)",
			trimmed.Bounds().Dy(), full.Bounds().Dy())
	}

	if _, err := RenderHTTP(&HTTPConfig{
		Request: testReq, Response: testRes,
		ResponseLines: []int{9000},
	}); err == nil {
		t.Error("a spec matching no response line should fail, not emit a blank panel")
	}
}

// Refusing is the point: a body this long renders taller than any viewer opens,
// and allocating it first costs about a gigabyte.
func TestRenderRefusesUnviewableHeight(t *testing.T) {
	huge := "HTTP/1.1 200 OK\n\n" + strings.Repeat("line\n", 1200)
	_, err := RenderHTTP(&HTTPConfig{Request: testReq, Response: huge, MaxRows: -1})
	if err == nil {
		t.Fatal("expected a height error")
	}
	for _, want := range []string{"decode", "--context"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q so the fix is obvious", err, want)
		}
	}
}

// A width below the floor would put one character per row.
func TestWrapWidthFloor(t *testing.T) {
	img, err := RenderHTTP(&HTTPConfig{
		Request: testReq, Response: testRes, WrapWidth: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	wide, err := RenderHTTP(&HTTPConfig{
		Request: testReq, Response: testRes, WrapWidth: minContentChars,
	})
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dy() != wide.Bounds().Dy() {
		t.Errorf("width 1 should clamp to %d: got height %d, want %d",
			minContentChars, img.Bounds().Dy(), wide.Bounds().Dy())
	}
}

func tokenizeFor(t *testing.T, src string) []highlight.LineTokens {
	t.Helper()
	lines, err := highlight.Tokenize(src, "http", "x.http", BurpTheme.Chroma)
	if err != nil {
		t.Fatal(err)
	}
	return lines
}

// HTTP evidence is Burp-coloured by construction, not by configuration. A
// caller cannot ask for another palette: a Dracula-tinted pane labelled
// "Request / Response / Pretty / Raw / Hex" is a screenshot of an application
// that does not exist, which is a false claim to put in a report.
func TestHTTPIsAlwaysBurp(t *testing.T) {
	if reflect.TypeOf(HTTPConfig{}).NumField() != len(httpConfigFields) {
		t.Fatalf("HTTPConfig changed shape; if a theme knob came back, remove it")
	}
	for _, name := range httpConfigFields {
		if _, ok := reflect.TypeOf(HTTPConfig{}).FieldByName(name); !ok {
			t.Errorf("HTTPConfig lost field %q", name)
		}
	}
	if _, ok := reflect.TypeOf(HTTPConfig{}).FieldByName("Theme"); ok {
		t.Error("HTTPConfig must not expose a theme")
	}

	// The only palette choice is which Burp: Dark is a bool, not a name.
	if f, _ := reflect.TypeOf(HTTPConfig{}).FieldByName("Dark"); f.Type.Kind() != reflect.Bool {
		t.Error("HTTPConfig.Dark must stay a bool: Burp light or Burp dark, nothing else")
	}

	img, err := RenderHTTP(&HTTPConfig{Request: testReq, Response: testRes, Dark: true})
	if err != nil {
		t.Fatal(err)
	}
	// Title strip, so the Burp palette really reached the panels.
	if got := img.NRGBAAt(1, 1); got != BurpTheme.UI.TabBarBg {
		t.Errorf("title strip = %v, want Burp's %v", got, BurpTheme.UI.TabBarBg)
	}
	if BurpTheme == nil || BurpTheme.Chroma == nil {
		t.Fatal("BurpTheme is not wired up")
	}
	if BurpLightTheme == nil || BurpLightTheme.HTTPSyntax == nil || !BurpLightTheme.BurpLayout {
		t.Fatal("BurpLightTheme is not wired up")
	}
}

var httpConfigFields = []string{
	"Request", "Response", "RequestHL", "ResponseHL", "Mark", "Caption",
	"RequestLines", "ResponseLines", "Context", "WrapWidth", "Dark", "Bold",
	"RawHTML", "MaxRows", "RequestCaret", "ResponseCaret",
}

// --mark and --context have to compose: the literal is what you have, and the
// window around it is what makes a 3000-line body readable. The marks are only
// known after tokenizing, so resolving the context outside the renderer left
// this combination silently doing nothing.
func TestMarkAndContextCompose(t *testing.T) {
	var body strings.Builder
	body.WriteString("HTTP/1.1 200 OK\n\n")
	for i := 1; i <= 400; i++ {
		fmt.Fprintf(&body, "filler line %d\n", i)
	}
	body.WriteString("needle=SECRET_VALUE\n")

	full, err := RenderHTTP(&HTTPConfig{Request: testReq, Response: body.String()})
	if err != nil {
		t.Fatal(err)
	}
	windowed, err := RenderHTTP(&HTTPConfig{
		Request: testReq, Response: body.String(),
		Mark: []string{"SECRET_VALUE"}, Context: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if windowed.Bounds().Dy() >= full.Bounds().Dy() {
		t.Errorf("context did not trim: %d vs full %d", windowed.Bounds().Dy(), full.Bounds().Dy())
	}

	// A literal that is nowhere leaves the pane untrimmed rather than empty.
	none, err := RenderHTTP(&HTTPConfig{
		Request: testReq, Response: body.String(),
		Mark: []string{"NOT_PRESENT"}, Context: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if none.Bounds().Dy() != full.Bounds().Dy() {
		t.Errorf("an unmatched mark should not trim anything: %d vs %d",
			none.Bounds().Dy(), full.Bounds().Dy())
	}
}
