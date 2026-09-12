// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package library

import (
	"database/sql"

	"melovian/internal/localmusic"
	"melovian/internal/store"
)

// LibraryIDsForUser returns the local library IDs visible to the user,
// honoring the multi-library preference.
func (h *Handler) LibraryIDsForUser(userID string) ([]string, error) {
	multi, err := h.preferences.GetMultiLocalLibrary(userID)
	if err != nil {
		return nil, err
	}
	if multi {
		libs, err := h.localLibraries.ListForUser(userID)
		if err != nil {
			return nil, err
		}
		ids := make([]string, len(libs))
		for i, lib := range libs {
			ids[i] = lib.ID
		}
		return ids, nil
	}
	lib, err := h.localLibraries.GetActiveForUser(userID)
	if err != nil {
		return nil, err
	}
	return []string{lib.ID}, nil
}

// LocalCatalogForUser loads (or returns the cached) catalog for the user's
// active or merged local libraries.
func (h *Handler) LocalCatalogForUser(userID string) (localmusic.Catalog, error) {
	multi, err := h.preferences.GetMultiLocalLibrary(userID)
	if err != nil {
		return localmusic.Catalog{}, err
	}
	if multi {
		return h.mergedLocalCatalog(userID)
	}
	lib, err := h.localLibraries.GetActiveForUser(userID)
	if err != nil {
		return localmusic.Catalog{}, err
	}
	version := localmusic.CatalogVersion(lib)
	if catalog, ok := h.catalogCache.Get(lib.ID, version); ok {
		return catalog, nil
	}
	catalog, err := localmusic.LoadCatalog(h.localTracks, lib.ID)
	if err != nil {
		return localmusic.Catalog{}, err
	}
	h.catalogCache.Set(lib.ID, version, catalog)
	return catalog, nil
}

func (h *Handler) mergedLocalCatalog(userID string) (localmusic.Catalog, error) {
	cacheKey := "merged:" + userID
	libs, err := h.localLibraries.ListForUser(userID)
	if err != nil {
		return localmusic.Catalog{}, err
	}
	if len(libs) == 0 {
		return localmusic.Catalog{}, sql.ErrNoRows
	}
	var version uint64
	for _, lib := range libs {
		version ^= localmusic.CatalogVersion(lib)
	}
	if catalog, ok := h.catalogCache.Get(cacheKey, version); ok {
		return catalog, nil
	}
	var tracks []store.CatalogTrack
	for _, lib := range libs {
		items, listErr := h.localTracks.ListForCatalog(lib.ID)
		if listErr != nil {
			return localmusic.Catalog{}, listErr
		}
		tracks = append(tracks, items...)
	}
	catalog := localmusic.BuildCatalog(tracks)
	h.catalogCache.Set(cacheKey, version, catalog)
	return catalog, nil
}

// CoverBytesForTrack returns the embedded or sidecar cover bytes for a track.
func (h *Handler) CoverBytesForTrack(lib store.LocalLibrary, track store.LocalTrack) ([]byte, string, error) {
	data, mime, ok := h.CoverBytesFromTrackFile(lib, track.ID)
	if !ok {
		return nil, "", sql.ErrNoRows
	}
	return data, mime, nil
}
