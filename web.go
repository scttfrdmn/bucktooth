// Package bucktooth provides the embedded web dashboard assets.
package bucktooth

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed web
var webFS embed.FS

// WebFileServer returns an http.Handler that serves the embedded dashboard.
func WebFileServer() http.Handler {
	sub, _ := fs.Sub(webFS, "web")
	return http.FileServer(http.FS(sub))
}
