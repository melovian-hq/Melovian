// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Package library serves the local-library, filesystem browse, local music
// catalog, and metadata routes.
package library

import (
	"log/slog"
	"net/http"

	"melovian/internal/api/apishared"
	"melovian/internal/api/realtime"
	"melovian/internal/appconfig"
	"melovian/internal/localmusic"
	"melovian/internal/metaloader"
	"melovian/internal/store"
)

// Handler serves the local library domain routes.
type Handler struct {
	cfg            *appconfig.Config
	instances      *store.InstanceStore
	preferences    *store.PreferencesStore
	listen         *store.ListenStore
	localLibraries *store.LocalLibraryStore
	localTracks    *store.LocalTrackStore
	catalogCache   *localmusic.CatalogCache
	coverCache     *localmusic.CoverCache
	libraryScanner *metaloader.Scanner
	libraryWatcher *metaloader.LibraryWatcher
	events         *realtime.EventHub
	resolver       *apishared.Resolver
}

type Deps struct {
	Config       *appconfig.Config
	Instances    *store.InstanceStore
	Preferences  *store.PreferencesStore
	Listen       *store.ListenStore
	Libraries    *store.LocalLibraryStore
	Tracks       *store.LocalTrackStore
	CatalogCache *localmusic.CatalogCache
	CoverCache   *localmusic.CoverCache
	Events       *realtime.EventHub
	Resolver     *apishared.Resolver
}

func New(d Deps) *Handler {
	h := &Handler{
		cfg:            d.Config,
		instances:      d.Instances,
		preferences:    d.Preferences,
		listen:         d.Listen,
		localLibraries: d.Libraries,
		localTracks:    d.Tracks,
		catalogCache:   d.CatalogCache,
		coverCache:     d.CoverCache,
		libraryScanner: metaloader.NewScanner(d.Libraries, d.Tracks),
		events:         d.Events,
		resolver:       d.Resolver,
	}
	h.libraryScanner.SetProgressHook(h.EmitScanProgress)
	if watcher, err := metaloader.NewLibraryWatcher(h.libraryScanner, h.handleLibraryWatchUpdate); err != nil {
		slog.Warn("local library file watching unavailable", "err", err)
	} else {
		h.libraryWatcher = watcher
		if h.Enabled() {
			if libs, listErr := d.Libraries.List(); listErr == nil {
				for _, lib := range libs {
					h.watchLocalLibrary(lib)
				}
			}
		}
	}
	return h
}

func (h *Handler) Register(mux *http.ServeMux) {
	h.registerLocalLibraryRoutes(mux)
	h.registerFilesystemRoutes(mux)
	h.registerLocalMusicRoutes(mux)
	h.registerLocalMetadataRoutes(mux)
}

// Close stops the library file watcher.
func (h *Handler) Close() {
	if h.libraryWatcher != nil {
		h.libraryWatcher.Close()
	}
}
