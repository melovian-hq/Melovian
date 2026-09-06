// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package osutil

import (
	"os/exec"
	"runtime"
)

func RevealPath(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start() //#nosec G204 -- OS file reveal. Path is user-selected library location
	case "windows":
		return exec.Command("explorer", path).Start() //#nosec G204 -- OS file reveal. Path is user-selected library location
	default:
		return exec.Command("xdg-open", path).Start() //#nosec G204 -- OS file reveal. Path is user-selected library location
	}
}
