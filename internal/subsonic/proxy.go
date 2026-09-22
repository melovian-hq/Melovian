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
	"melovian/internal/melog"
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

	target, err := client.parsedBaseURL()
	if err != nil {
		http.Error(w, "invalid server url", http.StatusInternalServerError)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/subsonic")
	if path == "" {
		path = "/"
	}

	// The cache key derives from the raw request, so the lookup runs before
	// query parsing and auth injection. On a hit those steps are skipped
	// entirely.
	cacheable := ShouldCacheRequest(r.Method, path) && !randomDraw(path, r.URL.RawQuery)
	key := cache.Key(r.Method, path, r.URL.RawQuery)
	if p.cacheScope != nil {
		if scope := p.cacheScope(r.Context()); scope != "" {
			key = scope + "|" + key
		}
	}

	if p.enabled && cacheable {
		if entry, ok := p.cache.Get(key); ok {
			cache.WriteCachedResponse(w, entry, "HIT")
			return
		}
	}

	query := r.URL.Query()
	query.Del("_instance")
	if !isPublicSharePath(path) {
		query = client.InjectAuth(query)
	}
	rewrittenQuery := query.Encode()

	upstreamURL := *target
	upstreamURL.Path = path
	upstreamURL.RawQuery = rewrittenQuery

	upstreamReq, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL.String(), r.Body)
	if err != nil {
		http.Error(w, "request error", http.StatusInternalServerError)
		return
	}

	httputil.CopyHeaders(upstreamReq.Header, r.Header)
	// Inbound credentials and routing headers must never reach the
	// upstream: a hostile or compromised server URL would otherwise get
	// a replayable session cookie, and Set-Cookie on the way back would
	// plant on the Melovian origin.
	upstreamReq.Header.Del("Cookie")
	upstreamReq.Header.Del("Authorization")
	upstreamReq.Header.Del("X-Instance-Id")
	upstreamReq.Header.Del("X-Device-Id")
	upstreamReq.Host = target.Host

	resp, err := httputil.DoWithRetry(r.Context(), p.httpClient, upstreamReq)
	if err != nil {
		// Transport errors embed the full request URL, including the auth
		// token query parameters. Strip them before logging or responding.
		err = httputil.SanitizeErrorURL(err)
		slog.Warn("subsonic upstream request failed",
			"request_id", httputil.RequestIDFromContext(r.Context()),
			"method", r.Method,
			"path", melog.Sanitize(path),
			"err", err,
		)
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	stripUpstreamCORS(resp.Header)
	resp.Header.Del("Set-Cookie")
	resp.Header.Del("WWW-Authenticate")

	if !p.enabled || !cacheable || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode >= 400 {
			slog.Warn("subsonic upstream error response",
				"request_id", httputil.RequestIDFromContext(r.Context()),
				"method", r.Method,
				"path", melog.Sanitize(path),
				"status", resp.StatusCode,
			)
		}
		httputil.CopyHeaders(w.Header(), resp.Header)
		if p.enabled && cacheable {
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

// randomDraw reports whether the endpoint must return a fresh random result
// on every call. Caching these serves the identical batch to continuous-mode
// queue refills, which then dedupe to nothing and stall playback.
func randomDraw(path, rawQuery string) bool {
	if strings.Contains(path, "getRandomSongs") {
		return true
	}
	if !strings.Contains(path, "getAlbumList") || !strings.Contains(rawQuery, "type=") {
		return false
	}
	// Only getAlbumList requests that mention a type reach this parse.
	values, err := url.ParseQuery(rawQuery)
	return err == nil && values.Get("type") == "random"
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
