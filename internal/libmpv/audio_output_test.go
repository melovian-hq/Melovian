// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package libmpv

import "testing"

func TestNormalizeOutputConfig(t *testing.T) {
	t.Parallel()

	got := NormalizeOutputConfig(OutputConfig{Mode: "PASSTHROUGH", Exclusive: true})
	if got.Mode != OutputPassthrough {
		t.Fatalf("mode = %q, want passthrough", got.Mode)
	}
	if !got.Exclusive {
		t.Fatal("expected exclusive to stay enabled for passthrough")
	}

	got = NormalizeOutputConfig(OutputConfig{Mode: "stereo", Exclusive: true})
	if got.Exclusive {
		t.Fatal("exclusive must be cleared outside passthrough")
	}

	got = NormalizeOutputConfig(OutputConfig{Mode: "nope"})
	if got.Mode != OutputAuto {
		t.Fatalf("mode = %q, want auto", got.Mode)
	}
}

func TestAudioOutputOptionsPassthrough(t *testing.T) {
	t.Parallel()

	opts := audioOutputOptions(OutputConfig{Mode: OutputPassthrough, Exclusive: true})
	byName := map[string]string{}
	for _, opt := range opts {
		byName[opt.name] = opt.value
	}
	if byName["audio-spdif"] != "ac3,eac3,dts,dts-hd,truehd" {
		t.Fatalf("audio-spdif = %q", byName["audio-spdif"])
	}
	if byName["audio-exclusive"] != "yes" {
		t.Fatalf("audio-exclusive = %q", byName["audio-exclusive"])
	}
}

func TestAudioOutputOptionsBinaural(t *testing.T) {
	t.Parallel()

	opts := audioOutputOptions(OutputConfig{Mode: OutputBinaural})
	byName := map[string]string{}
	for _, opt := range opts {
		byName[opt.name] = opt.value
	}
	if byName["audio-channels"] != "stereo" {
		t.Fatalf("audio-channels = %q", byName["audio-channels"])
	}
	if byName["af"] == "" {
		t.Fatal("expected binaural af filter")
	}
}
