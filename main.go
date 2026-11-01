package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"unsafe"

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

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	procLoadImageW  = user32.NewProc("LoadImageW")
	procSendMessage = user32.NewProc("SendMessageW")
	procDestroyIcon = user32.NewProc("DestroyIcon")
)

var lightParser = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		highlighting.NewHighlighting(
			highlighting.WithStyle("github"),
			highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
		),
	),
)

var darkParser = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		highlighting.NewHighlighting(
			highlighting.WithStyle("github-dark"),
			highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
		),
	),
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

func renderMarkdown(source string) string {
	return sanitizeMarkdown(lightParser, source)
}

func renderMarkdownDark(source string) string {
	return sanitizeMarkdown(darkParser, source)
}

func readAsset(name string) string {
	b, err := assets.ReadFile(name)
	if err != nil {
		log.Fatal(err)
	}
	return string(b)
}

func setWindowIcon(hwnd uintptr, iconData []byte) (func(), error) {
	file, err := os.CreateTemp("", "emdee-icon-*.ico")
	if err != nil {
		return nil, err
	}

	iconPath := file.Name()
	defer os.Remove(iconPath)

	if _, err := file.Write(iconData); err != nil {
		file.Close()
		return nil, err
	}
	file.Close()

	path, err := syscall.UTF16PtrFromString(iconPath)
	if err != nil {
		return nil, err
	}

	const (
		imageIcon      = 1
		lrLoadFromFile = 0x10
		lrDefaultSize  = 0x40
		wmSetIcon      = 0x80
		iconSmall      = 0
		iconBig        = 1
	)

	handle, _, callErr := procLoadImageW.Call(
		0,
		uintptr(unsafe.Pointer(path)),
		imageIcon,
		0,
		0,
		lrLoadFromFile|lrDefaultSize,
	)

	if handle == 0 {
		return nil, callErr
	}

	procSendMessage.Call(hwnd, wmSetIcon, iconSmall, handle)
	procSendMessage.Call(hwnd, wmSetIcon, iconBig, handle)

	return func() {
		procDestroyIcon.Call(handle)
	}, nil
}

func main() {
	w := webview.New(false)
	if w == nil {
		log.Fatal("Could not create WebView. Check Microsoft WebView2 Runtime.")
	}

	var cleanupIcon func()

	defer func() {
		w.Destroy()
		if cleanupIcon != nil {
			cleanupIcon()
		}
	}()

	w.SetTitle("Emdee Viewer - Markdown File Viewer")
	w.SetSize(1180, 820, webview.HintNone)

	iconData, err := assets.ReadFile("assets/app.ico")
	if err == nil {
		cleanupIcon, err = setWindowIcon(uintptr(w.Window()), iconData)
		if err != nil {
			log.Printf("Could not set window icon: %v", err)
		}
	}

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
			parts := strings.FieldsFunc(name, func(r rune) bool {
				return r == '/' || r == '\\'
			})

			if len(parts) > 0 {
				name = parts[len(parts)-1]
			}

			initial := fmt.Sprintf(
				`window.__INITIAL_DOCUMENT__ = {name:%q, source:%q};`,
				name,
				string(data),
			)

			page = strings.Replace(page, "/*__INITIAL_DOCUMENT__*/", initial, 1)
		}
	}

	w.SetHtml(page)
	w.Run()
}
