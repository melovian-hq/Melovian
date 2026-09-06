// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package desktop

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

func ShowMainWindow(window application.Window) {
	if window == nil {
		return
	}
	window.Show()
	window.UnMinimise()
	window.Restore()
	window.Focus()
}
