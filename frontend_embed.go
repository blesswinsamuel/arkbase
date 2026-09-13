package arkbase

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:frontend/dist
var FrontendFS embed.FS

func SPAHandler() http.Handler {
	distFS, err := fs.Sub(FrontendFS, "frontend/dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<!DOCTYPE html><html><body><h1>arkbase</h1><p>Frontend assets not embedded. Visit <a href="/docs">/docs</a> for API documentation.</p></body></html>`))
		})
	}

	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		f, err := distFS.Open(path)
		if err != nil {
			// Fallback to index.html for SPA routing
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		_ = f.Close()

		fileServer.ServeHTTP(w, r)
	})
}
