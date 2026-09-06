// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build linux && !android

package services

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
	"sync"
	"time"

	"github.com/Nadim147c/go-mpris/v2"
	"github.com/godbus/dbus/v5"

	"melovian/internal/brand"
)

type mprisController struct {
	svc      *MediaService
	conn     *dbus.Conn
	server   *mpris.Server
	pm       *mpris.PropertiesManager
	mu       sync.Mutex
	state    PlaybackState
	identity string
}

func newPlatformMediaController(svc *MediaService) (MediaController, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}

	name := mpris.BaseInterface + "." + brand.Slug
	server := mpris.NewServer(conn, name)
	pm := mpris.NewPropertiesManager(&mpris.Properties{
		CanQuit:          true,
		CanRaise:         true,
		CanSetFullscreen: false,
		HasTrackList:     false,
		Identity:         brand.Name,
		DesktopEntry:     brand.Slug,
		CanPlay:          true,
		CanPause:         true,
		CanGoNext:        true,
		CanGoPrevious:    true,
		CanSeek:          true,
		CanControl:       true,
		MinimumRate:      1.0,
		MaximumRate:      1.0,
		Rate:             1.0,
		Volume:           1.0,
		PlaybackStatus:   mpris.PlaybackStopped,
	})

	ctrl := &mprisController{
		svc:      svc,
		conn:     conn,
		server:   server,
		pm:       pm,
		identity: brand.Name,
	}

	if err := server.RegisterPropertiesManager(pm, mpris.NoOpPropertySetter{}); err != nil {
		return nil, err
	}
	if err := server.RegisterBaseHandler(ctrl); err != nil {
		return nil, err
	}
	if err := server.RegisterPlayerHandler(ctrl); err != nil {
		return nil, err
	}

	go func() {
		if err := server.Listen(context.Background()); err != nil {
			slog.Warn("mpris server stopped", "err", err)
		}
	}()

	return ctrl, nil
}

func mprisTrackID(trackID string) dbus.ObjectPath {
	if trackID == "" {
		return dbus.ObjectPath("/org/mpris/MediaPlayer2/NoTrack")
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(trackID))
	return dbus.ObjectPath(fmt.Sprintf("/org/mpris/MediaPlayer2/track/%x", h.Sum32()))
}

func (c *mprisController) applyState(state PlaybackState) {
	var changed map[string]dbus.Variant
	c.pm.SetProperty(func(p *mpris.Properties) {
		p.Identity = c.identity
		p.DesktopEntry = "melovian"
		p.CanPlay = state.CanPlay
		p.CanPause = state.CanPause
		p.CanGoNext = state.CanGoNext
		p.CanGoPrevious = state.CanGoPrevious
		p.CanSeek = state.TrackID != ""
		p.CanControl = state.TrackID != ""
		p.Position = time.Duration(state.PositionMs) * time.Millisecond
		p.MinimumRate = 1.0
		p.MaximumRate = 1.0
		p.Rate = 1.0
		p.Volume = 1.0

		meta := mpris.Metadata{}
		if state.TrackID != "" {
			meta.Set(mpris.KeyTrackID, mprisTrackID(state.TrackID))
			meta.Set(mpris.KeyTitle, state.Title)
			meta.Set(mpris.KeyArtist, []string{state.Artist})
			meta.Set(mpris.KeyAlbum, state.Album)
			if state.DurationMs > 0 {
				meta.Set(mpris.KeyLength, time.Duration(state.DurationMs)*time.Millisecond)
			}
			if state.CoverArtURL != "" {
				meta.Set(mpris.KeyArtURL, state.CoverArtURL)
			}
			p.Metadata = meta
		} else {
			p.Metadata = mpris.Metadata{}
		}

		if state.TrackID == "" {
			p.PlaybackStatus = mpris.PlaybackStopped
		} else if state.Playing {
			p.PlaybackStatus = mpris.PlaybackPlaying
		} else {
			p.PlaybackStatus = mpris.PlaybackPaused
		}

		emitMeta := p.Metadata
		if emitMeta == nil {
			emitMeta = mpris.Metadata{}
		}
		changed = map[string]dbus.Variant{
			"PlaybackStatus": dbus.MakeVariant(string(p.PlaybackStatus)),
			"CanPlay":        dbus.MakeVariant(p.CanPlay),
			"CanPause":       dbus.MakeVariant(p.CanPause),
			"CanGoNext":      dbus.MakeVariant(p.CanGoNext),
			"CanGoPrevious":  dbus.MakeVariant(p.CanGoPrevious),
			"CanSeek":        dbus.MakeVariant(p.CanSeek),
			"CanControl":     dbus.MakeVariant(p.CanControl),
			"Metadata":       dbus.MakeVariant(emitMeta),
		}
	})
	if len(changed) == 0 {
		return
	}
	if err := c.conn.Emit(
		mpris.DBusObjectPath,
		mpris.PropertiesChangedSignal,
		mpris.PlayerInterface,
		changed,
		[]string{},
	); err != nil {
		slog.Warn("mpris properties changed emit failed", "err", err)
	}
}

func (c *mprisController) patchPlaying(playing bool) {
	c.mu.Lock()
	if c.state.TrackID == "" {
		c.mu.Unlock()
		return
	}
	c.state.Playing = playing
	state := c.state
	c.mu.Unlock()
	c.applyState(state)
}

func (c *mprisController) Update(state PlaybackState) error {
	defer func() {
		if r := recover(); r != nil {
			slog.Warn("mpris update recovered from panic", "panic", r)
		}
	}()

	c.mu.Lock()
	c.state = state
	c.mu.Unlock()
	c.applyState(state)
	return nil
}

func (c *mprisController) Close() error {
	if c.server != nil {
		if err := c.server.Close(); err != nil {
			slog.Warn("mpris server close failed", "err", err)
		}
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *mprisController) Play() error {
	c.patchPlaying(true)
	c.svc.onPlay()
	return nil
}

func (c *mprisController) Pause() error {
	c.patchPlaying(false)
	c.svc.onPause()
	return nil
}

func (c *mprisController) PlayPause() error {
	c.mu.Lock()
	playing := c.state.Playing
	c.mu.Unlock()
	if playing {
		return c.Pause()
	}
	return c.Play()
}

func (c *mprisController) Stop() error {
	return c.Pause()
}

func (c *mprisController) Next() error {
	c.svc.onNext()
	return nil
}

func (c *mprisController) Previous() error {
	c.svc.onPrevious()
	return nil
}

func (c *mprisController) Seek(offset time.Duration) error {
	c.mu.Lock()
	position := c.state.PositionMs + offset.Milliseconds()
	c.mu.Unlock()
	if position < 0 {
		position = 0
	}
	c.svc.queueSeek(position)
	return nil
}

func (c *mprisController) SetPosition(_ dbus.ObjectPath, position time.Duration) error {
	c.svc.queueSeek(position.Milliseconds())
	return nil
}

func (c *mprisController) OpenURI(uri string) error {
	c.svc.queueOpenURI(uri)
	return nil
}
func (c *mprisController) Raise() error { c.svc.RaiseWindow(); return nil }
func (c *mprisController) Quit() error  { c.svc.QuitApp(); return nil }
