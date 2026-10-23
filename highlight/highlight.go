package highlight

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Token is a single syntax-highlighted token with resolved RGB color.
type Token struct {
	Text    string
	R, G, B uint8
	// Break ends the current visual row without ending the logical line:
	// pretty-printed markup expands one source line into several rows that
	// share its line number.
	Break bool
}

// LineTokens represents one display line.
type LineTokens struct {
	LineNum int
	IsGap   bool
	Skipped int // for gap rows: how many lines were omitted
	Tokens  []Token
}

// Tokenize tokenizes src and returns one LineTokens per source line.
// lang may be empty for auto-detection. filename is used for extension-based detection.
func Tokenize(src, lang, filename string, style *chroma.Style) ([]LineTokens, error) {
	var lexer chroma.Lexer
	if lang != "" {
		lexer = lexers.Get(lang)
	}
	if lexer == nil && filename != "" {
		lexer = lexers.Match(filename)
	}
	if lexer == nil {
		lexer = lexers.Analyse(src)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	if style == nil {
		style = styles.Get("monokai")
		if style == nil {
			style = styles.Fallback
		}
	}

	it, err := lexer.Tokenise(nil, src)
	if err != nil {
		return nil, err
	}

	var lines []LineTokens
	current := LineTokens{LineNum: 1}

	for tok := it(); tok != chroma.EOF; tok = it() {
		text := expandTabs(tok.Value, 4)
		entry := style.Get(tok.Type)
		r, g, b := colorToRGB(entry.Colour)
		// Use default text color if no color set
		if entry.Colour == 0 {
			defEntry := style.Get(chroma.Text)
			r, g, b = colorToRGB(defEntry.Colour)
			if r == 0 && g == 0 && b == 0 {
				r, g, b = 0xd4, 0xd4, 0xd4 // VS Code default text
			}
		}

		parts := strings.Split(text, "\n")
		for i, part := range parts {
			if i > 0 {
				lines = append(lines, current)
				current = LineTokens{LineNum: current.LineNum + 1}
			}
			if part != "" {
				current.Tokens = append(current.Tokens, Token{Text: part, R: r, G: g, B: b})
			}
		}
	}
	// Flush last line
	lines = append(lines, current)

	// A trailing "\n" closes the last line; it does not open a new one. Without
	// this, every file that ends in a newline renders one phantom blank line.
	if len(lines) > 1 && len(lines[len(lines)-1].Tokens) == 0 {
		lines = lines[:len(lines)-1]
	}

	return lines, nil
}

// Text is the plain text of the line, with tabs already expanded, so column
// numbers taken from it line up with what gets drawn.
func (lt LineTokens) Text() string {
	var b strings.Builder
	for _, tok := range lt.Tokens {
		b.WriteString(tok.Text)
	}
	return b.String()
}

// FindLiteral marks every occurrence of needle across the given lines.
//
// This is the difference between usable and not: counting columns to write
// "7:63-85" is error-prone, whereas pasting the secret you just found is the
// thing an analyst already has in the clipboard.
func FindLiteral(lines []LineTokens, needle string) Marks {
	if needle == "" {
		return nil
	}
	want := []rune(needle)
	found := Marks{}
	for _, lt := range lines {
		if lt.IsGap {
			continue
		}
		text := []rune(lt.Text())
		for i := 0; i+len(want) <= len(text); i++ {
			if string(text[i:i+len(want)]) != needle {
				continue
			}
			found[lt.LineNum] = append(found[lt.LineNum], Span{Start: i + 1, End: i + len(want)})
			i += len(want) - 1
		}
	}
	if len(found) == 0 {
		return nil
	}
	return found
}

// Merge returns the union of m and other. Neither input is modified: RenderHTTP
// merges literal matches into the caller's highlight map, and a render must not
// leave the config it was handed different from how it found it.
func (m Marks) Merge(other Marks) Marks {
	if len(other) == 0 {
		return m
	}
	out := make(Marks, len(m)+len(other))
	for line, spans := range m {
		out[line] = append([]Span(nil), spans...)
	}
	for line, spans := range other {
		out[line] = append(out[line], spans...)
	}
	return out
}

// FilterLines applies a LineSpec (with gap sentinels) to the full tokenized lines.
// Returns a filtered slice ready for rendering.
func FilterLines(all []LineTokens, spec []int) []LineTokens {
	if len(spec) == 0 {
		return all
	}

	// Build map for O(1) lookup
	byNum := make(map[int]LineTokens, len(all))
	for _, lt := range all {
		byNum[lt.LineNum] = lt
	}

	// Keep only lines that exist, then derive the gaps from those. Deriving from
	// the spec's sentinels instead would emit gaps for out-of-range requests —
	// a trailing "896 lines skipped" on a 115-line file, with nothing after it.
	var found []LineTokens
	for _, n := range spec {
		if n == GapSentinel {
			continue
		}
		if lt, ok := byNum[n]; ok {
			found = append(found, lt)
		}
	}

	var result []LineTokens
	for i, lt := range found {
		if i > 0 {
			if skipped := lt.LineNum - found[i-1].LineNum - 1; skipped > 0 {
				result = append(result, LineTokens{IsGap: true, Skipped: skipped})
			}
		}
		result = append(result, lt)
	}
	return result
}

func expandTabs(s string, tabWidth int) string {
	return strings.ReplaceAll(s, "\t", strings.Repeat(" ", tabWidth))
}

func colorToRGB(c chroma.Colour) (r, g, b uint8) {
	if c == 0 {
		return 0, 0, 0
	}
	return c.Red(), c.Green(), c.Blue()
}
