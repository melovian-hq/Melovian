// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build (android || ios) && !server

package main

import (
	"fmt"
	"os"
	"runtime/debug"
)

// handleFatalError logs a startup failure and parks this goroutine.
// os.Exit would kill the whole process and Android would restart it in a loop.
func handleFatalError(err error) {
	msg := fmt.Sprintf("fatal: %v\n%s", err, debug.Stack())
	mobileLog(msg)
	fmt.Fprintln(os.Stderr, msg)
	select {}
}
