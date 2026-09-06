// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// StaticAssetHandler serves the embedded SPA with cache rules that keep
// index.html fresh while allowing long-lived caching of hashed /assets/*.
func StaticAssetHandler(root fs.FS) http.Handler {
	files := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setStaticAssetCacheHeaders(w, r.URL.Path)
		files.ServeHTTP(w, r)
	})
}

func setStaticAssetCacheHeaders(w http.ResponseWriter, requestPath string) {
	clean := path.Clean("/" + strings.TrimPrefix(requestPath, "/"))
	switch {
	case clean == "/" || clean == "/index.html":
		// Always revalidate the shell so browsers pick up new hashed bundles.
		w.Header().Set("Cache-Control", "no-cache")
	case strings.HasPrefix(clean, "/assets/"):
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	default:
		// Favicons and other root static files: short cache is enough.
		if path.Ext(clean) != "" {
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}
	}
}
