// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package system

import (
	"log/slog"
	"net/http"
	"strings"

	"melovian/internal/httputil"
	"melovian/internal/metadata"
)

type artworkLookupResponse struct {
	URL string `json:"url,omitempty"`
}

func (h *Handler) registerArtworkRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/metadata/artwork", h.handleArtworkLookup)
}

// handleArtworkLookup resolves third party artwork server side. Browser
// clients fetch itunes.apple.com directly when possible, but blockers and
// CSP policies can kill those requests, so the API offers the same lookup
// as a fallback.
func (h *Handler) handleArtworkLookup(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	kind := strings.ToLower(strings.TrimSpace(q.Get("kind")))
	artist := strings.TrimSpace(q.Get("artist"))
	album := strings.TrimSpace(q.Get("album"))
	title := strings.TrimSpace(q.Get("title"))

	var (
		artURL string
		err    error
	)
	switch kind {
	case "artist":
		artURL, err = metadata.LookupArtistArtworkURL(r.Context(), artist)
	case "album":
		artURL, err = metadata.LookupAlbumArtworkURL(r.Context(), artist, album)
	case "song":
		artURL, err = metadata.LookupSongArtworkURL(r.Context(), artist, title, album)
	default:
		httputil.WriteError(w, http.StatusBadRequest, "bad_kind", "kind must be artist, album, or song")
		return
	}
	if err != nil {
		slog.Debug("artwork lookup failed", "kind", kind, "err", err)
	}
	httputil.WriteJSON(w, http.StatusOK, artworkLookupResponse{URL: artURL})
}
