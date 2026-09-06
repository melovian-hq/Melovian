// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const maxScriptBytes = 32 * 1024

var blockedScriptPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bimport\b`),
	regexp.MustCompile(`(?i)\brequire\s*\(`),
	regexp.MustCompile(`(?i)\bfetch\s*\(`),
	regexp.MustCompile(`(?i)\bXMLHttpRequest\b`),
	regexp.MustCompile(`(?i)\beval\s*\(`),
	regexp.MustCompile(`(?i)\bFunction\s*\(`),
	regexp.MustCompile(`(?i)\bwindow\b`),
	regexp.MustCompile(`(?i)\bdocument\b`),
	regexp.MustCompile(`(?i)\blocalStorage\b`),
	regexp.MustCompile(`(?i)\bsessionStorage\b`),
	regexp.MustCompile(`(?i)\b__proto__\b`),
	regexp.MustCompile(`(?i)\bconstructor\b`),
	regexp.MustCompile(`(?i)\bprocess\b`),
	regexp.MustCompile(`(?i)\bglobalThis\b`),
}

func ValidateScript(source string) error {
	source = strings.TrimSpace(source)
	if source == "" {
		return fmt.Errorf("empty script")
	}
	if len(source) > maxScriptBytes {
		return fmt.Errorf("script exceeds %d bytes", maxScriptBytes)
	}
	for _, pattern := range blockedScriptPatterns {
		if pattern.MatchString(source) {
			return fmt.Errorf("script contains blocked pattern: %s", pattern.String())
		}
	}
	return nil
}

func ScriptAllowed(source string) bool {
	return ValidateScript(source) == nil
}

// IsSafeRelPath reports whether rel is a relative path without traversal.
func IsSafeRelPath(rel string) bool {
	rel = strings.TrimSpace(rel)
	rel = strings.ReplaceAll(rel, "\\", "/")
	if rel == "" || strings.HasPrefix(rel, "/") || strings.Contains(rel, "..") {
		return false
	}
	if filepath.IsAbs(rel) {
		return false
	}
	return true
}

// SanitizeStyles returns only safe relative stylesheet paths from the manifest.
func SanitizeStyles(styles []string) []string {
	if len(styles) == 0 {
		return nil
	}
	out := make([]string, 0, len(styles))
	seen := map[string]bool{}
	for _, style := range styles {
		rel := strings.TrimSpace(style)
		rel = strings.ReplaceAll(rel, "\\", "/")
		if !IsSafeRelPath(rel) || seen[rel] {
			continue
		}
		seen[rel] = true
		out = append(out, rel)
	}
	return out
}
