// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package system

import (
	"net/http"
	"net/http/pprof"
	"runtime"
	"time"

	"melovian/internal/brand"
	"melovian/internal/httputil"
)

type memorySnapshot struct {
	Timestamp string `json:"timestamp"`

	Alloc         uint64  `json:"alloc"`
	TotalAlloc    uint64  `json:"totalAlloc"`
	Sys           uint64  `json:"sys"`
	HeapAlloc     uint64  `json:"heapAlloc"`
	HeapSys       uint64  `json:"heapSys"`
	HeapInuse     uint64  `json:"heapInuse"`
	HeapIdle      uint64  `json:"heapIdle"`
	HeapReleased  uint64  `json:"heapReleased"`
	StackInuse    uint64  `json:"stackInuse"`
	GCCPUFraction float64 `json:"gcCPUFraction"`
	NumGC         uint32  `json:"numGC"`
	NumGoroutine  int     `json:"numGoroutine"`

	ResponseCache cacheStatsJSON `json:"responseCache"`
	CoverCache    cacheStatsJSON `json:"coverCache"`
	CatalogCache  cacheStatsJSON `json:"catalogCache"`
	ClientCache   int            `json:"clientCache"`
	WSClients     int            `json:"wsClients"`
	DownloadSlots int            `json:"downloadSlots"`
	DownloadInUse int            `json:"downloadInUse"`
	DebugPprof    bool           `json:"debugPprof"`
	Notes         []string       `json:"notes"`
}

type cacheStatsJSON struct {
	Entries    int `json:"entries"`
	TotalBytes int `json:"totalBytes,omitempty"`
}

func (h *Handler) registerDebugRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/debug/memory", h.handleDebugMemory)
	mux.HandleFunc("POST /api/debug/gc", h.handleDebugGC)

	if !h.cfg.DebugPprof {
		return
	}

	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	mux.Handle("GET /debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("GET /debug/pprof/allocs", pprof.Handler("allocs"))
	mux.Handle("GET /debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("GET /debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("GET /debug/pprof/block", pprof.Handler("block"))
	mux.Handle("GET /debug/pprof/mutex", pprof.Handler("mutex"))
}

func (h *Handler) handleDebugMemory(w http.ResponseWriter, _ *http.Request) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	responseEntries, responseBytes := h.cache.Stats()
	coverEntries, coverBytes := h.coverCache.Stats()
	catalogEntries, catalogBytes := h.catalogCache.Stats()

	clientCount := h.resolver.ClientCacheSize()

	downloadInUse := 0
	if cap(h.downloadSem) > 0 {
		downloadInUse = len(h.downloadSem)
	}

	snapshot := memorySnapshot{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Alloc:         ms.Alloc,
		TotalAlloc:    ms.TotalAlloc,
		Sys:           ms.Sys,
		HeapAlloc:     ms.HeapAlloc,
		HeapSys:       ms.HeapSys,
		HeapInuse:     ms.HeapInuse,
		HeapIdle:      ms.HeapIdle,
		HeapReleased:  ms.HeapReleased,
		StackInuse:    ms.StackInuse,
		GCCPUFraction: ms.GCCPUFraction,
		NumGC:         ms.NumGC,
		NumGoroutine:  runtime.NumGoroutine(),
		ResponseCache: cacheStatsJSON{Entries: responseEntries, TotalBytes: responseBytes},
		CoverCache:    cacheStatsJSON{Entries: coverEntries, TotalBytes: coverBytes},
		CatalogCache:  cacheStatsJSON{Entries: catalogEntries, TotalBytes: catalogBytes},
		ClientCache:   clientCount,
		WSClients:     h.events.ClientCount(),
		DownloadSlots: cap(h.downloadSem),
		DownloadInUse: downloadInUse,
		DebugPprof:    h.cfg.DebugPprof,
		Notes: []string{
			brand.Name + " does not spawn ffmpeg. Large ffmpeg RSS usually comes from the upstream Subsonic/Navidrome transcoder or from libmpv/libvlc decoding.",
			"Enable heap profiles with MELOVIAN_DEBUG_PPROF=true or --debug-pprof, then: go tool pprof http://127.0.0.1:17337/debug/pprof/heap",
			"Force a GC sample with POST /api/debug/gc then re-check this endpoint.",
		},
	}

	httputil.WriteJSON(w, http.StatusOK, snapshot)
}

func (h *Handler) handleDebugGC(w http.ResponseWriter, _ *http.Request) {
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"heapAlloc": ms.HeapAlloc,
		"heapInuse": ms.HeapInuse,
		"sys":       ms.Sys,
		"numGC":     ms.NumGC,
	})
}
