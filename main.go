package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"strings"

	webview "github.com/abemedia/go-webview"
	_ "github.com/abemedia/go-webview/embedded"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
)

//go:embed assets/*
var assets embed.FS

var lightParser = goldmark.New(
	goldmark.WithExtensions(extension.GFM, highlighting.NewHighlighting(highlighting.WithStyle("github"), highlighting.WithFormatOptions(chromahtml.WithClasses(true)))),
)
var darkParser = goldmark.New(
	goldmark.WithExtensions(extension.GFM, highlighting.NewHighlighting(highlighting.WithStyle("github-dark"), highlighting.WithFormatOptions(chromahtml.WithClasses(true)))),
)

func sanitizeMarkdown(parser goldmark.Markdown, source string) string {
	var out strings.Builder
	if err := parser.Convert([]byte(source), &out); err != nil {
		return "<p>Unable to render this Markdown document.</p>"
	}
	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class").OnElements("span", "code", "pre", "div", "table", "thead", "tbody", "tr", "th", "td")
	policy.AllowAttrs("style").OnElements("span")
	return policy.Sanitize(out.String())
}

func renderMarkdown(source string) string     { return sanitizeMarkdown(lightParser, source) }
func renderMarkdownDark(source string) string { return sanitizeMarkdown(darkParser, source) }

func readAsset(name string) string {
	b, err := assets.ReadFile(name)
	if err != nil {
		log.Fatal(err)
	}
	return string(b)
}

func main() {
	w := webview.New(false)
	if w == nil {
		log.Fatal("Could not create WebView. Check Microsoft WebView2 Runtime.")
	}
	defer w.Destroy()
	w.SetTitle("Emdee Viewer - Markdown File Viewer")
	w.SetSize(1180, 820, webview.HintNone)
	if err := w.Bind("renderMarkdown", renderMarkdown); err != nil {
		log.Fatal(err)
	}
	if err := w.Bind("renderMarkdownDark", renderMarkdownDark); err != nil {
		log.Fatal(err)
	}
	if err := w.Bind("exitApp", func() { os.Exit(0) }); err != nil {
		log.Fatal(err)
	}

	page := readAsset("assets/index.html")
	page = strings.Replace(page, "/*__APP_CSS__*/", readAsset("assets/styles.css"), 1)
	page = strings.Replace(page, "/*__APP_JS__*/", readAsset("assets/app.js"), 1)

	if len(os.Args) > 1 {
		if data, err := os.ReadFile(os.Args[1]); err == nil {
			name := os.Args[1]
			parts := strings.FieldsFunc(name, func(r rune) bool { return r == '/' || r == '\\' })
			if len(parts) > 0 {
				name = parts[len(parts)-1]
			}
			initial := fmt.Sprintf(`window.__INITIAL_DOCUMENT__ = {name:%q, source:%q};`, name, string(data))
			page = strings.Replace(page, "/*__INITIAL_DOCUMENT__*/", initial, 1)
		}
	}
	w.SetHtml(page)
	w.Run()
}
