// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package sharing

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"melovian/internal/api/apishared"
	"melovian/internal/api/library"
	"melovian/internal/consts"
	"melovian/internal/httputil"
	"melovian/internal/localmusic"
	"melovian/internal/navidrome"
	"melovian/internal/smartplaylist"
	"melovian/internal/store"
)

// maxSmartPlaylistLimit bounds result capacity so a caller-supplied limit
// cannot trigger a huge allocation.
const maxSmartPlaylistLimit = 2000

func (h *Handler) registerSmartPlaylistRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/music/smart-playlists/support", h.handleSmartPlaylistSupport)
	mux.HandleFunc("POST /api/music/smart-playlists", h.handleCreateSmartPlaylist)
	mux.HandleFunc("POST /api/music/smart-playlists/preview", h.handlePreviewSmartPlaylist)
}

type smartPlaylistSupportResponse struct {
	Supported bool   `json:"supported"`
	Mode      string `json:"mode,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type createSmartPlaylistRequest struct {
	Name    string         `json:"name"`
	Comment string         `json:"comment,omitempty"`
	Public  bool           `json:"public,omitempty"`
	Rules   map[string]any `json:"rules"`
}

func (h *Handler) handleSmartPlaylistSupport(w http.ResponseWriter, r *http.Request) {
	supported, mode, reason, err := h.smartPlaylistCapability(r)
	if err != nil {
		writeSmartPlaylistError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, smartPlaylistSupportResponse{
		Supported: supported,
		Mode:      mode,
		Reason:    reason,
	})
}

func (h *Handler) handleCreateSmartPlaylist(w http.ResponseWriter, r *http.Request) {
	supported, mode, reason, err := h.smartPlaylistCapability(r)
	if err != nil {
		writeSmartPlaylistError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !supported {
		writeSmartPlaylistError(w, http.StatusBadRequest, reason)
		return
	}
	if mode != "navidrome" {
		writeSmartPlaylistError(
			w,
			http.StatusBadRequest,
			"Use client-side smart playlists for this source",
		)
		return
	}

	body, err := httputil.ReadJSONBytes(r)
	if err != nil {
		writeSmartPlaylistError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var req createSmartPlaylistRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeSmartPlaylistError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeSmartPlaylistError(w, http.StatusBadRequest, "playlist name is required")
		return
	}
	if req.Rules == nil {
		writeSmartPlaylistError(w, http.StatusBadRequest, "playlist rules are required")
		return
	}
	if !hasSmartPlaylistCriteria(req.Rules) {
		writeSmartPlaylistError(w, http.StatusBadRequest, "add at least one rule")
		return
	}

	inst, err := h.subsonicInstanceForRequest(r)
	if err != nil {
		writeSmartPlaylistError(w, http.StatusBadRequest, err.Error())
		return
	}

	client := navidrome.NewClient(inst.ServerURL, inst.Username, inst.Password)
	ctx, cancel := context.WithTimeout(r.Context(), consts.MetadataFetchTimeout)
	defer cancel()

	playlist, err := client.CreateSmartPlaylist(ctx, navidrome.CreateSmartPlaylistRequest{
		Name:    name,
		Comment: strings.TrimSpace(req.Comment),
		Public:  req.Public,
		Rules:   req.Rules,
	})
	if err != nil {
		status := http.StatusBadGateway
		message := err.Error()
		if errors.Is(err, navidrome.ErrNotAuthenticated) {
			status = http.StatusUnauthorized
			message = "Navidrome authentication failed"
		}
		writeSmartPlaylistError(w, status, message)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, map[string]string{
		"id":   playlist.ID,
		"name": playlist.Name,
	})
}

func (h *Handler) smartPlaylistCapability(r *http.Request) (bool, string, string, error) {
	userID := apishared.UserIDFromContext(r.Context())
	mode := apishared.SourceViewModeForUser(h.preferences, userID)
	if store.IsLocalSourceView(mode) {
		localLib, localErr := h.localLibraries.GetActiveForUser(userID)
		if localErr == nil && localLib.TrackCount > 0 {
			return true, "client", "", nil
		}
		return false, "", "Smart playlists require a music library", nil
	}

	inst, err := h.subsonicInstanceForRequest(r)
	if err != nil {
		if store.IsUnifiedSourceView(mode) {
			localLib, localErr := h.localLibraries.GetActiveForUser(userID)
			if localErr == nil && localLib.TrackCount > 0 {
				return true, "client", "", nil
			}
		}
		return false, "", err.Error(), nil
	}

	serverName := strings.TrimSpace(inst.ServerName)
	version := ""
	client := h.resolver.ForContext(r.Context())
	if client != nil && client.Enabled() {
		if pingName, pingVersion, pingErr := client.Ping(); pingErr == nil {
			// Prefer live ping identity over a custom display name.
			if pingName != "" {
				serverName = pingName
			}
			version = pingVersion
		}
	}

	looksNavidrome := navidrome.IsNavidromeServer(serverName, version)
	nd := navidrome.NewClient(inst.ServerURL, inst.Username, inst.Password)
	ctx, cancel := context.WithTimeout(r.Context(), consts.SmartPlaylistPreviewTimeout)
	defer cancel()
	ok, err := nd.PingNavidrome(ctx)
	if err != nil {
		if errors.Is(err, navidrome.ErrNotAuthenticated) {
			return false, "", "Navidrome authentication failed", nil
		}
		if looksNavidrome {
			return false, "", "", err
		}
		return true, "client", "", nil
	}
	if ok {
		return true, "navidrome", "", nil
	}

	return true, "client", "", nil
}

func (h *Handler) subsonicInstanceForRequest(r *http.Request) (store.SubsonicInstance, error) {
	instanceID, err := h.resolver.ResolveInstanceID(r)
	if err != nil {
		return store.SubsonicInstance{}, err
	}
	if strings.TrimSpace(instanceID) == "" {
		return store.SubsonicInstance{}, errors.New("no subsonic instance configured")
	}

	userID := apishared.UserIDFromContext(r.Context())
	inst, err := h.instances.GetForUser(userID, instanceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.SubsonicInstance{}, errors.New("unknown subsonic instance")
		}
		return store.SubsonicInstance{}, err
	}
	if strings.TrimSpace(inst.ServerURL) == "" {
		return store.SubsonicInstance{}, errors.New("subsonic server URL is missing")
	}
	return inst, nil
}

func hasSmartPlaylistCriteria(rules map[string]any) bool {
	for _, key := range []string{"all", "any"} {
		raw, ok := rules[key]
		if !ok {
			continue
		}
		items, ok := raw.([]any)
		if ok && len(items) > 0 {
			return true
		}
	}
	return false
}

func writeSmartPlaylistError(w http.ResponseWriter, status int, message string) {
	httputil.WriteJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) handlePreviewSmartPlaylist(w http.ResponseWriter, r *http.Request) {
	if !h.localLibraryEnabled() {
		writeSmartPlaylistError(w, http.StatusBadRequest, "local libraries are disabled")
		return
	}
	body, err := httputil.ReadJSONBytes(r)
	if err != nil {
		writeSmartPlaylistError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var payload struct {
		RulesJSON string `json:"rulesJson"`
		Limit     int    `json:"limit"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		writeSmartPlaylistError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	userID := apishared.UserIDFromContext(r.Context())
	catalog, err := h.library.LocalCatalogForUser(userID)
	if err != nil {
		writeSmartPlaylistError(w, http.StatusBadRequest, "no local library available")
		return
	}
	libraryIDs, err := h.library.LibraryIDsForUser(userID)
	if err != nil {
		writeSmartPlaylistError(w, http.StatusInternalServerError, err.Error())
		return
	}
	tracks, err := h.localTracks.ListPresentInLibraries(libraryIDs)
	if err != nil {
		writeSmartPlaylistError(w, http.StatusInternalServerError, err.Error())
		return
	}
	limit := payload.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > maxSmartPlaylistLimit {
		limit = maxSmartPlaylistLimit
	}
	progressUserID := apishared.ResolveProgressUserID(r.Context())
	favorites, _ := h.listen.ListFavorites(progressUserID, 5000)
	loved := make(map[string]bool, len(favorites))
	for _, fav := range favorites {
		loved[fav.TrackID] = true
	}
	matched := make([]localmusic.Song, 0, limit)
	for _, track := range tracks {
		trackCtx := smartplaylist.TrackFromLocal(track)
		if progress, err := h.listen.Get(progressUserID, track.ID); err == nil {
			smartplaylist.EnrichMeta(&trackCtx, progress, loved[track.ID])
		} else if loved[track.ID] {
			smartplaylist.EnrichMeta(&trackCtx, store.ListenProgress{}, true)
		}
		if !smartplaylist.MatchDraftJSON(payload.RulesJSON, trackCtx) {
			continue
		}
		song, ok := catalog.Song(track.ID)
		if !ok {
			continue
		}
		matched = append(matched, song)
		if len(matched) >= limit {
			break
		}
	}
	httputil.WriteJSON(w, http.StatusOK, library.LocalSongsResponse{
		Songs: library.LocalSongViewsFrom(matched),
	})
}
