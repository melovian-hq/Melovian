// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func settingsManifest() Manifest {
	return Manifest{
		ID: "test-settings",
		Settings: []SettingField{
			{Key: "enabled-look", Type: "boolean", Default: true},
			{Key: "accent", Type: "choice", Options: []string{"red", "blue"}, Default: "red"},
			{Key: "prefix", Type: "text", Default: "hi"},
			{Key: "nodefault", Type: "text"},
		},
	}
}

func TestLoadSettingsDefaults(t *testing.T) {
	dir := t.TempDir()
	got := LoadSettings(dir, settingsManifest())
	want := map[string]any{"enabled-look": true, "accent": "red", "prefix": "hi"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("key %q = %v, want %v", k, got[k], v)
		}
	}
}

func TestSaveAndLoadSettings(t *testing.T) {
	dir := t.TempDir()
	manifest := settingsManifest()
	if err := SaveSettings(dir, manifest, map[string]any{
		"enabled-look": false,
		"accent":       "blue",
		"prefix":       "yo",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	got := LoadSettings(dir, manifest)
	if got["enabled-look"] != false || got["accent"] != "blue" || got["prefix"] != "yo" {
		t.Fatalf("got %v", got)
	}
	info, err := os.Stat(settingsPath(dir, manifest.ID))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("settings file mode = %o, want 600", perm)
	}
}

func TestSaveSettingsRejectsUnknownKey(t *testing.T) {
	dir := t.TempDir()
	err := SaveSettings(dir, settingsManifest(), map[string]any{"evil": true})
	if err == nil {
		t.Fatal("expected rejection of undeclared key")
	}
}

func TestSaveSettingsRejectsWrongTypes(t *testing.T) {
	dir := t.TempDir()
	manifest := settingsManifest()
	for _, tc := range []map[string]any{
		{"enabled-look": "yes"},
		{"accent": "green"},
		{"accent": true},
		{"prefix": 42},
		{"prefix": strings.Repeat("x", 257)},
	} {
		if err := SaveSettings(dir, manifest, tc); err == nil {
			t.Fatalf("expected rejection of %v", tc)
		}
	}
}

func TestLoadSettingsIgnoresStoredGarbage(t *testing.T) {
	dir := t.TempDir()
	manifest := settingsManifest()
	path := settingsPath(dir, manifest.ID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	// Stored file holds a wrong-typed value and an undeclared key.
	data := `{"enabled-look": "nope", "extra": 1, "accent": "blue"}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	got := LoadSettings(dir, manifest)
	if got["enabled-look"] != true {
		t.Fatalf("wrong-typed stored value must fall back to default, got %v", got["enabled-look"])
	}
	if _, ok := got["extra"]; ok {
		t.Fatal("undeclared stored key must not reach the sandbox")
	}
	if got["accent"] != "blue" {
		t.Fatalf("valid stored value must win, got %v", got["accent"])
	}
}

func TestLoadSettingsCorruptFile(t *testing.T) {
	dir := t.TempDir()
	manifest := settingsManifest()
	path := settingsPath(dir, manifest.ID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := LoadSettings(dir, manifest)
	if got["accent"] != "red" {
		t.Fatalf("corrupt file must degrade to defaults, got %v", got)
	}
}

func TestValidateSettingsSchemas(t *testing.T) {
	cases := []struct {
		name    string
		fields  []SettingField
		wantErr bool
	}{
		{"ok", []SettingField{{Key: "a", Type: "boolean"}}, false},
		{"empty key", []SettingField{{Key: "", Type: "boolean"}}, true},
		{"key with space", []SettingField{{Key: "a b", Type: "boolean"}}, true},
		{"key with dot", []SettingField{{Key: "a.b", Type: "boolean"}}, true},
		{"duplicate", []SettingField{{Key: "a", Type: "boolean"}, {Key: "a", Type: "text"}}, true},
		{"unknown type", []SettingField{{Key: "a", Type: "number"}}, true},
		{"bad bool default", []SettingField{{Key: "a", Type: "boolean", Default: "x"}}, true},
		{"choice no options", []SettingField{{Key: "a", Type: "choice"}}, true},
		{"choice empty option", []SettingField{{Key: "a", Type: "choice", Options: []string{""}}}, true},
		{"choice bad default", []SettingField{{Key: "a", Type: "choice", Options: []string{"x"}, Default: "y"}}, true},
		{"choice ok", []SettingField{{Key: "a", Type: "choice", Options: []string{"x"}, Default: "x"}}, false},
		{"text long default", []SettingField{{Key: "a", Type: "text", Default: strings.Repeat("x", 300)}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSettings(Manifest{ID: "x", Settings: tc.fields})
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
