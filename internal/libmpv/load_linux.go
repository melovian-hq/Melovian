// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build linux && !android

package libmpv

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/ebitengine/purego"
)

// lcNumeric is the glibc category index for LC_NUMERIC.
const lcNumeric int32 = 1

var (
	setlocaleOnce sync.Once
	fnSetlocale   func(category int32, locale string) string
)

// prepareCreateEnv forces LC_NUMERIC to "C" before mpv_create runs. libmpv
// requires the C numeric locale. GTK and WebKit call setlocale(LC_ALL, "")
// during initialization, which switches LC_NUMERIC to the user's locale and
// makes mpv_create return NULL on systems whose locale uses a non-period
// decimal separator.
func prepareCreateEnv() {
	setlocaleOnce.Do(func() {
		handle, err := purego.Dlopen("libc.so.6", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			return
		}
		purego.RegisterLibFunc(&fnSetlocale, uintptr(handle), "setlocale")
	})
	if fnSetlocale != nil {
		fnSetlocale(lcNumeric, "C")
	}
}

func tryLoadLibrary() error {
	boostLibraryPath()

	candidates := libmpvCandidates()
	var lastErr error
	for _, name := range candidates {
		handle, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			lastErr = err
			continue
		}
		libHandle = uintptr(handle)
		return nil
	}
	if lastErr != nil {
		return fmt.Errorf("libmpv: %w", lastErr)
	}
	return errUnavailable
}

// boostLibraryPath prepends common library directories, including any bundled
// alongside the executable, to LD_LIBRARY_PATH. This mainly helps subprocesses
// and lazily resolved dependencies. The current process still resolves bundled
// libmpv through the absolute paths returned by libmpvCandidates.
func boostLibraryPath() {
	extra := append(bundledLibDirs(), systemLibDirs()...)
	current := os.Getenv("LD_LIBRARY_PATH")
	parts := append(extra, strings.Split(current, ":")...)
	seen := make(map[string]struct{}, len(parts))
	ordered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		ordered = append(ordered, part)
	}
	if len(ordered) > 0 {
		_ = os.Setenv("LD_LIBRARY_PATH", strings.Join(ordered, ":"))
	}
}

// bundledLibDirs returns directories that may hold a libmpv shipped alongside
// the application, such as an AppImage payload (usr/lib next to usr/bin) or a
// portable install placed beside the binary.
func bundledLibDirs() []string {
	var dirs []string
	if exe, err := os.Executable(); err == nil {
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
	}
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

// systemLibDirs lists the standard system library directories to search.
func systemLibDirs() []string {
	return []string{
		"/usr/lib",
		"/usr/lib64",
		"/usr/lib/x86_64-linux-gnu",
		"/usr/lib/aarch64-linux-gnu",
		"/usr/local/lib",
		"/usr/local/lib64",
	}
}

func libmpvCandidates() []string {
	seen := make(map[string]struct{})
	var paths []string
	add := func(candidate string, requireStat bool) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}
		if requireStat {
			info, err := os.Stat(candidate)
			if err != nil || info.IsDir() {
				return
			}
			if resolved, err := filepath.EvalSymlinks(candidate); err == nil && resolved != "" {
				candidate = resolved
			}
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		paths = append(paths, candidate)
	}

	globInto := func(dir string) {
		matches, err := filepath.Glob(filepath.Join(dir, "libmpv.so*"))
		if err != nil {
			return
		}
		sort.Slice(matches, func(i, j int) bool {
			return libmpvPreference(matches[i]) > libmpvPreference(matches[j])
		})
		for _, match := range matches {
			add(match, true)
		}
	}

	// Bundled libraries (AppImage / portable) are resolved by absolute path so
	// they load regardless of the LD_LIBRARY_PATH captured at process start, and
	// take priority over a missing or incompatible system copy.
	bundled := bundledLibDirs()
	system := systemLibDirs()
	for _, dir := range bundled {
		globInto(dir)
	}
	for _, dir := range system {
		globInto(dir)
	}

	// Explicit soname fallbacks in case globbing missed a versioned file.
	for _, dir := range append(append([]string{}, bundled...), system...) {
		for _, name := range []string{"libmpv.so.2", "libmpv.so.1", "libmpv.so"} {
			add(filepath.Join(dir, name), true)
		}
	}

	// Bare sonames rely on the dynamic loader search path (the ldconfig cache
	// and the LD_LIBRARY_PATH captured when the process started).
	for _, name := range []string{"libmpv.so.2", "libmpv.so.1", "libmpv.so"} {
		add(name, false)
	}

	return paths
}

func libmpvPreference(path string) int {
	switch {
	case strings.Contains(path, ".so.2"):
		return 3
	case strings.Contains(path, ".so.1"):
		return 2
	case strings.HasSuffix(path, ".so"):
		return 1
	default:
		return 0
	}
}
