// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Command export writes democatalog fixtures for the static frontend demo.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"melovian/internal/democatalog"
)

func main() {
	out := flag.String("out", "frontend/public/demo", "output directory")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o750); err != nil {
		fatal(err)
	}

	c := democatalog.Get()
	payload := map[string]any{
		"artists":   c.Artists,
		"albums":    c.Albums,
		"songs":     c.Songs,
		"playlists": c.Playlists,
		"genres":    c.Genres,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatal(err)
	}
	catalogPath := filepath.Join(*out, "catalog.json")
	if err := os.WriteFile(catalogPath, data, 0o600); err != nil {
		fatal(err)
	}

	wavPath := filepath.Join(*out, "silent.wav")
	if err := os.WriteFile(wavPath, democatalog.SilentWAV(), 0o600); err != nil {
		fatal(err)
	}

	fmt.Printf("Wrote %s and %s\n", catalogPath, wavPath)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "democatalog export: %v\n", err)
	os.Exit(1)
}
