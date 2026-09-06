// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build server

package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"melovian/internal/api"
	"melovian/internal/appconfig"
	"melovian/internal/democatalog"
	"melovian/internal/melog"
	"melovian/internal/observability"
	"melovian/internal/sandbox"
	"melovian/internal/store"
	"melovian/internal/termout"
)

//go:embed all:frontend/dist
var webAssets embed.FS

func main() {
	cli := appconfig.NewServerCLI()
	if err := cli.Parse(os.Args[1:]); err != nil {
		termout.Fail(err.Error())
		os.Exit(2)
	}

	if cli.ShowHelp {
		printServerHelp(cli)
		os.Exit(0)
	}

	defer melog.DeferredPanicHandler()
	defer observability.Flush(2 * time.Second)

	envLoaded, err := cli.LoadEnvFile()
	if err != nil {
		termout.Fail(err.Error())
		os.Exit(1)
	}

	cfg, err := appconfig.LoadConfig()
	if err != nil {
		termout.Fail(err.Error())
		os.Exit(1)
	}

	if err := cli.Apply(&cfg); err != nil {
		termout.Fail(err.Error())
		os.Exit(1)
	}

	democatalog.ApplyDefaults(&cfg)

	authRes, err := appconfig.PrepareServerAuth(&cfg, cli.StartupOptions())
	if err != nil {
		termout.Fail(err.Error())
		os.Exit(1)
	}

	if err := appconfig.ValidateServerConfig(cfg); err != nil {
		termout.Fail(err.Error())
		os.Exit(1)
	}

	if err := cli.ApplyDefaultLogging(); err != nil {
		termout.Fail(err.Error())
		os.Exit(1)
	}

	if _, err := melog.Init(cfg.DataDir); err != nil {
		termout.Fail("failed to init logging: " + err.Error())
		os.Exit(1)
	}

	printStartupBanner(envLoaded, cli, cfg, authRes)

	if cfg.DemoModeEffective() && authSecretConfigured(cli) {
		slog.Warn("demo mode overrides auth secret; account login is disabled")
		termout.Note("Demo mode active; account login is disabled")
	}
	if democatalog.UsesFakeCatalog(cfg) {
		termout.Note("Demo catalog: built-in fake library (no upstream Subsonic)")
	}
	if authRes.NoAuth {
		if authRes.IgnoredSecret {
			slog.Warn("no-auth disables authentication; ignoring auth secret")
			termout.Note("Authentication disabled; ignoring auth secret")
		} else {
			slog.Warn("authentication disabled; server is open without account login")
			termout.Note("Authentication disabled; server is open without account login")
		}
	}
	if authRes.Generated {
		slog.Warn(
			"generated auth secret for this run; save it to persist sessions across restarts",
			"secret", authRes.GeneratedSecret,
		)
		termout.Note("Generated auth secret (save to persist sessions across restarts):")
		termout.Line("auth-secret", authRes.GeneratedSecret)
	}

	db, err := store.Open(cfg)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		termout.Fail("Failed to open database: " + err.Error())
		os.Exit(1)
	}
	defer db.Close()

	if democatalog.UsesFakeCatalog(cfg) {
		instances := store.NewInstanceStore(db)
		inst, err := democatalog.EnsureInstance(instances)
		if err != nil {
			slog.Error("failed to ensure demo instance", "err", err)
			termout.Fail("Failed to set up demo catalog: " + err.Error())
			os.Exit(1)
		}
		listen := store.NewListenStore(db)
		scope := "instance:" + inst.ID
		if err := democatalog.SeedUserData(listen, scope); err != nil {
			slog.Warn("demo catalog seed failed", "err", err)
		}
		go democatalog.WarmArtworkCache()
	}

	landlockStatus := sandbox.Apply(cfg, sandbox.ModeServer)
	sandbox.LogStatus(landlockStatus)
	if landlockStatus.Enabled {
		termout.Line("landlock", landlockStatus.Detail)
	} else if landlockStatus.Reason != "" {
		termout.Line("landlock", "disabled: "+landlockStatus.Reason)
	}

	apiServer := api.NewServer(cfg, db)
	dist, err := fs.Sub(webAssets, "frontend/dist")
	if err != nil {
		slog.Error("failed to load web assets", "err", err)
		termout.Fail("Failed to load web assets: " + err.Error())
		os.Exit(1)
	}

	handler := api.IPAllowlistMiddleware(cfg.AllowedIPs, cfg.TrustProxy, &api.CombinedHandler{
		API:       apiServer.Handler(),
		Assets:    api.StaticAssetHandler(dist),
		Shell:     dist,
		PublicURL: cfg.PublicURL,
	})

	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		url := "http://" + cfg.ListenAddr
		slog.Info("melovian listening", "addr", cfg.ListenAddr, "url", url)
		termout.OK("Listening on " + url)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server failed", "err", err)
			termout.Fail("HTTP server failed: " + err.Error())
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
	slog.Info("shutdown complete")
	termout.Note("Shutdown complete")
}

func printServerHelp(cli *appconfig.ServerCLI) {
	termout.Banner()
	cli.PrintHelp()
}

func printStartupBanner(envLoaded bool, cli *appconfig.ServerCLI, cfg appconfig.Config, authRes appconfig.ServerAuthResolution) {
	termout.Banner()
	if envLoaded {
		if cli.Visited("env-file") {
			termout.Line("env", cli.EnvFile)
		} else {
			termout.Line("env", ".env")
		}
	}
	termout.Line("listen", cfg.ListenAddr)
	termout.Line("data", cfg.DataDir)
	switch {
	case cfg.DemoModeEffective():
		termout.Line("auth", "demo (read-only)")
	case authRes.NoAuth:
		termout.Line("auth", "disabled")
	case cfg.AuthEnabled():
		if authRes.Generated {
			termout.Line("auth", "enabled (generated secret)")
		} else {
			termout.Line("auth", "enabled")
		}
	}
	if cfg.PublicURL != "" {
		termout.Line("public-url", cfg.PublicURL)
	}
	if cfg.IPAllowlistEnabled() {
		termout.Line("allowed-ips", formatAllowedIPs(cfg))
	}
	mainLog, _, _ := melog.LogPaths(cfg.DataDir)
	termout.Line("log-file", mainLog)
	termout.Line("log-level", os.Getenv("MELOVIAN_LOG_LEVEL"))
	fmt.Fprintln(os.Stdout)
}

func authSecretConfigured(cli *appconfig.ServerCLI) bool {
	if cli.Visited("auth-secret") && strings.TrimSpace(cli.AuthSecret) != "" {
		return true
	}
	return strings.TrimSpace(os.Getenv("MELOVIAN_AUTH_SECRET")) != ""
}

func formatAllowedIPs(cfg appconfig.Config) string {
	parts := make([]string, 0, len(cfg.AllowedIPs))
	for _, prefix := range cfg.AllowedIPs {
		parts = append(parts, prefix.String())
	}
	return strings.Join(parts, ", ")
}
