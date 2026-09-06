// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package libmpv

import "strings"

// OutputMode controls how libmpv maps decoded audio to the system output.
type OutputMode string

const (
	OutputAuto        OutputMode = "auto"
	OutputStereo      OutputMode = "stereo"
	OutputSurround    OutputMode = "surround"
	OutputPassthrough OutputMode = "passthrough"
	OutputBinaural    OutputMode = "binaural"
)

// OutputConfig is applied to each mpv player instance.
type OutputConfig struct {
	Mode      OutputMode `json:"mode"`
	Exclusive bool       `json:"exclusive"`
	// PCMPath routes decoded audio to a named pipe instead of a sound device.
	// The stream is raw s16le stereo at 48 kHz with no WAV header. When set,
	// Mode and Exclusive are ignored because the PCM format is fixed.
	PCMPath string `json:"pcmPath,omitempty"`
}

// DefaultOutputConfig returns safe defaults for stereo music playback.
func DefaultOutputConfig() OutputConfig {
	return OutputConfig{Mode: OutputAuto}
}

// NormalizeOutputConfig clamps unknown values to safe defaults.
func NormalizeOutputConfig(cfg OutputConfig) OutputConfig {
	switch OutputMode(strings.ToLower(string(cfg.Mode))) {
	case OutputStereo, OutputSurround, OutputPassthrough, OutputBinaural, OutputAuto:
		cfg.Mode = OutputMode(strings.ToLower(string(cfg.Mode)))
	default:
		cfg.Mode = OutputAuto
	}
	if cfg.Mode != OutputPassthrough {
		cfg.Exclusive = false
	}
	return cfg
}

type mpvOption struct {
	name  string
	value string
}

func audioOutputOptions(cfg OutputConfig) []mpvOption {
	if cfg.PCMPath != "" {
		return []mpvOption{
			{name: "ao", value: "pcm"},
			{name: "ao-pcm-file", value: cfg.PCMPath},
			{name: "ao-pcm-waveheader", value: "no"},
			{name: "audio-samplerate", value: "48000"},
			{name: "audio-format", value: "s16"},
			{name: "audio-channels", value: "stereo"},
			{name: "audio-spdif", value: "no"},
			{name: "audio-exclusive", value: "no"},
			{name: "af", value: ""},
		}
	}

	cfg = NormalizeOutputConfig(cfg)
	opts := []mpvOption{}

	switch cfg.Mode {
	case OutputStereo:
		opts = append(opts,
			mpvOption{name: "audio-channels", value: "stereo"},
			mpvOption{name: "audio-spdif", value: "no"},
			mpvOption{name: "audio-exclusive", value: "no"},
			mpvOption{name: "af", value: ""},
		)
	case OutputSurround:
		opts = append(opts,
			mpvOption{name: "audio-channels", value: "7.1,5.1,stereo"},
			mpvOption{name: "audio-spdif", value: "no"},
			mpvOption{name: "audio-exclusive", value: "no"},
			mpvOption{name: "af", value: ""},
		)
	case OutputPassthrough:
		exclusive := "no"
		if cfg.Exclusive {
			exclusive = "yes"
		}
		opts = append(opts,
			mpvOption{name: "audio-channels", value: "7.1,5.1,stereo"},
			mpvOption{name: "audio-spdif", value: "ac3,eac3,dts,dts-hd,truehd"},
			mpvOption{name: "audio-exclusive", value: exclusive},
			mpvOption{name: "af", value: ""},
		)
	case OutputBinaural:
		// Decode full layouts then downmix with Bauer stereophonic-to-binaural
		// crossfeed so headphones keep a stable center image.
		opts = append(opts,
			mpvOption{name: "audio-channels", value: "stereo"},
			mpvOption{name: "audio-spdif", value: "no"},
			mpvOption{name: "audio-exclusive", value: "no"},
			mpvOption{name: "af", value: "lavfi=[bs2b=profile=default]"},
		)
	default:
		opts = append(opts,
			mpvOption{name: "audio-channels", value: "auto"},
			mpvOption{name: "audio-spdif", value: "no"},
			mpvOption{name: "audio-exclusive", value: "no"},
			mpvOption{name: "af", value: ""},
		)
	}
	return opts
}

// ApplyOutputConfig updates runtime audio properties on an initialized player.
func (p *Player) ApplyOutputConfig(cfg OutputConfig) error {
	p.apiMu.Lock()
	defer p.apiMu.Unlock()

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return errUnavailable
	}
	// PCM pipe output has a fixed format. Channel layout and passthrough
	// options do not apply and would disrupt the stream.
	if p.output.PCMPath != "" {
		p.mu.Unlock()
		return nil
	}
	handle := p.handle
	p.mu.Unlock()

	for _, opt := range audioOutputOptions(cfg) {
		if opt.name == "af" && opt.value == "" {
			_ = runCommand(handle, "af", "clr", "")
			continue
		}
		if err := runCommand(handle, "set", opt.name, opt.value); err != nil {
			if opt.name == "af" {
				continue
			}
			return err
		}
	}
	return nil
}
