// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package desktop

import (
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func RegisterQuitSignalHandler(app *application.App) {
	if runtime.GOOS == "windows" {
		return
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		slog.Info("shutdown signal received", "signal", sig.String())
		if app == nil {
			os.Exit(0)
		}
		application.InvokeSync(func() {
			app.Quit()
		})
	}()
}
