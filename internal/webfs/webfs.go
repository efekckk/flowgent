// Package webfs serves the built React SPA. The `dist/` subdirectory is
// expected to be populated before Go build — typically by `npm run build`
// followed by a copy into this package. See README for the build pipeline.
package webfs

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// Handler returns an http.Handler that serves the embedded SPA. Requests
// matching a file in dist/ are served directly; everything else falls back
// to index.html so client-side routes resolve correctly.
func Handler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("webfs: missing dist subdir: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			// Serve root via the file server — it will pick up index.html.
			fileServer.ServeHTTP(w, r)
			return
		}
		f, err := sub.Open(path)
		if err != nil {
			// Unknown path: serve index.html directly so SPA routing works.
			data, readErr := fs.ReadFile(sub, "index.html")
			if readErr != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
			return
		}
		_ = f.Close()
		fileServer.ServeHTTP(w, r)
	})
}
