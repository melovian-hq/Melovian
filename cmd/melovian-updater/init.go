// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"melovian/internal/brand"
)

// InitSystem is one of the supported service managers.
type InitSystem string

const (
	InitSystemd InitSystem = "systemd"
	InitOpenRC  InitSystem = "openrc"
	InitRunit   InitSystem = "runit"
	InitDinit   InitSystem = "dinit"
)

// ServiceSpec describes the periodic update job to install.
type ServiceSpec struct {
	ServiceName string // the melovian service to restart after an update
	BinaryPath  string // path of the server binary to replace
	UpdaterPath string // path of this tool
	Interval    string // "daily" (default), "weekly", "monthly", or a systemd OnCalendar expression
}

// UnitFile is one file the installer writes.
type UnitFile struct {
	Path    string
	Content string
	Mode    os.FileMode
}

const unitName = "melovian-updater"

// applyArgs is the command line each unit runs. The restart command is
// single-quoted so it survives being embedded inside sh -c strings.
func (s ServiceSpec) applyArgs(restart string) string {
	return fmt.Sprintf("apply --binary %s --restart '%s'", s.BinaryPath, restart)
}

// ServiceFiles renders every file needed to schedule updates on init.
func ServiceFiles(init InitSystem, spec ServiceSpec) ([]UnitFile, error) {
	switch init {
	case InitSystemd:
		return systemdFiles(spec), nil
	case InitOpenRC:
		return openrcFiles(spec), nil
	case InitRunit:
		return runitFiles(spec), nil
	case InitDinit:
		return dinitFiles(spec), nil
	}
	return nil, fmt.Errorf("unsupported init system %q (want systemd, openrc, runit, or dinit)", init)
}

// ServicePaths returns the files install-service would write, for removal.
func ServicePaths(init InitSystem, _ string) []string {
	name := unitName
	switch init {
	case InitSystemd:
		return []string{
			"/etc/systemd/system/" + name + ".service",
			"/etc/systemd/system/" + name + ".timer",
		}
	case InitOpenRC:
		return []string{"/etc/periodic/daily/" + name}
	case InitRunit:
		return []string{"/etc/sv/" + name + "/run", "/etc/sv/" + name + "/finish", "/etc/sv/" + name + "/log/run"}
	case InitDinit:
		return []string{"/etc/dinit.d/" + name}
	}
	return nil
}

// EnableInstructions prints the commands the operator runs after install.
func EnableInstructions(init InitSystem, _ string) string {
	switch init {
	case InitSystemd:
		return "To enable:\n" +
			"  systemctl daemon-reload\n" +
			"  systemctl enable --now " + unitName + ".timer\n"
	case InitOpenRC:
		return "The daily cron hook is active once cron runs. To trigger now:\n" +
			"  /etc/periodic/daily/" + unitName + "\n"
	case InitRunit:
		return "To enable:\n" +
			"  ln -s /etc/sv/" + unitName + " /var/service/   # or your service dir\n"
	case InitDinit:
		return "To enable:\n" +
			"  dinitctl enable " + unitName + "\n"
	}
	return ""
}

func systemdFiles(spec ServiceSpec) []UnitFile {
	calendar := spec.Interval
	switch calendar {
	case "", "daily":
		calendar = "daily"
	case "weekly", "monthly":
	default:
		// any other value is passed through as an OnCalendar expression
	}
	service := `[Unit]
Description=Melovian server self-update
Documentation=https://github.com/melovian-hq/Melovian
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=` + spec.UpdaterPath + ` ` + spec.applyArgs("systemctl restart "+spec.ServiceName) + `
Nice=19
IOSchedulingClass=idle
ProtectSystem=full
NoNewPrivileges=true
`
	timer := `[Unit]
Description=Check for Melovian updates

[Timer]
OnCalendar=` + calendar + `
Persistent=true

[Install]
WantedBy=timers.target
`
	return []UnitFile{
		{Path: "/etc/systemd/system/" + unitName + ".service", Content: service, Mode: 0o644},
		{Path: "/etc/systemd/system/" + unitName + ".timer", Content: timer, Mode: 0o644},
	}
}

func openrcFiles(spec ServiceSpec) []UnitFile {
	// OpenRC has no timer. Hook into the standard periodic cron directories.
	script := `#!/bin/sh
# Periodic ` + brand.Name + ` update check. Managed by ` + brand.Slug + `-updater install-service.
exec ` + spec.UpdaterPath + ` ` + spec.applyArgs("rc-service "+spec.ServiceName+" restart") + ` >>/var/log/` + unitName + `.log 2>&1
`
	return []UnitFile{
		{Path: "/etc/periodic/daily/" + unitName, Content: script, Mode: 0o755},
	}
}

func runitFiles(spec ServiceSpec) []UnitFile {
	// A runit service is a supervised loop: check, sleep, repeat.
	run := `#!/bin/sh
# ` + brand.Name + ` updater loop. Managed by ` + brand.Slug + `-updater install-service.
while true; do
  ` + spec.UpdaterPath + ` ` + spec.applyArgs("sv restart "+spec.ServiceName) + `
  sleep 86400
done
`
	finish := `#!/bin/sh
exit 0
`
	return []UnitFile{
		{Path: "/etc/sv/" + unitName + "/run", Content: run, Mode: 0o755},
		{Path: "/etc/sv/" + unitName + "/finish", Content: finish, Mode: 0o755},
	}
}

func dinitFiles(spec ServiceSpec) []UnitFile {
	// dinit runs a looping process service. log-type=file keeps output in
	// /var/log so journal-less systems still see progress.
	unit := `type            = process
command         = /bin/sh -c "while true; do ` + spec.UpdaterPath + ` ` + spec.applyArgs("dinitctl restart "+spec.ServiceName) + `; sleep 86400; done"
log-type        = file
logfile         = /var/log/` + unitName + `.log
restart         = true
smooth-recovery = true
`
	return []UnitFile{
		{Path: "/etc/dinit.d/" + unitName, Content: unit, Mode: 0o644},
	}
}
