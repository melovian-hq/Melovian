// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package subsonicserver

import (
	"context"
	"time"

	"melovian/internal/jukebox"
)

type ShareSummary struct {
	ID          string
	Token       string
	URL         string
	Description string
	ResourceID  string
	Expires     *time.Time
	Created     time.Time
	VisitCount  int
}

type ExtendedLibraryProvider interface {
	LibraryProvider
	CreatePlaylist(ctx context.Context, melovianUserID, name string, songIDs []string) (PlaylistSummary, error)
	UpdatePlaylist(ctx context.Context, melovianUserID, playlistID, name string, songIDs []string) (PlaylistSummary, error)
	DeletePlaylist(ctx context.Context, melovianUserID, playlistID string) error
	ListShares(ctx context.Context, melovianUserID string) ([]ShareSummary, error)
	CreateShare(ctx context.Context, melovianUserID, resourceID, description string, expires *time.Time) (ShareSummary, error)
	DeleteShare(ctx context.Context, melovianUserID, shareID string) error
	JukeboxStatus(ctx context.Context) jukebox.Status
	JukeboxControl(ctx context.Context, action string, track *jukebox.Track, positionSec int, gain float64, ids []string) jukebox.Status
}

func (s *Server) extended() ExtendedLibraryProvider {
	if lp, ok := s.provider.(ExtendedLibraryProvider); ok {
		return lp
	}
	return nil
}
