package static

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:webapp
var webappEmbed embed.FS

//go:embed all:widget
var widgetEmbed embed.FS

// WebappSub returns the built SPA as an fs.FS (paths like assets/..., index.html).
func WebappSub() (fs.FS, error) {
	return fs.Sub(webappEmbed, "webapp")
}

// WebappFS returns the built SPA filesystem (contents of webapp/).
func WebappFS() (http.FileSystem, error) {
	sub, err := WebappSub()
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}

// WidgetFS returns the built widget filesystem (contents of widget/).
func WidgetFS() (http.FileSystem, error) {
	sub, err := fs.Sub(widgetEmbed, "widget")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}

// WebappIndex returns index.html for SPA fallback.
func WebappIndex() ([]byte, error) {
	return webappEmbed.ReadFile("webapp/index.html")
}

// WidgetSnippet returns the HTML snippet injected into TMDB pages.
func WidgetSnippet() ([]byte, error) {
	return widgetEmbed.ReadFile("widget/snippet.html")
}
