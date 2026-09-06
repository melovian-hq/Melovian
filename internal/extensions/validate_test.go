// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import "testing"

func TestValidateScriptAllowsSimpleHook(t *testing.T) {
	source := `function register(api) {
  api.onTrack((track) => ({ accentColor: "#336699" }));
}`
	if err := ValidateScript(source); err != nil {
		t.Fatalf("expected allowed script, got %v", err)
	}
}

func TestValidateScriptBlocksFetch(t *testing.T) {
	source := `function register(api) { fetch("/secret"); }`
	if err := ValidateScript(source); err == nil {
		t.Fatal("expected blocked script")
	}
}

func TestSanitizeStyles(t *testing.T) {
	got := SanitizeStyles([]string{
		"assets/theme-app.css",
		"../escape.css",
		"/abs.css",
		"assets/theme-app.css",
		"assets/theme-player.css",
	})
	if len(got) != 2 {
		t.Fatalf("SanitizeStyles = %#v, want 2 safe paths", got)
	}
	if got[0] != "assets/theme-app.css" || got[1] != "assets/theme-player.css" {
		t.Fatalf("SanitizeStyles = %#v", got)
	}
}
