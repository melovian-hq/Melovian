// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package desktop

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"melovian/internal/brand"
)

var MainWindowCloseRequestedEvent = brand.Slug + ":window-close-requested"
var AppQuitRequestedEvent = brand.Slug + ":app-quit-requested"

func RegisterCloseHandler(window application.Window) {
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		e.Cancel()
		window.EmitEvent(MainWindowCloseRequestedEvent)
	})
}
