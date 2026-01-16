package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

// distFS contains the embedded production build output for the web UI.
//
// Build flow:
// - `pnpm -C web build` creates `web/dist`
// - build scripts copy `web/dist` -> `server/internal/webui/dist`
//
// The repo keeps a placeholder file in `dist/` so `go run`
// works even before the web UI is built.
//
//go:embed dist/*
var distFS embed.FS

func Handler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return placeholderHandler()
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If the UI hasn't been built, serve a friendly placeholder instead of a
		// directory listing or a confusing 404.
		if r.URL.Path == "/" {
			if _, err := fs.Stat(sub, "index.html"); err != nil {
				placeholderHandler().ServeHTTP(w, r)
				return
			}
		}

		fileServer.ServeHTTP(w, r)
	})
}

func placeholderHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!doctype html>
<html>
  <head><meta charset="utf-8"/><title>Calendar-app</title></head>
  <body>
    <h1>Calendar-app Server</h1>
    <p>Backend is running.</p>
    <p>Build the frontend with <code>pnpm -C web build</code> (or <code>make build</code>).</p>
    <p>Health check: <a href="/health">/health</a></p>
  </body>
</html>`))
	})
}
