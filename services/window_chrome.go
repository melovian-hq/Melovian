// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

func applyNativeTitleBar(window application.Window, enabled bool) {
	if window == nil {
		return
	}
	application.InvokeSync(func() {
		window.SetFrameless(!enabled)
	})
}
