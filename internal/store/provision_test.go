// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"os"
	"path/filepath"
	"testing"

	"melovian/internal/appconfig"
)

func provisionMusicDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "music")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	return dir
}

func localLibraryConfig(path string) appconfig.Config {
	return appconfig.Config{
		LocalLibrary:         appconfig.LocalLibraryConfig{Enabled: true, DefaultPath: path},
		LocalLibraryOverride: true,
	}
}

func TestProvisionLocalLibraryFromConfig(t *testing.T) {
	musicDir := provisionMusicDir(t)
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"), localLibraryConfig(musicDir))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer func() { _ = db.Close() }()

	libraries := NewLocalLibraryStore(db)
	items, err := libraries.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 provisioned library, got %d", len(items))
	}
	lib := items[0]
	if lib.Path != filepath.Clean(musicDir) {
		t.Fatalf("library path = %q, want %q", lib.Path, musicDir)
	}
	if lib.Name != "music" {
		t.Fatalf("library name = %q, want music", lib.Name)
	}
	if lib.UserID != "" {
		t.Fatalf("expected system-wide library, got user %q", lib.UserID)
	}
	activeID, err := libraries.GetActiveID()
	if err != nil {
		t.Fatalf("GetActiveID: %v", err)
	}
	if activeID != lib.ID {
		t.Fatalf("active library = %q, want %q", activeID, lib.ID)
	}
	mode, err := NewPreferencesStore(db).GetSourceViewMode("")
	if err != nil {
		t.Fatalf("GetSourceViewMode: %v", err)
	}
	if mode != SourceViewLocal {
		t.Fatalf("source view mode = %q, want local", mode)
	}
}

func TestProvisionLocalLibraryRootName(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"), localLibraryConfig("/"))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer func() { _ = db.Close() }()

	items, err := NewLocalLibraryStore(db).List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 || items[0].Name != "Music" {
		t.Fatalf("expected fallback name Music, got %+v", items)
	}
}

func TestProvisionBothSourcesUnified(t *testing.T) {
	musicDir := provisionMusicDir(t)
	cfg := localLibraryConfig(musicDir)
	cfg.LegacyServer = "https://navidrome.example.com"
	cfg.LegacyUser = "demo"
	cfg.LegacyPass = "secret"

	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"), cfg)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer func() { _ = db.Close() }()

	instances := NewInstanceStore(db)
	inst, err := instances.GetActive()
	if err != nil {
		t.Fatalf("GetActive instance: %v", err)
	}
	if inst.ServerURL != "https://navidrome.example.com" {
		t.Fatalf("instance url = %q", inst.ServerURL)
	}

	libraries := NewLocalLibraryStore(db)
	activeLib, err := libraries.GetActiveID()
	if err != nil {
		t.Fatalf("GetActiveID: %v", err)
	}
	if activeLib == "" {
		t.Fatal("expected active local library")
	}

	mode, err := NewPreferencesStore(db).GetSourceViewMode("")
	if err != nil {
		t.Fatalf("GetSourceViewMode: %v", err)
	}
	if mode != SourceViewUnified {
		t.Fatalf("source view mode = %q, want unified", mode)
	}
}

func TestProvisionLegacyInstanceOnly(t *testing.T) {
	t.Setenv("MELOVIAN_LOCAL_LIBRARY_PATH", "")
	cfg := appconfig.Config{
		LegacyServer: "https://navidrome.example.com",
		LegacyUser:   "demo",
		LegacyPass:   "secret",
	}
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"), cfg)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer func() { _ = db.Close() }()

	instances := NewInstanceStore(db)
	inst, err := instances.GetActive()
	if err != nil {
		t.Fatalf("GetActive: %v", err)
	}
	if inst.ServerURL != "https://navidrome.example.com" {
		t.Fatalf("instance url = %q", inst.ServerURL)
	}
	mode, err := NewPreferencesStore(db).GetSourceViewMode("")
	if err != nil {
		t.Fatalf("GetSourceViewMode: %v", err)
	}
	if mode != SourceViewSubsonic {
		t.Fatalf("source view mode = %q, want subsonic", mode)
	}
}

func TestProvisionInvalidLibraryPathSkipped(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"), localLibraryConfig(missing))
	if err != nil {
		t.Fatalf("OpenDB should not fail on invalid library path: %v", err)
	}
	defer func() { _ = db.Close() }()

	items, err := NewLocalLibraryStore(db).List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no libraries for invalid path, got %d", len(items))
	}
}

func TestProvisionRelativeLibraryPathSkipped(t *testing.T) {
	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"), localLibraryConfig("music"))
	if err != nil {
		t.Fatalf("OpenDB should not fail on relative library path: %v", err)
	}
	defer func() { _ = db.Close() }()

	items, err := NewLocalLibraryStore(db).List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no libraries for relative path, got %d", len(items))
	}
}

func TestProvisionIdempotentOnReopen(t *testing.T) {
	musicDir := provisionMusicDir(t)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db, err := OpenDB(dbPath, localLibraryConfig(musicDir))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	db, err = OpenDB(dbPath, localLibraryConfig(musicDir))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = db.Close() }()

	items, err := NewLocalLibraryStore(db).List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("reopen duplicated the library, got %d", len(items))
	}
}

func TestProvisionedSourcesClaimedByFirstUser(t *testing.T) {
	musicDir := provisionMusicDir(t)
	cfg := localLibraryConfig(musicDir)
	cfg.LegacyServer = "https://navidrome.example.com"
	cfg.LegacyUser = "demo"
	cfg.LegacyPass = "secret"

	db, err := OpenDB(filepath.Join(t.TempDir(), "test.db"), cfg)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer func() { _ = db.Close() }()

	auth := NewAuthStore(db, "test-secret")
	user, err := auth.CreateUser("admin", "password123")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	instances := NewInstanceStore(db)
	insts, err := instances.ListForUser(user.ID)
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	if len(insts) != 1 || insts[0].ServerURL != "https://navidrome.example.com" {
		t.Fatalf("expected provisioned instance visible to first user, got %+v", insts)
	}
	activeInstID, err := instances.GetActiveIDForUser(user.ID)
	if err != nil {
		t.Fatalf("GetActiveIDForUser: %v", err)
	}
	if activeInstID != insts[0].ID {
		t.Fatalf("active instance = %q, want %q", activeInstID, insts[0].ID)
	}

	libraries := NewLocalLibraryStore(db)
	libs, err := libraries.ListForUser(user.ID)
	if err != nil {
		t.Fatalf("ListForUser libraries: %v", err)
	}
	if len(libs) != 1 {
		t.Fatalf("expected provisioned library visible to first user, got %+v", libs)
	}
	activeLibID, err := libraries.GetActiveIDForUser(user.ID)
	if err != nil {
		t.Fatalf("GetActiveIDForUser library: %v", err)
	}
	if activeLibID != libs[0].ID {
		t.Fatalf("active library = %q, want %q", activeLibID, libs[0].ID)
	}

	mode, err := NewPreferencesStore(db).GetSourceViewMode(user.ID)
	if err != nil {
		t.Fatalf("GetSourceViewMode: %v", err)
	}
	if mode != SourceViewUnified {
		t.Fatalf("source view mode = %q, want unified", mode)
	}

	// A second user must not see the claimed sources.
	other, err := auth.CreateUser("other", "password123")
	if err != nil {
		t.Fatalf("CreateUser other: %v", err)
	}
	if insts, err := instances.ListForUser(other.ID); err != nil || len(insts) != 0 {
		t.Fatalf("second user must not see claimed instances, got %+v err %v", insts, err)
	}
	if libs, err := libraries.ListForUser(other.ID); err != nil || len(libs) != 0 {
		t.Fatalf("second user must not see claimed libraries, got %+v err %v", libs, err)
	}
}

func TestProvisionDoesNotResurrectDeletedSources(t *testing.T) {
	musicDir := provisionMusicDir(t)
	cfg := localLibraryConfig(musicDir)
	cfg.LegacyServer = "https://navidrome.example.com"
	cfg.LegacyUser = "demo"
	cfg.LegacyPass = "secret"
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db, err := OpenDB(dbPath, cfg)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	instances := NewInstanceStore(db)
	insts, err := instances.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, inst := range insts {
		if err := instances.Delete(inst.ID); err != nil {
			t.Fatalf("Delete instance: %v", err)
		}
	}
	libraries := NewLocalLibraryStore(db)
	libs, err := libraries.List()
	if err != nil {
		t.Fatalf("List libraries: %v", err)
	}
	for _, lib := range libs {
		if err := libraries.Delete(lib.ID); err != nil {
			t.Fatalf("Delete library: %v", err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	db, err = OpenDB(dbPath, cfg)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = db.Close() }()

	if insts, err := NewInstanceStore(db).List(); err != nil || len(insts) != 0 {
		t.Fatalf("deleted instances must not be recreated, got %+v err %v", insts, err)
	}
	if libs, err := NewLocalLibraryStore(db).List(); err != nil || len(libs) != 0 {
		t.Fatalf("deleted libraries must not be recreated, got %+v err %v", libs, err)
	}
}

func TestProvisionAppliesWhenConfigAddedLater(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db, err := OpenDB(dbPath, appconfig.Config{})
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	t.Setenv("MELOVIAN_LOCAL_LIBRARY_PATH", "")
	cfg := appconfig.Config{
		LegacyServer: "https://navidrome.example.com",
		LegacyUser:   "demo",
		LegacyPass:   "secret",
	}
	db, err = OpenDB(dbPath, cfg)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = db.Close() }()

	insts, err := NewInstanceStore(db).List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(insts) != 1 {
		t.Fatalf("config added later should still provision, got %+v", insts)
	}
}

func TestProvisionPartialSubsonicConfigRetries(t *testing.T) {
	t.Setenv("MELOVIAN_LOCAL_LIBRARY_PATH", "")
	dbPath := filepath.Join(t.TempDir(), "test.db")

	partial := appconfig.Config{
		LegacyServer: "https://navidrome.example.com",
		LegacyUser:   "demo",
	}
	db, err := OpenDB(dbPath, partial)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	if insts, err := NewInstanceStore(db).List(); err != nil || len(insts) != 0 {
		t.Fatalf("incomplete subsonic config must not provision, got %+v err %v", insts, err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	partial.LegacyPass = "secret"
	db, err = OpenDB(dbPath, partial)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = db.Close() }()

	insts, err := NewInstanceStore(db).List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(insts) != 1 {
		t.Fatalf("completed config should provision on the next start, got %+v", insts)
	}
}

func TestLegacyCredentialsEnvWinsOverSettings(t *testing.T) {
	db := OpenTestDB(t)
	for key, value := range map[string]string{
		appconfig.SettingNavidromeServer:   "http://settings-server",
		appconfig.SettingNavidromeUser:     "settings-user",
		appconfig.SettingNavidromePassword: "settings-pass",
	} {
		if err := db.setSetting(key, value); err != nil {
			t.Fatalf("setSetting %s: %v", key, err)
		}
	}

	server, user, pass := db.legacyCredentials(appconfig.Config{
		LegacyServer: "http://env-server",
		LegacyUser:   "env-user",
		LegacyPass:   "env-pass",
	})
	if server != "http://env-server" || user != "env-user" || pass != "env-pass" {
		t.Fatalf("env config should win over stored settings, got %q %q %q", server, user, pass)
	}

	// Gaps still fall back to the stored legacy settings.
	server, user, pass = db.legacyCredentials(appconfig.Config{LegacyServer: "http://env-server"})
	if server != "http://env-server" || user != "settings-user" || pass != "settings-pass" {
		t.Fatalf("stored settings should only fill gaps, got %q %q %q", server, user, pass)
	}
}

func TestProvisionSkippedWhenSourcesExist(t *testing.T) {
	t.Setenv("MELOVIAN_LOCAL_LIBRARY_PATH", "")
	musicDir := provisionMusicDir(t)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db, err := OpenDB(dbPath, appconfig.Config{})
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	libraries := NewLocalLibraryStore(db)
	if _, err := libraries.Create(CreateLocalLibraryInput{Name: "Mine", Path: musicDir}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	other := provisionMusicDir(t)
	db, err = OpenDB(dbPath, localLibraryConfig(other))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = db.Close() }()

	items, err := NewLocalLibraryStore(db).List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 || items[0].Name != "Mine" {
		t.Fatalf("provisioning should skip existing sources, got %+v", items)
	}
	mode, err := NewPreferencesStore(db).GetSourceViewMode("")
	if err != nil {
		t.Fatalf("GetSourceViewMode: %v", err)
	}
	if mode != SourceViewSubsonic {
		t.Fatalf("source view mode should stay default, got %q", mode)
	}
}
