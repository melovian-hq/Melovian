// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"log/slog"

	"melovian/internal/metaloader"
)

func (s *Server) emitScanProgress(libraryID string, progress metaloader.ScanProgress) {
	lib, err := s.localLibraries.Get(libraryID)
	if err != nil {
		return
	}
	s.events.BroadcastToUser(lib.UserID, Event{
		Type: EventScanProgress,
		Payload: ScanProgressPayload{
			LibraryID: libraryID,
			Name:      lib.Name,
			Processed: progress.Processed,
			Phase:     progress.Phase,
		},
	})
}

func (s *Server) emitScanComplete(libraryID string) {
	lib, err := s.localLibraries.Get(libraryID)
	if err != nil {
		slog.Warn("scan complete for unknown library", "library_id", libraryID, "err", err)
		return
	}
	s.events.BroadcastToUser(lib.UserID, Event{
		Type: EventScanComplete,
		Payload: ScanCompletePayload{
			LibraryID:  libraryID,
			Name:       lib.Name,
			TrackCount: lib.TrackCount,
		},
	})
}

func (s *Server) emitScanError(libraryID, message string) {
	lib, err := s.localLibraries.Get(libraryID)
	if err != nil {
		slog.Warn("scan error for unknown library", "library_id", libraryID, "err", err)
		return
	}
	s.events.BroadcastToUser(lib.UserID, Event{
		Type: EventScanError,
		Payload: ScanErrorPayload{
			LibraryID: libraryID,
			Name:      lib.Name,
			Error:     message,
		},
	})
}
