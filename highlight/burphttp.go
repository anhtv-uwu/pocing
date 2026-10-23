package highlight

import (
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

// RGB is an opaque color.
type RGB struct{ R, G, B uint8 }

// BurpPalette is the syntax coloring of Burp Suite's message editor. Burp does
// not colour by grammar the way an editor does: it marks the parts a tester
// acts on — header names, and the name/value halves of every parameter.
type BurpPalette struct {
	Text        RGB // punctuation and body text
	StartLine   RGB // request/status line; zero means Text
	HeaderValue RGB // header values; zero means Text
	HeaderName  RGB // header name and its colon
	ParamName   RGB // query, cookie and form parameter names, with their '='
	ParamValue  RGB // query, cookie and form parameter values
	JSONKey     RGB
	JSONString  RGB
	JSONNumber  RGB // numbers and true/false/null
	HTMLTag     RGB // element names
	HTMLAttr    RGB // attribute names
	HTMLValue   RGB // attribute values, quotes included unless HTMLQuote is set
	HTMLEquals  RGB // the '=' after an attribute name; zero means Text
	HTMLQuote   RGB // quotes around attribute values; zero means HTMLValue
}

// BurpOptions tunes TokenizeBurpHTTPWith.
type BurpOptions struct {
	// RawHTML keeps markup bodies as sent instead of re-indenting them the
	// way Burp's Pretty tab does.
	RawHTML bool
	// KeepFinalLine numbers the empty line after a trailing newline, the way
	// Burp's editor does (the caret can sit there).
	KeepFinalLine bool
}

func (c RGB) tok(s string) Token { return Token{Text: s, R: c.R, G: c.G, B: c.B} }

// requestLineRe splits "METHOD target VERSION" so the query string can be
// coloured as parameters.
var requestLineRe = regexp.MustCompile(`^([A-Z][A-Z0-9_-]*)( +)(\S+)(.*)$`)

// headerRe matches "Name: value". The name excludes separators, so a body
// line such as "a: b" only matches inside the header block.
var headerRe = regexp.MustCompile(`^([!#$%&'*+.^_` + "`" + `|~0-9A-Za-z-]+)(:)(.*)$`)

// TokenizeBurpHTTP colours an HTTP message the way Burp's Pretty view does.
// It returns one LineTokens per source line, numbered from 1, with tabs
// expanded like Tokenize so column marks line up.
func TokenizeBurpHTTP(src string, p BurpPalette) []LineTokens {
	return TokenizeBurpHTTPWith(src, p, BurpOptions{})
}

// TokenizeBurpHTTPWith is TokenizeBurpHTTP with options.
func TokenizeBurpHTTPWith(src string, p BurpPalette, opt BurpOptions) []LineTokens {
	lines := tokenizeBurp(src, p, opt)
	if opt.KeepFinalLine && strings.HasSuffix(src, "\n") && len(src) > 1 {
		lines = append(lines, LineTokens{LineNum: len(lines) + 1})
	}
	return lines
}

func tokenizeBurp(src string, p BurpPalette, opt BurpOptions) []LineTokens {
	if p.HeaderValue == (RGB{}) {
		p.HeaderValue = p.Text
	}
	if p.StartLine == (RGB{}) {
		p.StartLine = p.Text
	}
	if p.HTMLEquals == (RGB{}) {
		p.HTMLEquals = p.Text
	}
	if p.HTMLTag == (RGB{}) {
		p.HTMLTag = p.HeaderName
	}
	if p.HTMLAttr == (RGB{}) {
		p.HTMLAttr = p.ParamName
	}
	if p.HTMLValue == (RGB{}) {
		p.HTMLValue = p.ParamValue
	}
	if p.HTMLQuote == (RGB{}) {
		p.HTMLQuote = p.HTMLValue
	}
	src = strings.ReplaceAll(src, "\r\n", "\n")
	raw := strings.Split(src, "\n")
	// A trailing newline closes the last line; it does not open a new one.
	if len(raw) > 1 && raw[len(raw)-1] == "" {
		raw = raw[:len(raw)-1]
	}

	lines := make([]LineTokens, 0, len(raw))
	add := func(toks ...Token) {
		var kept []Token
		for _, t := range toks {
			if t.Break {
				kept = append(kept, t)
			} else if t.Text != "" {
				t.Text = expandTabs(t.Text, 4)
				kept = append(kept, t)
			}
		}
		lines = append(lines, LineTokens{LineNum: len(lines) + 1, Tokens: kept})
	}

	i := 0
	// Start line.
	if len(raw) > 0 {
		if m := requestLineRe.FindStringSubmatch(raw[0]); m != nil && !strings.HasPrefix(raw[0], "HTTP/") {
			toks := []Token{p.StartLine.tok(m[1] + m[2])}
			toks = append(toks, urlTokens(m[3], p)...)
			toks = append(toks, p.StartLine.tok(m[4]))
			add(toks...)
		} else {
			add(p.StartLine.tok(raw[0]))
		}
		i = 1
	}

	// Header block, up to the first blank line.
	contentType := ""
	for ; i < len(raw); i++ {
		line := raw[i]
		if strings.TrimSpace(line) == "" {
			add()
			i++
			break
		}
		m := headerRe.FindStringSubmatch(line)
		if m == nil {
			add(p.Text.tok(line)) // obsolete folding or junk: show it as is
			continue
		}
		name, value := m[1], m[3]
		switch strings.ToLower(name) {
		case "content-type":
			contentType = strings.ToLower(value)
		}
		if strings.EqualFold(name, "Cookie") {
			add(append([]Token{p.HeaderName.tok(name + ":")}, cookieTokens(value, p)...)...)
			continue
		}
		add(p.HeaderName.tok(name+":"), p.HeaderValue.tok(value))
	}
	if i >= len(raw) {
		return lines
	}

	body := raw[i:]
	joined := strings.Join(body, "\n")
	trimmed := strings.TrimSpace(joined)
	switch {
	case strings.Contains(contentType, "json") ||
		strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "["):
		for _, toks := range jsonTokens(joined, p) {
			add(toks...)
		}
	case strings.Contains(contentType, "html") || strings.Contains(contentType, "xml") ||
		strings.HasPrefix(trimmed, "<"):
		if opt.RawHTML {
			for _, line := range body {
				add(htmlRawTokens(line, p)...)
			}
			break
		}
		for _, toks := range htmlLines(body, p) {
			add(toks...)
		}
	case strings.Contains(contentType, "x-www-form-urlencoded") || looksLikeForm(trimmed):
		for _, line := range body {
			add(paramTokens(line, "&", p)...)
		}
	default:
		for _, line := range body {
			add(p.Text.tok(line))
		}
	}
	return lines
}

var formRe = regexp.MustCompile(`^[^\s=&]+=[^\s&]*(&[^\s=&]+=?[^\s&]*)*$`)

func looksLikeForm(s string) bool { return !strings.Contains(s, "\n") && formRe.MatchString(s) }

// urlTokens colours the query string of a request target.
func urlTokens(target string, p BurpPalette) []Token {
	q := strings.IndexByte(target, '?')
	if q < 0 {
		return []Token{p.StartLine.tok(target)}
	}
	return append([]Token{p.StartLine.tok(target[:q+1])}, paramTokens(target[q+1:], "&", p)...)
}

// cookieTokens colours "a=1; b=2" with the separators in the text color.
func cookieTokens(value string, p BurpPalette) []Token {
	// Keep the leading space after the colon as plain text.
	lead := value[:len(value)-len(strings.TrimLeft(value, " \t"))]
	toks := []Token{p.Text.tok(lead)}
	parts := strings.Split(value[len(lead):], ";")
	for j, part := range parts {
		if j > 0 {
			toks = append(toks, p.Text.tok(";"))
		}
		sp := part[:len(part)-len(strings.TrimLeft(part, " "))]
		toks = append(toks, p.Text.tok(sp))
		toks = append(toks, pair(part[len(sp):], p)...)
	}
	return toks
}

// paramTokens colours sep-separated name=value pairs.
func paramTokens(s, sep string, p BurpPalette) []Token {
	var toks []Token
	for j, part := range strings.Split(s, sep) {
		if j > 0 {
			toks = append(toks, p.Text.tok(sep))
		}
		toks = append(toks, pair(part, p)...)
	}
	return toks
}

func pair(s string, p BurpPalette) []Token {
	if s == "" {
		return nil
	}
	eq := strings.IndexByte(s, '=')
	if eq < 0 {
		return []Token{p.ParamName.tok(s)}
	}
	return []Token{p.ParamName.tok(s[:eq+1]), p.ParamValue.tok(s[eq+1:])}
}

// jsonTokens colours a JSON body line by line. It leans on chroma's lexer for
// the grammar and maps its token types onto Burp's palette.
func jsonTokens(body string, p BurpPalette) [][]Token {
	lexer := lexers.Get("json")
	var it chroma.Iterator
	if lexer != nil {
		it, _ = chroma.Coalesce(lexer).Tokenise(nil, body)
	}
	out := [][]Token{nil}
	if it == nil {
		for j, l := range strings.Split(body, "\n") {
			if j > 0 {
				out = append(out, nil)
			}
			out[len(out)-1] = append(out[len(out)-1], p.Text.tok(l))
		}
		return out
	}
	for t := it(); t != chroma.EOF; t = it() {
		c := p.Text
		switch {
		case t.Type == chroma.NameTag:
			c = p.JSONKey
		case t.Type.InSubCategory(chroma.LiteralString):
			c = p.JSONString
		case t.Type.InSubCategory(chroma.LiteralNumber), t.Type == chroma.KeywordConstant:
			c = p.JSONNumber
		}
		for j, part := range strings.Split(t.Value, "\n") {
			if j > 0 {
				out = append(out, nil)
			}
			if part != "" {
				out[len(out)-1] = append(out[len(out)-1], c.tok(part))
			}
		}
	}
	// The lexer reproduces the input exactly, so a trailing newline would add
	// a phantom row; the caller already dropped it from the line count.
	if want := strings.Count(body, "\n") + 1; len(out) > want {
		out = out[:want]
	}
	return out
}
