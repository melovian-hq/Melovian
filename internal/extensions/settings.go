// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Per-extension user settings live outside the extension folder so a
// reinstall does not wipe them.
func settingsDir(dataDir string) string {
	return filepath.Join(dataDir, "extension-settings")
}

func settingsPath(dataDir, id string) string {
	return filepath.Join(settingsDir(dataDir), id+".json")
}

// LoadSettings returns the stored values for id merged over the manifest
// defaults. Missing file or unreadable data degrades to defaults.
func LoadSettings(dataDir string, manifest Manifest) map[string]any {
	values := map[string]any{}
	for _, field := range manifest.Settings {
		if field.Default != nil {
			values[field.Key] = field.Default
		}
	}
	data, err := os.ReadFile(settingsPath(dataDir, manifest.ID))
	if err != nil {
		return values
	}
	var stored map[string]any
	if json.Unmarshal(data, &stored) != nil {
		return values
	}
	for key, val := range stored {
		if validSettingValue(manifest, key, val) {
			values[key] = val
		}
	}
	return values
}

// validSettingValue reports whether val is declared and correctly typed
// for key in the manifest. Undeclared keys never reach the sandbox.
func validSettingValue(manifest Manifest, key string, val any) bool {
	for _, field := range manifest.Settings {
		if field.Key != key {
			continue
		}
		switch field.Type {
		case "boolean":
			_, ok := val.(bool)
			return ok
		case "choice":
			s, ok := val.(string)
			if !ok {
				return false
			}
			for _, opt := range field.Options {
				if s == opt {
					return true
				}
			}
			return false
		case "text":
			s, ok := val.(string)
			return ok && len(s) <= 256
		default:
			return false
		}
	}
	return false
}

// SaveSettings validates values against the manifest schema and persists
// them. Unknown keys and wrong types are rejected rather than stored.
func SaveSettings(dataDir string, manifest Manifest, values map[string]any) error {
	clean := map[string]any{}
	for key, val := range values {
		if !validSettingValue(manifest, key, val) {
			return fmt.Errorf("invalid value for setting %q", key)
		}
		clean[key] = val
	}
	if err := os.MkdirAll(settingsDir(dataDir), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(clean, "", "  ")
	if err != nil {
		return err
	}
	// Write via temp file and rename so a crash cannot leave a truncated
	// settings file that silently resets values on next read.
	path := settingsPath(dataDir, manifest.ID)
	tmp, err := os.CreateTemp(settingsDir(dataDir), manifest.ID+"-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

// ValidateSettings checks a manifest's settings declarations. Called by
// the install validator so malformed schemas never reach the UI.
func ValidateSettings(manifest Manifest) error {
	seen := map[string]bool{}
	for _, field := range manifest.Settings {
		key := strings.TrimSpace(field.Key)
		if !IsValidExtensionID(strings.ReplaceAll(key, "_", "-")) ||
			strings.ContainsAny(key, " .\t\n") || key == "" {
			return fmt.Errorf("settings key %q must be a simple identifier", field.Key)
		}
		if seen[key] {
			return fmt.Errorf("duplicate settings key %q", key)
		}
		seen[key] = true
		switch field.Type {
		case "boolean":
			if field.Default != nil {
				if _, ok := field.Default.(bool); !ok {
					return fmt.Errorf("settings %q default must be boolean", key)
				}
			}
		case "choice":
			if len(field.Options) == 0 {
				return fmt.Errorf("settings %q of type choice needs options", key)
			}
			seenOptions := make(map[string]bool, len(field.Options))
			for _, opt := range field.Options {
				if opt == "" {
					return fmt.Errorf("settings %q has an empty option", key)
				}
				if seenOptions[opt] {
					return fmt.Errorf("settings %q has duplicate option %q", key, opt)
				}
				seenOptions[opt] = true
			}
			if field.Default != nil {
				def, ok := field.Default.(string)
				if !ok {
					return fmt.Errorf("settings %q default must be a string", key)
				}
				if !slices.Contains(field.Options, def) {
					return fmt.Errorf("settings %q default is not an option", key)
				}
			}
		case "text":
			if field.Default != nil {
				def, ok := field.Default.(string)
				if !ok {
					return fmt.Errorf("settings %q default must be a string", key)
				}
				if len(def) > 256 {
					return fmt.Errorf("settings %q default is too long", key)
				}
			}
		default:
			return fmt.Errorf("settings %q has unknown type %q", key, field.Type)
		}
	}
	return nil
}
