// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"melovian/internal/httputil"
	"melovian/internal/osutil"
)

const maxDirectoryListing = 500

func (s *Server) registerFilesystemRoutes() {
	s.mux.HandleFunc("GET /api/filesystem/directories", s.handleListDirectories)
}

func (s *Server) handleListDirectories(w http.ResponseWriter, r *http.Request) {
	cfg := s.localLibraryConfig()
	if !cfg.Enabled || !cfg.AllowCustomPath {
		httputil.WriteError(w, http.StatusForbidden, "filesystem_browse_disabled", "directory browsing is disabled")
		return
	}

	raw := strings.TrimSpace(r.URL.Query().Get("path"))
	path, err := s.resolveBrowsePath(raw)
	if err != nil {
		if os.IsNotExist(err) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "directory not found")
			return
		}
		httputil.WriteError(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}

	entries, err := listSubdirectories(path)
	if err != nil {
		if os.IsPermission(err) {
			httputil.WriteError(w, http.StatusForbidden, "permission_denied", "cannot read that directory. Stay under your home folder, /media, /mnt, or /run/media")
			return
		}
		if os.IsNotExist(err) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "directory not found")
			return
		}
		httputil.WriteError(w, http.StatusBadRequest, "list_failed", err.Error())
		return
	}

	parent := ""
	if path != filepath.Dir(path) {
		candidate := filepath.Dir(path)
		if s.pathAllowedForBrowse(candidate) {
			parent = candidate
		}
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"path":    path,
		"parent":  parent,
		"entries": entries,
	})
}

func (s *Server) resolveBrowsePath(raw string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" {
		for _, candidate := range s.browseStartCandidates() {
			info, err := os.Stat(candidate)
			if err != nil || !info.IsDir() {
				continue
			}
			if _, err := os.Open(candidate); err != nil { //#nosec G304 -- candidate is from fixed browse roots
				continue
			}
			return filepath.Clean(candidate), nil
		}
		return "", filesystemBrowseError("no readable folder found under home, /media, /mnt, or /run/media")
	}
	if !filepath.IsAbs(path) {
		return "", errAbsolutePathRequired
	}
	// Resolve before the allowlist check so a symlink inside an allowed root
	// cannot point the listing somewhere else.
	resolved, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	if !s.pathAllowedForBrowse(resolved) {
		return "", errBrowseOutsideRoots
	}
	info, err := os.Stat(resolved) //#nosec G703 -- resolved passed the browse allowlist after symlink resolution
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errNotDirectory
	}
	return resolved, nil
}

// pathAllowedForBrowse reports whether path is under a configured browse root.
// Callers must pass a symlink-resolved path.
func (s *Server) pathAllowedForBrowse(path string) bool {
	path = filepath.Clean(path)
	for _, root := range s.browseStartCandidates() {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		if err := osutil.PathEscapesRoot(filepath.Clean(root), path); err == nil {
			return true
		}
	}
	return false
}

func (s *Server) browseStartCandidates() []string {
	var out []string
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		out = append(out, home)
	}
	out = append(out, "/run/media", "/media", "/mnt")
	if strings.TrimSpace(s.cfg.DataDir) != "" {
		out = append(out, s.cfg.DataDir)
	}
	out = append(out, s.localLibraryConfig().Roots...)
	return out
}

var (
	errAbsolutePathRequired = filesystemBrowseError("path must be absolute")
	errNotDirectory         = filesystemBrowseError("path is not a directory")
	errBrowseOutsideRoots   = filesystemBrowseError("path must stay under your home folder, /media, /mnt, /run/media, or the app data directory")
)

type filesystemBrowseError string

func (e filesystemBrowseError) Error() string { return string(e) }

type directoryEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func listSubdirectories(path string) ([]directoryEntry, error) {
	dir, err := os.Open(path) //#nosec G304,G703 -- path already passed browse allowlist
	if err != nil {
		return nil, err
	}
	defer func() { _ = dir.Close() }()

	names, err := dir.Readdirnames(maxDirectoryListing + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	sort.Strings(names)

	entries := make([]directoryEntry, 0, len(names))
	for _, name := range names {
		if name == "." || name == ".." {
			continue
		}
		if strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(path, name)
		info, err := os.Lstat(full) //#nosec G703 -- full is under allowlisted browse path
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if !info.IsDir() {
			continue
		}
		entries = append(entries, directoryEntry{Name: name, Path: full})
		if len(entries) >= maxDirectoryListing {
			break
		}
	}
	return entries, nil
}
