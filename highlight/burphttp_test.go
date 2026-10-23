package highlight

import (
	"strings"
	"testing"
)

var testPalette = BurpPalette{
	Text:       RGB{0x14, 0x14, 0x14},
	HeaderName: RGB{0x00, 0x00, 0x75},
	ParamName:  RGB{0x00, 0x00, 0xc0},
	ParamValue: RGB{0xa0, 0x10, 0x10},
	JSONKey:    RGB{0x00, 0x00, 0x75},
	JSONString: RGB{0xa0, 0x10, 0x10},
	JSONNumber: RGB{0x00, 0x00, 0xc0},
}

// colorOf returns the color of the token containing text on a line.
func colorOf(t *testing.T, lt LineTokens, text string) RGB {
	t.Helper()
	for _, tok := range lt.Tokens {
		if strings.Contains(tok.Text, text) {
			return RGB{tok.R, tok.G, tok.B}
		}
	}
	t.Fatalf("no token containing %q in %q", text, lt.Text())
	return RGB{}
}

func TestBurpHTTPRequestColors(t *testing.T) {
	src := "GET /a?x=1&y=two HTTP/2\r\nHost: example.com\r\nCookie: sid=abc; t=xyz\r\n\r\nuser=bob&pw=hunter2\r\n"
	lines := TokenizeBurpHTTP(src, testPalette)
	if len(lines) != 5 {
		t.Fatalf("got %d lines, want 5 (CRLF normalised, trailing blank dropped)", len(lines))
	}
	for i, lt := range lines {
		if lt.LineNum != i+1 {
			t.Errorf("line %d numbered %d", i+1, lt.LineNum)
		}
	}
	p := testPalette
	checks := []struct {
		line int
		text string
		want RGB
	}{
		{0, "GET", p.Text},
		{0, "x=", p.ParamName},
		{0, "two", p.ParamValue},
		{1, "Host:", p.HeaderName},
		{1, "example.com", p.Text},
		{2, "Cookie:", p.HeaderName},
		{2, "sid=", p.ParamName},
		{2, "abc", p.ParamValue},
		{2, "t=", p.ParamName},
		{4, "pw=", p.ParamName},
		{4, "hunter2", p.ParamValue},
	}
	for _, c := range checks {
		if got := colorOf(t, lines[c.line], c.text); got != c.want {
			t.Errorf("line %d %q = %v, want %v", c.line+1, c.text, got, c.want)
		}
	}
	// Coloring never changes the text.
	want := strings.Split(strings.TrimSuffix(strings.ReplaceAll(src, "\r\n", "\n"), "\n"), "\n")
	for i, lt := range lines {
		if lt.Text() != want[i] {
			t.Errorf("line %d text %q, want %q", i+1, lt.Text(), want[i])
		}
	}
}

func TestBurpHTTPResponseColors(t *testing.T) {
	src := "HTTP/2 200 OK\nContent-Type: application/json\nCache-Control: max-age=0\n\n{\"token\": \"abc\", \"n\": 3}\n"
	lines := TokenizeBurpHTTP(src, testPalette)
	p := testPalette
	if got := colorOf(t, lines[0], "200"); got != p.Text {
		t.Errorf("status line = %v, want text color", got)
	}
	// Only Cookie values are split into params; other headers stay plain.
	if got := colorOf(t, lines[2], "max-age=0"); got != p.Text {
		t.Errorf("Cache-Control value = %v, want text color", got)
	}
	body := lines[4]
	if got := colorOf(t, body, `"token"`); got != p.JSONKey {
		t.Errorf("json key = %v, want %v", got, p.JSONKey)
	}
	if got := colorOf(t, body, `"abc"`); got != p.JSONString {
		t.Errorf("json string = %v, want %v", got, p.JSONString)
	}
	if got := colorOf(t, body, "3"); got != p.JSONNumber {
		t.Errorf("json number = %v, want %v", got, p.JSONNumber)
	}
}
