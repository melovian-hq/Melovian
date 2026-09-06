// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package sandbox

import (
	"path/filepath"
	"testing"

	"melovian/internal/appconfig"
)

func TestEnabledRespectsEnv(t *testing.T) {
	t.Setenv("MELOVIAN_LANDLOCK", "0")
	if Enabled() {
		t.Fatal("expected landlock disabled when MELOVIAN_LANDLOCK=0")
	}

	t.Setenv("MELOVIAN_LANDLOCK", "off")
	if Enabled() {
		t.Fatal("expected landlock disabled when MELOVIAN_LANDLOCK=off")
	}

	t.Setenv("MELOVIAN_LANDLOCK", "1")
	if !Enabled() {
		t.Fatal("expected landlock enabled when MELOVIAN_LANDLOCK=1")
	}

	t.Setenv("MELOVIAN_LANDLOCK", "")
	if !Enabled() {
		t.Fatal("expected landlock enabled by default")
	}
}

func TestCollectPathEntriesIncludesDataDirRW(t *testing.T) {
	dataDir := t.TempDir()
	cfg := appconfig.Config{DataDir: dataDir}

	entries, err := collectPathEntries(cfg, ModeDesktop)
	if err != nil {
		t.Fatalf("collectPathEntries: %v", err)
	}

	absDataDir, err := absClean(dataDir)
	if err != nil {
		t.Fatalf("absClean: %v", err)
	}

	found := false
	for _, entry := range entries {
		if entry.path == absDataDir && entry.readWrite {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected RW data dir entry, got %+v", entries)
	}
}

func TestCollectPathEntriesIncludesLocalLibraryRO(t *testing.T) {
	dataDir := t.TempDir()
	libraryDir := filepath.Join(t.TempDir(), "music")
	cfg := appconfig.Config{
		DataDir: dataDir,
		LocalLibrary: appconfig.LocalLibraryConfig{
			Enabled:     true,
			DefaultPath: libraryDir,
		},
		LocalLibraryOverride: true,
	}

	entries, err := collectPathEntries(cfg, ModeServer)
	if err != nil {
		t.Fatalf("collectPathEntries: %v", err)
	}

	absLibraryDir, err := absClean(libraryDir)
	if err != nil {
		t.Fatalf("absClean: %v", err)
	}

	found := false
	for _, entry := range entries {
		if entry.path == absLibraryDir && !entry.readWrite {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected RO local library entry, got %+v", entries)
	}
}

func TestCollectPathEntriesCustomPathAddsBrowseRoots(t *testing.T) {
	dataDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := appconfig.Config{
		DataDir: dataDir,
		LocalLibrary: appconfig.LocalLibraryConfig{
			Enabled:         true,
			AllowCustomPath: true,
		},
		LocalLibraryOverride: true,
	}

	entries, err := collectPathEntries(cfg, ModeServer)
	if err != nil {
		t.Fatalf("collectPathEntries: %v", err)
	}

	absHome, err := absClean(home)
	if err != nil {
		t.Fatalf("absClean: %v", err)
	}

	foundHome := false
	foundMedia := false
	for _, entry := range entries {
		if entry.path == absHome && !entry.readWrite {
			foundHome = true
		}
		if entry.path == "/run/media" && !entry.readWrite {
			foundMedia = true
		}
	}
	if !foundHome {
		t.Fatalf("expected home browse root, got %+v", entries)
	}
	if !foundMedia {
		t.Fatalf("expected /run/media browse root, got %+v", entries)
	}
}

func TestCollectPathEntriesDesktopIncludesResolveUnixRuntime(t *testing.T) {
	dataDir := t.TempDir()
	runtimeDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)

	cfg := appconfig.Config{DataDir: dataDir}
	entries, err := collectPathEntries(cfg, ModeDesktop)
	if err != nil {
		t.Fatalf("collectPathEntries: %v", err)
	}

	absRuntimeDir, err := absClean(runtimeDir)
	if err != nil {
		t.Fatalf("absClean: %v", err)
	}

	found := false
	for _, entry := range entries {
		if entry.path == absRuntimeDir && entry.resolveUnix {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected resolve-unix runtime entry, got %+v", entries)
	}
}

func TestDedupePathEntriesPrefersReadWrite(t *testing.T) {
	entries := dedupePathEntries([]pathEntry{
		{path: "/data", readWrite: false},
		{path: "/data", readWrite: true},
	})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if !entries[0].readWrite {
		t.Fatal("expected read-write rule to win dedupe")
	}
}

func TestApplyDisabledByEnv(t *testing.T) {
	t.Setenv("MELOVIAN_LANDLOCK", "false")
	status := Apply(appconfig.Config{DataDir: t.TempDir()}, ModeServer)
	if status.Enabled {
		t.Fatal("expected landlock to stay disabled")
	}
	if status.Reason == "" {
		t.Fatal("expected disabled reason")
	}
}
