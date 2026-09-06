// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"melovian/internal/brand"
)

type PlaybackState struct {
	Title         string `json:"title"`
	Artist        string `json:"artist"`
	Album         string `json:"album"`
	TrackID       string `json:"trackId"`
	CoverArtID    string `json:"coverArtId"`
	CoverArtURL   string `json:"coverArtUrl"`
	DurationMs    int64  `json:"durationMs"`
	PositionMs    int64  `json:"positionMs"`
	Playing       bool   `json:"playing"`
	CanPlay       bool   `json:"canPlay"`
	CanPause      bool   `json:"canPause"`
	CanGoNext     bool   `json:"canGoNext"`
	CanGoPrevious bool   `json:"canGoPrevious"`
}

const MainWindowName = "main"

var MainWindowCloseRequestedEvent = brand.Slug + ":window-close-requested"
var AppQuitRequestedEvent = brand.Slug + ":app-quit-requested"

type PendingMediaAction struct {
	Action  string `json:"action"`
	SeekMs  int64  `json:"seekMs"`
	OpenURI string `json:"openUri"`
}

type MediaService struct {
	app     *application.App
	dataDir string
	mu      sync.Mutex
	state   PlaybackState
	ctrl    MediaController

	actionMu      sync.Mutex
	pendingAction PendingMediaAction
}

func NewMediaService(app *application.App) *MediaService {
	return &MediaService{
		app:  app,
		ctrl: noopMediaController{},
	}
}

func (s *MediaService) SetApp(app *application.App) {
	s.app = app
	if _, noop := s.ctrl.(noopMediaController); noop {
		s.ctrl = newMediaController(s)
	}
}

func (s *MediaService) UpdatePlayback(state PlaybackState) error {
	s.mu.Lock()
	if s.state == state {
		s.mu.Unlock()
		return nil
	}
	s.state = state
	s.mu.Unlock()
	return s.ctrl.Update(state)
}

func (s *MediaService) PollMediaAction() PendingMediaAction {
	s.actionMu.Lock()
	defer s.actionMu.Unlock()
	out := s.pendingAction
	s.pendingAction = PendingMediaAction{}
	return out
}

func (s *MediaService) queueAction(action string) {
	s.actionMu.Lock()
	s.pendingAction = PendingMediaAction{Action: action}
	s.actionMu.Unlock()
}

func (s *MediaService) queueSeek(positionMs int64) {
	s.actionMu.Lock()
	s.pendingAction = PendingMediaAction{Action: "seek", SeekMs: positionMs}
	s.actionMu.Unlock()
}

func (s *MediaService) queueOpenURI(uri string) {
	s.actionMu.Lock()
	s.pendingAction = PendingMediaAction{Action: "openUri", OpenURI: uri}
	s.actionMu.Unlock()
}

func (s *MediaService) onPlay()     { s.queueAction("play") }
func (s *MediaService) onPause()    { s.queueAction("pause") }
func (s *MediaService) onNext()     { s.queueAction("next") }
func (s *MediaService) onPrevious() { s.queueAction("previous") }

func (s *MediaService) TogglePlayback() {
	s.mu.Lock()
	playing := s.state.Playing
	s.mu.Unlock()
	if playing {
		s.onPause()
	} else {
		s.onPlay()
	}
}

func (s *MediaService) GetPlaybackState() PlaybackState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *MediaService) Play() {
	s.onPlay()
}

func (s *MediaService) Pause() {
	s.onPause()
}

func (s *MediaService) Next() {
	s.onNext()
}

func (s *MediaService) Previous() {
	s.onPrevious()
}

func (s *MediaService) SeekTo(positionMs int64) {
	s.queueSeek(positionMs)
}

func (s *MediaService) mainWindow() application.Window {
	if s.app == nil {
		return nil
	}
	window, ok := s.app.Window.GetByName(MainWindowName)
	if !ok {
		window = s.app.Window.Current()
	}
	return window
}

func (s *MediaService) RaiseWindow() {
	if s.app == nil {
		return
	}
	application.InvokeSync(func() {
		if window := s.mainWindow(); window != nil {
			window.Show()
			window.UnMinimise()
			window.Restore()
			window.Focus()
		}
	})
}

func (s *MediaService) HideMainWindow() {
	if s.app == nil {
		return
	}
	application.InvokeSync(func() {
		if window := s.mainWindow(); window != nil {
			window.Hide()
		}
	})
}

func (s *MediaService) MinimizeMainWindow() {
	if s.app == nil {
		return
	}
	application.InvokeSync(func() {
		if window := s.mainWindow(); window != nil {
			window.Minimise()
		}
	})
}

func (s *MediaService) ToggleMaximizeMainWindow() {
	if s.app == nil {
		return
	}
	application.InvokeSync(func() {
		if window := s.mainWindow(); window != nil {
			window.ToggleMaximise()
		}
	})
}

func (s *MediaService) HandleTitleBarDoubleClick() {
	if s.app == nil {
		return
	}
	application.InvokeSync(func() {
		window := s.mainWindow()
		if window == nil {
			return
		}
		if window.IsMaximised() {
			window.UnMaximise()
			return
		}
		window.Maximise()
	})
}

func (s *MediaService) RequestMainWindowClose() {
	if s.app == nil {
		return
	}
	application.InvokeSync(func() {
		if window := s.mainWindow(); window != nil {
			window.EmitEvent(MainWindowCloseRequestedEvent)
		}
	})
}

func (s *MediaService) SetNativeTitleBarEnabled(enabled bool) {
	if s.app == nil {
		return
	}
	application.InvokeSync(func() {
		applyNativeTitleBar(s.mainWindow(), enabled)
	})
}

func (s *MediaService) SetTaskbarIntegrationEnabled(enabled bool) {
	setTaskbarIntegrationEnabled(enabled)
}

func (s *MediaService) QuitApp() {
	if s.app == nil {
		return
	}
	application.InvokeSync(func() {
		s.app.Quit()
	})
}

func (s *MediaService) Shutdown() {
	state := PlaybackState{}
	s.mu.Lock()
	s.state = state
	s.mu.Unlock()
	_ = s.ctrl.Update(state)
	if err := s.ctrl.Close(); err != nil {
		slog.Warn("media controller shutdown failed", "err", err)
	}
}

type MediaController interface {
	Update(state PlaybackState) error
	Close() error
}

func newMediaController(svc *MediaService) MediaController {
	ctrl, err := newPlatformMediaController(svc)
	if err != nil {
		slog.Warn("media controller unavailable", "err", err)
		return noopMediaController{}
	}
	return ctrl
}

type noopMediaController struct{}

func (noopMediaController) Update(_ PlaybackState) error { return nil }
func (noopMediaController) Close() error                 { return nil }

func ShutdownMediaService(svc *MediaService) {
	if svc == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = ctx
	svc.Shutdown()
}
