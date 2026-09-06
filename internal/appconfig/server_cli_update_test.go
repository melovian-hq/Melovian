// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package appconfig

import "testing"

func TestUpdateFlagBare(t *testing.T) {
	cli := NewServerCLI()
	if err := cli.Parse([]string{"--update"}); err != nil {
		t.Fatal(err)
	}
	if !cli.Update {
		t.Fatal("expected Update to be set")
	}
	if cli.UpdateVersion != "" {
		t.Fatalf("unexpected version %q", cli.UpdateVersion)
	}
}

func TestUpdateFlagWithEquals(t *testing.T) {
	cli := NewServerCLI()
	if err := cli.Parse([]string{"--update=v1.2.3"}); err != nil {
		t.Fatal(err)
	}
	if !cli.Update || cli.UpdateVersion != "1.2.3" {
		t.Fatalf("got update=%v version=%q", cli.Update, cli.UpdateVersion)
	}
}

func TestUpdateFlagWithPositionalVersion(t *testing.T) {
	cli := NewServerCLI()
	if err := cli.Parse([]string{"--update", "v0.5.0"}); err != nil {
		t.Fatal(err)
	}
	if !cli.Update || cli.UpdateVersion != "0.5.0" {
		t.Fatalf("got update=%v version=%q", cli.Update, cli.UpdateVersion)
	}
	if len(cli.Args) != 0 {
		t.Fatalf("leftover args: %v", cli.Args)
	}
}

func TestNoUpdateFlag(t *testing.T) {
	cli := NewServerCLI()
	if err := cli.Parse([]string{"--port", "9090"}); err != nil {
		t.Fatal(err)
	}
	if cli.Update {
		t.Fatal("update unexpectedly set")
	}
}
