// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build !linux

package sandbox

import (
	"fmt"
	"runtime"

	"melovian/internal/appconfig"
)

func Apply(_ appconfig.Config, _ Mode) Status {
	return Status{
		Reason: fmt.Sprintf("Landlock requires Linux (running on %s)", runtime.GOOS),
	}
}
