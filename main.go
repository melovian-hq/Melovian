// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build !server && !android && !ios

package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"melovian/internal/api"
	"melovian/internal/appconfig"
	"melovian/internal/brand"
	"melovian/internal/desktop"
	"melovian/internal/melog"
	"melovian/internal/observability"
	"melovian/internal/sandbox"
	"melovian/internal/store"
	"melovian/services"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	defer melog.DeferredPanicHandler()
	defer observability.Flush(2 * time.Second)

	cfg, err := appconfig.LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	desktop.ApplyWebKitStabilityEnv(cfg.DataDir)

	if _, err := melog.Init(cfg.DataDir); err != nil {
		fmt.Fprintln(os.Stderr, "failed to init logging:", err)
		os.Exit(1)
	}

	db, err := store.Open(cfg)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	apiServer := api.NewServer(cfg, db)
	if err := apiServer.Start(); err != nil {
		slog.Error("failed to start api server", "err", err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = apiServer.Stop(ctx)
	}()

	landlockStatus := sandbox.Apply(cfg, sandbox.ModeDesktop)
	sandbox.LogStatus(landlockStatus)
	desktop.DisableWebKitSandboxIfLandlocked(landlockStatus.Enabled)

	mediaSvc := services.NewMediaService(nil)
	mediaSvc.SetDataDir(cfg.DataDir)
	audioSvc := services.NewAudioService()
	updateSvc := services.NewUpdateService()
	wireUpdatePrefs(updateSvc, db)

	app := application.New(application.Options{
		Name:        brand.Name,
		Description: brand.Description,
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(mediaSvc),
			application.NewService(audioSvc),
			application.NewService(updateSvc),
		},
		Assets: application.AssetOptions{
			Handler: &api.CombinedHandler{
				API:    apiServer.Handler(),
				Assets: application.BundledAssetFileServer(assets),
			},
		},
		Linux: application.LinuxOptions{
			ProgramName:                   brand.Slug,
			DisableQuitOnLastWindowClosed: true,
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	mediaSvc.SetApp(app)
	updateSvc.SetApp(app)
	apiServer.SetUpdateHooks(&api.DesktopUpdateHooks{
		Apply:   updateSvc.ApplyUpdate,
		Restart: updateSvc.RestartToApply,
		Status:  updateSvc.GetStatus,
	})
	if len(appIcon) > 0 {
		app.SetIcon(appIcon)
	}

	windowStateStore := desktop.NewWindowStateStore(cfg.DataDir)
	savedWindowState, restoredWindowState := windowStateStore.Load(desktop.ResetWindowRequested())

	windowOpts := application.WebviewWindowOptions{
		Name:      services.MainWindowName,
		Title:     brand.Name,
		Frameless: runtime.GOOS != "darwin",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
			WebviewPreferences: application.MacWebviewPreferences{
				EnableAutoplayWithoutUserAction: application.Enabled,
			},
		},
		BackgroundColour: application.NewRGB(9, 9, 9),
		URL:              "/music",
	}
	desktop.ApplyWindowState(&windowOpts, savedWindowState, restoredWindowState)
	window := app.Window.NewWithOptions(windowOpts)

	windowStateStore.Attach(window)
	desktop.RegisterCloseHandler(window)
	trayManager := desktop.NewTrayManager(app, window, mediaSvc, appIcon)
	trayManager.Setup()
	services.SetTaskbarIntegrationHook(trayManager.SetEnabled)
	desktop.RegisterQuitSignalHandler(app)

	if runtime.GOOS == "darwin" {
		app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(*application.ApplicationEvent) {
			desktop.ShowMainWindow(window)
		})
	}

	err = app.Run()

	slog.Info("application run loop ended")
	services.ShutdownMediaService(mediaSvc)
	audioSvc.Shutdown()
	updateSvc.Shutdown()
	slog.Info("shutdown complete")
	if err != nil {
		slog.Error("application exited with error", "err", err)
		os.Exit(1)
	}
}

// wireUpdatePrefs connects the update service to the local preferences
// store. Desktop installs are single-user, so the empty user scope holds
// the auto-update opt-in and release channel.
func wireUpdatePrefs(updateSvc *services.UpdateService, db *store.DB) {
	prefs := store.NewPreferencesStore(db)
	read := func() (map[string]any, error) {
		raw, err := prefs.Get("", store.PrefKeyUpdateSettings)
		if err != nil {
			return map[string]any{}, nil
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return map[string]any{}, nil
		}
		return m, nil
	}
	updateSvc.LoadAutoUpdate = func() (bool, string) {
		m, err := read()
		if err != nil {
			return false, "stable"
		}
		auto, _ := m["autoUpdate"].(bool)
		channel, _ := m["channel"].(string)
		if channel == "" {
			channel = "stable"
		}
		return auto, channel
	}
	updateSvc.SaveAutoUpdate = func(enabled bool) error {
		m, _ := read()
		m["autoUpdate"] = enabled
		data, err := json.Marshal(m)
		if err != nil {
			return err
		}
		return prefs.Set("", store.PrefKeyUpdateSettings, string(data))
	}
}
