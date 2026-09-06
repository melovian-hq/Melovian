// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build ios && !server

package main

import (
	"fmt"
	"os"
)

func mobileLog(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}
