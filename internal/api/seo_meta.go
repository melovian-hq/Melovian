// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"strings"

	"melovian/internal/brand"
	"melovian/internal/ogimage"
	"melovian/internal/seo"
	"melovian/internal/store"
)

// PageMeta upgrades generic route meta with entity data for the public link
// surfaces: /share/{token} and /listen/{token}. Library routes behind the
// login wall keep their static meta so nothing about private collections
// leaks into crawler-visible HTML.
func (s *Server) PageMeta(r *http.Request, path string, fallback seo.Page) seo.Page {
	switch {
	case strings.HasPrefix(path, "/share/"):
		return s.sharePageMeta(path, fallback)
	case strings.HasPrefix(path, "/listen/"):
		return s.listenPageMeta(path, fallback)
	default:
		return fallback
	}
}

func pathToken(path, prefix string) string {
	token := strings.TrimSuffix(strings.TrimPrefix(path, prefix), "/")
	if token == "" || strings.ContainsAny(token, "/?#") {
		return ""
	}
	return token
}

func (s *Server) sharePageMeta(path string, fallback seo.Page) seo.Page {
	token := pathToken(path, "/share/")
	if token == "" {
		return fallback
	}
	share, err := s.shares.GetByToken(token)
	if err != nil || share.Expired() {
		fallback.Index = false
		return fallback
	}
	if share.AccessMode != store.ShareAccessPublic {
		// Gated shares get a generic card. The title and artwork belong to
		// the owner until the visitor passes the password or login check.
		return seo.Page{
			Title:       "Shared music",
			Description: "Open music shared with you on " + brand.Name + ".",
			Index:       false,
		}
	}
	title, subtitle := s.sharingH.SharePreview(share)
	if title == "" {
		return fallback
	}
	desc := "Listen on " + brand.Name + "."
	if subtitle != "" {
		desc = subtitle + " - Listen on " + brand.Name + "."
	}
	return seo.Page{
		Title:       title,
		Description: desc,
		Image:       "/og/share/" + token + ".png",
		ImageWidth:  ogimage.Width,
		ImageHeight: ogimage.Height,
		Type:        "music." + shareOGType(share.ResourceType),
		Index:       true,
	}
}

func shareOGType(resourceType string) string {
	switch resourceType {
	case "album":
		return "album"
	case "playlist":
		return "playlist"
	default:
		return "song"
	}
}

func (s *Server) listenPageMeta(path string, fallback seo.Page) seo.Page {
	token := pathToken(path, "/listen/")
	if token == "" {
		return fallback
	}
	_, snap, ok := s.devices.SessionByInviteToken(token)
	if !ok {
		fallback.Index = false
		return fallback
	}
	page := seo.Page{
		Title:       "Listen together",
		Description: "Join a live listen-together session on " + brand.Name + ".",
		Image:       "/og/listen/" + token + ".png",
		ImageWidth:  ogimage.Width,
		ImageHeight: ogimage.Height,
		// Invite tokens are credentials for a live session. Rich previews
		// help unfurlers, but the links should not be indexed.
		Index: false,
	}
	if snap != nil && snap.TrackTitle != "" {
		desc := "Now playing: " + snap.TrackTitle
		if snap.ArtistName != "" {
			desc += " - " + snap.ArtistName
		}
		page.Description = desc
	}
	return page
}
