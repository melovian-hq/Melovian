// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"

	"melovian/internal/compat"
	"melovian/internal/update"
)

// autoUpdateInterval is how often the opt-in background checker polls the
// update feed.
const autoUpdateInterval = 6 * time.Hour

// UpdateService wraps the Wails in-app updater with Melovian's Atom-feed
// provider. It is registered as a Wails service and also surfaced through
// the HTTP API via desktop hooks, so the settings UI works identically in
// desktop and server mode.
//
// Auto-update is strictly opt-in: the periodic background check only runs
// when the user has enabled it in settings.
type UpdateService struct {
	app     *application.App
	mu      sync.Mutex
	inited  bool
	initErr string
	rel     *updater.Release

	// bgCancel and bgWG manage the detached feed check started by
	// SetAutoUpdate. checkRunning dedups it so repeated toggles never
	// stack concurrent checks. stopped is set by Shutdown.
	bgCancel     context.CancelFunc
	bgWG         sync.WaitGroup
	checkRunning bool
	stopped      bool

	// LoadAutoUpdate returns (enabled, channel). Set from main.go.
	LoadAutoUpdate func() (bool, string)
	// SaveAutoUpdate persists the opt-in flag. Set from main.go.
	SaveAutoUpdate func(bool) error
}

func NewUpdateService() *UpdateService {
	return &UpdateService{}
}

func (s *UpdateService) SetApp(app *application.App) {
	s.mu.Lock()
	s.app = app
	s.mu.Unlock()
}

func (s *UpdateService) updater() (*updater.Updater, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.app == nil {
		return nil, errors.New("application not ready")
	}
	if s.inited {
		return s.app.Updater, s.initErrOrNil()
	}
	auto, channelName := s.loadSettings()
	channel := update.ChannelStable
	if channelName == string(update.ChannelPrerelease) {
		channel = update.ChannelPrerelease
	}
	cfg := updater.Config{
		CurrentVersion: compat.Version,
		Providers:      []updater.Provider{&update.AtomProvider{Channel: channel}},
	}
	if key := update.PinnedKeyBytes(); len(key) > 0 {
		cfg.PublicKey = key
	}
	// Opt-in auto-updates: only poll when the user enabled them.
	if auto {
		cfg.CheckInterval = autoUpdateInterval
	}
	err := s.app.Updater.Init(cfg)
	s.inited = true
	if err != nil {
		s.initErr = err.Error()
		return nil, err
	}
	return s.app.Updater, nil
}

func (s *UpdateService) initErrOrNil() error {
	if s.initErr == "" {
		return nil
	}
	return errors.New(s.initErr)
}

func (s *UpdateService) loadSettings() (bool, string) {
	if s.LoadAutoUpdate == nil {
		return false, string(update.ChannelStable)
	}
	return s.LoadAutoUpdate()
}

// CheckForUpdates runs a feed check and returns the resulting status.
// Manual-only releases (installers, AppImage, containers) report
// StatusManual with a releaseUrl the UI can open.
func (s *UpdateService) CheckForUpdates(ctx context.Context) map[string]any {
	u, err := s.updater()
	if err != nil {
		return s.statusPayload(err)
	}
	rel, err := u.Check(ctx)
	s.mu.Lock()
	s.rel = rel
	s.mu.Unlock()
	if err != nil {
		if manual, ok := errors.AsType[*update.ErrManualOnly](err); ok {
			return s.statusPayloadManual(manual.ReleaseURL)
		}
		return s.statusPayload(err)
	}
	return s.statusPayload(nil)
}

// ApplyUpdate downloads, verifies, and stages the pending release. It runs
// Check first if none is pending. Progress is observable through the
// wails:updater:* events and GetStatus.
func (s *UpdateService) ApplyUpdate(ctx context.Context) map[string]any {
	u, err := s.updater()
	if err != nil {
		return s.statusPayload(err)
	}
	s.mu.Lock()
	rel := s.rel
	s.mu.Unlock()
	if rel == nil {
		if _, err := u.Check(ctx); err != nil {
			if manual, ok := errors.AsType[*update.ErrManualOnly](err); ok {
				return s.statusPayloadManual(manual.ReleaseURL)
			}
			return s.statusPayload(err)
		}
	}
	if err := u.DownloadAndInstall(ctx); err != nil {
		return s.statusPayload(err)
	}
	return s.statusPayload(nil)
}

// RestartToApply relaunches into the staged update. The process exits.
func (s *UpdateService) RestartToApply(ctx context.Context) error {
	u, err := s.updater()
	if err != nil {
		return err
	}
	return u.Restart(ctx)
}

// GetStatus returns the current updater state for polling UIs.
func (s *UpdateService) GetStatus() map[string]any {
	return s.statusPayload(nil)
}

// GetAutoUpdate reports whether background auto-updates are enabled.
func (s *UpdateService) GetAutoUpdate() bool {
	auto, _ := s.loadSettings()
	return auto
}

// SetAutoUpdate flips the opt-in flag and starts or stops the background
// checker for the running session.
func (s *UpdateService) SetAutoUpdate(enabled bool) error {
	if s.SaveAutoUpdate != nil {
		if err := s.SaveAutoUpdate(enabled); err != nil {
			return err
		}
	}
	u, err := s.updater()
	if err != nil {
		return nil
	}
	if enabled {
		// CheckInterval is fixed at Init. Emulate a fresh periodic loop by
		// doing a check now. Subsequent ticks need a restart of the app or
		// a fresh Init, so persist the flag and run one immediate check.
		s.startCheck(func(ctx context.Context) { _, _ = u.Check(ctx) })
	} else {
		s.stopCheck()
		u.StopPeriodicCheck()
	}
	return nil
}

// startCheck runs check on a detached, cancellable context. A check that
// is already running is left alone so repeated SetAutoUpdate calls never
// stack concurrent feed fetches.
func (s *UpdateService) startCheck(check func(context.Context)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped || s.checkRunning {
		return
	}
	if s.bgCancel != nil {
		s.bgCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.bgCancel = cancel
	s.checkRunning = true
	s.bgWG.Go(func() {
		check(ctx)
		s.mu.Lock()
		s.checkRunning = false
		s.mu.Unlock()
	})
}

// stopCheck cancels the in-flight detached check, if any.
func (s *UpdateService) stopCheck() {
	s.mu.Lock()
	if s.bgCancel != nil {
		s.bgCancel()
		s.bgCancel = nil
	}
	s.mu.Unlock()
}

// Shutdown cancels background update work and waits for it to drain.
func (s *UpdateService) Shutdown() {
	s.mu.Lock()
	s.stopped = true
	if s.bgCancel != nil {
		s.bgCancel()
		s.bgCancel = nil
	}
	s.mu.Unlock()
	s.bgWG.Wait()
}

func (s *UpdateService) statusPayload(err error) map[string]any {
	p := map[string]any{
		"currentVersion": compat.Version,
		"desktop":        true,
		"signed":         update.Pinned(),
	}
	if s.app != nil && s.inited {
		p["state"] = string(s.app.Updater.State())
	}
	s.mu.Lock()
	if s.rel != nil {
		p["latestVersion"] = s.rel.Version
		p["notes"] = s.rel.Notes
		if u, ok := s.rel.Metadata["releaseUrl"]; ok {
			p["releaseUrl"] = u
		}
	}
	initErr := s.initErr
	s.mu.Unlock()
	if err != nil {
		p["error"] = err.Error()
	} else if initErr != "" {
		p["error"] = initErr
	}
	return p
}

func (s *UpdateService) statusPayloadManual(releaseURL string) map[string]any {
	p := s.statusPayload(nil)
	p["manual"] = true
	p["releaseUrl"] = releaseURL
	return p
}
