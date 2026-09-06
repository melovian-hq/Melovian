// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build ios

package main

import (
	"C"
	"fmt"
	"os"
	"runtime/debug"
)

// WailsIOSMain runs main() after UIKit has launched. The WailsAppDelegate
// calls it from didFinishLaunchingWithOptions on a background thread so the
// Go runtime does not race UIApplicationMain.

//export WailsIOSMain
func WailsIOSMain() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "application panic: %v\n%s\n", r, debug.Stack())
		}
	}()
	main()
}
