// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package metaloader

import (
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchDebounce is exported for tests that need to wait out the flush delay.
const WatchDebounce = 700 * time.Millisecond

// LibraryWatcher keeps local library catalogs in sync by watching library roots
// for filesystem changes and applying them incrementally through the Scanner.
type LibraryWatcher struct {
	scanner  *Scanner
	onChange func(libraryID string)

	fsw *fsnotify.Watcher

	mu      sync.Mutex
	roots   map[string]string                // libraryID -> absolute root path
	dirs    map[string]string                // watched dir -> libraryID
	pending map[string]map[string]FileChange // libraryID -> relPath -> change
	timers  map[string]*time.Timer           // libraryID -> debounce timer
	closed  bool
	done    chan struct{}
	wg      sync.WaitGroup
}

// NewLibraryWatcher creates a watcher. onChange runs after each debounced batch
// is applied and may be nil.
func NewLibraryWatcher(scanner *Scanner, onChange func(libraryID string)) (*LibraryWatcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &LibraryWatcher{
		scanner:  scanner,
		onChange: onChange,
		fsw:      fsw,
		roots:    make(map[string]string),
		dirs:     make(map[string]string),
		pending:  make(map[string]map[string]FileChange),
		timers:   make(map[string]*time.Timer),
		done:     make(chan struct{}),
	}
	w.wg.Add(1)
	go w.loop()
	return w, nil
}

// WatchLibrary starts watching a library root recursively. It is safe to call
// again to rewatch after the root path changes.
func (w *LibraryWatcher) WatchLibrary(libraryID, root string) error {
	root = filepath.Clean(root)
	w.UnwatchLibrary(libraryID)

	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fs.ErrNotExist
	}

	w.mu.Lock()
	w.roots[libraryID] = root
	w.mu.Unlock()

	if err := w.addDirRecursive(libraryID, root); err != nil {
		w.UnwatchLibrary(libraryID)
		return err
	}
	slog.Debug("local library watch started", "library_id", libraryID, "path", root)
	return nil
}

// UnwatchLibrary stops watching a library and drops its pending changes.
func (w *LibraryWatcher) UnwatchLibrary(libraryID string) {
	w.mu.Lock()
	root := w.roots[libraryID]
	delete(w.roots, libraryID)
	for dir, id := range w.dirs {
		if id == libraryID {
			delete(w.dirs, dir)
			_ = w.fsw.Remove(dir)
		}
	}
	delete(w.pending, libraryID)
	if timer, ok := w.timers[libraryID]; ok {
		timer.Stop()
		delete(w.timers, libraryID)
	}
	w.mu.Unlock()
	if root != "" {
		slog.Debug("local library watch stopped", "library_id", libraryID, "path", root)
	}
}

// Watching reports whether a library currently has an active watch.
func (w *LibraryWatcher) Watching(libraryID string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, ok := w.roots[libraryID]
	return ok
}

// Close stops the watcher and releases inotify handles.
func (w *LibraryWatcher) Close() {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.closed = true
	for _, timer := range w.timers {
		timer.Stop()
	}
	w.timers = make(map[string]*time.Timer)
	w.mu.Unlock()
	close(w.done)
	_ = w.fsw.Close()
	w.wg.Wait()
}

func (w *LibraryWatcher) loop() {
	defer w.wg.Done()
	for {
		select {
		case <-w.done:
			return
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			w.handleEvent(ev)
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			slog.Warn("local library watch error", "err", err)
		}
	}
}

func (w *LibraryWatcher) handleEvent(ev fsnotify.Event) {
	w.mu.Lock()
	libID, root := w.libraryForPathLocked(ev.Name)
	w.mu.Unlock()
	if libID == "" {
		return
	}

	rel, err := filepath.Rel(root, ev.Name)
	if err != nil {
		return
	}
	rel = filepath.ToSlash(rel)

	if ev.Has(fsnotify.Create) {
		if info, statErr := os.Stat(ev.Name); statErr == nil && info.IsDir() {
			w.mu.Lock()
			if addErr := w.addDirRecursiveLocked(libID, ev.Name); addErr != nil {
				slog.Debug("local library watch: add dir failed", "path", ev.Name, "err", addErr)
			}
			w.mu.Unlock()
			w.enqueueDirContents(libID, root, ev.Name)
			return
		}
		if !IsMediaFile(ev.Name) {
			return
		}
		w.enqueue(libID, FileChange{RelPath: rel})
		return
	}

	if ev.Has(fsnotify.Write) {
		if !IsMediaFile(ev.Name) {
			return
		}
		w.enqueue(libID, FileChange{RelPath: rel})
		return
	}

	if ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename) {
		if rel == "." {
			// The library root itself was removed or renamed. Mark every track
			// missing so the catalog reflects reality and drop the dead watch.
			w.mu.Lock()
			w.forgetDirsLocked(libID)
			w.mu.Unlock()
			w.enqueue(libID, FileChange{RelPath: "", Dir: true, Removed: true})
			return
		}
		w.mu.Lock()
		_, wasDir := w.dirs[filepath.Clean(ev.Name)]
		if wasDir {
			w.forgetDirsUnderLocked(filepath.Clean(ev.Name))
		}
		w.mu.Unlock()
		if wasDir {
			w.enqueue(libID, FileChange{RelPath: rel, Dir: true, Removed: true})
			return
		}
		if !IsMediaFile(ev.Name) {
			return
		}
		w.enqueue(libID, FileChange{RelPath: rel, Removed: true})
	}
}

// libraryForPathLocked returns the library owning path by longest watched dir
// prefix. Caller must hold w.mu.
func (w *LibraryWatcher) libraryForPathLocked(path string) (string, string) {
	path = filepath.Clean(path)
	best := ""
	libID := ""
	for dir, id := range w.dirs {
		if path == dir || strings.HasPrefix(path, dir+string(filepath.Separator)) {
			if len(dir) > len(best) {
				best = dir
				libID = id
			}
		}
	}
	if libID == "" {
		return "", ""
	}
	return libID, w.roots[libID]
}

func (w *LibraryWatcher) enqueue(libraryID string, change FileChange) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	if _, ok := w.roots[libraryID]; !ok {
		return
	}
	batch := w.pending[libraryID]
	if batch == nil {
		batch = make(map[string]FileChange)
		w.pending[libraryID] = batch
	}
	// A removal always wins over a pending upsert for the same path; an upsert
	// after a removal means the file was recreated.
	if _, exists := batch[change.RelPath]; !exists || !change.Removed || change.Dir {
		batch[change.RelPath] = change
	}
	if timer, ok := w.timers[libraryID]; ok {
		timer.Stop()
	}
	w.timers[libraryID] = time.AfterFunc(WatchDebounce, func() {
		w.flush(libraryID)
	})
}

func (w *LibraryWatcher) flush(libraryID string) {
	w.mu.Lock()
	batch := w.pending[libraryID]
	delete(w.pending, libraryID)
	delete(w.timers, libraryID)
	root, watching := w.roots[libraryID]
	onChange := w.onChange
	w.mu.Unlock()

	if len(batch) == 0 {
		return
	}
	if !watching {
		return
	}

	// Apply removals before upserts so a rename marks the old row missing
	// before the new path is processed, letting content-hash adoption run.
	changes := make([]FileChange, 0, len(batch))
	for _, change := range batch {
		if change.Removed {
			changes = append(changes, change)
		}
	}
	for _, change := range batch {
		if !change.Removed {
			changes = append(changes, change)
		}
	}
	if err := w.scanner.ApplyChanges(libraryID, changes); err != nil {
		slog.Warn("local library watch apply failed",
			"library_id", libraryID,
			"path", root,
			"err", err,
		)
		return
	}
	slog.Debug("local library watch applied changes",
		"library_id", libraryID,
		"path", root,
		"changes", len(changes),
	)
	if onChange != nil {
		onChange(libraryID)
	}
}

func (w *LibraryWatcher) addDirRecursive(libraryID, dir string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.addDirRecursiveLocked(libraryID, dir)
}

// addDirRecursiveLocked registers fsnotify watches for dir and all subdirs.
// Caller must hold w.mu.
func (w *LibraryWatcher) addDirRecursiveLocked(libraryID, dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return filepath.SkipDir
		}
		path = filepath.Clean(path)
		if _, ok := w.dirs[path]; ok {
			return nil
		}
		if err := w.fsw.Add(path); err != nil {
			slog.Debug("local library watch: watch dir failed", "path", path, "err", err)
			return nil
		}
		w.dirs[path] = libraryID
		return nil
	})
}

// enqueueDirContents upserts media files found inside a newly created
// directory, covering moves where only the directory create event is seen.
func (w *LibraryWatcher) enqueueDirContents(libraryID, root, dir string) {
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 || !IsMediaFile(d.Name()) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		w.enqueue(libraryID, FileChange{RelPath: filepath.ToSlash(rel)})
		return nil
	})
}

// forgetDirsLocked drops all watch registrations for a library.
// Caller must hold w.mu.
func (w *LibraryWatcher) forgetDirsLocked(libraryID string) {
	for dir, id := range w.dirs {
		if id == libraryID {
			delete(w.dirs, dir)
			_ = w.fsw.Remove(dir)
		}
	}
}

// forgetDirsUnderLocked drops watch registrations below a removed directory.
// Caller must hold w.mu.
func (w *LibraryWatcher) forgetDirsUnderLocked(dir string) {
	prefix := dir + string(filepath.Separator)
	for watched := range w.dirs {
		if watched == dir || strings.HasPrefix(watched, prefix) {
			delete(w.dirs, watched)
			_ = w.fsw.Remove(watched)
		}
	}
}
