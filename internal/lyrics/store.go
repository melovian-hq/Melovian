// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lyrics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	RootDir string
}

func NewStore(rootDir string) *Store {
	return &Store{RootDir: rootDir}
}

func (s *Store) ResolveRoot(dataDir, configuredDir string) string {
	configuredDir = strings.TrimSpace(configuredDir)
	if configuredDir != "" {
		return configuredDir
	}
	if strings.TrimSpace(s.RootDir) != "" {
		return s.RootDir
	}
	return filepath.Join(dataDir, "lyrics")
}

func (s *Store) trackPath(root, instanceID, trackID string) string {
	sum := sha256.Sum256([]byte(trackID))
	name := hex.EncodeToString(sum[:]) + ".json"
	return filepath.Join(root, instanceID, name)
}

func (s *Store) Load(root, instanceID, trackID string) (*Document, error) {
	path := s.trackPath(root, instanceID, trackID)
	data, err := os.ReadFile(path) //#nosec G304 -- path derived from hashed track id within lyrics cache dir
	if err != nil {
		return nil, err
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("decode lyrics cache: %w", err)
	}
	return NormalizeDocument(&doc)
}

func (s *Store) Save(root, instanceID, trackID string, doc *Document) error {
	normalized, err := NormalizeDocument(doc)
	if err != nil {
		return err
	}
	dir := filepath.Join(root, instanceID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	path := s.trackPath(root, instanceID, trackID)
	// Unique temp name avoids concurrent Save races on path+".tmp".
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(encoded); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

func (s *Store) Delete(root, instanceID, trackID string) error {
	path := s.trackPath(root, instanceID, trackID)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Store) ClearInstance(root, instanceID string) error {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		// Empty instance id stores files directly under root. Only remove
		// those json files. Never RemoveAll(root), which would wipe every
		// nested instance directory.
		return clearRootLevelLyricsFiles(root)
	}
	dir := filepath.Join(root, instanceID)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(dir)
}

func clearRootLevelLyricsFiles(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			return removeErr
		}
	}
	return nil
}

func (s *Store) Stats(root, instanceID string) (count int, bytes int64, err error) {
	dir := filepath.Join(root, instanceID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, statErr := entry.Info()
		if statErr != nil {
			continue
		}
		count++
		bytes += info.Size()
	}
	return count, bytes, nil
}

type Fetcher struct {
	Store  *Store
	Client *http.Client
}

func (f *Fetcher) FetchFromProviders(
	ctx context.Context,
	providers []Provider,
	in FetchInput,
) (*Document, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("no lyrics providers enabled")
	}
	var best *Document
	var lastErr error
	for _, provider := range providers {
		pctx, cancel := context.WithTimeout(ctx, ProviderRequestTimeout)
		doc, err := provider.Fetch(pctx, f.Client, in)
		cancel()
		if err != nil {
			lastErr = err
			continue
		}
		normalized, normErr := NormalizeDocument(doc)
		if normErr != nil {
			lastErr = normErr
			continue
		}
		if normalized.Synced {
			return normalized, nil
		}
		if best == nil {
			best = normalized
		}
	}
	if best != nil {
		return best, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("lyrics not found")
	}
	return nil, lastErr
}
