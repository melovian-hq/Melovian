// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RegistryOverride is a user-configured registry saved under the data dir.
// Environment variables still win so an admin can force a registry that
// the UI cannot change.
type RegistryOverride struct {
	URL  string   `json:"url"`
	Keys []string `json:"keys"`
}

func registryOverridePath(dataDir string) string {
	return filepath.Join(dataDir, "extension-registry.json")
}

// LoadRegistryOverride reads the saved custom registry, if any.
func LoadRegistryOverride(dataDir string) (RegistryOverride, bool) {
	data, err := os.ReadFile(registryOverridePath(dataDir))
	if err != nil {
		return RegistryOverride{}, false
	}
	var o RegistryOverride
	if json.Unmarshal(data, &o) != nil {
		return RegistryOverride{}, false
	}
	o.URL = strings.TrimSpace(o.URL)
	var keys []string
	for _, k := range o.Keys {
		if k = strings.TrimSpace(k); k != "" {
			keys = append(keys, k)
		}
	}
	o.Keys = keys
	if o.URL == "" {
		return RegistryOverride{}, false
	}
	return o, true
}

// SaveRegistryOverride validates and stores a custom registry. The URL
// must satisfy remoteURLOK and every key must be a full Ed25519 public
// key in hex. Keys are mandatory: an unsigned custom registry would be a
// downgrade path around the official signature check.
func SaveRegistryOverride(dataDir, rawURL string, keys []string) error {
	u := strings.TrimSpace(rawURL)
	if !remoteURLOK(u) {
		return fmt.Errorf("registry URL must be https (http is allowed for loopback only)")
	}
	clean := make([]string, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if _, err := decodeRegistryKey(k); err != nil {
			return fmt.Errorf("invalid registry public key")
		}
		clean = append(clean, k)
	}
	if len(clean) == 0 {
		return fmt.Errorf("a custom registry needs at least one public key")
	}
	data, err := json.MarshalIndent(RegistryOverride{URL: u, Keys: clean}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(registryOverridePath(dataDir), data, 0o600)
}

// ClearRegistryOverride removes the saved custom registry so the app
// returns to the official index.
func ClearRegistryOverride(dataDir string) error {
	err := os.Remove(registryOverridePath(dataDir))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
