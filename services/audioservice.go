// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"errors"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"melovian/internal/brand"
	"melovian/internal/libmpv"
	"melovian/internal/libvlc"
	"melovian/internal/pcmsink"
)

var errNativeUnavailable = errors.New("native audio unavailable")

const (
	BackendAuto = "auto"
	BackendMPV  = "mpv"
	BackendVLC  = "vlc"
)

// AudioCapabilities describes native playback support on this host.
type AudioCapabilities struct {
	NativeAvailable bool                 `json:"nativeAvailable"`
	Backend         string               `json:"backend"`
	MpvAvailable    bool                 `json:"mpvAvailable"`
	VlcAvailable    bool                 `json:"vlcAvailable"`
	MpvLoadError    string               `json:"mpvLoadError,omitempty"`
	InitError       string               `json:"initError,omitempty"`
	PcmActive       bool                 `json:"pcmActive"`
	PcmError        string               `json:"pcmError,omitempty"`
	Sinks           []pcmsink.SinkStatus `json:"sinks,omitempty"`
}

// AudioPlaybackState is polled by the frontend native engine.
type AudioPlaybackState struct {
	CurrentTime float64 `json:"currentTime"`
	Duration    float64 `json:"duration"`
	Playing     bool    `json:"playing"`
	Ended       bool    `json:"ended"`
	Error       string  `json:"error"`
}

type nativePlayer interface {
	LoadURL(url string) error
	Play() error
	Pause()
	Seek(seconds float64) error
	SetVolume(volume float64)
	PlaybackState() (currentTime, duration float64, playing, ended bool, errMsg string)
	ClearEnded()
	Close()
}

// AudioOutputConfig controls immersive / surround output for native backends.
type AudioOutputConfig struct {
	Mode      string `json:"mode"`
	Exclusive bool   `json:"exclusive"`
	// Targets lists extra PCM outputs. Supported specs: "device", "stdout",
	// "fifo:<path>", "tcp:<host:port>", "tcp-listen:<addr>",
	// "unix:<path>", "unix-listen:<path>". When non-empty, mpv decodes to a
	// raw 48 kHz s16le stereo pipe and the audio fans out to every target.
	// Pipe targets require the mpv backend.
	Targets []string `json:"targets,omitempty"`
}

type audioBackend interface {
	LoadURL(url string) error
	Play() error
	Pause()
	Seek(seconds float64) error
	SetVolume(volume float64)
	GetState() AudioPlaybackState
	ClearEnded()
	Close()
}

type noopAudioBackend struct{}

func (noopAudioBackend) LoadURL(string) error { return errNativeUnavailable }
func (noopAudioBackend) Play() error          { return errNativeUnavailable }
func (noopAudioBackend) Pause()               {}
func (noopAudioBackend) Seek(float64) error   { return errNativeUnavailable }
func (noopAudioBackend) SetVolume(float64)    {}
func (noopAudioBackend) GetState() AudioPlaybackState {
	return AudioPlaybackState{}
}
func (noopAudioBackend) ClearEnded() {}
func (noopAudioBackend) Close()      {}

type nativeAudioBackend struct {
	player nativePlayer
}

type outputConfigurable interface {
	ApplyOutputConfig(cfg libmpv.OutputConfig) error
}

func (b *nativeAudioBackend) LoadURL(url string) error {
	return b.player.LoadURL(url)
}

func (b *nativeAudioBackend) Play() error {
	return b.player.Play()
}

func (b *nativeAudioBackend) Pause() {
	b.player.Pause()
}

func (b *nativeAudioBackend) Seek(seconds float64) error {
	return b.player.Seek(seconds)
}

func (b *nativeAudioBackend) SetVolume(volume float64) {
	b.player.SetVolume(volume)
}

func (b *nativeAudioBackend) GetState() AudioPlaybackState {
	currentTime, duration, playing, ended, errMsg := b.player.PlaybackState()
	return AudioPlaybackState{
		CurrentTime: currentTime,
		Duration:    duration,
		Playing:     playing,
		Ended:       ended,
		Error:       errMsg,
	}
}

func (b *nativeAudioBackend) ClearEnded() {
	b.player.ClearEnded()
}

func (b *nativeAudioBackend) Close() {
	b.player.Close()
}

// pcmPlayer releases the player's FIFO source when the player closes.
type pcmPlayer struct {
	nativePlayer
	src    *pcmsink.Source
	router *pcmsink.Router
}

func (p *pcmPlayer) Close() {
	p.nativePlayer.Close()
	if p.router != nil && p.src != nil {
		p.router.RemoveSource(p.src)
		p.src = nil
	}
}

// AudioService exposes native playback to the frontend when libmpv or libvlc is available.
//
// Lock order is opMu then mu. opMu serializes player operations while mu
// guards the backend, standby, and config fields. Code holding mu must
// never acquire opMu.
type AudioService struct {
	mu          sync.Mutex
	opMu        sync.Mutex
	backend     audioBackend
	standby     audioBackend
	activating  bool // a crossfade swap is running outside opMu
	preparedURL string
	volume      float64
	caps        AudioCapabilities
	preferred   string
	output      AudioOutputConfig
	router      *pcmsink.Router
	fanout      *pcmsink.Fanout
	pcmDir      string
}

// NewAudioService probes native backends once and keeps a backend when loading succeeds.
func NewAudioService() *AudioService {
	s := &AudioService{
		preferred: BackendAuto,
		volume:    1,
		output:    AudioOutputConfig{Mode: "auto"},
	}
	s.probeCapabilities()
	s.initBackend()
	return s
}

func (s *AudioService) probeCapabilities() {
	s.caps.MpvAvailable = libmpv.Available()
	s.caps.VlcAvailable = libvlc.Available()
	s.caps.MpvLoadError = ""
	if !s.caps.MpvAvailable {
		if err := libmpv.LoadError(); err != nil {
			s.caps.MpvLoadError = err.Error()
		}
	}
}

func resolveBackend(preferred string, mpvOK, vlcOK bool) string {
	switch preferred {
	case BackendMPV:
		if mpvOK {
			return BackendMPV
		}
	case BackendVLC:
		if vlcOK {
			return BackendVLC
		}
	}
	if mpvOK {
		return BackendMPV
	}
	if vlcOK {
		return BackendVLC
	}
	return ""
}

func (s *AudioService) mpvOutputConfig() libmpv.OutputConfig {
	return libmpv.NormalizeOutputConfig(libmpv.OutputConfig{
		Mode:      libmpv.OutputMode(s.output.Mode),
		Exclusive: s.output.Exclusive,
	})
}

func (s *AudioService) newNativePlayer(name string) (nativePlayer, error) {
	switch name {
	case BackendMPV:
		cfg := s.mpvOutputConfig()
		if s.router == nil {
			return libmpv.NewPlayerWithConfig(cfg)
		}
		src, err := s.router.NewSource()
		if err != nil {
			return nil, err
		}
		cfg.PCMPath = src.Path()
		player, err := libmpv.NewPlayerWithConfig(cfg)
		if err != nil {
			s.router.RemoveSource(src)
			return nil, err
		}
		return &pcmPlayer{nativePlayer: player, src: src, router: s.router}, nil
	case BackendVLC:
		return libvlc.NewPlayerWithConfig(libvlc.OutputConfig{
			Mode:      s.output.Mode,
			Exclusive: s.output.Exclusive,
		})
	default:
		return nil, errNativeUnavailable
	}
}

func (s *AudioService) backendCandidates() []string {
	// Pipe output targets decode through mpv's PCM audio output. VLC has no
	// equivalent raw stream, so targets restrict candidates to mpv.
	if len(s.output.Targets) > 0 {
		if s.caps.MpvAvailable && s.preferred != BackendVLC {
			return []string{BackendMPV}
		}
		return nil
	}
	switch s.preferred {
	case BackendMPV:
		if s.caps.MpvAvailable {
			return []string{BackendMPV}
		}
	case BackendVLC:
		if s.caps.VlcAvailable {
			return []string{BackendVLC}
		}
	default:
		var out []string
		if s.caps.MpvAvailable {
			out = append(out, BackendMPV)
		}
		if s.caps.VlcAvailable {
			out = append(out, BackendVLC)
		}
		return out
	}
	return nil
}

func (s *AudioService) clearPreparedLocked() {
	if s.standby != nil {
		s.standby.Close()
		s.standby = nil
	}
	s.preparedURL = ""
}

func (s *AudioService) openStandbyLocked() (audioBackend, error) {
	if !s.caps.NativeAvailable || s.caps.Backend == "" {
		return nil, errNativeUnavailable
	}
	player, err := s.newNativePlayer(s.caps.Backend)
	if err != nil {
		return nil, err
	}
	return &nativeAudioBackend{player: player}, nil
}

func (s *AudioService) initBackend() {
	s.clearPreparedLocked()
	candidates := s.backendCandidates()
	if len(candidates) == 0 {
		s.backend = noopAudioBackend{}
		s.caps.NativeAvailable = false
		s.caps.Backend = ""
		if len(s.output.Targets) > 0 {
			s.caps.InitError = "PCM output targets require the mpv backend"
		} else {
			s.caps.InitError = "no native audio libraries found"
		}
		return
	}

	var lastErr error
	for _, backendName := range candidates {
		player, err := s.newNativePlayer(backendName)
		if err == nil {
			s.backend = &nativeAudioBackend{player: player}
			s.caps.NativeAvailable = true
			s.caps.Backend = backendName
			s.caps.InitError = ""
			return
		}
		lastErr = err
	}

	s.backend = noopAudioBackend{}
	s.caps.NativeAvailable = false
	s.caps.Backend = ""
	if lastErr != nil {
		s.caps.InitError = lastErr.Error()
	} else {
		s.caps.InitError = "failed to initialize native audio backend"
	}
}

// normalizeTargets trims and drops empty output target specs.
func normalizeTargets(targets []string) []string {
	var out []string
	for _, t := range targets {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// rebuildPCMOutputLocked creates the sink fanout and FIFO router for the
// configured targets. Called with s.mu held.
func (s *AudioService) rebuildPCMOutputLocked() {
	s.teardownPCMOutputLocked()
	if len(s.output.Targets) == 0 {
		return
	}
	fanout, err := pcmsink.NewFanout(s.output.Targets)
	if err != nil {
		s.caps.PcmError = err.Error()
	}
	dir, err := os.MkdirTemp("", brand.Slug+"-pcm-*")
	if err != nil {
		s.caps.PcmError = err.Error()
		fanout.Close()
		return
	}
	router, err := pcmsink.NewRouter(dir, fanout)
	if err != nil {
		s.caps.PcmError = err.Error()
		fanout.Close()
		_ = os.RemoveAll(dir)
		return
	}
	s.fanout = fanout
	s.router = router
	s.pcmDir = dir
	s.caps.PcmActive = true
}

// teardownPCMOutputLocked stops the router and fanout. Called with s.mu held.
// The router owns and removes the FIFO directory.
func (s *AudioService) teardownPCMOutputLocked() {
	if s.router != nil {
		s.router.Close()
		s.router = nil
	}
	if s.fanout != nil {
		s.fanout.Close()
		s.fanout = nil
	}
	s.pcmDir = ""
	s.caps.PcmActive = false
	s.caps.PcmError = ""
	s.caps.Sinks = nil
}

// SetAudioOutputConfig applies immersive audio settings to native players.
// Mode or target changes recreate the backend so mpv and VLC start options
// take effect.
func (s *AudioService) SetAudioOutputConfig(cfg AudioOutputConfig) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	normalized := libmpv.NormalizeOutputConfig(libmpv.OutputConfig{
		Mode:      libmpv.OutputMode(cfg.Mode),
		Exclusive: cfg.Exclusive,
	})
	next := AudioOutputConfig{
		Mode:      string(normalized.Mode),
		Exclusive: normalized.Exclusive,
		Targets:   normalizeTargets(cfg.Targets),
	}
	same := next.Mode == s.output.Mode &&
		next.Exclusive == s.output.Exclusive &&
		slices.Equal(next.Targets, s.output.Targets)
	s.output = next

	if same {
		if nb, ok := s.backend.(*nativeAudioBackend); ok {
			if configurable, ok := nb.player.(outputConfigurable); ok {
				_ = configurable.ApplyOutputConfig(normalized)
			}
		}
		return
	}

	if s.backend != nil {
		s.backend.Close()
	}
	s.rebuildPCMOutputLocked()
	s.probeCapabilities()
	s.initBackend()
}

// GetAudioOutputConfig returns the active immersive audio settings.
func (s *AudioService) GetAudioOutputConfig() AudioOutputConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.output
}

// GetSinkStatus reports the health of each configured PCM output target.
func (s *AudioService) GetSinkStatus() []pcmsink.SinkStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fanout == nil {
		return nil
	}
	return s.fanout.Status()
}

// SetPreferredBackend switches the native backend. Pass "auto", "mpv", or "vlc".
func (s *AudioService) SetPreferredBackend(preferred string) {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	switch preferred {
	case BackendMPV, BackendVLC, BackendAuto:
	default:
		preferred = BackendAuto
	}

	if preferred == s.preferred && s.caps.NativeAvailable {
		if _, isNoop := s.backend.(noopAudioBackend); !isNoop {
			return
		}
	}

	s.preferred = preferred

	if s.backend != nil {
		s.backend.Close()
	}
	s.probeCapabilities()
	s.initBackend()
}

// Capabilities reports whether native playback can be used.
func (s *AudioService) Capabilities() AudioCapabilities {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fanout != nil {
		s.caps.Sinks = s.fanout.Status()
	}
	return s.caps
}

// LoadURL opens a stream URL in the native player.
func (s *AudioService) LoadURL(url string) error {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.Lock()
	s.clearPreparedLocked()
	backend := s.backend
	s.mu.Unlock()
	return backend.LoadURL(url)
}

// Play starts or resumes native playback.
func (s *AudioService) Play() error {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	s.mu.Lock()
	backend := s.backend
	s.mu.Unlock()
	return backend.Play()
}

// Pause pauses native playback.
func (s *AudioService) Pause() {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	s.mu.Lock()
	backend := s.backend
	s.mu.Unlock()
	backend.Pause()
}

// Seek moves playback to the given time in seconds.
func (s *AudioService) Seek(seconds float64) error {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	s.mu.Lock()
	backend := s.backend
	s.mu.Unlock()
	return backend.Seek(seconds)
}

// SetVolume sets native output volume from 0 to 1.
func (s *AudioService) SetVolume(volume float64) {
	if volume < 0 {
		volume = 0
	}
	if volume > 1 {
		volume = 1
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.Lock()
	s.volume = volume
	backend := s.backend
	s.mu.Unlock()
	backend.SetVolume(volume)
}

// GetState returns the current native playback state.
func (s *AudioService) GetState() AudioPlaybackState {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	s.mu.Lock()
	backend := s.backend
	s.mu.Unlock()
	return backend.GetState()
}

// ClearEnded resets the ended latch after the frontend handles track completion.
func (s *AudioService) ClearEnded() {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	s.mu.Lock()
	backend := s.backend
	s.mu.Unlock()
	backend.ClearEnded()
}

// PrepareURL loads the next track into a standby player for gapless handoff.
func (s *AudioService) PrepareURL(url string) error {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if url == "" {
		s.clearPreparedLocked()
		return nil
	}
	if s.preparedURL == url && s.standby != nil {
		return nil
	}
	s.clearPreparedLocked()
	standby, err := s.openStandbyLocked()
	if err != nil {
		return err
	}
	if err := standby.LoadURL(url); err != nil {
		standby.Close()
		return err
	}
	standby.SetVolume(0)
	s.standby = standby
	s.preparedURL = url
	return nil
}

// HasPrepared reports whether the standby player is ready for the given URL.
func (s *AudioService) HasPrepared(url string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return url != "" && s.preparedURL == url && s.standby != nil
}

// ActivatePrepared swaps the standby player into the primary slot.
// When crossfadeSec > 0, volumes are ramped between the two players.
// The timed fade runs without opMu so other playback ops stay responsive;
// the final swap re-locks and is dropped if teardown ran during the fade.
func (s *AudioService) ActivatePrepared(crossfadeSec float64) error {
	s.opMu.Lock()

	s.mu.Lock()
	if s.activating || s.standby == nil || s.preparedURL == "" {
		s.mu.Unlock()
		s.opMu.Unlock()
		return errNativeUnavailable
	}
	s.activating = true
	standby := s.standby
	primary := s.backend
	volume := s.volume
	s.mu.Unlock()

	if crossfadeSec < 0 {
		crossfadeSec = 0
	}

	if err := standby.Play(); err != nil {
		s.mu.Lock()
		s.activating = false
		s.clearPreparedLocked()
		s.mu.Unlock()
		s.opMu.Unlock()
		return err
	}

	// opMu stays released for the fade so Play, Pause, and GetState are
	// not blocked for the whole ramp. Players ignore SetVolume after
	// Close, so teardown racing the loop is safe.
	s.opMu.Unlock()

	if crossfadeSec > 0 {
		steps := max(int(crossfadeSec*20), 1)
		interval := time.Duration(float64(time.Second) * crossfadeSec / float64(steps))
		for i := 0; i <= steps; i++ {
			progress := float64(i) / float64(steps)
			primary.SetVolume(volume * (1 - progress))
			standby.SetVolume(volume * progress)
			if i < steps {
				time.Sleep(interval)
			}
		}
	} else {
		standby.SetVolume(volume)
	}

	s.opMu.Lock()
	defer s.opMu.Unlock()
	s.mu.Lock()
	s.activating = false
	if s.standby != standby || s.backend != primary {
		// Shutdown, ClearPrepared, or a backend rebuild tore this pair
		// down mid-fade. Drop the standby instead of reviving it.
		s.mu.Unlock()
		standby.Close()
		return nil
	}
	s.backend = standby
	s.standby = nil
	s.preparedURL = ""
	volume = s.volume
	s.mu.Unlock()

	standby.SetVolume(volume)
	primary.Pause()
	primary.SetVolume(volume)
	primary.Close()
	return nil
}

// ClearPrepared discards any standby preparation.
func (s *AudioService) ClearPrepared() {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearPreparedLocked()
}

// Shutdown closes the native backend.
func (s *AudioService) Shutdown() {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearPreparedLocked()
	s.backend.Close()
	s.backend = noopAudioBackend{}
	s.teardownPCMOutputLocked()
	s.caps.NativeAvailable = false
	s.caps.Backend = ""
	s.caps.InitError = ""
}
