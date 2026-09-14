// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package desktop

import (
	"net/url"
	"strings"

	"melovian/internal/extensions"
)

// DeepLinkScheme is the URL scheme the desktop app registers for extension
// installs, e.g. melovian://install-extension/genre-palette.
const DeepLinkScheme = "melovian"

// ParseExtensionDeepLink extracts an extension id from a melovian:// URL.
// Accepted forms: melovian://install-extension/<id>,
// melovian://extension/<id>, melovian://extensions/<id>. Returns "" for
// anything else.
func ParseExtensionDeepLink(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != DeepLinkScheme {
		return ""
	}
	host := strings.ToLower(u.Host)
	path := strings.Trim(u.Path, "/")
	var id string
	switch host {
	case "install-extension", "extension", "extensions":
		id = path
	default:
		// melovian:///install-extension/<id> without a host.
		if strings.HasPrefix(strings.ToLower(path), "install-extension/") {
			id = path[len("install-extension/"):]
		}
	}
	id = strings.ToLower(strings.TrimSpace(id))
	if !extensions.IsValidExtensionID(id) {
		return ""
	}
	return id
}

// DeepLinkArgs scans argv for the first melovian:// extension deep link.
func DeepLinkArgs(args []string) string {
	for _, arg := range args {
		if id := ParseExtensionDeepLink(arg); id != "" {
			return id
		}
	}
	return ""
}
