// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build !unix

package pcmsink

import (
	"errors"
	"os"
)

var errPlatform = errors.New("pcmsink: FIFO targets are not supported on this platform")

func ensureFIFO(string) error { return errPlatform }

func removeSocketFile(string) error { return nil }

func newFIFOSink(string, string) (Sink, error) { return nil, errPlatform }

func openFIFOReader(string) (*os.File, error) { return nil, errPlatform }
