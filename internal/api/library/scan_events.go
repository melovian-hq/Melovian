// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package library

import (
	"log/slog"

	"melovian/internal/api/realtime"
	"melovian/internal/metaloader"
)

func (h *Handler) EmitScanProgress(libraryID string, progress metaloader.ScanProgress) {
	lib, err := h.localLibraries.Get(libraryID)
	if err != nil {
		return
	}
	h.events.BroadcastToUser(lib.UserID, realtime.Event{
		Type: realtime.EventScanProgress,
		Payload: realtime.ScanProgressPayload{
			LibraryID: libraryID,
			Name:      lib.Name,
			Processed: progress.Processed,
			Phase:     progress.Phase,
		},
	})
}

func (h *Handler) EmitScanComplete(libraryID string) {
	lib, err := h.localLibraries.Get(libraryID)
	if err != nil {
		slog.Warn("scan complete for unknown library", "library_id", libraryID, "err", err)
		return
	}
	h.events.BroadcastToUser(lib.UserID, realtime.Event{
		Type: realtime.EventScanComplete,
		Payload: realtime.ScanCompletePayload{
			LibraryID:  libraryID,
			Name:       lib.Name,
			TrackCount: lib.TrackCount,
		},
	})
}

func (h *Handler) EmitScanError(libraryID, message string) {
	lib, err := h.localLibraries.Get(libraryID)
	if err != nil {
		slog.Warn("scan error for unknown library", "library_id", libraryID, "err", err)
		return
	}
	h.events.BroadcastToUser(lib.UserID, realtime.Event{
		Type: realtime.EventScanError,
		Payload: realtime.ScanErrorPayload{
			LibraryID: libraryID,
			Name:      lib.Name,
			Error:     message,
		},
	})
}
