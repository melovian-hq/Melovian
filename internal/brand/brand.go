// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Package brand holds the product identity in one place so downstreams can
// rebrand the app. Override at link time, for example:
// -X melovian/internal/brand.Name=Acme -X melovian/internal/brand.Slug=acme
//
// Name is the display name users see. Slug is the lowercase identifier used
// for binary names, temp dirs, and file names.
//
// Warning: changing Slug changes the extension manifest file name and other
// on-disk identifiers. Keep it stable unless the rebrand intends a clean
// break. The X-Melovian-* HTTP headers live in internal/compat and are part
// of the wire protocol, so they are not derived from Name.
package brand

var (
	// Name is the display name ("Melovian").
	Name = "Melovian"
	// Slug is the lowercase identifier ("melovian").
	Slug = "melovian"
	// Description is the one-line product blurb for meta tags and installers.
	Description = "Music player for local libraries and Subsonic-compatible servers"
	// OGImage is the default Open Graph image path on the site root.
	OGImage = "/og.png"
)

// ManifestName is the extension manifest file name.
func ManifestName() string {
	return Slug + "-extension.json"
}
