// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metaloader

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"melovian/internal/store"
)

const maxScanErrors = 64

type ScanResult struct {
	Added      int      `json:"added"`
	Updated    int      `json:"updated"`
	Removed    int      `json:"removed"`
	Duplicates int      `json:"duplicates"`
	Unchanged  int      `json:"unchanged"`
	Total      int      `json:"total"`
	Errors     []string `json:"errors,omitempty"`
}

type ScanProgress struct {
	Processed int    `json:"processed"`
	Phase     string `json:"phase"`
}

type Scanner struct {
	libraries  *store.LocalLibraryStore
	tracks     *store.LocalTrackStore
	scanMu     sync.Map
	progress   sync.Map
	onProgress func(libraryID string, progress ScanProgress)
}

func NewScanner(libraries *store.LocalLibraryStore, tracks *store.LocalTrackStore) *Scanner {
	return &Scanner{libraries: libraries, tracks: tracks}
}

func (s *Scanner) SetProgressHook(fn func(libraryID string, progress ScanProgress)) {
	s.onProgress = fn
}

func (s *Scanner) ForgetLibrary(libraryID string) {
	s.scanMu.Delete(libraryID)
	s.progress.Delete(libraryID)
}

func (s *Scanner) libraryScanMu(libraryID string) *sync.Mutex {
	mu, _ := s.scanMu.LoadOrStore(libraryID, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

func (s *Scanner) Progress(libraryID string) (ScanProgress, bool) {
	value, ok := s.progress.Load(libraryID)
	if !ok {
		return ScanProgress{}, false
	}
	return value.(ScanProgress), true
}

func (s *Scanner) setProgress(libraryID, phase string, processed int) {
	progress := ScanProgress{Processed: processed, Phase: phase}
	s.progress.Store(libraryID, progress)
	if s.onProgress != nil {
		s.onProgress(libraryID, progress)
	}
}

func (s *Scanner) clearProgress(libraryID string) {
	s.progress.Delete(libraryID)
}

func (s *Scanner) StartScan(libraryID string, done func(ScanResult, error)) bool {
	mu := s.libraryScanMu(libraryID)
	if !mu.TryLock() {
		return false
	}
	go func() {
		defer mu.Unlock()
		defer s.clearProgress(libraryID)
		result, err := s.scan(libraryID)
		if done != nil {
			done(result, err)
		}
	}()
	return true
}

func (s *Scanner) Scan(libraryID string) (ScanResult, error) {
	mu := s.libraryScanMu(libraryID)
	mu.Lock()
	defer mu.Unlock()
	defer s.clearProgress(libraryID)
	return s.scan(libraryID)
}

func (s *Scanner) scan(libraryID string) (ScanResult, error) {
	lib, err := s.libraries.Get(libraryID)
	if err != nil {
		return ScanResult{}, err
	}

	root := filepath.Clean(lib.Path)
	info, err := os.Stat(root)
	if err != nil {
		return ScanResult{}, fmt.Errorf("library path: %w", err)
	}
	if !info.IsDir() {
		return ScanResult{}, fmt.Errorf("library path is not a directory")
	}

	if err := s.libraries.SetScanState(libraryID, "scanning", ""); err != nil {
		return ScanResult{}, err
	}

	slog.Info("local library scan started", "library_id", libraryID, "path", root)

	result := ScanResult{}
	seen := make(map[string]struct{})
	s.setProgress(libraryID, "scanning", 0)

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if len(result.Errors) < maxScanErrors {
				result.Errors = append(result.Errors, walkErr.Error())
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if !IsMediaFile(d.Name()) {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			if len(result.Errors) < maxScanErrors {
				result.Errors = append(result.Errors, err.Error())
			}
			return nil
		}
		rel = filepath.ToSlash(rel)
		seen[rel] = struct{}{}

		fileInfo, err := d.Info()
		if err != nil {
			if len(result.Errors) < maxScanErrors {
				result.Errors = append(result.Errors, err.Error())
			}
			return nil
		}

		action, err := s.processFile(libraryID, root, rel, path, fileInfo)
		if err != nil {
			if len(result.Errors) < maxScanErrors {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", rel, err))
			}
			return nil
		}
		switch action {
		case "added":
			result.Added++
		case "updated":
			result.Updated++
		case "unchanged":
			result.Unchanged++
		}
		result.Total++
		if result.Total%25 == 0 {
			s.setProgress(libraryID, "scanning", result.Total)
		}
		return nil
	})
	if walkErr != nil {
		_ = s.libraries.SetScanState(libraryID, "error", walkErr.Error())
		slog.Warn("local library scan failed",
			"library_id", libraryID,
			"path", root,
			"err", walkErr,
		)
		return result, walkErr
	}

	s.setProgress(libraryID, "reconciling", result.Total)

	removed, err := s.tracks.MarkMissingExcept(libraryID, seen)
	if err != nil {
		_ = s.libraries.SetScanState(libraryID, "error", err.Error())
		return result, err
	}
	result.Removed = removed

	dupes, err := s.tracks.ReconcileDuplicates(libraryID)
	if err != nil {
		_ = s.libraries.SetScanState(libraryID, "error", err.Error())
		return result, err
	}
	result.Duplicates = dupes

	if err := s.updateLibraryStats(libraryID); err != nil {
		_ = s.libraries.SetScanState(libraryID, "error", err.Error())
		return result, err
	}

	slog.Info("local library scan finished",
		"library_id", libraryID,
		"path", root,
		"added", result.Added,
		"updated", result.Updated,
		"removed", result.Removed,
		"duplicates", result.Duplicates,
		"unchanged", result.Unchanged,
		"total", result.Total,
		"errors", len(result.Errors),
	)
	if len(result.Errors) > 0 {
		slog.Warn("local library scan completed with file errors",
			"library_id", libraryID,
			"error_count", len(result.Errors),
			"first_error", result.Errors[0],
		)
	}

	return result, nil
}

func (s *Scanner) updateLibraryStats(libraryID string) error {
	present, missing, duplicates, err := s.tracks.CountByStatus(libraryID)
	if err != nil {
		return err
	}
	return s.libraries.UpdateScanStats(libraryID, present, missing, duplicates)
}

// FileChange describes a single filesystem change observed by the library
// watcher. RelPath is slash-separated and relative to the library root.
type FileChange struct {
	RelPath string
	Dir     bool
	Removed bool
}

// ApplyChanges incrementally updates the catalog for files that were created,
// modified, renamed, or removed while the library was being watched. A file
// whose content hash matches a track currently marked missing is treated as a
// rename or move and keeps its track identity.
func (s *Scanner) ApplyChanges(libraryID string, changes []FileChange) error {
	if len(changes) == 0 {
		return nil
	}
	mu := s.libraryScanMu(libraryID)
	mu.Lock()
	defer mu.Unlock()

	lib, err := s.libraries.Get(libraryID)
	if err != nil {
		return err
	}
	root := filepath.Clean(lib.Path)

	for _, change := range changes {
		if change.Removed && change.Dir && (change.RelPath == "" || change.RelPath == ".") {
			// The library root itself went away. Every tracked file is missing.
			if _, err := s.tracks.MarkMissingExcept(libraryID, map[string]struct{}{}); err != nil {
				slog.Warn("local library watch: mark all missing failed",
					"library_id", libraryID, "err", err)
			}
			continue
		}
		if change.RelPath == "" || change.RelPath == "." {
			continue
		}
		var err error
		if change.Removed {
			err = s.removePath(libraryID, change.RelPath, change.Dir)
		} else {
			err = s.syncFile(libraryID, root, change.RelPath)
		}
		if err != nil {
			slog.Warn("local library watch update failed",
				"library_id", libraryID,
				"path", change.RelPath,
				"err", err,
			)
		}
	}
	if _, err := s.tracks.ReconcileDuplicates(libraryID); err != nil {
		slog.Warn("local library duplicate reconcile failed", "library_id", libraryID, "err", err)
	}
	return s.updateLibraryStats(libraryID)
}

func (s *Scanner) removePath(libraryID, relPath string, isDir bool) error {
	if isDir {
		_, err := s.tracks.MarkMissingByPrefix(libraryID, relPath)
		return err
	}
	_, err := s.tracks.MarkMissingByRelPath(libraryID, relPath)
	return err
}

// syncFile upserts one file into the catalog. When no row exists at relPath but
// a missing row carries the same content hash, the missing row is reassigned to
// the new path so renames keep play counts, favorites, and queue references.
func (s *Scanner) syncFile(libraryID, root, relPath string) error {
	absPath := filepath.Join(root, filepath.FromSlash(relPath))
	info, err := os.Stat(absPath)
	if err != nil {
		return s.removePath(libraryID, relPath, false)
	}
	if info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
		return nil
	}
	if !IsMediaFile(absPath) {
		return nil
	}

	_, lookupErr := s.tracks.GetByRelPath(libraryID, relPath)
	if lookupErr != nil && lookupErr != sql.ErrNoRows {
		return lookupErr
	}

	if lookupErr == sql.ErrNoRows {
		size := info.Size()
		mtime := info.ModTime().Unix()
		if hash, herr := ContentHash(absPath, size); herr == nil {
			if missing, merr := s.tracks.FindMissingByContentHash(libraryID, hash); merr == nil {
				sig := FileSignature(absPath, size, mtime)
				if rerr := s.tracks.ReassignTrackPath(missing.ID, relPath, absPath, sig, size, mtime); rerr == nil {
					slog.Info("local library track moved",
						"library_id", libraryID,
						"track_id", missing.ID,
						"from", missing.RelPath,
						"to", relPath,
					)
					return nil
				}
			}
		}
	}

	_, err = s.processFile(libraryID, root, relPath, absPath, info)
	return err
}

func (s *Scanner) processFile(libraryID, root, relPath, absPath string, info fs.FileInfo) (string, error) {
	size := info.Size()
	mtime := info.ModTime().Unix()
	sig := FileSignature(absPath, size, mtime)

	existing, lookupErr := s.tracks.GetByRelPath(libraryID, relPath)
	if lookupErr == nil && existing.FileSig == sig && existing.Status == store.TrackStatusPresent {
		return "unchanged", nil
	}

	contentHash, err := ContentHash(absPath, size)
	if err != nil {
		return "", err
	}

	meta := readMetadata(absPath)
	input := store.UpsertLocalTrackInput{
		LibraryID:   libraryID,
		RelPath:     relPath,
		AbsPath:     absPath,
		FileSig:     sig,
		ContentHash: contentHash,
		Size:        size,
		Mtime:       mtime,
		Title:       meta.title,
		Artist:      meta.artist,
		Album:       meta.album,
		AlbumArtist: meta.albumArtist,
		TrackNum:    meta.trackNum,
		DiscNum:     meta.discNum,
		DurationMs:  meta.durationMs,
		Genre:       meta.genre,
		Year:        meta.year,
		Format:      FormatFromPath(absPath),
		MediaKind:   MediaKindFromPath(absPath),
	}

	if _, err := s.tracks.Upsert(input); err != nil {
		return "", err
	}
	if lookupErr != nil {
		return "added", nil
	}
	return "updated", nil
}

type trackMeta struct {
	title       string
	artist      string
	album       string
	albumArtist string
	trackNum    int
	discNum     int
	durationMs  int
	genre       string
	year        int
}

func readMetadata(path string) trackMeta {
	meta, err := ReadTrackMetadata(path)
	if err != nil {
		return trackMeta{}
	}
	return trackMeta{
		title:       meta.Title,
		artist:      meta.Artist,
		album:       meta.Album,
		albumArtist: meta.AlbumArtist,
		trackNum:    meta.TrackNum,
		discNum:     meta.DiscNum,
		genre:       meta.Genre,
		year:        meta.Year,
	}
}

// RescanFile re-reads tags for one library file and updates the database row.
func (s *Scanner) RescanFile(libraryID, trackID string) (store.LocalTrack, error) {
	lib, err := s.libraries.Get(libraryID)
	if err != nil {
		return store.LocalTrack{}, err
	}
	track, err := s.tracks.Get(libraryID, trackID)
	if err != nil {
		return store.LocalTrack{}, err
	}
	if track.Status != store.TrackStatusPresent || track.DuplicateOf != "" {
		return store.LocalTrack{}, fmt.Errorf("track is not editable")
	}

	absPath := filepath.Join(lib.Path, filepath.FromSlash(track.RelPath))
	info, err := os.Stat(absPath)
	if err != nil {
		return store.LocalTrack{}, err
	}

	action, err := s.processFile(libraryID, lib.Path, track.RelPath, absPath, info)
	if err != nil {
		return store.LocalTrack{}, err
	}
	if action == "" {
		return store.LocalTrack{}, fmt.Errorf("rescan file")
	}
	return s.tracks.Get(libraryID, trackID)
}
