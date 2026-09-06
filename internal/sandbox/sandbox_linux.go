// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package sandbox

import (
	"fmt"

	"github.com/landlock-lsm/go-landlock/landlock"
	"melovian/internal/appconfig"
)

func Apply(cfg appconfig.Config, mode Mode) Status {
	if !Enabled() {
		return Status{Reason: "disabled by MELOVIAN_LANDLOCK"}
	}

	entries, err := collectPathEntries(cfg, mode)
	if err != nil {
		return Status{Supported: true, Reason: err.Error()}
	}
	if len(entries) == 0 {
		return Status{Supported: true, Reason: "no filesystem paths to restrict"}
	}

	rules, err := landlockRulesFromEntries(entries)
	if err != nil {
		return Status{Supported: true, Reason: err.Error()}
	}

	cfgLL := landlock.V9.BestEffort()
	if err := cfgLL.RestrictPaths(rules...); err != nil {
		return Status{Supported: true, Reason: err.Error()}
	}
	if mode == ModeServer {
		_ = cfgLL.RestrictScoped()
	}

	return Status{
		Enabled:   true,
		Supported: true,
		Detail:    fmt.Sprintf("ABI V9 BestEffort, %d path rules", len(rules)),
	}
}

func landlockRulesFromEntries(entries []pathEntry) ([]landlock.Rule, error) {
	var rwPaths []string
	var roPaths []string
	var resolveUnixPaths []string

	for _, entry := range entries {
		if entry.readWrite {
			rwPaths = append(rwPaths, entry.path)
			continue
		}
		if entry.resolveUnix {
			resolveUnixPaths = append(resolveUnixPaths, entry.path)
			continue
		}
		roPaths = append(roPaths, entry.path)
	}

	rules := make([]landlock.Rule, 0, 3)
	if len(rwPaths) > 0 {
		rules = append(rules, landlock.RWDirs(rwPaths...))
	}
	if len(roPaths) > 0 {
		rules = append(rules, landlock.RODirs(roPaths...).IgnoreIfMissing())
	}
	if len(resolveUnixPaths) > 0 {
		rules = append(rules, landlock.RODirs(resolveUnixPaths...).WithResolveUnix().IgnoreIfMissing())
	}
	if len(rules) == 0 {
		return nil, fmt.Errorf("no landlock rules generated")
	}
	return rules, nil
}
