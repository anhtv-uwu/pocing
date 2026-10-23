package highlight

import (
	"strings"
	"testing"
)

func rowsOf(lt LineTokens) []string {
	rows := []string{""}
	for _, t := range lt.Tokens {
		if t.Break {
			rows = append(rows, "")
			continue
		}
		rows[len(rows)-1] += t.Text
	}
	return rows
}

// Burp's Pretty view re-indents markup: one tag or text run per row, five
// columns per level, all rows of a source line under one line number.
func TestBurpHTMLPrettyPrint(t *testing.T) {
	src := "HTTP/1.1 200 OK\nContent-Type: text/html\n\n<HTML><HEAD><TITLE>x</TITLE></HEAD>\n</HTML>\n"
	lines := TokenizeBurpHTTP(src, BurpPalette{})
	if len(lines) != 5 {
		t.Fatalf("got %d lines, want 5", len(lines))
	}
	got := strings.Join(rowsOf(lines[3]), "|")
	want := "<HTML>|     <HEAD>|          <TITLE>|               x|          </TITLE>|     </HEAD>"
	if got != want {
		t.Errorf("rows = %q\nwant  %q", got, want)
	}
	if got := strings.Join(rowsOf(lines[4]), "|"); got != "</HTML>" {
		t.Errorf("closing line = %q", got)
	}

	raw := TokenizeBurpHTTPWith(src, BurpPalette{}, BurpOptions{RawHTML: true})
	if got := strings.Join(rowsOf(raw[3]), "|"); got != "<HTML><HEAD><TITLE>x</TITLE></HEAD>" {
		t.Errorf("raw html = %q", got)
	}
}

// Burp numbers the empty line after a trailing newline; the light renderer
// asks for that, other callers keep the old count.
func TestBurpKeepFinalLine(t *testing.T) {
	src := "GET / HTTP/1.1\nHost: a\n\n"
	if n := len(TokenizeBurpHTTP(src, BurpPalette{})); n != 3 {
		t.Errorf("default: %d lines, want 3", n)
	}
	if n := len(TokenizeBurpHTTPWith(src, BurpPalette{}, BurpOptions{KeepFinalLine: true})); n != 4 {
		t.Errorf("KeepFinalLine: %d lines, want 4", n)
	}
}

// Burp's dark theme colours the start line, '=' and the quotes on their own;
// a palette that leaves them zero keeps the light behaviour.
func TestBurpDarkOnlyColours(t *testing.T) {
	text, start, eq, val, quote := RGB{1, 1, 1}, RGB{2, 2, 2}, RGB{3, 3, 3}, RGB{4, 4, 4}, RGB{5, 5, 5}
	src := "HTTP/1.1 200 OK\nContent-Type: text/html\n\n<a href=\"x\">\n"
	colourOf := func(lines []LineTokens, line int, s string) RGB {
		for _, tk := range lines[line].Tokens {
			if tk.Text == s {
				return RGB{tk.R, tk.G, tk.B}
			}
		}
		t.Fatalf("token %q not on line %d", s, line)
		return RGB{}
	}

	dark := TokenizeBurpHTTP(src, BurpPalette{Text: text, StartLine: start, HTMLEquals: eq, HTMLValue: val, HTMLQuote: quote})
	if got := colourOf(dark, 0, "HTTP/1.1 200 OK"); got != start {
		t.Errorf("start line = %v, want %v", got, start)
	}
	if got := colourOf(dark, 3, "="); got != eq {
		t.Errorf("'=' = %v, want %v", got, eq)
	}
	if got := colourOf(dark, 3, `"`); got != quote {
		t.Errorf("quote = %v, want %v", got, quote)
	}
	if got := colourOf(dark, 3, "x"); got != val {
		t.Errorf("value = %v, want %v", got, val)
	}

	light := TokenizeBurpHTTP(src, BurpPalette{Text: text, HTMLValue: val})
	if got := colourOf(light, 0, "HTTP/1.1 200 OK"); got != text {
		t.Errorf("light start line = %v, want text %v", got, text)
	}
	if got := colourOf(light, 3, `"x"`); got != val {
		t.Errorf("light value with quotes = %v, want %v", got, val)
	}
}
