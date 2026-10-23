package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/anhtv-uwu/pocing/highlight"
	"github.com/anhtv-uwu/pocing/render"
)

// SnapCmd renders a single code file.
type SnapCmd struct {
	File       string   `arg:"" optional:"" help:"Source file to render (reads stdin if omitted)."`
	Lang       string   `short:"l" help:"Language for syntax highlighting (auto-detected if omitted)."`
	Lines      string   `short:"n" help:"Lines to include, e.g. '1,5,10-20'. Ignored when --context is set."`
	Highlight  string   `short:"H" help:"Regions to highlight: '15', '13,15', '10-20', or '15:20-45' for columns."`
	Mark       []string `short:"m" help:"Highlight every occurrence of this text. Repeatable."`
	Label      string   `short:"L" help:"Caption strip stating what the image proves."`
	Context    int      `short:"c" default:"0" help:"Include N context lines around each highlighted line (requires --highlight)."`
	Wrap       bool     `short:"w" help:"Wrap long lines at a column boundary."`
	WrapWidth  int      `name:"wrap-width" default:"0" help:"Columns per row (default: 100, minimum 20). Implies --wrap."`
	Output     string   `short:"o" default:"output.png" help:"Output PNG file path."`
	Title      string   `short:"t" help:"Title shown in the tab bar (defaults to file basename)."`
	Theme      string   `short:"T" default:"vscode-dark" help:"Color theme. Use --list-themes to see options."`
	ListThemes bool     `name:"list-themes" help:"Print available themes and exit."`
}

// HTTPCmd renders a side-by-side HTTP request/response evidence image.
type HTTPCmd struct {
	Request   string   `short:"r" required:"" help:"File containing the HTTP request."`
	Response  string   `short:"s" required:"" help:"File containing the HTTP response."`
	ReqHL     string   `name:"hr" help:"Regions to highlight in request: '8-9' or '8:20-45' for columns."`
	ResHL     string   `name:"hs" help:"Regions to highlight in response: '4' or '4:12-60' for columns."`
	Mark      []string `short:"m" help:"Highlight every occurrence of this text in either panel. Repeatable."`
	Label     string   `short:"L" help:"Caption strip stating what the image proves."`
	ReqLines  string   `name:"nr" help:"Lines to include from the request, e.g. '1-20'."`
	ResLines  string   `name:"ns" help:"Lines to include from the response, e.g. '1-30'."`
	Context   int      `short:"c" default:"0" help:"Include N context lines around each highlighted line; applies per panel."`
	WrapWidth int      `name:"wrap-width" default:"0" help:"Columns per panel (default: 71; minimum 20)."`
	Theme     string   `name:"theme" default:"light" enum:"light,dark,burp-light,burp-dark,burp" help:"Burp palette: light (default) or dark."`
	Bold      bool     `name:"bold" help:"Draw message text in DejaVu Sans Mono Bold (Burp itself uses regular)."`
	Regular   bool     `name:"regular" hidden:"" help:"Deprecated no-op: regular weight is now the default."`
	RawHTML   bool     `name:"raw-html" help:"Show HTML bodies as sent instead of Burp's re-indented Pretty view."`
	MaxLines  int      `name:"max-lines" default:"32" help:"Visible rows per panel; longer panels get a Burp-style scrollbar (0 = show everything)."`
	CaretReq  int      `name:"caret-r" default:"1" help:"Request line carrying Burp's grey current-line band (0 = none)."`
	CaretRes  int      `name:"caret-s" default:"1" help:"Response line carrying Burp's grey current-line band (0 = none)."`
	Output    string   `short:"o" default:"output.png" help:"Output PNG file path."`
}

var cli struct {
	Snap    SnapCmd `cmd:"" default:"withargs" help:"Render a code file as a PNG screenshot."`
	HTTP    HTTPCmd `cmd:"http" help:"Render HTTP request/response evidence (side-by-side)."`
	Version bool    `short:"v" name:"version" help:"Print version and exit."`
}

// version is set by GoReleaser via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	// Show help when called with no arguments and stdin is a terminal.
	if len(os.Args) == 1 {
		fi, err := os.Stdin.Stat()
		if err != nil || (fi.Mode()&os.ModeCharDevice) != 0 {
			fmt.Fprintln(os.Stderr, "pocing: no input. Run 'pocing --help' for usage.")
			os.Exit(1)
		}
	}

	ctx := kong.Parse(&cli,
		kong.Name("pocing"),
		kong.Description("Generate professional code & HTTP evidence screenshots for security audits."),
		kong.UsageOnError(),
	)

	if cli.Version {
		fmt.Println(version)
		os.Exit(0)
	}

	switch ctx.Command() {
	case "http":
		runHTTP(&cli.HTTP)
	default:
		runSnap(&cli.Snap)
	}
}

func runSnap(cmd *SnapCmd) {
	if cmd.ListThemes {
		names := make([]string, 0, len(render.Themes))
		for k := range render.Themes {
			names = append(names, k)
		}
		sort.Strings(names)
		fmt.Println("Available themes:")
		for _, name := range names {
			note := ""
			switch {
			case name == "vscode-dark":
				note = "  (default)"
			case render.Themes[name].HTTPOnly:
				note = "  (`pocing http` only \u2014 not for code)"
			}
			fmt.Printf("    %-20s  %s%s\n", name, render.Themes[name].Name, note)
		}
		os.Exit(0)
	}

	theme := resolveTheme(cmd.Theme)

	// Asking for a column width and not getting wrapping is never what anyone
	// meant, and the flag used to be silently ignored without --wrap.
	cmd.WrapWidth = clampWrapWidth(cmd.WrapWidth, false)
	if cmd.WrapWidth > 0 {
		cmd.Wrap = true
	}

	var src []byte
	var err error
	if cmd.File != "" {
		src, err = os.ReadFile(cmd.File)
	} else {
		// Not /dev/stdin: that path does not exist on Windows.
		src, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "pocing: read input: %v\n", err)
		os.Exit(1)
	}

	title := cmd.Title
	if title == "" && cmd.File != "" {
		// Keep the relative path — in evidence, auth/views.py beats views.py.
		// Absolute paths leak the analyst's filesystem, so trim those.
		if filepath.IsAbs(cmd.File) {
			title = filepath.Base(cmd.File)
		} else {
			title = filepath.ToSlash(filepath.Clean(cmd.File))
		}
	}
	if title == "" {
		title = "code"
	}

	all, err := highlight.Tokenize(string(src), cmd.Lang, cmd.File, theme.Chroma)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pocing: tokenize: %v\n", err)
		os.Exit(1)
	}

	hlLines, err := highlight.ParseHighlightSpec(cmd.Highlight)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pocing: parse highlight: %v\n", err)
		os.Exit(1)
	}

	// Merge before the guard: -m supplies marks just as -H does, and --context
	// centers on either.
	hlLines = hlLines.Merge(findMarks(all, cmd.Mark))
	if cmd.Context > 0 && len(hlLines) == 0 {
		fmt.Fprintf(os.Stderr, "pocing: --context needs --highlight or --mark to center on\n")
		os.Exit(1)
	}

	lineSpec := resolveLineSpec("--lines", cmd.Lines, hlLines, cmd.Context)
	lines := highlight.FilterLines(all, lineSpec)
	// A spec that matches nothing would emit a blank PNG at exit 0 — exactly the
	// kind of thing that reaches a report unnoticed.
	if len(lines) == 0 {
		fmt.Fprintf(os.Stderr, "pocing: no lines matched (file has %d lines)\n", len(all))
		os.Exit(1)
	}

	img, err := render.Render(&render.Config{
		Lines:          lines,
		Title:          title,
		HighlightLines: hlLines,
		Theme:          theme,
		Wrap:           cmd.Wrap,
		WrapWidth:      cmd.WrapWidth,
		Caption:        cmd.Label,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "pocing: render: %v\n", err)
		os.Exit(1)
	}

	writePNG(img, cmd.Output)
}

func runHTTP(cmd *HTTPCmd) {
	dark := cmd.Theme == "dark" || cmd.Theme == "burp-dark" || cmd.Theme == "burp"
	httpTheme := render.BurpLightTheme
	if dark {
		httpTheme = render.BurpTheme
	}
	cmd.WrapWidth = clampWrapWidthFor(cmd.WrapWidth, true, httpTheme)
	maxRows := cmd.MaxLines
	switch {
	case maxRows == 0:
		maxRows = -1 // no cap
	case maxRows < 0:
		fmt.Fprintf(os.Stderr, "pocing: --max-lines must be 0 or positive\n")
		os.Exit(1)
	case maxRows < 5:
		maxRows = 5
	}
	caret := func(n int) int {
		if n <= 0 {
			return -1 // no band
		}
		return n
	}

	reqSrc, err := os.ReadFile(cmd.Request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pocing: read request: %v\n", err)
		os.Exit(1)
	}
	resSrc, err := os.ReadFile(cmd.Response)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pocing: read response: %v\n", err)
		os.Exit(1)
	}
	// An empty capture is a mistake upstream, not evidence of anything.
	if len(bytes.TrimSpace(reqSrc)) == 0 {
		fmt.Fprintf(os.Stderr, "pocing: %s is empty\n", cmd.Request)
		os.Exit(1)
	}
	if len(bytes.TrimSpace(resSrc)) == 0 {
		fmt.Fprintf(os.Stderr, "pocing: %s is empty\n", cmd.Response)
		os.Exit(1)
	}

	reqHL, err := highlight.ParseHighlightSpec(cmd.ReqHL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pocing: parse --hr: %v\n", err)
		os.Exit(1)
	}
	resHL, err := highlight.ParseHighlightSpec(cmd.ResHL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pocing: parse --hs: %v\n", err)
		os.Exit(1)
	}

	// --mark's matches are found while rendering, so they count here too.
	if cmd.Context > 0 && len(reqHL) == 0 && len(resHL) == 0 && len(cmd.Mark) == 0 {
		fmt.Fprintf(os.Stderr, "pocing: --context needs --hr, --hs or --mark to center on\n")
		os.Exit(1)
	}
	if cmd.Context > 0 {
		if cmd.ReqLines != "" {
			fmt.Fprintf(os.Stderr, "pocing: --context overrides --nr; ignoring it\n")
		}
		if cmd.ResLines != "" {
			fmt.Fprintf(os.Stderr, "pocing: --context overrides --ns; ignoring it\n")
		}
	}

	img, err := render.RenderHTTP(&render.HTTPConfig{
		Request:       string(reqSrc),
		Response:      string(resSrc),
		RequestHL:     reqHL,
		ResponseHL:    resHL,
		Mark:          cmd.Mark,
		RequestLines:  parseLineSpec("--nr", cmd.ReqLines),
		ResponseLines: parseLineSpec("--ns", cmd.ResLines),
		Context:       cmd.Context,
		WrapWidth:     cmd.WrapWidth,
		Caption:       cmd.Label,
		Dark:          dark,
		Bold:          cmd.Bold,
		RawHTML:       cmd.RawHTML,
		MaxRows:       maxRows,
		RequestCaret:  caret(cmd.CaretReq),
		ResponseCaret: caret(cmd.CaretRes),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "pocing: render: %v\n", err)
		os.Exit(1)
	}

	writePNG(img, cmd.Output)
}

// resolveLineSpec picks which lines to render: --context expands around each
// highlighted line, otherwise the explicit spec (empty = every line). Shared by
// both commands so `pocing http` trims the same way `pocing` does.
//
// Context only applies to a panel that has highlights to center on. In HTTP mode
// one side often carries none — the request in a response-only finding — and
// that side must still render, not abort the run.
func resolveLineSpec(flag, spec string, hl highlight.Marks, context int) []int {
	if s := highlight.ContextSpec(hl, context); s != nil {
		if spec != "" {
			fmt.Fprintf(os.Stderr, "pocing: --context overrides %s; ignoring it\n", flag)
		}
		return s
	}
	return parseLineSpec(flag, spec)
}

// parseLineSpec exits with the flag named, so an error points at what to edit.
func parseLineSpec(flag, spec string) []int {
	parsed, err := highlight.ParseLineSpec(spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pocing: parse %s: %v\n", flag, err)
		os.Exit(1)
	}
	return parsed
}

// findMarks locates every literal the analyst passed with --mark. Anything that
// matches nothing is reported: a silently unmarked screenshot is the failure
// mode this whole tool exists to avoid.
func findMarks(lines []highlight.LineTokens, needles []string) highlight.Marks {
	var out highlight.Marks
	for _, needle := range needles {
		found := highlight.FindLiteral(lines, needle)
		if found == nil {
			fmt.Fprintf(os.Stderr, "pocing: warning: --mark %q matched nothing\n", needle)
			continue
		}
		out = out.Merge(found)
	}
	return out
}

// clampWrapWidth reports a too-narrow width once, at the flag level. Render
// clamps silently too, but doing it here keeps a two-panel HTTP run from
// printing the same complaint for each panel.
func clampWrapWidth(w int, httpMode bool) int { return clampWrapWidthFor(w, httpMode, nil) }

func clampWrapWidthFor(w int, httpMode bool, theme *render.Theme) int {
	const minCols = 20
	if w <= 0 {
		return w
	}
	if w < minCols {
		fmt.Fprintf(os.Stderr, "pocing: --wrap-width %d is below the %d-column minimum; using %d\n",
			w, minCols, minCols)
		return minCols
	}
	if max := render.MaxWrapColsFor(httpMode, theme); w > max {
		fmt.Fprintf(os.Stderr, "pocing: --wrap-width %d is wider than the canvas; wrapping at %d columns\n", w, max)
		return max
	}
	return w
}

func resolveTheme(name string) *render.Theme {
	theme, ok := render.Themes[strings.ToLower(name)]
	if !ok {
		fmt.Fprintf(os.Stderr, "pocing: unknown theme %q. Run --list-themes to see options.\n", name)
		os.Exit(1)
	}
	// Its style only covers the HTTP lexer's token types, so source code comes
	// out flat — every keyword falling back to the plain text color.
	if theme.HTTPOnly {
		fmt.Fprintf(os.Stderr,
			"pocing: theme %q is only for `pocing http`; source code renders without highlighting.\n"+
				"       Use vscode-dark (default) or run --list-themes.\n", name)
		os.Exit(1)
	}
	return theme
}

// writePNG encodes to a temporary file and renames it into place.
//
// Writing straight to path would truncate whatever is already there before
// encoding, and os.Exit skips deferred closes — so a failure part-way left a
// corrupt PNG where a good piece of evidence used to be.
func writePNG(img image.Image, path string) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".pocing-*.png")
	if err != nil {
		fmt.Fprintf(os.Stderr, "pocing: create output: %v\n", err)
		os.Exit(1)
	}
	tmpName := tmp.Name()

	fail := func(format string, args ...any) {
		tmp.Close()
		os.Remove(tmpName)
		fmt.Fprintf(os.Stderr, "pocing: "+format+"\n", args...)
		os.Exit(1)
	}
	if err := png.Encode(tmp, img); err != nil {
		fail("encode png: %v", err)
	}
	if err := tmp.Close(); err != nil {
		fail("write output: %v", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		fmt.Fprintf(os.Stderr, "pocing: save output: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("pocing: wrote %s\n", path)
}
