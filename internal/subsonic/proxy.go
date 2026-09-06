// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonic

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"melovian/internal/cache"
	"melovian/internal/democatalog"
	"melovian/internal/httputil"
)

type ClientResolver func(context.Context) *Client

type Proxy struct {
	resolve    ClientResolver
	httpClient *http.Client
	cache      *cache.ResponseCache
	enabled    bool
	cacheScope func(context.Context) string
}

func NewProxy(resolve ClientResolver, responseCache *cache.ResponseCache, enabled bool, cacheScope func(context.Context) string) http.Handler {
	return &Proxy{
		resolve:    resolve,
		httpClient: httputil.NewRetryHTTPClient(),
		cache:      responseCache,
		enabled:    enabled,
		cacheScope: cacheScope,
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Range, X-Instance-Id, X-Device-Id")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	client := p.resolve(r.Context())
	if !client.Enabled() {
		http.Error(w, "no subsonic instance configured", http.StatusServiceUnavailable)
		return
	}

	if democatalog.IsFakeURL(client.ServerURL) {
		democatalog.Handler().ServeHTTP(w, r)
		return
	}

	target, err := url.Parse(client.ServerURL)
	if err != nil {
		http.Error(w, "invalid server url", http.StatusInternalServerError)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/subsonic")
	if path == "" {
		path = "/"
	}

	query := r.URL.Query()
	query.Del("_instance")
	if !isPublicSharePath(path) {
		query = client.InjectAuth(query)
	}
	rewrittenQuery := query.Encode()
	key := cache.Key(r.Method, path, r.URL.RawQuery)
	if p.cacheScope != nil {
		if scope := p.cacheScope(r.Context()); scope != "" {
			key = scope + "|" + key
		}
	}

	if p.enabled && ShouldCacheRequest(r.Method, path) {
		if entry, ok := p.cache.Get(key); ok {
			cache.WriteCachedResponse(w, entry, "HIT")
			return
		}
	}

	upstreamURL := *target
	upstreamURL.Path = path
	upstreamURL.RawQuery = rewrittenQuery

	upstreamReq, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL.String(), r.Body)
	if err != nil {
		http.Error(w, "request error", http.StatusInternalServerError)
		return
	}

	httputil.CopyHeaders(upstreamReq.Header, r.Header)
	upstreamReq.Host = target.Host

	resp, err := httputil.DoWithRetry(r.Context(), p.httpClient, upstreamReq)
	if err != nil {
		slog.Warn("subsonic upstream request failed", //#nosec G706 -- path is request URL path for diagnostics
			"request_id", httputil.RequestIDFromContext(r.Context()),
			"method", r.Method,
			"path", path,
			"err", err,
		)
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	stripUpstreamCORS(resp.Header)

	if !p.enabled || !ShouldCacheRequest(r.Method, path) || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode >= 400 {
			slog.Warn("subsonic upstream error response", //#nosec G706 -- path is request URL path for diagnostics
				"request_id", httputil.RequestIDFromContext(r.Context()),
				"method", r.Method,
				"path", path,
				"status", resp.StatusCode,
			)
		}
		httputil.CopyHeaders(w.Header(), resp.Header)
		if p.enabled && ShouldCacheRequest(r.Method, path) {
			w.Header().Set("X-Cache", "BYPASS")
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
		return
	}

	limited := io.LimitReader(resp.Body, int64(cache.MaxBodyBytes)+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		http.Error(w, "upstream read error", http.StatusBadGateway)
		return
	}
	if len(body) > cache.MaxBodyBytes {
		httputil.CopyHeaders(w.Header(), resp.Header)
		w.Header().Set("X-Cache", "BYPASS")
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
		_, _ = io.Copy(w, resp.Body)
		return
	}

	entry := cache.Entry{
		StatusCode: resp.StatusCode,
		Header:     cache.CloneHeader(resp.Header),
		Body:       body,
		ExpiresAt:  time.Now().Add(CacheTTL(path)),
	}
	p.cache.Set(key, entry)
	cache.WriteCachedResponse(w, entry, "MISS")
}

func stripUpstreamCORS(header http.Header) {
	header.Del("Access-Control-Allow-Origin")
	header.Del("Access-Control-Allow-Methods")
	header.Del("Access-Control-Allow-Headers")
	header.Del("Access-Control-Expose-Headers")
	header.Del("Access-Control-Allow-Credentials")
}

func isPublicSharePath(path string) bool {
	return strings.HasPrefix(path, "/share/")
}
