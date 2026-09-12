// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package compat

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var (
	tsStringConstRE = regexp.MustCompile(`export const ([A-Z0-9_]+)\s*=\s*"([^"]*)"`)
	tsNumberConstRE = regexp.MustCompile(`export const ([A-Z0-9_]+)\s*=\s*(\d+)`)
	tsCapBlockRE    = regexp.MustCompile(`(?s)export const Cap = \{(.*?)\} as const`)
	tsCapEntryRE    = regexp.MustCompile(`(\w+)\s*:\s*"([^"]+)"`)
	goCapConstRE    = regexp.MustCompile(`\b(Cap\w+)\s*=\s*"([^"]+)"`)
)

func readFrontendCompatSource(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "..", "frontend", "src", "lib", "compat", "index.ts")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func parseTSConsts(source string) map[string]string {
	consts := make(map[string]string)
	for _, m := range tsStringConstRE.FindAllStringSubmatch(source, -1) {
		consts[m[1]] = m[2]
	}
	for _, m := range tsNumberConstRE.FindAllStringSubmatch(source, -1) {
		consts[m[1]] = m[2]
	}
	return consts
}

func parseTSCaps(t *testing.T, source string) map[string]string {
	t.Helper()
	block := tsCapBlockRE.FindStringSubmatch(source)
	if block == nil {
		t.Fatal("index.ts is missing the export const Cap block")
	}
	caps := make(map[string]string)
	for _, m := range tsCapEntryRE.FindAllStringSubmatch(block[1], -1) {
		caps[m[1]] = m[2]
	}
	return caps
}

func parseGoCaps(t *testing.T) map[string]string {
	t.Helper()
	data, err := os.ReadFile("compat.go")
	if err != nil {
		t.Fatalf("read compat.go: %v", err)
	}
	caps := make(map[string]string)
	for _, m := range goCapConstRE.FindAllStringSubmatch(string(data), -1) {
		caps[m[1]] = m[2]
	}
	if len(caps) == 0 {
		t.Fatal("no Cap* constants found in compat.go")
	}
	return caps
}

// tsCapKey converts a Go Cap constant name to the Cap key used in index.ts.
// CapPersonalRadio becomes personalRadio and all caps names like CapWS become ws.
func tsCapKey(goName string) string {
	rest := strings.TrimPrefix(goName, "Cap")
	if rest == strings.ToUpper(rest) {
		return strings.ToLower(rest)
	}
	return strings.ToLower(rest[:1]) + rest[1:]
}

// expectedTSConsts lists every literal export index.ts must mirror. Values come
// from the compiled constants so the check compares real wire values.
// MinClientVersion is enforced server side only and has no index.ts export.
func expectedTSConsts() map[string]string {
	return map[string]string{
		"HEADER_CLIENT_VERSION": HeaderClientVersion,
		"HEADER_API_VERSION":    HeaderAPIVersion,
		"HEADER_CAPABILITIES":   HeaderCapabilities,
		"HEADER_SERVER_VERSION": HeaderServerVersion,
		"MIN_SERVER_VERSION":    MinServerVersion,
		"API_VERSION":           strconv.Itoa(APIVersion),
	}
}

func TestCompatContractMatchesFrontend(t *testing.T) {
	tsSource := readFrontendCompatSource(t)
	tsConsts := parseTSConsts(tsSource)
	tsCaps := parseTSCaps(t, tsSource)
	goCaps := parseGoCaps(t)
	expected := expectedTSConsts()

	var problems []string

	for name, want := range expected {
		got, ok := tsConsts[name]
		if !ok {
			problems = append(problems, fmt.Sprintf("index.ts is missing export const %s", name))
			continue
		}
		if got != want {
			problems = append(problems, fmt.Sprintf("index.ts %s = %q but compat.go has %q", name, got, want))
		}
	}
	for name, value := range tsConsts {
		if _, ok := expected[name]; !ok {
			problems = append(problems, fmt.Sprintf("index.ts exports %s = %q with no compat.go counterpart", name, value))
		}
	}

	goCapKeys := make(map[string]string, len(goCaps))
	for goName, goValue := range goCaps {
		goCapKeys[tsCapKey(goName)] = goValue
	}
	for key, want := range goCapKeys {
		got, ok := tsCaps[key]
		if !ok {
			problems = append(problems, fmt.Sprintf("index.ts Cap is missing %s (compat.go value %q)", key, want))
			continue
		}
		if got != want {
			problems = append(problems, fmt.Sprintf("index.ts Cap.%s = %q but compat.go has %q", key, got, want))
		}
	}
	for key, value := range tsCaps {
		if _, ok := goCapKeys[key]; !ok {
			problems = append(problems, fmt.Sprintf("index.ts Cap.%s = %q has no compat.go counterpart", key, value))
		}
	}

	if len(problems) == 0 {
		return
	}
	sort.Strings(problems)
	var b strings.Builder
	b.WriteString("compat contract drift between internal/compat/compat.go and frontend/src/lib/compat/index.ts:\n")
	for _, p := range problems {
		b.WriteString("  - " + p + "\n")
	}
	b.WriteString("Update both files so the X-Melovian wire contract stays in sync.")
	t.Fatal(b.String())
}
