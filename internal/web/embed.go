// Package web embeds compiled static assets so the server binary has no
// runtime filesystem dependency on the source tree.
//
// # Rebuilding CSS
//
// After editing templates or input.css, regenerate output.css:
//
//	go generate ./internal/web/
//	# or
//	make css
//
//go:generate ./static/css/tailwind -i ./static/css/input.css -o ./static/css/output.css
package web

import (
	"embed"
	"io/fs"
	"net/http"
)

// Files is the embedded static asset tree.
//
//go:embed static
var Files embed.FS

// StaticFS returns an http.FileSystem rooted at the embedded static/ directory,
// ready to be passed to gin's r.StaticFS("/static", ...).
func StaticFS() http.FileSystem {
	sub, err := fs.Sub(Files, "static")
	if err != nil {
		panic("web: embedded static/ subtree missing: " + err.Error())
	}
	return http.FS(sub)
}

// StaticFile returns the bytes of a single embedded file (e.g. "robots.txt"),
// useful for serving root-level files.
func StaticFile(name string) ([]byte, error) {
	return Files.ReadFile("static/" + name)
}
