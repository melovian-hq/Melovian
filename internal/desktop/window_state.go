// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package desktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const (
	defaultWindowWidth  = 1280
	defaultWindowHeight = 800
	minWindowWidth      = 320
	minWindowHeight     = 240
	maxWindowDimension  = 16_384

	windowStateFileName = "window-state.json"
	saveDebounce        = 400 * time.Millisecond
)

type WindowState struct {
	X         int  `json:"x"`
	Y         int  `json:"y"`
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximised bool `json:"maximised,omitempty"`
}

type WindowStateStore struct {
	dataDir string
	mu      sync.Mutex
	timer   *time.Timer
}

func WindowStatePath(dataDir string) string {
	return filepath.Join(dataDir, windowStateFileName)
}

func ResetWindowRequested() bool {
	if envTruthy(os.Getenv("MELOVIAN_RESET_WINDOW")) {
		return true
	}
	return slices.Contains(os.Args[1:], "--reset-window")
}

func NewWindowStateStore(dataDir string) *WindowStateStore {
	return &WindowStateStore{dataDir: dataDir}
}

func (s *WindowStateStore) Load(reset bool) (WindowState, bool) {
	path := WindowStatePath(s.dataDir)
	if reset {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Warn("failed to reset window state", "path", path, "err", err)
		}
		return defaultWindowState(), false
	}

	data, err := os.ReadFile(path) //#nosec G304 -- path is app data dir window state file
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultWindowState(), false
		}
		slog.Warn("failed to read window state", "path", path, "err", err)
		return defaultWindowState(), false
	}

	var state WindowState
	if err := json.Unmarshal(data, &state); err != nil {
		slog.Warn("failed to parse window state, using defaults", "path", path, "err", err)
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			slog.Warn("failed to remove corrupt window state", "path", path, "err", removeErr)
		}
		return defaultWindowState(), false
	}

	normalized, ok := normalizeWindowState(state)
	if !ok {
		slog.Warn("window state failed validation, using defaults", "path", path)
		return defaultWindowState(), false
	}
	return normalized, true
}

func (s *WindowStateStore) Save(state WindowState) error {
	normalized, ok := normalizeWindowState(state)
	if !ok {
		return fmt.Errorf("invalid window state")
	}

	if err := os.MkdirAll(s.dataDir, 0o700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	encoded, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return fmt.Errorf("encode window state: %w", err)
	}

	path := WindowStatePath(s.dataDir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, encoded, 0o600); err != nil {
		return fmt.Errorf("write window state: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("commit window state: %w", err)
	}
	return nil
}

func (s *WindowStateStore) SaveFromWindow(window application.Window) {
	if window == nil {
		return
	}
	state := captureWindowState(window)
	if screen, err := window.GetScreen(); err == nil && screen != nil {
		state = fitWindowStateToWorkArea(state, screen.WorkArea)
	}
	if err := s.Save(state); err != nil {
		slog.Warn("failed to save window state", "err", err)
	}
}

func (s *WindowStateStore) ScheduleSave(window application.Window) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.timer != nil {
		s.timer.Stop()
	}
	s.timer = time.AfterFunc(saveDebounce, func() {
		s.SaveFromWindow(window)
	})
}

func (s *WindowStateStore) Attach(window application.Window) {
	if window == nil {
		return
	}

	schedule := func() {
		s.ScheduleSave(window)
	}

	window.RegisterHook(events.Common.WindowDidMove, func(*application.WindowEvent) {
		schedule()
	})
	window.RegisterHook(events.Common.WindowDidResize, func(*application.WindowEvent) {
		schedule()
	})
	window.RegisterHook(events.Common.WindowMaximise, func(*application.WindowEvent) {
		schedule()
	})
	window.RegisterHook(events.Common.WindowUnMaximise, func(*application.WindowEvent) {
		schedule()
	})
	window.RegisterHook(events.Common.WindowHide, func(*application.WindowEvent) {
		s.SaveFromWindow(window)
	})
	window.RegisterHook(events.Common.WindowClosing, func(*application.WindowEvent) {
		s.SaveFromWindow(window)
	})

	window.RegisterHook(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		s.ensureOnScreen(window)
	})
}

func (s *WindowStateStore) ensureOnScreen(window application.Window) {
	screen, err := window.GetScreen()
	if err != nil || screen == nil {
		return
	}

	state := captureWindowState(window)
	if windowIntersectsWorkArea(state, screen.WorkArea) {
		return
	}

	fitted := fitWindowStateToWorkArea(state, screen.WorkArea)
	window.SetPosition(fitted.X, fitted.Y)
	if !window.IsMaximised() {
		window.SetSize(fitted.Width, fitted.Height)
	}
	if err := s.Save(fitted); err != nil {
		slog.Warn("failed to save corrected window state", "err", err)
	}
}

func ApplyWindowState(opts *application.WebviewWindowOptions, state WindowState, restored bool) {
	if opts == nil {
		return
	}

	opts.Width = state.Width
	opts.Height = state.Height
	if !restored {
		opts.InitialPosition = application.WindowCentered
		return
	}

	if state.Maximised {
		opts.StartState = application.WindowStateMaximised
		return
	}

	opts.InitialPosition = application.WindowXY
	opts.X = state.X
	opts.Y = state.Y
}

func defaultWindowState() WindowState {
	return WindowState{
		Width:  defaultWindowWidth,
		Height: defaultWindowHeight,
	}
}

func captureWindowState(window application.Window) WindowState {
	width, height := window.Size()
	if width <= 0 {
		width = window.Width()
	}
	if height <= 0 {
		height = window.Height()
	}
	x, y := window.Position()
	return WindowState{
		X:         x,
		Y:         y,
		Width:     width,
		Height:    height,
		Maximised: window.IsMaximised(),
	}
}

func normalizeWindowState(state WindowState) (WindowState, bool) {
	if state.Maximised {
		state.Width = clampDimension(state.Width, defaultWindowWidth)
		state.Height = clampDimension(state.Height, defaultWindowHeight)
		if state.Width < minWindowWidth || state.Height < minWindowHeight {
			return WindowState{}, false
		}
		return state, true
	}

	width := clampDimension(state.Width, defaultWindowWidth)
	height := clampDimension(state.Height, defaultWindowHeight)
	if width < minWindowWidth || height < minWindowHeight {
		return WindowState{}, false
	}

	state.Width = width
	state.Height = height
	return state, true
}

func fitWindowStateToWorkArea(state WindowState, workArea application.Rect) WindowState {
	if state.Maximised {
		return state
	}

	width := clampDimension(state.Width, defaultWindowWidth)
	height := clampDimension(state.Height, defaultWindowHeight)
	if workArea.Width > 0 && width > workArea.Width {
		width = workArea.Width
	}
	if workArea.Height > 0 && height > workArea.Height {
		height = workArea.Height
	}

	x := state.X
	y := state.Y
	if workArea.Width > 0 {
		maxX := workArea.X + workArea.Width - width
		if x < workArea.X {
			x = workArea.X
		}
		if x > maxX {
			x = maxX
		}
	}
	if workArea.Height > 0 {
		maxY := workArea.Y + workArea.Height - height
		if y < workArea.Y {
			y = workArea.Y
		}
		if y > maxY {
			y = maxY
		}
	}

	state.X = x
	state.Y = y
	state.Width = width
	state.Height = height
	return state
}

func windowIntersectsWorkArea(state WindowState, workArea application.Rect) bool {
	if state.Maximised {
		return true
	}
	if workArea.Width <= 0 || workArea.Height <= 0 {
		return true
	}

	left := state.X
	top := state.Y
	right := state.X + state.Width
	bottom := state.Y + state.Height

	areaRight := workArea.X + workArea.Width
	areaBottom := workArea.Y + workArea.Height

	return right > workArea.X && left < areaRight && bottom > workArea.Y && top < areaBottom
}

func clampDimension(value, fallback int) int {
	if value <= 0 || value > maxWindowDimension {
		return fallback
	}
	return value
}

func envTruthy(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
