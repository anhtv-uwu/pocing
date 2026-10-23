package highlight

import (
	"regexp"
	"strings"
)

// Burp's Pretty view re-indents markup: every tag and text run starts its own
// row, nested one indent step deeper than its parent. The line number stays
// with the source line, so the rows a source line expands into are drawn as
// unnumbered continuation rows — exactly how a wrapped line looks.
const htmlIndent = 5 // columns per nesting level, measured from Burp 2026.x

// voidElements never get a closing tag, so they do not open a level.
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true,
	"img": true, "input": true, "link": true, "meta": true, "param": true,
	"source": true, "track": true, "wbr": true, "!doctype": true,
}

var (
	htmlPieceRe = regexp.MustCompile(`(?s)<!--.*?-->|<[^>]*>?|[^<]+`)
	htmlAttrRe  = regexp.MustCompile(`([^\s=/>]+)(\s*=\s*)?("[^"]*"?|'[^']*'?|[^\s>]+)?`)
	tagNameRe   = regexp.MustCompile(`^</?\s*([!A-Za-z][^\s/>]*)`)
)

// Break is a zero-width token that starts a new visual row in the same
// logical line.
func breakTok() Token { return Token{Break: true} }

// htmlLines pretty-prints a markup body into one token slice per source line.
func htmlLines(body []string, p BurpPalette) [][]Token {
	level := 0
	out := make([][]Token, 0, len(body))
	for _, line := range body {
		var toks []Token
		first := true
		emitRow := func(lvl int, row []Token) {
			if !first {
				toks = append(toks, breakTok())
			}
			first = false
			if lvl > 0 {
				toks = append(toks, p.Text.tok(strings.Repeat(" ", lvl*htmlIndent)))
			}
			toks = append(toks, row...)
		}
		for _, piece := range htmlPieceRe.FindAllString(line, -1) {
			switch {
			case strings.HasPrefix(piece, "<!--"):
				emitRow(level, []Token{p.Text.tok(piece)})
			case strings.HasPrefix(piece, "</"):
				level = max(level-1, 0)
				emitRow(level, tagTokens(piece, p))
			case strings.HasPrefix(piece, "<"):
				emitRow(level, tagTokens(piece, p))
				name := ""
				if m := tagNameRe.FindStringSubmatch(piece); m != nil {
					name = strings.ToLower(m[1])
				}
				if name != "" && !voidElements[name] && !strings.HasSuffix(piece, "/>") &&
					!strings.HasPrefix(name, "?") {
					level++
				}
			default:
				if t := strings.TrimSpace(piece); t != "" {
					emitRow(level, []Token{p.Text.tok(t)})
				}
			}
		}
		out = append(out, toks)
	}
	return out
}

// tagTokens colours one tag: brackets in the text colour, the element name
// magenta, attribute names blue and quoted values dark red.
func tagTokens(tag string, p BurpPalette) []Token {
	m := tagNameRe.FindStringSubmatchIndex(tag)
	if m == nil {
		return []Token{p.Text.tok(tag)}
	}
	toks := []Token{p.Text.tok(tag[:m[2]]), p.HTMLTag.tok(tag[m[2]:m[3]])}
	rest := tag[m[3]:]
	end := ""
	switch {
	case strings.HasSuffix(rest, "/>"):
		rest, end = rest[:len(rest)-2], "/>"
	case strings.HasSuffix(rest, ">"):
		rest, end = rest[:len(rest)-1], ">"
	}
	pos := 0
	for _, a := range htmlAttrRe.FindAllStringSubmatchIndex(rest, -1) {
		toks = append(toks, p.Text.tok(rest[pos:a[0]]))
		toks = append(toks, p.HTMLAttr.tok(rest[a[2]:a[3]]))
		if a[4] >= 0 {
			toks = append(toks, p.HTMLEquals.tok(rest[a[4]:a[5]]))
		}
		if a[6] >= 0 {
			toks = append(toks, valueTokens(rest[a[6]:a[7]], p)...)
		}
		pos = a[1]
	}
	toks = append(toks, p.Text.tok(rest[pos:]), p.Text.tok(end))
	return toks
}

// valueTokens colours an attribute value, splitting off its quotes when the
// palette gives them their own colour (Burp's dark theme does).
func valueTokens(v string, p BurpPalette) []Token {
	if p.HTMLQuote == p.HTMLValue || len(v) < 2 || (v[0] != '"' && v[0] != '\'') || v[len(v)-1] != v[0] {
		return []Token{p.HTMLValue.tok(v)}
	}
	q := v[:1]
	return []Token{p.HTMLQuote.tok(q), p.HTMLValue.tok(v[1 : len(v)-1]), p.HTMLQuote.tok(q)}
}

// htmlRawTokens colours one markup line without re-indenting it.
func htmlRawTokens(line string, p BurpPalette) []Token {
	var toks []Token
	for _, piece := range htmlPieceRe.FindAllString(line, -1) {
		if strings.HasPrefix(piece, "<") && !strings.HasPrefix(piece, "<!--") {
			toks = append(toks, tagTokens(piece, p)...)
		} else {
			toks = append(toks, p.Text.tok(piece))
		}
	}
	return toks
}
