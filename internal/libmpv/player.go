// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package libmpv

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"unsafe"
)

// Player wraps a libmpv instance for audio-only streaming playback.
type Player struct {
	mu           sync.Mutex
	apiMu        sync.Mutex
	handle       uintptr
	closed       bool
	hasFile      bool
	playing      bool
	ended        bool
	errMsg       string
	lastTimePos  float64
	lastDuration float64
	stopCh       chan struct{}
	wg           sync.WaitGroup
	output       OutputConfig
}

// NewPlayer creates and initializes an mpv instance configured for music playback.
func NewPlayer() (*Player, error) {
	return NewPlayerWithConfig(DefaultOutputConfig())
}

// NewPlayerWithConfig creates an mpv instance with immersive audio output options.
func NewPlayerWithConfig(cfg OutputConfig) (*Player, error) {
	if err := ensureLoaded(); err != nil {
		return nil, err
	}

	prepareCreateEnv()

	handle := fnCreate()
	if handle == 0 {
		return nil, errors.New("libmpv: mpv_create returned NULL (client initialization failed. Check the libmpv version and LC_NUMERIC locale)")
	}

	cfg = NormalizeOutputConfig(cfg)
	p := &Player{
		handle: handle,
		stopCh: make(chan struct{}),
		output: cfg,
	}

	options := []mpvOption{
		{"vo", "null"},
		{"gapless-audio", "weak"},
		{"keep-open", "no"},
		{"audio-display", "no"},
		{"input-default-bindings", "no"},
		{"input-vo-keyboard", "no"},
		{"osc", "no"},
		{"ytdl", "no"},
		{"load-scripts", "no"},
		{"force-window", "no"},
		{"demuxer-max-bytes", "12MiB"},
		{"demuxer-max-back-bytes", "4MiB"},
		{"demuxer-readahead-secs", "8"},
		{"cache", "no"},
	}
	options = append(options, audioOutputOptions(cfg)...)
	for _, opt := range options {
		if opt.name == "af" {
			continue
		}
		if err := mpvError(fnSetOptionString(handle, opt.name, opt.value)); err != nil {
			fnTerminateDestroy(handle)
			return nil, fmt.Errorf("libmpv: setting option %q=%q: %w", opt.name, opt.value, err)
		}
	}

	if err := mpvError(fnInitialize(handle)); err != nil {
		fnTerminateDestroy(handle)
		return nil, fmt.Errorf("libmpv: mpv_initialize: %w", err)
	}

	// Audio filters such as bs2b may be missing from the host ffmpeg build.
	// Apply them after initialize and ignore failures so playback still works.
	for _, opt := range audioOutputOptions(cfg) {
		if opt.name != "af" || opt.value == "" {
			continue
		}
		_ = runCommand(handle, "set", opt.name, opt.value)
	}

	p.wg.Add(1)
	go p.eventLoop()
	return p, nil
}

func (p *Player) eventLoop() {
	defer p.wg.Done()
	defer func() {
		if recover() != nil {
			p.mu.Lock()
			p.errMsg = "playback failed"
			p.ended = true
			p.playing = false
			p.mu.Unlock()
		}
	}()
	for {
		select {
		case <-p.stopCh:
			return
		default:
		}

		ev := fnWaitEvent(p.handle, 0.2)
		if ev == nil {
			continue
		}
		if ev.EventID == eventEnd && ev.Data != nil {
			end := (*eventEndFile)(ev.Data)
			p.mu.Lock()
			switch end.Reason {
			case endFileEOF:
				p.ended = true
				p.playing = false
			case endFileError:
				p.ended = true
				p.playing = false
				if end.Error != 0 && fnErrorString != nil {
					p.errMsg = fnErrorString(end.Error)
				} else {
					p.errMsg = "playback failed"
				}
			default:
				p.playing = false
			}
			p.mu.Unlock()
		}
	}
}

// LoadURL opens a remote or local stream URL.
func (p *Player) LoadURL(url string) error {
	p.apiMu.Lock()
	defer p.apiMu.Unlock()

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return errUnavailable
	}
	p.ended = false
	p.errMsg = ""
	p.playing = false
	p.hasFile = false
	p.lastTimePos = 0
	p.lastDuration = 0
	handle := p.handle
	p.mu.Unlock()

	pause := int32(1)
	_ = mpvError(fnSetProperty(handle, "pause", formatFlag, unsafe.Pointer(&pause))) //#nosec G103 -- required for libmpv property access

	cmd := []string{"loadfile", url, "replace"}
	if err := runCommand(handle, cmd...); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return errUnavailable
	}
	p.hasFile = true
	return nil
}

// Play resumes playback.
func (p *Player) Play() error {
	p.apiMu.Lock()
	defer p.apiMu.Unlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return errUnavailable
	}
	pause := int32(0)
	err := mpvError(fnSetProperty(p.handle, "pause", formatFlag, unsafe.Pointer(&pause))) //#nosec G103 -- required for libmpv property access
	if err != nil {
		return err
	}
	p.playing = true
	p.ended = false
	return nil
}

// Pause stops playback.
func (p *Player) Pause() {
	p.apiMu.Lock()
	defer p.apiMu.Unlock()

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.playing = false
	handle := p.handle
	p.mu.Unlock()

	_ = runCommand(handle, "set", "pause", "yes")
}

// Seek moves playback to the given time in seconds.
func (p *Player) Seek(seconds float64) error {
	p.apiMu.Lock()
	defer p.apiMu.Unlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return errUnavailable
	}
	if seconds < 0 {
		seconds = 0
	}
	cmd := []string{"seek", strconv.FormatFloat(seconds, 'f', 3, 64), "absolute"}
	return runCommand(p.handle, cmd...)
}

// SetVolume sets output volume from 0 to 1.
func (p *Player) SetVolume(volume float64) {
	p.apiMu.Lock()
	defer p.apiMu.Unlock()

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
	v := volume * 100
	_ = mpvError(fnSetProperty(p.handle, "volume", formatDouble, unsafe.Pointer(&v))) //#nosec G103 -- required for libmpv property access
}

// CurrentTime returns playback position in seconds.
func (p *Player) CurrentTime() float64 {
	p.apiMu.Lock()
	defer p.apiMu.Unlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0
	}
	var value float64
	err := mpvError(fnGetProperty(p.handle, "time-pos", formatDouble, unsafe.Pointer(&value))) //#nosec G103 -- required for libmpv property access
	if err != nil {
		return 0
	}
	return value
}

// Duration returns track duration in seconds.
func (p *Player) Duration() float64 {
	p.apiMu.Lock()
	defer p.apiMu.Unlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0
	}
	var value float64
	err := mpvError(fnGetProperty(p.handle, "duration", formatDouble, unsafe.Pointer(&value))) //#nosec G103 -- required for libmpv property access
	if err != nil {
		return 0
	}
	return value
}

// PlaybackState returns position, duration, and playback flags under one lock.
//
// Properties are polled whenever a file is loaded, including while paused,
// so the duration becomes known during the initial (paused) load rather than
// only after playback starts.
func (p *Player) PlaybackState() (currentTime, duration float64, playing, ended bool, errMsg string) {
	p.apiMu.Lock()
	defer p.apiMu.Unlock()

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return 0, 0, false, false, ""
	}
	if !p.hasFile {
		ct, dur := p.lastTimePos, p.lastDuration
		playing, ended, errMsg := p.playing, p.ended, p.errMsg
		p.mu.Unlock()
		return ct, dur, playing, ended, errMsg
	}
	handle := p.handle
	p.mu.Unlock()

	var timePos float64
	timeOK := mpvError(fnGetProperty(handle, "time-pos", formatDouble, unsafe.Pointer(&timePos))) == nil //#nosec G103 -- required for libmpv property access

	var dur float64
	durOK := mpvError(fnGetProperty(handle, "duration", formatDouble, unsafe.Pointer(&dur))) == nil //#nosec G103 -- required for libmpv property access

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return 0, 0, false, false, ""
	}
	if timeOK {
		p.lastTimePos = timePos
	}
	if durOK {
		p.lastDuration = dur
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

// Close shuts down the mpv instance.
func (p *Player) Close() {
	p.apiMu.Lock()
	defer p.apiMu.Unlock()

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	handle := p.handle
	p.mu.Unlock()

	close(p.stopCh)
	p.wg.Wait()
	if handle != 0 {
		fnTerminateDestroy(handle)
	}
}
