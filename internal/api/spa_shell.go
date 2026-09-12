// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"io"
	"io/fs"
	"net"
	"net/http"
	"strings"

	"melovian/internal/seo"
)

// CombinedHandler routes API traffic, static assets, and the SPA shell.
type CombinedHandler struct {
	API    http.Handler
	Assets http.Handler
	// Shell is the embedded frontend dist used to inject per-route meta into
	// index.html. When nil, SPA fallbacks are served through Assets unchanged.
	Shell fs.FS
	// PublicURL is the preferred absolute origin for og:url and og:image.
	// When empty, the request Host and scheme are used.
	PublicURL string
}

func (h *CombinedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if isAPIPath(r.URL.Path) {
		h.API.ServeHTTP(w, r)
		return
	}

	// In dev, Wails proxies to the Vite server. Do not SPA-fallback paths like
	// /@vite/client here or the webview receives HTML with a JS MIME type.
	if frontendDevServerEnabled() {
		h.Assets.ServeHTTP(w, r)
		return
	}

	if isStaticAssetPath(r.URL.Path) {
		h.Assets.ServeHTTP(w, r)
		return
	}

	if r.Method == http.MethodGet && (r.URL.Path == "/" || needsSPAFallback(r.URL.Path)) {
		if h.serveShell(w, r, r.URL.Path) {
			return
		}
		cloned := r.Clone(r.Context())
		cloned.URL.Path = "/"
		h.Assets.ServeHTTP(w, cloned)
		return
	}

	h.Assets.ServeHTTP(w, r)
}

func (h *CombinedHandler) serveShell(w http.ResponseWriter, r *http.Request, pagePath string) bool {
	if h.Shell == nil {
		return false
	}
	f, err := h.Shell.Open("index.html")
	if err != nil {
		return false
	}
	defer f.Close()
	raw, err := io.ReadAll(f)
	if err != nil {
		return false
	}

	base := strings.TrimRight(strings.TrimSpace(h.PublicURL), "/")
	if base == "" {
		base = requestOrigin(r)
	}
	page := seo.ForPath(pagePath)
	body := seo.Inject(raw, page, base, pagePath)

	setStaticAssetCacheHeaders(w, "/")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body) //#nosec G705 -- seo.Inject escapes all injected values
	return true
}

func requestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); proto != "" {
		scheme = strings.Split(proto, ",")[0]
		scheme = strings.TrimSpace(scheme)
	}
	host := r.Host
	if fwd := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); fwd != "" {
		host = strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	if host == "" {
		return ""
	}
	// Strip default ports for cleaner canonical URLs.
	if h, p, err := net.SplitHostPort(host); err == nil {
		if (scheme == "http" && p == "80") || (scheme == "https" && p == "443") {
			host = h
		}
	}
	return scheme + "://" + host
}
