package render

import (
	"image/color"
	"strings"
)

// fileIcon is a Nerd Font codepoint plus the color it is drawn in.
//
// Codepoints are written as escapes on purpose: a literal PUA character in the
// source is unreviewable. TestIconGlyphsExist proves every codepoint here is
// present in the embedded Symbols Nerd Font. It cannot prove a glyph is the
// *right* logo. Those were identified by eye from a rendered contact sheet —
// which is how e609 was caught drawing a down-arrow for Markdown. Render your
// own before adding an entry:
//
//	ICON_SHEET=/tmp go test ./render/ -run ContactSheet -v
type fileIcon struct {
	glyph rune
	color color.NRGBA
}

// Shared colors for categories with no brand of their own.
var (
	cConfig  = rgb(0x8f, 0x9b, 0xa8)
	cData    = rgb(0xd7, 0x99, 0x21)
	cDocs    = rgb(0x51, 0x9a, 0xba)
	cImage   = rgb(0xa0, 0x74, 0xc4)
	cMedia   = rgb(0xe3, 0x7e, 0x33)
	cArchive = rgb(0xb0, 0x70, 0x19)
	cSecret  = rgb(0xf1, 0x4c, 0x4c)
	cShell   = rgb(0x4e, 0xaa, 0x25)
	cPlain   = rgb(0xb0, 0xb0, 0xb0)
	cGeneric = rgb(0x89, 0x89, 0x89)
)

// Glyphs reused across many entries.
const (
	gCode      = '\ue60e' // <> chevrons
	gGear      = '\ue615' // settings
	gSliders   = '\ue690' // tuned settings
	gText      = '\ue612' // plain lines
	gLock      = '\ue672'
	gKeys      = '\ue60a'
	gImage     = '\ue60d'
	gZip       = '\ue6aa'
	gDB        = '\ue64d'
	gGit       = '\ue65d'
	gMoustache = '\ue60f' // handlebars / mustache family
	gLambda    = '\ue6b1' // functional languages with no logo of their own
	gFile      = '\uf15b' // nf-fa-file
)

var defaultIcon = fileIcon{gFile, cGeneric}

// iconByExt is keyed on a lowercased extension without its dot.
var iconByExt = map[string]fileIcon{
	// ── Systems ──────────────────────────────────────────────────────────
	"go":    {'\ue627', rgb(0x00, 0xad, 0xd8)}, // gopher
	"rs":    {'\ue7a8', rgb(0xde, 0x80, 0x27)}, // rust gear
	"zig":   {'\ue6a9', rgb(0xf7, 0xa4, 0x1d)},
	"c":     {'\ue649', rgb(0x00, 0x5f, 0x87)},
	"h":     {'\ue649', rgb(0x00, 0x5f, 0x87)},
	"cpp":   {'\ue646', rgb(0x65, 0x9a, 0xd2)},
	"cc":    {'\ue646', rgb(0x65, 0x9a, 0xd2)},
	"cxx":   {'\ue646', rgb(0x65, 0x9a, 0xd2)},
	"hpp":   {'\ue646', rgb(0x65, 0x9a, 0xd2)},
	"hh":    {'\ue646', rgb(0x65, 0x9a, 0xd2)},
	"hxx":   {'\ue646', rgb(0x65, 0x9a, 0xd2)},
	"m":     {'\ue711', rgb(0x43, 0x8e, 0xff)}, // objective-c
	"mm":    {'\ue711', rgb(0x43, 0x8e, 0xff)},
	"swift": {'\ue755', rgb(0xf0, 0x51, 0x38)},
	"d":     {'\ue651', rgb(0xba, 0x59, 0x5e)},
	"nim":   {'\ue6ae', rgb(0xff, 0xe9, 0x53)},
	"v":     {'\ue6ac', rgb(0x5d, 0x87, 0xbf)},
	"asm":   {'\ue637', rgb(0xd0, 0xb0, 0x76)},
	"s":     {'\ue637', rgb(0xd0, 0xb0, 0x76)},

	// ── JVM / .NET ───────────────────────────────────────────────────────
	"java":   {'\ue738', rgb(0xb0, 0x70, 0x19)},
	"kt":     {'\ue607', rgb(0xa9, 0x7b, 0xff)}, // kotlin
	"kts":    {'\ue607', rgb(0xa9, 0x7b, 0xff)},
	"scala":  {'\ue737', rgb(0xdc, 0x32, 0x2f)},
	"sc":     {'\ue737', rgb(0xdc, 0x32, 0x2f)},
	"groovy": {'\ue775', rgb(0x42, 0x98, 0xb8)},
	"gradle": {'\ue660', rgb(0x7f, 0x9e, 0xa8)}, // gradle elephant
	"clj":    {'\ue768', rgb(0x91, 0xdc, 0x47)},
	"cljs":   {'\ue768', rgb(0x91, 0xdc, 0x47)},
	"cljc":   {'\ue768', rgb(0x91, 0xdc, 0x47)},
	"edn":    {'\ue768', rgb(0x91, 0xdc, 0x47)},
	"cs":     {'\ue648', rgb(0x9a, 0x4f, 0x96)}, // c#
	"csproj": {'\ue77f', rgb(0x51, 0x8c, 0xff)}, // .net
	"fs":     {gLambda, rgb(0x37, 0x8b, 0xba)},  // f#: no logo in font
	"fsx":    {gLambda, rgb(0x37, 0x8b, 0xba)},
	"vb":     {'\ue77f', rgb(0x51, 0x8c, 0xff)},

	// ── Functional ───────────────────────────────────────────────────────
	"hs":  {'\ue777', rgb(0x5e, 0x50, 0x86)}, // haskell
	"lhs": {'\ue777', rgb(0x5e, 0x50, 0x86)},
	"ml":  {gLambda, rgb(0xec, 0x68, 0x13)}, // ocaml: no logo in font
	"mli": {gLambda, rgb(0xec, 0x68, 0x13)},
	"ex":  {'\ue62d', rgb(0xa0, 0x74, 0xc4)}, // elixir droplet
	"exs": {'\ue62d', rgb(0xa0, 0x74, 0xc4)},
	// Erlang's brand maroon (#a2003e) is too dark to read against a tab bar;
	// lifted into the same hue at a luminance that survives the background.
	"erl": {'\ue7b1', rgb(0xd4, 0x35, 0x6b)},
	"hrl": {'\ue7b1', rgb(0xd4, 0x35, 0x6b)},

	// ── Scripting ────────────────────────────────────────────────────────
	"py":    {'\ue606', rgb(0x3d, 0x79, 0xd8)},
	"pyi":   {'\ue606', rgb(0x3d, 0x79, 0xd8)},
	"pyw":   {'\ue606', rgb(0x3d, 0x79, 0xd8)},
	"ipynb": {'\ue678', rgb(0xf3, 0x77, 0x26)}, // jupyter notebook
	"rb":    {'\ue739', rgb(0xcc, 0x34, 0x2d)},
	"erb":   {'\ue739', rgb(0xcc, 0x34, 0x2d)},
	"php":   {'\ue73d', rgb(0x77, 0x7b, 0xb4)},
	"pl":    {'\ue67e', rgb(0x02, 0x98, 0xc3)}, // perl camel
	"pm":    {'\ue67e', rgb(0x02, 0x98, 0xc3)},
	"lua":   {'\ue620', rgb(0x4e, 0x7e, 0xe8)},
	"r":     {'\ue68a', rgb(0x27, 0x6d, 0xc3)},
	"rmd":   {'\ue68a', rgb(0x27, 0x6d, 0xc3)},
	"jl":    {'\ue624', rgb(0x95, 0x58, 0xb2)}, // julia
	"dart":  {'\ue798', rgb(0x01, 0x75, 0xc2)},
	"vim":   {'\ue62b', rgb(0x01, 0x98, 0x33)},

	// ── Web ──────────────────────────────────────────────────────────────
	"js":     {'\ue74e', rgb(0xf1, 0xbe, 0x42)},
	"mjs":    {'\ue74e', rgb(0xf1, 0xbe, 0x42)},
	"cjs":    {'\ue74e', rgb(0xf1, 0xbe, 0x42)},
	"jsx":    {'\ue625', rgb(0x61, 0xda, 0xfb)}, // react atom
	"ts":     {'\ue628', rgb(0x02, 0x7a, 0xcb)},
	"mts":    {'\ue628', rgb(0x02, 0x7a, 0xcb)},
	"cts":    {'\ue628', rgb(0x02, 0x7a, 0xcb)},
	"tsx":    {'\ue625', rgb(0x61, 0xda, 0xfb)},
	"vue":    {'\ue6a0', rgb(0x41, 0xb8, 0x83)},
	"svelte": {'\ue697', rgb(0xff, 0x3e, 0x00)},
	"astro":  {'\ue6b3', rgb(0xff, 0x5d, 0x01)},
	"coffee": {'\ue61b', rgb(0x6f, 0x4e, 0x37)},
	"html":   {'\ue736', rgb(0xe3, 0x4c, 0x26)}, // html5 shield
	"htm":    {'\ue736', rgb(0xe3, 0x4c, 0x26)},
	"xhtml":  {'\ue736', rgb(0xe3, 0x4c, 0x26)},
	"css":    {'\ue749', rgb(0x15, 0x72, 0xb6)}, // css3 shield
	"scss":   {'\ue74b', rgb(0xcf, 0x64, 0x9a)}, // sass
	"sass":   {'\ue74b', rgb(0xcf, 0x64, 0x9a)},
	"less":   {'\ue758', rgb(0x5a, 0x7f, 0xc0)},
	"styl":   {'\ue759', rgb(0xb3, 0xd1, 0x07)},
	"wasm":   {'\ue6a2', rgb(0x65, 0x4f, 0xf0)},
	"wat":    {'\ue6a2', rgb(0x65, 0x4f, 0xf0)},

	// ── Templates ────────────────────────────────────────────────────────
	"hbs":        {gMoustache, rgb(0xf0, 0x77, 0x2b)},
	"handlebars": {gMoustache, rgb(0xf0, 0x77, 0x2b)},
	"mustache":   {gMoustache, rgb(0xf0, 0x77, 0x2b)},
	"pug":        {'\ue686', rgb(0xa8, 0x64, 0x54)},
	"jade":       {'\ue686', rgb(0xa8, 0x64, 0x54)},
	"ejs":        {gCode, rgb(0xf0, 0x77, 0x2b)},
	"twig":       {gCode, rgb(0x78, 0xa8, 0x3b)},
	"jinja":      {gCode, rgb(0xb4, 0x1a, 0x1a)},
	"j2":         {gCode, rgb(0xb4, 0x1a, 0x1a)},
	"liquid":     {gCode, rgb(0x67, 0xb2, 0x8f)},
	"blade":      {'\ue73f', rgb(0xf0, 0x51, 0x38)}, // laravel
	"jsp":        {'\ue738', rgb(0xb0, 0x70, 0x19)},
	"aspx":       {'\ue77f', rgb(0x51, 0x8c, 0xff)},
	"cshtml":     {'\ue77f', rgb(0x51, 0x8c, 0xff)},
	"gohtml":     {'\ue627', rgb(0x00, 0xad, 0xd8)},
	"gotmpl":     {'\ue627', rgb(0x00, 0xad, 0xd8)},

	// ── Shell ────────────────────────────────────────────────────────────
	"sh":   {'\ue795', cShell},
	"bash": {'\ue795', cShell},
	"zsh":  {'\ue795', cShell},
	"ksh":  {'\ue795', cShell},
	"fish": {'\ue795', cShell},
	"awk":  {'\ue795', cShell},
	"ps1":  {'\uebc7', rgb(0x53, 0x91, 0xfe)}, // powershell
	"psm1": {'\uebc7', rgb(0x53, 0x91, 0xfe)},
	"psd1": {'\uebc7', rgb(0x53, 0x91, 0xfe)},
	"bat":  {'\ue62a', rgb(0xc1, 0xf1, 0x2e)}, // windows
	"cmd":  {'\ue62a', rgb(0xc1, 0xf1, 0x2e)},

	// ── Infra ────────────────────────────────────────────────────────────
	"tf":      {'\ue69a', rgb(0x7b, 0x42, 0xbc)}, // terraform
	"tfvars":  {'\ue69a', rgb(0x7b, 0x42, 0xbc)},
	"tfstate": {'\ue69a', rgb(0x7b, 0x42, 0xbc)},
	"hcl":     {'\ue69a', rgb(0x7b, 0x42, 0xbc)},
	"nix":     {'\uf313', rgb(0x52, 0x77, 0xc3)}, // nixos snowflake
	"pp":      {gGear, rgb(0xff, 0xa6, 0x1a)},    // puppet: no logo in font
	"service": {gGear, cConfig},

	// ── Captured traffic ─────────────────────────────────────────────────
	"http":  {gCode, rgb(0x4e, 0xaa, 0xd8)},
	"https": {gCode, rgb(0x4e, 0xaa, 0xd8)},
	"rest":  {gCode, rgb(0x4e, 0xaa, 0xd8)},
	"har":   {gCode, rgb(0x4e, 0xaa, 0xd8)},
	"pcap":  {gCode, rgb(0x4e, 0xaa, 0xd8)},
	"burp":  {gCode, rgb(0xe8, 0x74, 0x3b)},

	// ── Data / query ─────────────────────────────────────────────────────
	"sql":     {'\ue75b', rgb(0xff, 0xd0, 0x00)},
	"psql":    {'\ue76e', rgb(0x33, 0x67, 0x91)}, // postgres
	"pgsql":   {'\ue76e', rgb(0x33, 0x67, 0x91)},
	"mysql":   {'\ue704', rgb(0x44, 0x79, 0xa1)},
	"db":      {gDB, cData},
	"sqlite":  {gDB, cData},
	"sqlite3": {gDB, cData},
	"graphql": {'\ue662', rgb(0xe1, 0x00, 0x98)},
	"gql":     {'\ue662', rgb(0xe1, 0x00, 0x98)},
	"sol":     {'\ue656', rgb(0x62, 0x7e, 0xea)}, // ethereum
	"proto":   {gCode, rgb(0x42, 0x85, 0xf4)},
	"prisma":  {gDB, rgb(0x5a, 0x67, 0xd8)},

	// ── Markup / config ──────────────────────────────────────────────────
	"json":       {'\ue60b', rgb(0xf1, 0xbe, 0x42)}, // braces
	"jsonc":      {'\ue60b', rgb(0xf1, 0xbe, 0x42)},
	"json5":      {'\ue60b', rgb(0xf1, 0xbe, 0x42)},
	"yaml":       {gGear, rgb(0xcb, 0x17, 0x1e)},
	"yml":        {gGear, rgb(0xcb, 0x17, 0x1e)},
	"toml":       {'\ue6b2', rgb(0x9c, 0x42, 0x21)},
	"ini":        {gSliders, cConfig},
	"cfg":        {gSliders, cConfig},
	"conf":       {gSliders, cConfig},
	"properties": {gSliders, cConfig},
	"env":        {gKeys, cSecret},
	"xml":        {gCode, rgb(0xe3, 0x7e, 0x33)},
	"xsl":        {gCode, rgb(0xe3, 0x7e, 0x33)},
	"xsd":        {gCode, rgb(0xe3, 0x7e, 0x33)},
	"plist":      {gCode, rgb(0xe3, 0x7e, 0x33)},
	"svg":        {gImage, cImage},
	"lock":       {gLock, rgb(0xe5, 0xc0, 0x7b)},

	// ── Docs ─────────────────────────────────────────────────────────────
	"md":       {'\ue73e', cDocs}, // markdown M-down
	"markdown": {'\ue73e', cDocs},
	"mdx":      {'\ue73e', cDocs},
	"rst":      {gText, cDocs},
	"adoc":     {gText, cDocs},
	"tex":      {'\ue69b', rgb(0x3d, 0x61, 0x17)},
	"bib":      {'\ue69b', rgb(0x3d, 0x61, 0x17)},
	"txt":      {gText, cPlain},
	"text":     {gText, cPlain},
	"log":      {'\uf4ed', rgb(0xa0, 0xa0, 0xa0)}, // scroll
	"csv":      {'\ue64a', cShell},
	"tsv":      {'\ue64a', cShell},
	"pdf":      {'\ue67d', rgb(0xe5, 0x25, 0x2a)},
	"doc":      {'\ue6a5', rgb(0x2b, 0x57, 0x97)},
	"docx":     {'\ue6a5', rgb(0x2b, 0x57, 0x97)},
	"xls":      {'\ue6a6', rgb(0x1f, 0x6f, 0x43)},
	"xlsx":     {'\ue6a6', rgb(0x1f, 0x6f, 0x43)},
	"patch":    {gGit, rgb(0xf1, 0x4c, 0x28)},
	"diff":     {gGit, rgb(0xf1, 0x4c, 0x28)},

	// ── Secrets: worth spotting at a glance in evidence ───────────────────
	"pem":  {gKeys, cSecret},
	"key":  {gKeys, cSecret},
	"pub":  {gKeys, cSecret},
	"crt":  {gLock, cSecret},
	"cer":  {gLock, cSecret},
	"der":  {gLock, cSecret},
	"p12":  {gLock, cSecret},
	"pfx":  {gLock, cSecret},
	"jks":  {gLock, cSecret},
	"kdbx": {gLock, cSecret},
	"asc":  {gKeys, cSecret},
	"gpg":  {gKeys, cSecret},

	// ── Binary / media ───────────────────────────────────────────────────
	"zip":   {gZip, cArchive},
	"tar":   {gZip, cArchive},
	"gz":    {gZip, cArchive},
	"tgz":   {gZip, cArchive},
	"bz2":   {gZip, cArchive},
	"xz":    {gZip, cArchive},
	"zst":   {gZip, cArchive},
	"7z":    {gZip, cArchive},
	"rar":   {gZip, cArchive},
	"jar":   {'\ue738', rgb(0xb0, 0x70, 0x19)},
	"war":   {'\ue738', rgb(0xb0, 0x70, 0x19)},
	"apk":   {'\ue70e', rgb(0x3d, 0xdc, 0x84)}, // android
	"exe":   {'\ue62a', rgb(0x00, 0xa4, 0xef)},
	"dll":   {'\ue62a', rgb(0x00, 0xa4, 0xef)},
	"so":    {'\ue712', rgb(0xcc, 0xcc, 0xcc)}, // linux
	"dylib": {'\ue711', rgb(0xcc, 0xcc, 0xcc)},
	"png":   {gImage, cImage},
	"jpg":   {gImage, cImage},
	"jpeg":  {gImage, cImage},
	"gif":   {gImage, cImage},
	"webp":  {gImage, cImage},
	"ico":   {gImage, cImage},
	"bmp":   {gImage, cImage},
	"mp4":   {'\ue602', cMedia}, // play
	"mov":   {'\ue602', cMedia},
	"mkv":   {'\ue602', cMedia},
	"avi":   {'\ue602', cMedia},
	"mp3":   {'\ue638', cMedia}, // speaker
	"wav":   {'\ue638', cMedia},
	"flac":  {'\ue638', cMedia},
	"ogg":   {'\ue638', cMedia},
	"ttf":   {'\ue659', rgb(0xcc, 0xcc, 0xcc)}, // AA
	"otf":   {'\ue659', rgb(0xcc, 0xcc, 0xcc)},
	"woff":  {'\ue659', rgb(0xcc, 0xcc, 0xcc)},
	"woff2": {'\ue659', rgb(0xcc, 0xcc, 0xcc)},
}

// iconByName covers files whose identity is the whole name, not an extension.
var iconByName = map[string]fileIcon{
	// Containers
	"dockerfile":          {'\ue650', rgb(0x24, 0x96, 0xed)},
	"containerfile":       {'\ue650', rgb(0x24, 0x96, 0xed)},
	"docker-compose.yml":  {'\ue650', rgb(0x24, 0x96, 0xed)},
	"docker-compose.yaml": {'\ue650', rgb(0x24, 0x96, 0xed)},
	"compose.yml":         {'\ue650', rgb(0x24, 0x96, 0xed)},
	"compose.yaml":        {'\ue650', rgb(0x24, 0x96, 0xed)},
	".dockerignore":       {'\ue650', rgb(0x24, 0x96, 0xed)},

	// Build
	"makefile":       {'\uf0ad', rgb(0x6d, 0x80, 0x86)}, // wrench
	"gnumakefile":    {'\uf0ad', rgb(0x6d, 0x80, 0x86)},
	"cmakelists.txt": {'\uf0ad', rgb(0x6d, 0x80, 0x86)},
	"justfile":       {'\uf0ad', rgb(0x6d, 0x80, 0x86)},
	"rakefile":       {'\ue739', rgb(0xcc, 0x34, 0x2d)},
	"gemfile":        {'\ue739', rgb(0xcc, 0x34, 0x2d)},
	"jenkinsfile":    {'\ue66e', rgb(0xd3, 0x38, 0x33)},
	"vagrantfile":    {gGear, rgb(0x14, 0x63, 0xff)},
	"procfile":       {gGear, rgb(0x79, 0x58, 0x9f)},

	// Manifests
	"go.mod":            {'\ue627', rgb(0x00, 0xad, 0xd8)},
	"go.sum":            {'\ue627', rgb(0x00, 0xad, 0xd8)},
	"package.json":      {'\ue71e', rgb(0xcb, 0x34, 0x37)}, // npm
	"package-lock.json": {'\ue71e', rgb(0xcb, 0x34, 0x37)},
	"yarn.lock":         {'\ue71e', rgb(0x2c, 0x8e, 0xbb)},
	"pnpm-lock.yaml":    {'\ue71e', rgb(0xf6, 0x9b, 0x35)},
	"cargo.toml":        {'\ue7a8', rgb(0xde, 0x80, 0x27)},
	"cargo.lock":        {'\ue7a8', rgb(0xde, 0x80, 0x27)},
	"pyproject.toml":    {'\ue606', rgb(0x3d, 0x79, 0xd8)},
	"requirements.txt":  {'\ue606', rgb(0x3d, 0x79, 0xd8)},
	"pipfile":           {'\ue606', rgb(0x3d, 0x79, 0xd8)},
	"poetry.lock":       {'\ue606', rgb(0x3d, 0x79, 0xd8)},
	"composer.json":     {'\ue73d', rgb(0x77, 0x7b, 0xb4)},
	"composer.lock":     {'\ue73d', rgb(0x77, 0x7b, 0xb4)},
	"build.gradle":      {'\ue660', rgb(0x7f, 0x9e, 0xa8)},
	"build.gradle.kts":  {'\ue660', rgb(0x7f, 0x9e, 0xa8)},
	"pom.xml":           {'\ue738', rgb(0xc7, 0x1a, 0x36)},

	// Tooling dotfiles
	".gitignore":     {'\ue668', rgb(0xf1, 0x4c, 0x28)}, // crossed-out eye
	".gitattributes": {gGit, rgb(0xf1, 0x4c, 0x28)},
	".gitmodules":    {gGit, rgb(0xf1, 0x4c, 0x28)},
	".npmignore":     {'\ue71e', rgb(0xcb, 0x34, 0x37)},
	".npmrc":         {'\ue71e', rgb(0xcb, 0x34, 0x37)},
	".editorconfig":  {gSliders, cConfig},
	".babelrc":       {gGear, rgb(0xf5, 0xda, 0x55)},
	".eslintrc":      {gGear, rgb(0x4b, 0x32, 0xc3)},
	".prettierrc":    {gGear, rgb(0x56, 0xb3, 0xb4)},
	".bashrc":        {'\ue795', cShell},
	".zshrc":         {'\ue795', cShell},
	".profile":       {'\ue795', cShell},
	".vimrc":         {'\ue62b', rgb(0x01, 0x98, 0x33)},
	".htaccess":      {'\ue72b', rgb(0xd2, 0x2c, 0x2c)}, // apache feather
	"nginx.conf":     {gSliders, rgb(0x00, 0x9f, 0x63)},

	// Credential files
	".env":            {gKeys, cSecret},
	".netrc":          {gKeys, cSecret},
	".pgpass":         {gKeys, cSecret},
	"id_rsa":          {gKeys, cSecret},
	"id_ed25519":      {gKeys, cSecret},
	"authorized_keys": {gKeys, cSecret},
	"known_hosts":     {gKeys, cSecret},
	"credentials":     {gLock, cSecret},
	"shadow":          {gLock, cSecret},
	"passwd":          {gLock, cSecret},

	"license": {gText, cDocs},
}

// iconFor resolves the tab icon for a title, which may be a path.
func iconFor(title string) fileIcon {
	name := strings.ToLower(title)
	if i := strings.LastIndexAny(name, "/\\"); i >= 0 {
		name = name[i+1:]
	}

	if ic, ok := iconByName[name]; ok {
		return ic
	}
	// Dockerfile.prod, .env.local and friends.
	switch {
	case strings.HasPrefix(name, "dockerfile"):
		return iconByName["dockerfile"]
	case strings.HasPrefix(name, ".env"):
		return iconByName[".env"]
	}

	// i > 0, so a leading dot marks a whole name (".env") not an extension.
	if i := strings.LastIndex(name, "."); i > 0 {
		if ic, ok := iconByExt[name[i+1:]]; ok {
			return ic
		}
	}
	return defaultIcon
}
