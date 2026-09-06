// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// melovian-updater is a standalone update agent for Melovian servers. It
// checks the releases Atom feed, verifies signed checksums, applies delta
// patches when published, swaps the server binary, and restarts the
// service. The install-service command writes a periodic update unit for
// systemd, OpenRC, runit, or dinit.
//
// Typical use:
//
//	melovian-updater check
//	melovian-updater apply --restart "systemctl restart melovian"
//	melovian-updater install-service --init systemd --service melovian
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"melovian/internal/brand"
	"melovian/internal/compat"
	"melovian/internal/update"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "check":
		err = cmdCheck(os.Args[2:])
	case "apply":
		err = cmdApply(os.Args[2:])
	case "install-service":
		err = cmdInstallService(os.Args[2:])
	case "uninstall-service":
		err = cmdUninstallService(os.Args[2:])
	case "version":
		fmt.Println(brand.Slug+"-updater", compat.Version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `%s-updater %s - update agent for %s servers

Commands:
  check               Report the latest release and whether one is newer
  apply               Download, verify, and install an update
  install-service     Install a periodic update unit for an init system
  uninstall-service   Remove the periodic update unit
  version             Print this tool's version

Run "<command> -h" for flags.
`, brand.Slug, compat.Version, brand.Name)
}

func progressPrinter(p update.Progress) {
	switch p.Stage {
	case update.StageDownload:
		if p.Total > 0 {
			fmt.Fprintf(os.Stderr, "\rdownloading %.0f%% (%d/%d bytes)", float64(p.Written)/float64(p.Total)*100, p.Written, p.Total)
		} else {
			fmt.Fprintf(os.Stderr, "\rdownloading %d bytes", p.Written)
		}
	default:
		fmt.Fprint(os.Stderr, "\r\033[K")
		fmt.Println(p.Stage+":", p.Message)
	}
}

func cmdCheck(args []string) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	version := fs.String("version", "", "version to look up instead of latest")
	channel := fs.String("channel", "stable", "release channel: stable or prerelease")
	current := fs.String("current", compat.Version, "version to compare against")
	jsonOut := fs.Bool("json", false, "print machine-readable status")
	if err := fs.Parse(args); err != nil {
		return err
	}
	res, err := update.Check(context.Background(), *current, update.Options{
		Channel:       update.Channel(*channel),
		TargetVersion: *version,
	})
	if err != nil {
		return err
	}
	if *jsonOut {
		return json.NewEncoder(os.Stdout).Encode(res)
	}
	fmt.Println("current:", res.Current)
	if res.UpToDate {
		fmt.Println("status: up to date")
		return nil
	}
	fmt.Println("latest:", res.Latest.Version)
	fmt.Println("notes:", res.Latest.NotesURL)
	fmt.Println("status: update available")
	return nil
}

func cmdApply(args []string) error {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	version := fs.String("version", "", "version to install instead of latest")
	channel := fs.String("channel", "stable", "release channel: stable or prerelease")
	target := fs.String("binary", "", "path of the binary to replace (default: the running "+brand.Slug+" server binary)")
	restart := fs.String("restart", "", "command run after install (for example \"systemctl restart melovian\")")
	current := fs.String("current", compat.Version, "installed version to update from")
	if err := fs.Parse(args); err != nil {
		return err
	}
	res, err := update.Apply(context.Background(), *current, update.Options{
		Channel:        update.Channel(*channel),
		TargetVersion:  *version,
		Target:         *target,
		RestartCommand: *restart,
		OnProgress:     progressPrinter,
	})
	if err != nil {
		fmt.Fprint(os.Stderr, "\r\033[K")
		return err
	}
	if res.Method == "none" {
		fmt.Println("already up to date")
		return nil
	}
	fmt.Printf("updated to v%s via %s (%d bytes downloaded)\n", res.Version, res.Method, res.BytesDownloaded)
	if !res.Restarted && *restart == "" {
		fmt.Println("restart the service to run the new version")
	}
	return nil
}

func cmdInstallService(args []string) error {
	fs := flag.NewFlagSet("install-service", flag.ContinueOnError)
	initSystem := fs.String("init", "", "init system: systemd, openrc, runit, dinit")
	service := fs.String("service", brand.Slug, "name of the running service to restart after updates")
	binary := fs.String("binary", "/usr/local/bin/"+brand.Slug+"-server", "path of the server binary to update")
	updaterPath := fs.String("updater", "/usr/local/bin/"+brand.Slug+"-updater", "path of this tool")
	interval := fs.String("interval", "daily", "check interval: systemd calendar expression or daily/weekly/monthly")
	dryRun := fs.Bool("print", false, "print the unit files instead of writing them")
	if err := fs.Parse(args); err != nil {
		return err
	}
	files, err := ServiceFiles(InitSystem(*initSystem), ServiceSpec{
		ServiceName: *service,
		BinaryPath:  *binary,
		UpdaterPath: *updaterPath,
		Interval:    *interval,
	})
	if err != nil {
		return err
	}
	for _, f := range files {
		if *dryRun {
			fmt.Printf("=== %s ===\n%s\n", f.Path, f.Content)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(f.Path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(f.Path, []byte(f.Content), f.Mode); err != nil {
			return fmt.Errorf("write %s: %w", f.Path, err)
		}
		fmt.Println("wrote", f.Path)
	}
	if !*dryRun {
		fmt.Println()
		fmt.Print(EnableInstructions(InitSystem(*initSystem), *service))
	}
	return nil
}

func cmdUninstallService(args []string) error {
	fs := flag.NewFlagSet("uninstall-service", flag.ContinueOnError)
	initSystem := fs.String("init", "", "init system: systemd, openrc, runit, dinit")
	service := fs.String("service", brand.Slug, "service name used at install time")
	if err := fs.Parse(args); err != nil {
		return err
	}
	for _, p := range ServicePaths(InitSystem(*initSystem), *service) {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", p, err)
		}
		fmt.Println("removed", p)
	}
	return nil
}
