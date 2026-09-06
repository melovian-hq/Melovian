// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package pcmsink

import (
	"errors"
	"io"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// ensureFIFO creates path as a named pipe when it does not already exist.
func ensureFIFO(path string) error {
	info, err := os.Stat(path)
	switch {
	case err == nil:
		if info.Mode()&os.ModeNamedPipe == 0 {
			return errors.New("pcmsink: path exists and is not a FIFO")
		}
		return nil
	case !os.IsNotExist(err):
		return err
	}
	return unix.Mkfifo(path, 0o600)
}

// removeSocketFile unlinks a stale Unix socket path before listening.
func removeSocketFile(path string) error {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return err
	}
	return os.Remove(path)
}

// newFIFOSink creates path when missing and writes PCM into the pipe.
// Opening is nonblocking so the sink reconnects as readers come and go.
func newFIFOSink(spec, path string) (Sink, error) {
	if err := ensureFIFO(path); err != nil {
		return nil, err
	}
	s := &dialSink{
		spec:     spec,
		retryMin: 500 * time.Millisecond,
	}
	s.open = func() (io.WriteCloser, error) {
		fd, err := unix.Open(path, unix.O_WRONLY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
		if err != nil {
			return nil, &os.PathError{Op: "open", Path: path, Err: err}
		}
		return os.NewFile(uintptr(fd), path), nil
	}
	return s, nil
}

// openFIFOReader opens a FIFO for reading without waiting for a writer.
// The descriptor stays nonblocking: the router polls with a short sleep,
// which also lets RemoveSource interrupt the read loop cleanly.
func openFIFOReader(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	return os.NewFile(uintptr(fd), path), nil
}
