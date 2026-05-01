package server

import (
	"io/fs"
	"net/http"

	"github.com/ogcode/ogcode/web"
)

func (s *Server) serveStatic(r chiRouter) {
	sub, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>ogcode</title></head>
<body>
<h1>ogcode</h1>
<p>Server is running. Web UI not built.</p>
</body>
</html>`))
		})
		return
	}

	// Read index.html once for SPA fallback
	indexHTML, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		indexHTML = []byte("<!DOCTYPE html><html><body><h1>ogcode</h1></body></html>")
	}

	fileServer := http.FileServer(http.FS(sub))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve static assets first
		if r.URL.Path != "/" {
			// Check if a static file exists
			filePath := r.URL.Path[1:] // strip leading /
			if f, err := sub.Open(filePath); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		// SPA fallback: serve index.html for all other routes
		w.Header().Set("Content-Type", "text/html")
		w.Write(indexHTML)
	})
}

type chiRouter interface {
	Get(pattern string, handler http.HandlerFunc)
	NotFound(handler http.HandlerFunc)
}