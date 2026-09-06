// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonic

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"melovian/internal/cache"
	"melovian/internal/httputil"
)

func scopedCacheKey(scope func(context.Context) string, r *http.Request, path, query string) string {
	key := cache.Key(http.MethodGet, path, query)
	if scope != nil {
		if prefix := scope(r.Context()); prefix != "" {
			key = prefix + "|" + key
		}
	}
	return key
}

func StatusHandler(client *Client, responseCache *cache.ResponseCache, cacheEnabled bool, cacheScope func(context.Context) string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if !client.Enabled() {
			httputil.WriteJSON(w, http.StatusOK, StatusResponse{Enabled: false})
			return
		}

		key := scopedCacheKey(cacheScope, r, "/api/music/status", "")
		if cacheEnabled {
			if entry, ok := responseCache.Get(key); ok {
				cache.WriteCachedResponse(w, entry, "HIT")
				return
			}
		}

		name, version, err := client.Ping()
		if err != nil {
			payload, _ := json.Marshal(StatusResponse{
				Enabled:   true,
				Connected: false,
				Error:     "Subsonic server is unreachable",
			})
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write(payload)
			return
		}

		payload, _ := json.Marshal(StatusResponse{
			Enabled:    true,
			Connected:  true,
			ServerName: name,
			Version:    version,
		})

		entry := cache.Entry{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       payload,
			ExpiresAt:  time.Now().Add(15 * time.Minute),
		}
		if cacheEnabled {
			responseCache.Set(key, entry)
			cache.WriteCachedResponse(w, entry, "MISS")
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}
}

func LibraryStatsHandler(client *Client, responseCache *cache.ResponseCache, cacheEnabled bool, cacheScope func(context.Context) string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !client.Enabled() {
			httputil.WriteJSON(w, http.StatusOK, LibraryStatsResponse{})
			return
		}

		bypassCache := r.URL.Query().Has("_refresh")
		key := scopedCacheKey(cacheScope, r, "/api/music/library-stats", "")
		if cacheEnabled && !bypassCache {
			if entry, ok := responseCache.Get(key); ok {
				cache.WriteCachedResponse(w, entry, "HIT")
				return
			}
		}

		stats, err := client.LibraryStats()
		if err != nil {
			http.Error(w, "failed to load library stats", http.StatusBadGateway)
			return
		}

		payload, _ := json.Marshal(stats)
		entry := cache.Entry{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       payload,
			ExpiresAt:  time.Now().Add(30 * time.Minute),
		}
		if cacheEnabled {
			responseCache.Set(key, entry)
			cache.WriteCachedResponse(w, entry, "MISS")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}
}
