// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Package pcmsink routes decoded PCM audio to flexible output targets such as
// the system audio device, stdout, FIFOs, and Unix or TCP sockets.
//
// The stream format is fixed little-endian signed 16-bit stereo at 48 kHz
// (192000 bytes per second) so every sink sees identical bytes.
package pcmsink

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Stream parameters shared by every source and sink.
const (
	SampleRate   = 48000
	Channels     = 2
	BytesPerSec  = SampleRate * Channels * 2
	frameBytes   = Channels * 2
	chunkBytes   = frameBytes * 1024
	sourceBufCap = 1 << 20
)

// Sink consumes raw s16le stereo PCM. Implementations may block briefly but
// should treat a stalled consumer as an error instead of blocking forever.
type Sink interface {
	io.WriteCloser
	// String returns the target spec for status reporting.
	String() string
}

// New creates a sink for a target spec:
//
//	device                  system audio device
//	stdout                  process standard output
//	fifo:<path>             named pipe (created when missing)
//	tcp:<host:port>         connect to a TCP listener
//	tcp-listen:<addr>       listen and stream to each TCP client
//	unix:<path>             connect to a Unix socket
//	unix-listen:<path>      listen on a Unix socket for clients
func New(spec string) (Sink, error) {
	spec = strings.TrimSpace(spec)
	kind, arg, _ := strings.Cut(spec, ":")
	kind = strings.ToLower(kind)
	arg = strings.TrimSpace(arg)

	switch kind {
	case "device":
		return newDeviceSink(spec)
	case "stdout":
		return writerSink{w: nopCloser{os.Stdout}, spec: spec}, nil
	case "fifo":
		if arg == "" {
			return nil, fmt.Errorf("pcmsink: fifo target needs a path")
		}
		return newFIFOSink(spec, arg)
	case "tcp":
		if arg == "" {
			return nil, fmt.Errorf("pcmsink: tcp target needs an address")
		}
		return newDialSink(spec, "tcp", arg), nil
	case "tcp-listen":
		if arg == "" {
			return nil, fmt.Errorf("pcmsink: tcp-listen target needs an address")
		}
		return newListenSink(spec, "tcp", arg)
	case "unix":
		if arg == "" {
			return nil, fmt.Errorf("pcmsink: unix target needs a path")
		}
		return newDialSink(spec, "unix", arg), nil
	case "unix-listen":
		if arg == "" {
			return nil, fmt.Errorf("pcmsink: unix-listen target needs a path")
		}
		return newListenSink(spec, "unix", arg)
	default:
		return nil, fmt.Errorf("pcmsink: unknown output target %q", spec)
	}
}
