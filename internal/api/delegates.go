// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"melovian/internal/api/music"
	"melovian/internal/api/system"
	"melovian/internal/api/videos"
	"melovian/internal/appconfig"
	"melovian/internal/localmusic"
	"melovian/internal/metaloader"
	"melovian/internal/store"
)

// Delegates to domain handlers. They keep the subsonic/DLNA adapters and the
// package-internal tests working while the route registrations live in the
// domain packages.

func (s *Server) localCatalogForUser(userID string) (localmusic.Catalog, error) {
	return s.libraryH.LocalCatalogForUser(userID)
}

func (s *Server) libraryIDsForUser(userID string) ([]string, error) {
	return s.libraryH.LibraryIDsForUser(userID)
}

func (s *Server) coverBytesForTrack(lib store.LocalLibrary, track store.LocalTrack) ([]byte, string, error) {
	return s.libraryH.CoverBytesForTrack(lib, track)
}

func (s *Server) localLibraryEnabled() bool {
	return s.cfg.LocalLibraryEffective().Enabled
}

func (s *Server) resolveLibraryPath(inputPath string) (string, error) {
	return s.libraryH.ResolveLibraryPath(inputPath)
}

func (s *Server) emitScanProgress(libraryID string, progress metaloader.ScanProgress) {
	s.libraryH.EmitScanProgress(libraryID, progress)
}

// LyricsSettings aliases the music domain type for existing tests.
type LyricsSettings = music.LyricsSettings

func (s *Server) lyricsRoot(settings music.LyricsSettings) string {
	return s.lyricsH.LyricsRoot(settings)
}

func (s *Server) evictDownloads(instanceID string, limitBytes int64, reserveBytes int64) error {
	return s.downloadsH.EvictDownloads(instanceID, limitBytes, reserveBytes)
}

// VideoSettings aliases the videos domain type for the public api surface.
type VideoSettings = videos.VideoSettings

func (s *Server) shareCoverAllowed(share store.Share, coverID string) bool {
	return s.sharingH.ShareCoverAllowed(share, coverID)
}

func (s *Server) shareAccessAllowed(r *http.Request, share store.Share) bool {
	return s.sharingH.ShareAccessAllowed(r, share)
}

// DesktopUpdateHooks aliases the system domain type for the public api
// surface used by main.go.
type DesktopUpdateHooks = system.DesktopUpdateHooks

// SetUpdateHooks installs the desktop updater bridge.
func (s *Server) SetUpdateHooks(h *DesktopUpdateHooks) {
	s.systemH.SetUpdateHooks(h)
}

func (s *Server) initSentryFromStore() error {
	return s.systemH.InitSentryFromStore()
}

func (s *Server) effectiveSentryConfig() appconfig.SentryConfig {
	return s.systemH.EffectiveSentryConfig()
}

func (s *Server) loadSentryClientSettings(userID string) (store.SentryClientSettings, error) {
	return s.systemH.LoadSentryClientSettings(userID)
}

func (s *Server) refreshSentryRuntime(stored appconfig.StoredSentrySettings) error {
	return s.systemH.RefreshSentryRuntime(stored)
}
