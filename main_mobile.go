// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build (android || ios) && !server

package main

import (
	"embed"
	"fmt"
	"log/slog"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"melovian/internal/api"
	"melovian/internal/appconfig"
	"melovian/internal/brand"
	"melovian/internal/melog"
	"melovian/internal/observability"
	"melovian/internal/store"
	"melovian/services"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	defer func() {
		observability.Flush(2 * time.Second)
		if recovered := recover(); recovered != nil {
			handleFatalError(fmt.Errorf("panic: %v", recovered))
		}
	}()

	cfg, err := appconfig.LoadConfig()
	if err != nil {
		handleFatalError(err)
	}

	if _, err := melog.Init(cfg.DataDir); err != nil {
		handleFatalError(fmt.Errorf("init logging: %w", err))
	}
	slog.Info("logging initialized", "dataDir", cfg.DataDir)

	db, err := store.Open(cfg)
	if err != nil {
		handleFatalError(fmt.Errorf("open database: %w", err))
	}
	defer db.Close()

	apiServer := api.NewServer(cfg, db)
	mediaSvc := services.NewMediaService(nil)
	mediaSvc.SetDataDir(cfg.DataDir)
	audioSvc := services.NewAudioService()

	opts := application.Options{
		Name:        brand.Name,
		Description: "Music player for local libraries and Subsonic-compatible servers",
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(mediaSvc),
			application.NewService(audioSvc),
		},
		Assets: application.AssetOptions{
			Handler: &api.CombinedHandler{
				API:    apiServer.Handler(),
				Assets: application.BundledAssetFileServer(assets),
			},
		},
	}
	modifyOptionsForIOS(&opts)

	app := application.New(opts)

	mediaSvc.SetApp(app)
	if len(appIcon) > 0 {
		app.SetIcon(appIcon)
	}

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             services.MainWindowName,
		Title:            brand.Name,
		BackgroundColour: application.NewRGB(9, 9, 9),
		URL:              "/music",
	})
	_ = window

	err = app.Run()
	slog.Info("application run loop ended")
	services.ShutdownMediaService(mediaSvc)
	audioSvc.Shutdown()
	slog.Info("shutdown complete")
	if err != nil {
		handleFatalError(fmt.Errorf("application exited: %w", err))
	}
}
