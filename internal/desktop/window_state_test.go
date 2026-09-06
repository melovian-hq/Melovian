// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestResetWindowRequested(t *testing.T) {
	t.Setenv("MELOVIAN_RESET_WINDOW", "")

	if ResetWindowRequested() {
		t.Fatal("expected false with no flag or env")
	}

	t.Setenv("MELOVIAN_RESET_WINDOW", "true")
	if !ResetWindowRequested() {
		t.Fatal("expected true when env is set")
	}
	t.Setenv("MELOVIAN_RESET_WINDOW", "")

	original := os.Args
	defer func() { os.Args = original }()
	os.Args = []string{"melovian", "--reset-window"}
	if !ResetWindowRequested() {
		t.Fatal("expected true when --reset-window is passed")
	}
}

func TestWindowStateStoreLoadMissingFile(t *testing.T) {
	store := NewWindowStateStore(t.TempDir())
	state, restored := store.Load(false)
	if restored {
		t.Fatal("expected restored=false for missing file")
	}
	if state.Width != defaultWindowWidth || state.Height != defaultWindowHeight {
		t.Fatalf("unexpected defaults: %+v", state)
	}
}

func TestWindowStateStoreLoadCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := WindowStatePath(dir)
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	store := NewWindowStateStore(dir)
	state, restored := store.Load(false)
	if restored {
		t.Fatal("expected restored=false for corrupt file")
	}
	if state.Width != defaultWindowWidth || state.Height != defaultWindowHeight {
		t.Fatalf("unexpected defaults: %+v", state)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected corrupt window state file to be removed")
	}
}

func TestWindowStateStoreLoadInvalidDimensions(t *testing.T) {
	dir := t.TempDir()
	path := WindowStatePath(dir)
	payload, err := json.Marshal(WindowState{X: 10, Y: 20, Width: 10, Height: 10})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}

	store := NewWindowStateStore(dir)
	_, restored := store.Load(false)
	if restored {
		t.Fatal("expected restored=false for invalid dimensions")
	}
}

func TestWindowStateStoreSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewWindowStateStore(dir)
	want := WindowState{X: 120, Y: 80, Width: 1024, Height: 768, Maximised: true}

	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}

	got, restored := store.Load(false)
	if !restored {
		t.Fatal("expected restored=true")
	}
	if got != want {
		t.Fatalf("round trip mismatch: got %+v want %+v", got, want)
	}
}

func TestWindowStateStoreLoadReset(t *testing.T) {
	dir := t.TempDir()
	store := NewWindowStateStore(dir)
	if err := store.Save(WindowState{X: 1, Y: 2, Width: 900, Height: 700}); err != nil {
		t.Fatal(err)
	}

	state, restored := store.Load(true)
	if restored {
		t.Fatal("expected restored=false after reset")
	}
	if state.Width != defaultWindowWidth || state.Height != defaultWindowHeight {
		t.Fatalf("unexpected defaults after reset: %+v", state)
	}
	if _, err := os.Stat(WindowStatePath(dir)); !os.IsNotExist(err) {
		t.Fatal("expected window state file to be removed on reset")
	}
}

func TestFitWindowStateToWorkArea(t *testing.T) {
	workArea := application.Rect{X: 100, Y: 50, Width: 1000, Height: 800}
	state := WindowState{X: 2000, Y: 2000, Width: 1200, Height: 900}

	fitted := fitWindowStateToWorkArea(state, workArea)
	if fitted.Width != workArea.Width || fitted.Height != workArea.Height {
		t.Fatalf("expected size clamped to work area, got %+v", fitted)
	}
	if fitted.X != workArea.X || fitted.Y != workArea.Y {
		t.Fatalf("expected position clamped to work area, got %+v", fitted)
	}
}

func TestApplyWindowState(t *testing.T) {
	opts := application.WebviewWindowOptions{}
	ApplyWindowState(&opts, WindowState{X: 40, Y: 60, Width: 960, Height: 640}, true)
	if opts.InitialPosition != application.WindowXY || opts.X != 40 || opts.Y != 60 {
		t.Fatalf("unexpected restored position: %+v", opts)
	}
	if opts.Width != 960 || opts.Height != 640 {
		t.Fatalf("unexpected restored size: %+v", opts)
	}

	opts = application.WebviewWindowOptions{}
	ApplyWindowState(&opts, defaultWindowState(), false)
	if opts.InitialPosition != application.WindowCentered {
		t.Fatalf("expected centered defaults, got %+v", opts)
	}

	opts = application.WebviewWindowOptions{}
	ApplyWindowState(&opts, WindowState{Width: 800, Height: 600, Maximised: true}, true)
	if opts.StartState != application.WindowStateMaximised {
		t.Fatalf("expected maximised start state, got %+v", opts)
	}
}

func TestWindowStatePathUsesDataDir(t *testing.T) {
	dir := filepath.Join("tmp", "melovian")
	if got := WindowStatePath(dir); got != filepath.Join(dir, windowStateFileName) {
		t.Fatalf("unexpected path %q", got)
	}
}
