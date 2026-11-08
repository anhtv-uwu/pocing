package main

import (
	"testing"
	"unicode/utf8"
)

// decode is the only place a capture is normalised, and these four repairs are
// fixed at the door on purpose: every later stage — the tokenizer, the word
// wrapper, the column arithmetic behind --highlight — counts runes and would
// faithfully lay out the damage. A BOM left in place becomes the first
// character of line 1 and the request line stops being recognised; a broken
// byte sequence splits a rune mid-glyph; and a combining mark that reaches the
// renderer takes a whole monospace cell of its own, printing a tone beside its
// letter instead of above it. What comes out of decode is valid, precomposed
// UTF-8, or nothing downstream can be trusted to be honest about the capture.
func TestDecode(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []byte
		want string
	}{{
		name: "ascii passes through",
		in:   []byte("GET /login HTTP/1.1\nHost: app.example.com\n"),
		want: "GET /login HTTP/1.1\nHost: app.example.com\n",
	}, {
		name: "leading BOM is stripped",
		in:   []byte("\ufeffPOST /login HTTP/1.1\n"),
		want: "POST /login HTTP/1.1\n",
	}, {
		// Only the leading one breaks the request line. A U+FEFF anywhere else
		// is part of the body the server actually sent, and evidence must show
		// what was sent.
		name: "BOM inside the text is kept",
		in:   []byte("GET / HTTP/1.1\nX-Note: a\ufeffb\n"),
		want: "GET / HTTP/1.1\nX-Note: a\ufeffb\n",
	}, {
		name: "invalid byte becomes U+FFFD",
		in:   []byte{'a', 0xff, 'b'},
		want: "a\ufffdb",
	}, {
		// The input is literally decomposed — "tie" U+0302 U+0301 "ng Vie"
		// U+0323 U+0302 "t" — which is the form macOS filesystems and some
		// HTTP clients hand over. It must come back as the precomposed runes
		// U+1EBF and U+1EC7. Do not let an editor "tidy" this line: if the
		// bytes are composed in place, the test silently stops testing.
		name: "NFD Vietnamese is composed",
		in:   []byte("tiếng Việt"),
		want: "tiếng Việt",
	}, {
		// Byte-identical: string comparison here is a byte comparison, so an
		// already-NFC capture must survive decode untouched.
		name: "NFC Vietnamese is unchanged",
		in:   []byte(`{"trạng_thái":"ĐẶT KIỂM","ghi_chú":"kết nối ổn"}`),
		want: `{"trạng_thái":"ĐẶT KIỂM","ghi_chú":"kết nối ổn"}`,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			got := decode(tc.in)
			if got != tc.want {
				t.Errorf("decode(%q) = %q, want %q", tc.in, got, tc.want)
			}
			// Whatever the input was, the renderer may never be handed bytes
			// it cannot step through rune by rune.
			if !utf8.ValidString(got) {
				t.Errorf("decode(%q) = %q, which is not valid UTF-8", tc.in, got)
			}
		})
	}
}
