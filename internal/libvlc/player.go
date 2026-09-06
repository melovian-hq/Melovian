// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package libvlc

import (
	"fmt"
	"math"
	"strings"
	"sync"
)

// Player wraps a libvlc media player for audio-only streaming playback.
type Player struct {
	mu           sync.Mutex
	instance     uintptr
	player       uintptr
	closed       bool
	playing      bool
	ended        bool
	errMsg       string
	lastTimePos  float64
	lastDuration float64
	argStore     [][]byte
}

// OutputConfig mirrors libmpv immersive modes for VLC where support exists.
type OutputConfig struct {
	Mode      string
	Exclusive bool
}

// NewPlayer creates and initializes a libvlc instance configured for music playback.
func NewPlayer() (*Player, error) {
	return NewPlayerWithConfig(OutputConfig{Mode: "auto"})
}

// NewPlayerWithConfig creates a libvlc player. Passthrough enables S/PDIF when available.
func NewPlayerWithConfig(cfg OutputConfig) (*Player, error) {
	if err := ensureLoaded(); err != nil {
		return nil, err
	}
	prepareVLCEnvironment()

	options := []string{
		"--no-video",
		"--no-osd",
		"--no-stats",
		"--intf=dummy",
		"--quiet",
		"--network-caching=3000",
		"--file-caching=1000",
	}
	if strings.EqualFold(cfg.Mode, "passthrough") {
		options = append(options, "--spdif")
	}
	argv, argStore := vlcArgs(options)
	argc := len(options)
	if argc > math.MaxInt32 {
		return nil, fmt.Errorf("too many libvlc options: %d", argc)
	}
	instance := fnNew(int32(argc), argv) //#nosec G115 -- argc bounded above
	if instance == 0 {
		return nil, errUnavailable
	}

	player := fnPlayerNew(instance)
	if player == 0 {
		fnRelease(instance)
		return nil, errUnavailable
	}

	return &Player{
		instance: instance,
		player:   player,
		argStore: argStore,
	}, nil
}

// LoadURL opens a remote or local stream URL.
func (p *Player) LoadURL(url string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return errUnavailable
	}
	p.ended = false
	p.errMsg = ""
	p.playing = false

	fnPlayerPause(p.player, 1)

	media := fnMediaNewLocation(p.instance, url)
	if media == 0 {
		return errUnavailable
	}
	fnPlayerSetMedia(p.player, media)
	fnMediaRelease(media)
	return nil
}

// Play resumes playback.
func (p *Player) Play() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return errUnavailable
	}
	if fnPlayerPlay(p.player) != 0 {
		return errUnavailable
	}
	p.playing = true
	p.ended = false
	p.errMsg = ""
	return nil
}

// Pause stops playback.
func (p *Player) Pause() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	player := p.player
	p.playing = false
	p.mu.Unlock()

	fnPlayerPause(player, 1)
}

// Seek moves playback to the given time in seconds.
func (p *Player) Seek(seconds float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return errUnavailable
	}
	if seconds < 0 {
		seconds = 0
	}
	ms := int64(seconds * 1000)
	if fnPlayerSetTime(p.player, ms) != 0 {
		return errUnavailable
	}
	return nil
}

// SetVolume sets output volume from 0 to 1.
func (p *Player) SetVolume(volume float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	if volume < 0 {
		volume = 0
	}
	if volume > 1 {
		volume = 1
	}
	v := int32(volume * 100)
	_ = fnAudioSetVolume(p.player, v)
}

// CurrentTime returns playback position in seconds.
func (p *Player) CurrentTime() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0
	}
	ms := fnPlayerGetTime(p.player)
	if ms < 0 {
		return 0
	}
	return float64(ms) / 1000
}

// Duration returns track duration in seconds.
func (p *Player) Duration() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0
	}
	ms := fnPlayerGetLength(p.player)
	if ms < 0 {
		return 0
	}
	return float64(ms) / 1000
}

// PlaybackState returns position, duration, and playback flags under one lock.
func (p *Player) PlaybackState() (currentTime, duration float64, playing, ended bool, errMsg string) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return 0, 0, false, false, ""
	}
	if !p.playing {
		ct, dur := p.lastTimePos, p.lastDuration
		ended, errMsg := p.ended, p.errMsg
		p.mu.Unlock()
		return ct, dur, false, ended, errMsg
	}
	player := p.player
	p.mu.Unlock()

	state := fnPlayerGetState(player)
	ms := fnPlayerGetTime(player)
	length := fnPlayerGetLength(player)

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0, 0, false, false, ""
	}
	if !p.playing {
		return p.lastTimePos, p.lastDuration, false, p.ended, p.errMsg
	}
	switch state {
	case statePlaying:
		p.playing = true
		p.ended = false
		p.errMsg = ""
	case statePaused:
		p.playing = false
	case stateEnded:
		p.playing = false
		p.ended = true
	case stateError:
		p.playing = false
		p.ended = true
		if p.errMsg == "" {
			p.errMsg = "playback failed"
		}
	}
	if ms >= 0 {
		p.lastTimePos = float64(ms) / 1000
	}
	if length >= 0 {
		p.lastDuration = float64(length) / 1000
	}
	return p.lastTimePos, p.lastDuration, p.playing, p.ended, p.errMsg
}

// Snapshot returns the current playback state.
func (p *Player) Snapshot() (playing bool, ended bool, errMsg string) {
	_, _, playing, ended, errMsg = p.PlaybackState()
	return playing, ended, errMsg
}

// ClearEnded resets the ended latch after the consumer handles it.
func (p *Player) ClearEnded() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ended = false
	p.errMsg = ""
}

// Close shuts down the libvlc instance.
func (p *Player) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	player := p.player
	instance := p.instance
	p.mu.Unlock()

	if player != 0 {
		fnPlayerRelease(player)
	}
	if instance != 0 {
		fnRelease(instance)
	}
}
