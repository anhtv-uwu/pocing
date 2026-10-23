package render

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"sort"
	"strings"
	"testing"

	customfont "github.com/anhtv-uwu/pocing/font"
	"github.com/anhtv-uwu/pocing/highlight"
	"golang.org/x/image/font/sfnt"
)

// Every codepoint in the icon tables must exist in the embedded font. A missing
// glyph renders as tofu in the tab bar, which is invisible until someone opens
// the PNG. sfnt.GlyphIndex is the check that works: font.Face.GlyphAdvance
// reports ok for a missing rune because it falls back to .notdef.
func TestIconGlyphsExist(t *testing.T) {
	f, err := sfnt.Parse(customfont.SymbolsTTF)
	if err != nil {
		t.Fatal(err)
	}
	var buf sfnt.Buffer

	check := func(what, key string, ic fileIcon) {
		g, err := f.GlyphIndex(&buf, ic.glyph)
		if err != nil {
			t.Errorf("%s[%q]: glyph %#x lookup failed: %v", what, key, ic.glyph, err)
			return
		}
		if g == 0 {
			t.Errorf("%s[%q]: glyph %#x is not in the embedded font", what, key, ic.glyph)
		}
		if ic.color.A == 0 {
			t.Errorf("%s[%q]: color is transparent", what, key)
		}
	}

	for ext, ic := range iconByExt {
		check("iconByExt", ext, ic)
	}
	for name, ic := range iconByName {
		check("iconByName", name, ic)
	}
	check("defaultIcon", "-", defaultIcon)
}

func TestIconFor(t *testing.T) {
	cases := []struct {
		title string
		want  rune
	}{
		// Extension lookup, case-insensitive, path-tolerant.
		{"main.go", iconByExt["go"].glyph},
		{"MAIN.GO", iconByExt["go"].glyph},
		{"src/auth/views.py", iconByExt["py"].glyph},
		{`src\auth\views.py`, iconByExt["py"].glyph},
		{"app.tsx", iconByExt["tsx"].glyph},

		// Whole-name lookup wins over the extension.
		{"go.mod", iconByName["go.mod"].glyph},
		{"package.json", iconByName["package.json"].glyph},
		{"deploy/Dockerfile", iconByName["dockerfile"].glyph},
		{"Makefile", iconByName["makefile"].glyph},

		// Prefix fallbacks.
		{"Dockerfile.prod", iconByName["dockerfile"].glyph},
		{".env.local", iconByName[".env"].glyph},
		{".env", iconByName[".env"].glyph},

		// A leading dot is a name, not an extension: ".gitignore" must not be
		// looked up as the extension "gitignore".
		{".gitignore", iconByName[".gitignore"].glyph},

		// Unknown falls back.
		{"weird.qqq", defaultIcon.glyph},
		{"noextension", defaultIcon.glyph},
		{"", defaultIcon.glyph},
	}
	for _, c := range cases {
		if got := iconFor(c.title).glyph; got != c.want {
			t.Errorf("iconFor(%q) = %#x, want %#x", c.title, got, c.want)
		}
	}
}

// Both Burp palettes draw the macOS panel: the title strip and tab row sit on
// the editor background and are separated by hairlines, so the colours that
// carry the header must stand out from that background and from each other.
func TestBurpThemeHeaderContrast(t *testing.T) {
	for _, name := range []string{"burp", "burp-light"} {
		th, ok := Themes[name]
		if !ok {
			t.Fatalf("%s theme missing", name)
		}
		if !th.BurpLayout || th.HTTPSyntax == nil {
			t.Errorf("%s: must use the Burp layout and its HTTP colorizer", name)
		}
		ui := th.UI
		pairs := []struct {
			name string
			a, b color.NRGBA
		}{
			{"separator vs editor", ui.TabBorder, ui.Bg},
			{"active tab underline vs tab row", ui.TabAccent, ui.ActiveTab},
			{"active vs inactive tab text", ui.TabText, ui.TabMutedFg},
			{"toggle pill vs tab row", ui.ControlBg, ui.ActiveTab},
			{"current-line band vs editor", ui.CaretLineBg, ui.Bg},
			{"scroll thumb vs track", ui.ScrollThumb, ui.ScrollTrack},
			{"search field vs footer", ui.FieldBorder, ui.Bg},
		}
		for _, p := range pairs {
			if p.a == p.b {
				t.Errorf("%s: %s: both %v", name, p.name, p.a)
			}
		}
		if s := th.HTTPSyntax; s.HeaderName == s.HeaderValue {
			t.Errorf("%s: header names and values share %v", name, s.HeaderName)
		}
	}
}

// The two modes must not share geometry: an HTTP panel is denser than an editor
// and its header carries a tab row the editor tab bar does not.
func TestGeometryDiffersByMode(t *testing.T) {
	snap := geometryFor(false)
	burp := geometryFor(true)

	if burp.lineMul >= snap.lineMul {
		t.Errorf("http line height %v should be tighter than snap %v", burp.lineMul, snap.lineMul)
	}
	if burp.headerH <= snap.headerH {
		t.Errorf("http header %d should be taller than snap %d (title + tabs)", burp.headerH, snap.headerH)
	}
	if burp.headerH != burpTitleH+burpTabRowH {
		t.Errorf("http header %d != title %d + tabs %d", burp.headerH, burpTitleH, burpTabRowH)
	}
	if burp.gutterL >= snap.gutterL || burp.gutterR >= snap.gutterR {
		t.Error("Burp's gutter is narrower than an editor's")
	}
}

func TestBurpTabs(t *testing.T) {
	req := burpTabs("Request", "GET / HTTP/1.1\nHost: x\n")
	res := burpTabs("Response", "HTTP/1.1 200 OK\n")
	if req[0].label != "Pretty" || res[0].label != "Pretty" {
		t.Error("Pretty must be first; the header underlines index 0 as active")
	}
	if req[0].disabled || res[0].disabled {
		t.Error("the active tab must not be dimmed")
	}
	if len(res) <= len(req) {
		t.Error("the response panel has an extra view (Render)")
	}
	if !res[len(res)-1].disabled {
		t.Error("Render is dimmed for a JSON body, like Burp")
	}

	// Burp adds an editor tab per detected payload type.
	withJWT := burpTabs("Request",
		"GET / HTTP/1.1\nAuthorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjMifQ.sig\n")
	if len(withJWT) != len(req)+1 || withJWT[len(withJWT)-1].label != "JSON Web Token" {
		t.Errorf("a request bearing a JWT should gain the JWT tab, got %+v", withJWT)
	}
	// Not every base64 blob is a JWT: both segments must decode to an object.
	if got := burpTabs("Request", "Authorization: Basic Zm9vOmJhcg==\n"); len(got) != len(req) {
		t.Errorf("Basic auth must not add a JWT tab, got %+v", got)
	}

	// Burp only enables Render for markup.
	html := burpTabs("Response", "HTTP/1.1 200 OK\nContent-Type: text/html\n\n<html></html>")
	if html[len(html)-1].disabled {
		t.Error("Render should be enabled for an HTML body")
	}
}

func TestIsHTML(t *testing.T) {
	yes := []string{
		"HTTP/1.1 200 OK\nContent-Type: text/html; charset=utf-8\n",
		"HTTP/1.1 200 OK\nCONTENT-TYPE: TEXT/HTML\n",
		"HTTP/1.1 200 OK\n\n<!DOCTYPE html>\n",
		"HTTP/1.1 200 OK\n\n<html lang=\"en\">",
	}
	no := []string{
		"HTTP/1.1 200 OK\nContent-Type: application/json\n\n{}",
		"HTTP/1.1 204 No Content\n",
		"HTTP/1.1 200 OK\nContent-Type: text/plain\n\nnot <html-ish",
	}
	for _, m := range yes {
		if !isHTML(m) {
			t.Errorf("isHTML should be true for %q", m)
		}
	}
	for _, m := range no {
		if isHTML(m) {
			t.Errorf("isHTML should be false for %q", m)
		}
	}
}

// Guards against the class of bug this table replaced: two extensions of the
// same language drifting to different icons.
func TestIconFamiliesAgree(t *testing.T) {
	families := [][]string{
		{"c", "h"},
		{"cpp", "cc", "cxx", "hpp", "hh", "hxx"},
		{"js", "mjs", "cjs"},
		{"ts", "mts", "cts"},
		{"jsx", "tsx"},
		{"yaml", "yml"},
		{"html", "htm", "xhtml"},
		{"scss", "sass"},
		{"tf", "tfvars", "tfstate", "hcl"},
		{"sh", "bash", "zsh", "ksh", "fish"},
		{"ps1", "psm1", "psd1"},
		{"md", "markdown", "mdx"},
		{"clj", "cljs", "cljc", "edn"},
		{"erl", "hrl"},
		{"hs", "lhs"},
		{"pl", "pm"},
		{"ex", "exs"},
		{"kt", "kts"},
	}
	for _, fam := range families {
		first, ok := iconByExt[fam[0]]
		if !ok {
			t.Errorf("family head %q missing from iconByExt", fam[0])
			continue
		}
		for _, ext := range fam[1:] {
			ic, ok := iconByExt[ext]
			if !ok {
				t.Errorf("%q missing from iconByExt", ext)
				continue
			}
			if ic != first {
				t.Errorf("%q and %q disagree: %#x/%v vs %#x/%v",
					fam[0], ext, first.glyph, first.color, ic.glyph, ic.color)
			}
		}
	}
}

// TestContactSheet renders every table entry to labeled PNGs so a human can
// confirm each glyph is the logo it claims to be — the one thing
// TestIconGlyphsExist cannot check. Off by default:
//
//	ICON_SHEET=/tmp go test ./render/ -run ContactSheet -v
func TestContactSheet(t *testing.T) {
	dir := os.Getenv("ICON_SHEET")
	if dir == "" {
		t.Skip("set ICON_SHEET=<dir> to render the sheets")
	}

	m, err := loadMetrics()
	if err != nil {
		t.Fatal(err)
	}
	iconFace, err := customfont.LoadIconFace(22, fontDPI)
	if err != nil {
		t.Fatal(err)
	}

	type entry struct {
		key string
		ic  fileIcon
	}
	var all []entry
	for k, v := range iconByExt {
		all = append(all, entry{"." + k, v})
	}
	for k, v := range iconByName {
		all = append(all, entry{k, v})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].key < all[j].key })

	const cols, cellW, cellH, perSheet = 8, 190, 120, 80
	for s := 0; s*perSheet < len(all); s++ {
		chunk := all[s*perSheet:]
		if len(chunk) > perSheet {
			chunk = chunk[:perSheet]
		}
		rows := (len(chunk) + cols - 1) / cols
		img := image.NewNRGBA(image.Rect(0, 0, cols*cellW, rows*cellH))
		fillRect(img, img.Bounds(), rgb(0x1e, 0x1e, 0x1e))
		for i, e := range chunk {
			cx, cy := (i%cols)*cellW, (i/cols)*cellH
			fillRect(img, image.Rect(cx, cy, cx+cellW, cy+1), rgb(0x3c, 0x3c, 0x3c))
			fillRect(img, image.Rect(cx, cy, cx+1, cy+cellH), rgb(0x3c, 0x3c, 0x3c))
			adv, _ := iconFace.GlyphAdvance(e.ic.glyph)
			drawString(img, iconFace, string(e.ic.glyph), cx+(cellW-int(adv>>6))/2, cy+62, e.ic.color)
			drawString(img, m.face, e.key, cx+8, cy+92, rgb(0xd4, 0xd4, 0xd4))
			drawString(img, m.face, fmt.Sprintf("%04x", e.ic.glyph), cx+8, cy+112, rgb(0x6a, 0x99, 0x55))
		}
		path := fmt.Sprintf("%s/icons-%d.png", dir, s)
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			t.Fatal(err)
		}
		f.Close()
		t.Logf("wrote %s", path)
	}
	t.Logf("%d entries across %d sheets", len(all), (len(all)+perSheet-1)/perSheet)
}

// An icon drawn too dark disappears against the tab bar. That failure is
// invisible in review — the codepoint is right, the color compiles — and was
// found three separate times by eye. This checks the whole table at once.
func TestIconColorsAreVisible(t *testing.T) {
	// Rec. 709 luminance. Tab backgrounds sit near 0x22–0x25; the darkest
	// entry verified as legible by eye is #005f87 at ~78.
	const floor = 55
	lum := func(c color.NRGBA) float64 {
		return 0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)
	}
	check := func(what, key string, ic fileIcon) {
		if l := lum(ic.color); l < floor {
			t.Errorf("%s[%q] luminance %.0f < %d; invisible on the tab bar (%v)",
				what, key, l, floor, ic.color)
		}
	}
	for ext, ic := range iconByExt {
		check("iconByExt", ext, ic)
	}
	for name, ic := range iconByName {
		check("iconByName", name, ic)
	}
	check("defaultIcon", "-", defaultIcon)
}

// Keys that iconFor can never produce would be dead entries.
func TestIconKeysAreReachable(t *testing.T) {
	for ext := range iconByExt {
		if ext != strings.ToLower(ext) {
			t.Errorf("iconByExt[%q] is not lowercase; lookups are lowercased", ext)
		}
		if strings.HasPrefix(ext, ".") {
			t.Errorf("iconByExt[%q] has a leading dot; the dot is stripped before lookup", ext)
		}
	}
	for name := range iconByName {
		if name != strings.ToLower(name) {
			t.Errorf("iconByName[%q] is not lowercase", name)
		}
	}
}

// A palette built for one lexer must not be offered for code: it defines only
// that lexer's token types, so every other language falls back to the plain
// text color and renders with no highlighting at all.
func TestHTTPOnlyThemeCoversOnlyHTTP(t *testing.T) {
	if !BurpTheme.HTTPOnly || !BurpLightTheme.HTTPOnly {
		t.Error("burp and burp-light must be marked HTTPOnly; offering them for code renders flat")
	}
	for name, th := range Themes {
		if name != "burp" && name != "burp-light" && th.HTTPOnly {
			t.Errorf("theme %q is marked HTTPOnly; is that intended?", name)
		}
	}

	// The trap, made concrete: Java keywords carry no color under this style.
	java := "class Foo { void bar() { return; } }\n"
	flat, err := highlight.Tokenize(java, "java", "T.java", BurpStyle)
	if err != nil {
		t.Fatal(err)
	}
	lit, err := highlight.Tokenize(java, "java", "T.java", VSCodeDarkStyle)
	if err != nil {
		t.Fatal(err)
	}
	// Counting distinct colors is not enough: BurpStyle happens to define
	// NameFunction, so Java gets #e4e4e4 and #e8e8e8 — two values four levels
	// apart, which is flat to a human eye. Measure the spread instead.
	spread := func(lines []highlight.LineTokens) int {
		lo, hi := [3]int{255, 255, 255}, [3]int{0, 0, 0}
		for _, l := range lines {
			for _, tok := range l.Tokens {
				for i, v := range [3]int{int(tok.R), int(tok.G), int(tok.B)} {
					if v < lo[i] {
						lo[i] = v
					}
					if v > hi[i] {
						hi[i] = v
					}
				}
			}
		}
		widest := 0
		for i := range lo {
			if d := hi[i] - lo[i]; d > widest {
				widest = d
			}
		}
		return widest
	}
	const perceptible = 24
	if got := spread(flat); got >= perceptible {
		t.Errorf("BurpStyle now spreads Java colors by %d; if it highlights code, drop HTTPOnly", got)
	}
	if got := spread(lit); got < perceptible {
		t.Errorf("VSCodeDarkStyle should colour Java; spread is only %d", got)
	}
}
