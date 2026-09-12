// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package music

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"melovian/internal/api/apishared"
	"melovian/internal/appconfig"
	"melovian/internal/extensions"
	"melovian/internal/httputil"
	"melovian/internal/lyrics"
	"melovian/internal/observability"
	"melovian/internal/store"
	"melovian/internal/subsonic"
)

const lyricsFetchTimeout = 45 * time.Second

// LyricsHandler serves the lyrics settings, lookup, fetch, and whisper
// transcription routes.
type LyricsHandler struct {
	cfg         appconfig.Config
	preferences *store.PreferencesStore
	resolver    *apishared.Resolver
}

func NewLyricsHandler(cfg appconfig.Config, preferences *store.PreferencesStore, resolver *apishared.Resolver) *LyricsHandler {
	return &LyricsHandler{cfg: cfg, preferences: preferences, resolver: resolver}
}

func (h *LyricsHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/music/settings/lyrics", h.handleGetLyricsSettings)
	mux.HandleFunc("PUT /api/music/settings/lyrics", h.handlePutLyricsSettings)
	mux.HandleFunc("GET /api/music/lyrics/{trackId}", h.handleGetTrackLyrics)
	mux.HandleFunc("POST /api/music/lyrics/{trackId}/fetch", h.handleFetchTrackLyrics)
	mux.HandleFunc("POST /api/music/lyrics/{trackId}/whisper", h.handleWhisperTrackLyrics)
	mux.HandleFunc("DELETE /api/music/lyrics/cache", h.handleClearLyricsCache)
}

func (h *LyricsHandler) handleGetLyricsSettings(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	settings, err := h.loadLyricsSettings(userID)
	if err != nil {
		writeLyricsError(w, http.StatusInternalServerError, "failed to load lyrics settings")
		return
	}
	root := h.LyricsRoot(settings)
	instanceID := apishared.InstanceIDFromContext(r.Context())
	store := lyrics.NewStore(root)
	count, bytes, err := store.Stats(root, instanceID)
	if err != nil {
		writeLyricsError(w, http.StatusInternalServerError, "failed to load lyrics cache stats")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"storageDir":         settings.StorageDir,
		"resolvedStorageDir": root,
		"defaultStorageDir":  DefaultLyricsStorageDir(h.cfg.DataDir),
		"autoFetch":          settings.AutoFetch,
		"providers":          settings.Providers,
		"builtinProviders":   lyrics.BuiltInProviders(),
		"whisperUrl":         settings.WhisperURL,
		"whisperEnabled":     extensions.IsEnabled(h.cfg.DataDir, lyricsWhisperExtensionID),
		"trackCount":         count,
		"usedBytes":          bytes,
	})
}

func (h *LyricsHandler) handlePutLyricsSettings(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	body, err := httputil.ReadJSONBytes(r)
	if err != nil {
		writeLyricsError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if !json.Valid(body) {
		writeLyricsError(w, http.StatusBadRequest, "invalid json")
		return
	}
	settings, err := MergeLyricsSettings(body, h.cfg.DataDir)
	if err != nil {
		writeLyricsError(w, http.StatusBadRequest, "invalid lyrics settings")
		return
	}
	encoded, err := json.Marshal(settings)
	if err != nil {
		writeLyricsError(w, http.StatusInternalServerError, "encode lyrics settings")
		return
	}
	if err := h.preferences.Set(userID, store.PrefKeyLyricsSettings, string(encoded)); err != nil {
		writeLyricsError(w, http.StatusBadRequest, err.Error())
		return
	}
	root := h.LyricsRoot(settings)
	instanceID := apishared.InstanceIDFromContext(r.Context())
	lyricsStore := lyrics.NewStore(root)
	count, bytes, err := lyricsStore.Stats(root, instanceID)
	if err != nil {
		writeLyricsError(w, http.StatusInternalServerError, "failed to load lyrics cache stats")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"storageDir":         settings.StorageDir,
		"resolvedStorageDir": root,
		"defaultStorageDir":  DefaultLyricsStorageDir(h.cfg.DataDir),
		"autoFetch":          settings.AutoFetch,
		"providers":          settings.Providers,
		"builtinProviders":   lyrics.BuiltInProviders(),
		"whisperUrl":         settings.WhisperURL,
		"whisperEnabled":     extensions.IsEnabled(h.cfg.DataDir, lyricsWhisperExtensionID),
		"trackCount":         count,
		"usedBytes":          bytes,
	})
}

func (h *LyricsHandler) handleGetTrackLyrics(w http.ResponseWriter, r *http.Request) {
	trackID := strings.TrimSpace(r.PathValue("trackId"))
	if trackID == "" {
		writeLyricsError(w, http.StatusBadRequest, "track id required")
		return
	}
	in := lyricsFetchInputFromQuery(r, trackID)
	doc, err := h.resolveTrackLyrics(r, in, false)
	if err != nil {
		writeLyricsError(w, http.StatusNotFound, err.Error())
		return
	}
	writeLyricsDocument(w, doc)
}

func (h *LyricsHandler) handleFetchTrackLyrics(w http.ResponseWriter, r *http.Request) {
	trackID := strings.TrimSpace(r.PathValue("trackId"))
	if trackID == "" {
		writeLyricsError(w, http.StatusBadRequest, "track id required")
		return
	}
	in := lyricsFetchInputFromQuery(r, trackID)
	doc, err := h.resolveTrackLyrics(r, in, true)
	if err != nil {
		writeLyricsError(w, http.StatusNotFound, err.Error())
		return
	}
	writeLyricsDocument(w, doc)
}

func (h *LyricsHandler) handleClearLyricsCache(w http.ResponseWriter, r *http.Request) {
	userID := apishared.ResolveProgressUserID(r.Context())
	settings, err := h.loadLyricsSettings(userID)
	if err != nil {
		writeLyricsError(w, http.StatusInternalServerError, "failed to load lyrics settings")
		return
	}
	root := h.LyricsRoot(settings)
	instanceID := apishared.InstanceIDFromContext(r.Context())
	if err := lyrics.NewStore(root).ClearInstance(root, instanceID); err != nil {
		writeLyricsError(w, http.StatusInternalServerError, "clear lyrics cache: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func lyricsFetchInputFromQuery(r *http.Request, trackID string) lyrics.FetchInput {
	durationSec := parseDurationSec(r.URL.Query().Get("durationSec"))
	return lyrics.FetchInput{
		TrackID:     trackID,
		Artist:      strings.TrimSpace(r.URL.Query().Get("artist")),
		Title:       strings.TrimSpace(r.URL.Query().Get("title")),
		Album:       strings.TrimSpace(r.URL.Query().Get("album")),
		DurationSec: durationSec,
	}
}

func parseDurationSec(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if n, err := strconv.Atoi(raw); err == nil && n > 0 {
		return n
	}
	if f, err := strconv.ParseFloat(raw, 64); err == nil && f > 0 {
		return int(f + 0.5)
	}
	return 0
}

func (h *LyricsHandler) resolveTrackLyrics(r *http.Request, in lyrics.FetchInput, forceFetch bool) (*lyrics.Document, error) {
	if in.TrackID == "" {
		return nil, errors.New("track id required")
	}

	ctx, cancel := context.WithTimeout(r.Context(), lyricsFetchTimeout)
	defer cancel()

	userID := apishared.ResolveProgressUserID(r.Context())
	settings, err := h.loadLyricsSettings(userID)
	if err != nil {
		return nil, err
	}
	root := h.LyricsRoot(settings)
	instanceID := apishared.InstanceIDFromContext(r.Context())
	store := lyrics.NewStore(root)

	if !forceFetch {
		if cached, loadErr := store.Load(root, instanceID, in.TrackID); loadErr == nil && cached != nil {
			return cached, nil
		}
	} else {
		_ = store.Delete(root, instanceID, in.TrackID)
	}

	subsonicFn := h.subsonicLyricsFetcher(r)
	providers := h.enabledLyricsProviders(settings, subsonicFn)
	fetcher := &lyrics.Fetcher{
		Store:  store,
		Client: lyrics.NewProviderHTTPClient(),
	}

	var doc *lyrics.Document
	if forceFetch {
		// Refresh must try online providers first so plain Subsonic lyrics
		// can be replaced with synced LRCLIB results.
		doc, err = fetcher.FetchFromProviders(ctx, preferOnlineLyricsProviders(providers), in)
	} else if !settings.AutoFetch {
		doc, err = fetcher.FetchFromProviders(ctx, filterProviders(providers, "subsonic"), in)
	} else {
		doc, err = fetcher.FetchFromProviders(ctx, providers, in)
	}
	if err != nil {
		return nil, err
	}
	if saveErr := store.Save(root, instanceID, in.TrackID, doc); saveErr != nil {
		slog.Warn("lyrics cache save failed",
			"track_id", in.TrackID,
			"instance_id", instanceID,
			"source", doc.Source,
			"err", saveErr,
		)
		observability.CaptureError(saveErr, map[string]string{
			"kind":     "lyrics-cache-save",
			"track_id": in.TrackID,
			"source":   doc.Source,
			"force":    strconv.FormatBool(forceFetch),
			"instance": instanceID,
		})
		return doc, nil
	}
	return doc, nil
}

func filterProviders(providers []lyrics.Provider, id string) []lyrics.Provider {
	out := make([]lyrics.Provider, 0, 1)
	for _, provider := range providers {
		if provider.ID() == id {
			out = append(out, provider)
		}
	}
	return out
}

func excludeProviders(providers []lyrics.Provider, id string) []lyrics.Provider {
	out := make([]lyrics.Provider, 0, len(providers))
	for _, provider := range providers {
		if provider.ID() == id {
			continue
		}
		out = append(out, provider)
	}
	return out
}

// preferOnlineLyricsProviders puts non-subsonic providers first so a refresh
// can replace server-stored lyrics. Subsonic remains as a last fallback.
func preferOnlineLyricsProviders(providers []lyrics.Provider) []lyrics.Provider {
	online := excludeProviders(providers, "subsonic")
	subsonicOnly := filterProviders(providers, "subsonic")
	if len(online) == 0 {
		return subsonicOnly
	}
	return append(online, subsonicOnly...)
}

func (h *LyricsHandler) subsonicLyricsFetcher(r *http.Request) lyrics.SubsonicFetcher {
	return func(ctx context.Context, in lyrics.FetchInput) (*lyrics.Document, error) {
		client := h.resolver.ForContext(r.Context())
		if client == nil || !client.Enabled() {
			return nil, errors.New("subsonic unavailable")
		}
		if doc, err := client.GetLyricsBySongID(in.TrackID); err == nil && doc != nil {
			return subsonicToLyricsDocument(doc)
		}
		if in.Artist == "" {
			return nil, errors.New("subsonic: artist required")
		}
		doc, err := client.GetLyrics(in.Artist, in.Title)
		if err != nil {
			return nil, err
		}
		return subsonicToLyricsDocument(doc)
	}
}

func subsonicToLyricsDocument(doc *subsonic.LyricsDocument) (*lyrics.Document, error) {
	if doc == nil {
		return nil, errors.New("subsonic: empty lyrics")
	}
	out := &lyrics.Document{
		Source:   "subsonic",
		Artist:   doc.Artist,
		Title:    doc.Title,
		Synced:   doc.Synced,
		OffsetMs: doc.OffsetMs,
		RawValue: doc.RawValue,
	}
	for _, line := range doc.Lines {
		entry := lyrics.Line{Text: line.Value}
		if line.Start != nil {
			entry.StartMs = line.Start
		}
		out.Lines = append(out.Lines, entry)
	}
	return lyrics.NormalizeDocument(out)
}

func writeLyricsDocument(w http.ResponseWriter, doc *lyrics.Document) {
	normalized, err := lyrics.NormalizeDocument(doc)
	if err != nil {
		writeLyricsError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.WriteJSON(w, http.StatusOK, lyricsDocumentToMap(normalized))
}

func writeLyricsError(w http.ResponseWriter, status int, message string) {
	httputil.WriteJSON(w, status, map[string]any{
		"error":   http.StatusText(status),
		"message": message,
	})
}
