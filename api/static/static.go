package static

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var embedded embed.FS

// dist is the build output, rooted so request paths map directly to assets.
var dist = mustSub(embedded, "dist")

func mustSub(f fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(f, dir)
	if err != nil {
		panic("static: embed " + dir + ": " + err.Error())
	}
	return sub
}

// Handler serves the embedded SPA, falling back to index.html so TanStack
// Router deep-links / hard refreshes resolve client-side.
func Handler() http.Handler {
	fileServer := http.FileServerFS(dist)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p != "" {
			if _, err := fs.Stat(dist, p); err != nil {
				r.URL.Path = "/" // unknown path → SPA shell
			}
		}
		if strings.HasPrefix(p, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		fileServer.ServeHTTP(w, r)
	})
}
