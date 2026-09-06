// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package sandbox

import (
	"os"
	"testing"

	"melovian/internal/appconfig"
)

func TestApplyLinux(t *testing.T) {
	if os.Getenv("MELOVIAN_LANDLOCK_INTEGRATION") == "" {
		t.Skip("set MELOVIAN_LANDLOCK_INTEGRATION=1 to run in-process landlock apply test")
	}
	if !Enabled() {
		t.Skip("landlock disabled by MELOVIAN_LANDLOCK")
	}

	dataDir, err := os.MkdirTemp(".", "melovian-landlock-*")
	if err != nil {
		t.Fatalf("mkdir temp data dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dataDir)
	})

	cfg := appconfig.Config{
		DataDir:    dataDir,
		ServerMode: true,
	}

	status := Apply(cfg, ModeServer)
	t.Logf("landlock status: enabled=%v supported=%v reason=%q detail=%q", status.Enabled, status.Supported, status.Reason, status.Detail)
	if status.Reason != "" && !status.Enabled && !status.Supported {
		t.Skipf("landlock unavailable: %s", status.Reason)
	}
	if !status.Enabled {
		t.Fatalf("expected landlock enabled, got reason=%q", status.Reason)
	}
}
