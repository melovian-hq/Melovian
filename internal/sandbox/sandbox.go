// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package sandbox

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"melovian/internal/appconfig"
)

type Mode int

const (
	ModeDesktop Mode = iota
	ModeServer
)

type Status struct {
	Enabled   bool
	Supported bool
	Reason    string
	Detail    string
}

type pathEntry struct {
	path        string
	readWrite   bool
	resolveUnix bool
}

func Enabled() bool {
	raw := strings.TrimSpace(os.Getenv("MELOVIAN_LANDLOCK"))
	if raw == "" {
		return true
	}
	switch strings.ToLower(raw) {
	case "0", "false", "off", "no", "disabled":
		return false
	default:
		return true
	}
}

func LogStatus(status Status) {
	if status.Enabled {
		slog.Info("landlock enabled", "detail", status.Detail)
		return
	}
	if status.Supported {
		slog.Warn("landlock disabled", "reason", status.Reason)
		return
	}
	slog.Info("landlock disabled", "reason", status.Reason)
}

func collectPathEntries(cfg appconfig.Config, mode Mode) ([]pathEntry, error) {
	dataDir, err := absClean(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("data dir: %w", err)
	}
	if dataDir == "" {
		return nil, fmt.Errorf("data dir is empty")
	}

	entries := []pathEntry{{path: dataDir, readWrite: true}}

	for _, path := range systemROPaths() {
		entries = append(entries, pathEntry{path: path})
	}

	for _, path := range executableDirs() {
		entries = append(entries, pathEntry{path: path})
	}

	if lib := cfg.LocalLibraryEffective(); lib.Enabled {
		if libPath, err := absClean(lib.DefaultPath); err != nil {
			return nil, fmt.Errorf("local library path: %w", err)
		} else if libPath != "" {
			entries = append(entries, pathEntry{path: libPath})
		}
		if lib.AllowCustomPath {
			for _, root := range localLibraryBrowseRoots() {
				clean, err := absClean(root)
				if err != nil || clean == "" {
					continue
				}
				entries = append(entries, pathEntry{path: clean})
			}
		}
	}

	if mode == ModeDesktop {
		if runtimeDir := desktopRuntimeDir(); runtimeDir != "" {
			entries = append(entries, pathEntry{path: runtimeDir, resolveUnix: true})
		}
	}

	return dedupePathEntries(entries), nil
}

func dedupePathEntries(entries []pathEntry) []pathEntry {
	seen := make(map[string]pathEntry, len(entries))
	for _, entry := range entries {
		if entry.path == "" {
			continue
		}
		existing, ok := seen[entry.path]
		if !ok {
			seen[entry.path] = entry
			continue
		}
		if !existing.readWrite && entry.readWrite {
			entry.resolveUnix = entry.resolveUnix || existing.resolveUnix
			seen[entry.path] = entry
			continue
		}
		if existing.readWrite && entry.resolveUnix {
			existing.resolveUnix = true
			seen[entry.path] = existing
		}
	}

	out := make([]pathEntry, 0, len(seen))
	for _, entry := range seen {
		out = append(out, entry)
	}
	return out
}

func systemROPaths() []string {
	return []string{"/usr", "/lib", "/lib64", "/etc"}
}

// localLibraryBrowseRoots are read-only roots opened when custom library paths
// are enabled. Landlock cannot expand after startup, so browse and scan need
// these roots up front.
func localLibraryBrowseRoots() []string {
	roots := []string{"/media", "/mnt", "/run/media"}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		roots = append([]string{home}, roots...)
	}
	return roots
}

func executableDirs() []string {
	var dirs []string
	exe, err := os.Executable()
	if err != nil {
		return dirs
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil && resolved != "" {
		exe = resolved
	}
	exeDir := filepath.Dir(exe)
	dirs = append(dirs,
		exeDir,
		filepath.Join(exeDir, "lib"),
		filepath.Join(exeDir, "..", "lib"),
		filepath.Join(exeDir, "..", "lib64"),
		filepath.Join(exeDir, "..", "lib", "x86_64-linux-gnu"),
		filepath.Join(exeDir, "..", "lib", "aarch64-linux-gnu"),
	)
	if appdir := strings.TrimSpace(os.Getenv("APPDIR")); appdir != "" {
		dirs = append(dirs,
			filepath.Join(appdir, "usr", "lib"),
			filepath.Join(appdir, "usr", "lib64"),
			filepath.Join(appdir, "usr", "lib", "x86_64-linux-gnu"),
			filepath.Join(appdir, "usr", "lib", "aarch64-linux-gnu"),
		)
	}
	return dirs
}

func desktopRuntimeDir() string {
	if dir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")); dir != "" {
		return dir
	}
	if uid := os.Getuid(); uid >= 0 {
		return filepath.Join("/run/user", strconv.Itoa(uid))
	}
	return ""
}

func absClean(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}
