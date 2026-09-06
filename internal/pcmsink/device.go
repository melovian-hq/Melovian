// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package pcmsink

import (
	"errors"
	"io"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"

	"melovian/internal/brand"
)

// deviceSink plays PCM on the system audio device. The oto player pulls from
// an io.Pipe so Write simply feeds the pipe; the sound device paces reads.
type deviceSink struct {
	spec   string
	ctx    *oto.Context
	player *oto.Player
	pw     *io.PipeWriter
	pr     *io.PipeReader
	once   sync.Once
}

func newDeviceSink(spec string) (Sink, error) {
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:      SampleRate,
		ChannelCount:    Channels,
		Format:          oto.FormatSignedInt16LE,
		BufferSize:      100 * time.Millisecond,
		ApplicationName: brand.Name,
	})
	if err != nil {
		return nil, err
	}
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		return nil, errDeviceTimeout
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	player := ctx.NewPlayer(pr)
	player.Play()
	return &deviceSink{spec: spec, ctx: ctx, player: player, pw: pw, pr: pr}, nil
}

var errDeviceTimeout = errors.New("pcmsink: audio device did not initialize")

func (s *deviceSink) Write(p []byte) (int, error) {
	if err := s.player.Err(); err != nil {
		return 0, err
	}
	return s.pw.Write(p)
}

func (s *deviceSink) Close() error {
	s.once.Do(func() {
		_ = s.pw.Close()
		_ = s.pr.Close()
	})
	return nil
}

func (s *deviceSink) String() string { return s.spec }
