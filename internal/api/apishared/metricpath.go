// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package apishared

import "strings"

// MetricPath normalizes request paths into bounded metric label values.
func MetricPath(path string) string {
	switch {
	case path == "/health":
		return "/health"
	case path == "/metrics":
		return "/metrics"
	case path == "/api/debug/memory", path == "/api/debug/gc":
		return path
	case strings.HasPrefix(path, "/debug/"):
		return "/debug/"
	case strings.HasPrefix(path, "/api/subsonic"):
		return "/api/subsonic/"
	case strings.HasPrefix(path, "/api/"):
		return path
	default:
		return "other"
	}
}
