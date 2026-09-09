// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package services

import (
	"log/slog"
	"sync"
	"time"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

type smtcController struct {
	svc         *MediaService
	mu          sync.Mutex
	state       PlaybackState
	player      *ole.IDispatch
	controls    *ole.IDispatch
	initialized bool
}

func newPlatformMediaController(svc *MediaService) (MediaController, error) {
	ctrl := &smtcController{svc: svc}
	if err := ctrl.init(); err != nil {
		slog.Warn("windows smtc unavailable", "err", err)
		return noopMediaController{}, nil
	}
	return ctrl, nil
}

func (c *smtcController) init() error {
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return err
	}
	unknown, err := oleutil.CreateObject("Windows.Media.Playback.MediaPlayer")
	if err != nil {
		ole.CoUninitialize()
		return err
	}
	player, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		ole.CoUninitialize()
		return err
	}
	controlsRaw, err := oleutil.GetProperty(player, "SystemMediaTransportControls")
	if err != nil {
		player.Release()
		ole.CoUninitialize()
		return err
	}
	controls := controlsRaw.ToIDispatch()
	_, _ = oleutil.PutProperty(controls, "IsEnabled", true)
	_, _ = oleutil.PutProperty(controls, "IsPlayEnabled", true)
	_, _ = oleutil.PutProperty(controls, "IsPauseEnabled", true)
	_, _ = oleutil.PutProperty(controls, "IsNextEnabled", true)
	_, _ = oleutil.PutProperty(controls, "IsPreviousEnabled", true)

	c.player = player
	c.controls = controls
	c.initialized = true
	c.wireEvents()
	return nil
}

func (c *smtcController) wireEvents() {
	// WinRT button events require IAsyncOperation handlers. Transport state is polled from Update.
}

func (c *smtcController) Update(state PlaybackState) error {
	if !c.initialized {
		return nil
	}
	c.mu.Lock()
	c.state = state
	c.mu.Unlock()

	if state.TrackID == "" {
		_, _ = oleutil.PutProperty(c.controls, "PlaybackStatus", 4)
		return nil
	}

	display, err := oleutil.GetProperty(c.controls, "DisplayUpdater")
	if err == nil {
		updater := display.ToIDispatch()
		_, _ = oleutil.PutProperty(updater, "Type", 1)
		_, _ = oleutil.PutProperty(updater, "MusicProperties", map[string]any{
			"Title":  state.Title,
			"Artist": state.Artist,
			"Album":  state.Album,
		})
		_, _ = oleutil.CallMethod(updater, "Update")
		updater.Release()
	}

	status := 2
	if state.Playing {
		status = 4
	}
	_, _ = oleutil.PutProperty(c.controls, "PlaybackStatus", status)

	if state.DurationMs > 0 {
		timeline, err := oleutil.GetProperty(c.controls, "TimelineProperties")
		if err == nil {
			tl := timeline.ToIDispatch()
			_, _ = oleutil.PutProperty(tl, "StartTime", time.Unix(0, 0))
			_, _ = oleutil.PutProperty(tl, "EndTime", time.Unix(0, int64(state.DurationMs)*int64(time.Millisecond)))
			_, _ = oleutil.PutProperty(tl, "Position", time.Unix(0, int64(state.PositionMs)*int64(time.Millisecond)))
			_, _ = oleutil.CallMethod(c.controls, "UpdateTimelineProperties", tl)
			tl.Release()
		}
	}

	_, _ = oleutil.PutProperty(c.controls, "IsPlayEnabled", state.CanPlay)
	_, _ = oleutil.PutProperty(c.controls, "IsPauseEnabled", state.CanPause)
	_, _ = oleutil.PutProperty(c.controls, "IsNextEnabled", state.CanGoNext)
	_, _ = oleutil.PutProperty(c.controls, "IsPreviousEnabled", state.CanGoPrevious)
	return nil
}

func (c *smtcController) Close() error {
	if c.controls != nil {
		c.controls.Release()
	}
	if c.player != nil {
		c.player.Release()
	}
	if c.initialized {
		ole.CoUninitialize()
	}
	return nil
}
