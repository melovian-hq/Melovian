// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package desktop

import (
	"runtime"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/icons"
	"melovian/internal/brand"
)

type TrayManager struct {
	app    *application.App
	window application.Window
	media  MediaController
	icon   []byte

	mu      sync.Mutex
	enabled bool
	tray    *application.SystemTray
}

func NewTrayManager(
	app *application.App,
	window application.Window,
	media MediaController,
	icon []byte,
) *TrayManager {
	return &TrayManager{
		app:     app,
		window:  window,
		media:   media,
		icon:    icon,
		enabled: true,
	}
}

func (tm *TrayManager) SetEnabled(enabled bool) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tm.enabled == enabled {
		return
	}
	tm.enabled = enabled
	if enabled {
		tm.createTrayLocked()
		return
	}
	tm.destroyTrayLocked()
}

func (tm *TrayManager) createTrayLocked() {
	if tm.tray != nil || len(tm.icon) == 0 {
		return
	}

	tray := tm.app.SystemTray.New()
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(icons.SystrayMacTemplate)
	} else {
		tray.SetIcon(tm.icon)
	}
	tray.SetTooltip(brand.Name)

	menu := tm.app.NewMenu()
	menu.Add("Show " + brand.Name).OnClick(func(*application.Context) {
		ShowMainWindow(tm.window)
	})
	menu.Add("Hide").OnClick(func(*application.Context) {
		tm.window.Hide()
	})
	menu.AddSeparator()
	menu.Add("Play / Pause").OnClick(func(*application.Context) {
		tm.media.TogglePlayback()
	})
	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(*application.Context) {
		tm.window.EmitEvent(AppQuitRequestedEvent)
	})
	tray.SetMenu(menu)

	tray.AttachWindow(tm.window).WindowOffset(8)
	tray.OnClick(func() {
		tray.ToggleWindow()
	})

	tm.tray = tray
}

func (tm *TrayManager) destroyTrayLocked() {
	if tm.tray == nil {
		return
	}
	tm.tray.Destroy()
	tm.tray = nil
}

func (tm *TrayManager) Setup() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if tm.enabled {
		tm.createTrayLocked()
	}
}
