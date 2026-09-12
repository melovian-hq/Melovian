// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package system

import (
	"bufio"
	"melovian/internal/api/apishared"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "melovian_http_requests_total",
			Help: "Total number of HTTP requests processed by Melovian.",
		},
		[]string{"method", "path", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "melovian_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	cacheEntries = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "melovian_cache_entries",
			Help: "Number of entries currently held in in-memory caches.",
		},
		[]string{"cache"},
	)
	cacheBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "melovian_cache_bytes",
			Help: "Approximate bytes currently held in in-memory caches.",
		},
		[]string{"cache"},
	)
	wsClients = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "melovian_ws_clients",
			Help: "Number of connected websocket clients.",
		},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration, cacheEntries, cacheBytes, wsClients)
}

func (h *Handler) registerMetricsRoutes(mux *http.ServeMux) {
	mux.Handle("GET /metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.refreshRuntimeMetrics()
		promhttp.Handler().ServeHTTP(w, r)
	}))
}

func (h *Handler) refreshRuntimeMetrics() {
	entries, bytes := h.cache.Stats()
	cacheEntries.WithLabelValues("response").Set(float64(entries))
	cacheBytes.WithLabelValues("response").Set(float64(bytes))

	entries, bytes = h.coverCache.Stats()
	cacheEntries.WithLabelValues("cover").Set(float64(entries))
	cacheBytes.WithLabelValues("cover").Set(float64(bytes))

	entries, bytes = h.catalogCache.Stats()
	cacheEntries.WithLabelValues("catalog").Set(float64(entries))
	cacheBytes.WithLabelValues("catalog").Set(float64(bytes))

	wsClients.Set(float64(h.events.ClientCount()))
}

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return apishared.DelegateHijack(w.ResponseWriter)
}

func (w *statusResponseWriter) Flush() {
	_ = apishared.DelegateFlush(w.ResponseWriter)
}

func (w *statusResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := apishared.MetricPath(r.URL.Path)
		if path == "/metrics" || path == "/api/debug/memory" || strings.HasPrefix(path, "/debug/") {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		sw := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)

		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}
		statusLabel := strconv.Itoa(status)
		httpRequestsTotal.WithLabelValues(r.Method, path, statusLabel).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path).Observe(time.Since(start).Seconds())
	})
}
