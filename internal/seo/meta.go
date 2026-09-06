// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Package seo builds per-route HTML meta for the SPA shell.
package seo

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"melovian/internal/brand"
)

// Page holds title and description for a route.
type Page struct {
	Title       string
	Description string
	Image       string
	Type        string
	Index       bool
}

type routeSpec struct {
	pattern     *regexp.Regexp
	title       string
	description string
	index       bool
}

var routes = []routeSpec{
	{regexp.MustCompile(`^/account/login/?$`), "Sign in", "Sign in to your Melovian account.", true},
	{regexp.MustCompile(`^/setup/?$`), "Set up music source", "Connect a Subsonic server or local music folder.", true},
	{regexp.MustCompile(`^/share/[^/]+/?$`), "Shared playlist", "Open a playlist shared with you.", true},
	{regexp.MustCompile(`^/listen/[^/]+/?$`), "Join listen together", "Join a listen-together session.", true},
	{regexp.MustCompile(`^/music/search/?$`), "Search", "Search your music library.", true},
	{regexp.MustCompile(`^/music/artists/?$`), "Artists", "Browse artists in your library.", true},
	{regexp.MustCompile(`^/music/albums/?$`), "Albums", "Browse albums in your library.", true},
	{regexp.MustCompile(`^/music/genres/?$`), "Genres", "Browse genres in your library.", true},
	{regexp.MustCompile(`^/music/genre/[^/]+/?$`), "Genre", "Browse tracks in this genre.", true},
	{regexp.MustCompile(`^/music/album/[^/]+/?$`), "Album", "Album details and track list.", true},
	{regexp.MustCompile(`^/music/artist/[^/]+/?$`), "Artist", "Artist discography and related artists.", true},
	{regexp.MustCompile(`^/music/mix/[^/]+/?$`), "Mix", "A personalized Melovian mix.", true},
	{regexp.MustCompile(`^/music/playlists/?$`), "Playlists", "Your playlists and smart playlists.", true},
	{regexp.MustCompile(`^/music/shared/?$`), "Shared with you", "Playlists others have shared with you.", true},
	{regexp.MustCompile(`^/music/favorites/?$`), "Favorites", "Starred tracks, albums, and artists.", true},
	{regexp.MustCompile(`^/music/history/?$`), "History", "Recently played tracks.", true},
	{regexp.MustCompile(`^/music/videos/?$`), "Videos", "Music videos linked to your library.", true},
	{regexp.MustCompile(`^/play/[^/]+/?$`), "Video", "Watch a music video.", true},
	{regexp.MustCompile(`^/music/now-playing/?$`), "Now playing", "Full-screen now playing view.", true},
	{regexp.MustCompile(`^/music/lyrics/?$`), "Lyrics", "Synced and plain lyrics for the current track.", true},
	{regexp.MustCompile(`^/music/server-playlist/[^/]+/?$`), "Server playlist", "A playlist from your music server.", true},
	{regexp.MustCompile(`^/music/playlist/[^/]+/?$`), "Playlist", "Playlist tracks and editing.", true},
	{regexp.MustCompile(`^/music/metadata/?$`), "Metadata editor", "Edit track and album metadata.", false},
	{regexp.MustCompile(`^/settings(?:/[^/]+)?/?$`), "Settings", "App, library, and playback settings.", false},
	{regexp.MustCompile(`^/instances/?$`), "Settings", "Manage music sources and servers.", false},
	{regexp.MustCompile(`^/(?:music|login)?/?$`), "Music", "Home for your local and Subsonic libraries.", true},
}

// ForPath returns meta for a request path. Unknown paths use brand defaults.
func ForPath(path string) Page {
	clean := path
	if i := strings.IndexByte(clean, '?'); i >= 0 {
		clean = clean[:i]
	}
	if clean == "" {
		clean = "/"
	}
	page := Page{
		Title:       brand.Name,
		Description: brand.Description,
		Image:       brand.OGImage,
		Type:        "website",
		Index:       true,
	}
	for _, route := range routes {
		if route.pattern.MatchString(clean) {
			page.Title = route.title
			page.Description = strings.ReplaceAll(route.description, "Melovian", brand.Name)
			page.Index = route.index
			break
		}
	}
	return page
}

var (
	titleRE     = regexp.MustCompile(`(?i)<title>[^<]*</title>`)
	metaNameRE  = regexp.MustCompile(`(?i)<meta\s+[^>]*name=["']([^"']+)["'][^>]*>`)
	metaPropRE  = regexp.MustCompile(`(?i)<meta\s+[^>]*property=["']([^"']+)["'][^>]*>`)
	canonicalRE = regexp.MustCompile(`(?i)<link\s+[^>]*rel=["']canonical["'][^>]*>`)
	headCloseRE = regexp.MustCompile(`(?i)</head>`)
)

// Inject rewrites title and upserts description / Open Graph / Twitter tags.
func Inject(raw []byte, page Page, baseURL, path string) []byte {
	title := documentTitle(page.Title)
	desc := page.Description
	if desc == "" {
		desc = brand.Description
	}
	image := page.Image
	if image == "" {
		image = brand.OGImage
	}
	imageURL := absURL(baseURL, image)
	pageURL := absURL(baseURL, path)
	ogType := page.Type
	if ogType == "" {
		ogType = "website"
	}
	robots := "index, follow"
	if !page.Index {
		robots = "noindex, nofollow"
	}

	htmlStr := string(raw)
	titleTag := "<title>" + html.EscapeString(title) + "</title>"
	if titleRE.MatchString(htmlStr) {
		htmlStr = titleRE.ReplaceAllString(htmlStr, titleTag)
	}

	tags := map[string]string{
		"description":         desc,
		"robots":              robots,
		"twitter:card":        "summary_large_image",
		"twitter:title":       title,
		"twitter:description": desc,
		"twitter:image":       imageURL,
	}
	props := map[string]string{
		"og:type":        ogType,
		"og:site_name":   brand.Name,
		"og:title":       title,
		"og:description": desc,
		"og:url":         pageURL,
		"og:image":       imageURL,
	}

	htmlStr = upsertMeta(htmlStr, "name", tags)
	htmlStr = upsertMeta(htmlStr, "property", props)
	canonical := fmt.Sprintf(`<link rel="canonical" href="%s" />`, html.EscapeString(pageURL))
	if canonicalRE.MatchString(htmlStr) {
		htmlStr = canonicalRE.ReplaceAllString(htmlStr, canonical)
	} else {
		htmlStr = insertBeforeHeadClose(htmlStr, canonical)
	}

	return []byte(htmlStr)
}

func documentTitle(pageTitle string) string {
	trimmed := strings.TrimSpace(pageTitle)
	if trimmed == "" || trimmed == brand.Name {
		return brand.Name
	}
	suffix := " · " + brand.Name
	if strings.HasSuffix(trimmed, suffix) || strings.HasSuffix(trimmed, " - "+brand.Name) {
		return trimmed
	}
	return trimmed + suffix
}

func absURL(base, pathOrURL string) string {
	if strings.HasPrefix(pathOrURL, "http://") || strings.HasPrefix(pathOrURL, "https://") {
		return pathOrURL
	}
	base = strings.TrimRight(base, "/")
	if pathOrURL == "" {
		pathOrURL = "/"
	}
	if !strings.HasPrefix(pathOrURL, "/") {
		pathOrURL = "/" + pathOrURL
	}
	if base == "" {
		return pathOrURL
	}
	return base + pathOrURL
}

func upsertMeta(htmlStr, attr string, values map[string]string) string {
	re := metaNameRE
	if attr == "property" {
		re = metaPropRE
	}
	present := map[string]bool{}
	htmlStr = re.ReplaceAllStringFunc(htmlStr, func(tag string) string {
		m := re.FindStringSubmatch(tag)
		if len(m) < 2 {
			return tag
		}
		key := strings.ToLower(m[1])
		val, ok := values[key]
		if !ok {
			return tag
		}
		present[key] = true
		return fmt.Sprintf(`<meta %s="%s" content="%s" />`, attr, html.EscapeString(key), html.EscapeString(val))
	})
	var missing strings.Builder
	for key, val := range values {
		if present[key] {
			continue
		}
		fmt.Fprintf(&missing, `  <meta %s="%s" content="%s" />`+"\n", attr, html.EscapeString(key), html.EscapeString(val))
	}
	if missing.Len() > 0 {
		htmlStr = insertBeforeHeadClose(htmlStr, strings.TrimRight(missing.String(), "\n"))
	}
	return htmlStr
}

func insertBeforeHeadClose(htmlStr, snippet string) string {
	if headCloseRE.MatchString(htmlStr) {
		return headCloseRE.ReplaceAllString(htmlStr, snippet+"\n  </head>")
	}
	return htmlStr + "\n" + snippet
}
