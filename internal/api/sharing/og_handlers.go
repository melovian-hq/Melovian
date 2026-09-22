// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package sharing

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"
	"time"

	_ "golang.org/x/image/webp"

	"melovian/internal/api/realtime"
	"melovian/internal/brand"
	"melovian/internal/cache"
	"melovian/internal/httputil"
	"melovian/internal/ogimage"
	"melovian/internal/store"
)

const (
	ogShareTTL  = 10 * time.Minute
	ogListenTTL = 30 * time.Second
	// ogCoverSize asks upstream for artwork a bit larger than the card tile
	// so scaling stays sharp.
	ogCoverSize   = 600
	maxCoverBytes = 8 << 20
)

// SharePreview resolves the display title and a short subtitle for a share.
// Results are cached briefly because resolving a playlist title can hit the
// upstream server, and crawlers hit the meta path on every link unfurl.
func (h *Handler) SharePreview(share store.Share) (title, subtitle string) {
	key := share.Token + "|" + share.AccessMode
	h.previewMu.Lock()
	if entry, ok := h.previews[key]; ok && time.Now().Before(entry.expiresAt) {
		h.previewMu.Unlock()
		return entry.title, entry.subtitle
	}
	h.previewMu.Unlock()

	title, subtitle = h.resolveSharePreview(share)

	h.previewMu.Lock()
	if h.previews == nil {
		h.previews = make(map[string]sharePreviewEntry)
	}
	if len(h.previews) > 512 {
		h.previews = make(map[string]sharePreviewEntry)
	}
	h.previews[key] = sharePreviewEntry{title: title, subtitle: subtitle, expiresAt: time.Now().Add(5 * time.Minute)}
	h.previewMu.Unlock()
	return title, subtitle
}

type sharePreviewEntry struct {
	title     string
	subtitle  string
	expiresAt time.Time
}

func (h *Handler) resolveSharePreview(share store.Share) (title, subtitle string) {
	songs, resolved, err := h.shareSongsWithTitle(share)
	if err == nil {
		title = resolved
	}
	if title == "" {
		title = share.Description
	}
	switch share.ResourceType {
	case "playlist":
		if n := len(songs); n == 1 {
			subtitle = "1 track"
		} else if n > 1 {
			subtitle = fmt.Sprintf("%d tracks", n)
		}
	case "album":
		if len(songs) > 0 && songs[0].Artist != "" {
			subtitle = songs[0].Artist
		}
	default:
		if len(songs) > 0 {
			s := songs[0]
			switch {
			case s.Artist != "" && s.Album != "":
				subtitle = s.Artist + " · " + s.Album
			case s.Artist != "":
				subtitle = s.Artist
			case s.Album != "":
				subtitle = s.Album
			}
		}
	}
	return title, subtitle
}

// HandleOGImage serves generated Open Graph cards for public links:
// /og/share/{token}.png and /og/listen/{token}.png.
func (h *Handler) HandleOGImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		httputil.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/og/"), "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	token := strings.TrimSuffix(strings.TrimSpace(parts[1]), ".png")
	if token == "" || strings.ContainsAny(token, "/?#") {
		http.NotFound(w, r)
		return
	}
	switch parts[0] {
	case "share":
		h.serveShareOGImage(w, r, token)
	case "listen":
		h.serveListenOGImage(w, r, token)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) serveShareOGImage(w http.ResponseWriter, r *http.Request, token string) {
	share, err := h.shares.GetByToken(token)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if share.Expired() {
		httputil.WriteError(w, http.StatusGone, "share_expired", "share expired")
		return
	}
	cacheKey := "og:share:" + token + ":" + share.AccessMode
	if h.serveCachedOG(w, cacheKey) {
		return
	}

	spec := ogimage.Spec{
		Eyebrow: "SHARED MUSIC",
		Title:   "Shared music",
	}
	if share.AccessMode == store.ShareAccessPublic {
		title, subtitle := h.SharePreview(share)
		spec = ogimage.Spec{
			Eyebrow:  "SHARED " + shareEyebrowKind(share.ResourceType),
			Title:    title,
			Subtitle: subtitle,
			Cover:    h.shareCoverImage(share),
		}
	}
	h.writeOGImage(w, r, cacheKey, spec, ogShareTTL)
}

func shareEyebrowKind(resourceType string) string {
	switch resourceType {
	case "album":
		return "ALBUM"
	case "playlist":
		return "PLAYLIST"
	default:
		return "TRACK"
	}
}

func (h *Handler) serveListenOGImage(w http.ResponseWriter, r *http.Request, token string) {
	session, snap, ok := h.devices.SessionByInviteToken(token)
	if !ok {
		http.NotFound(w, r)
		return
	}
	trackKey := ""
	if snap != nil {
		trackKey = snap.TrackID
	}
	cacheKey := "og:listen:" + token + ":" + trackKey
	if h.serveCachedOG(w, cacheKey) {
		return
	}

	spec := ogimage.Spec{
		Eyebrow: "LISTEN TOGETHER",
		Title:   "Listen together",
	}
	if snap != nil && snap.TrackTitle != "" {
		spec.Title = snap.TrackTitle
		spec.Subtitle = snap.ArtistName
		spec.Cover = h.listenCoverImage(session, snap.CoverArt)
	}
	h.writeOGImage(w, r, cacheKey, spec, ogListenTTL)
}

func (h *Handler) serveCachedOG(w http.ResponseWriter, key string) bool {
	entry, ok := h.ogCache.Get(key)
	if !ok {
		return false
	}
	cache.WriteCachedResponse(w, entry, "HIT")
	return true
}

func (h *Handler) writeOGImage(w http.ResponseWriter, r *http.Request, cacheKey string, spec ogimage.Spec, ttl time.Duration) {
	body, err := ogimage.Render(spec, brand.Name)
	if err != nil {
		httputil.WriteInternalError(w, r, "ogimage.Render", err)
		return
	}
	maxAge := int(ttl / time.Second)
	entry := cache.Entry{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type":  {"image/png"},
			"Cache-Control": {fmt.Sprintf("public, max-age=%d", maxAge)},
		},
		Body:      body,
		ExpiresAt: time.Now().Add(ttl),
	}
	h.ogCache.Set(cacheKey, entry)
	cache.WriteCachedResponse(w, entry, "MISS")
}

// shareCoverImage fetches and decodes the share's artwork. Returns nil when
// no cover exists so the card falls back to the branded tile.
func (h *Handler) shareCoverImage(share store.Share) image.Image {
	data, _, ok := h.shareCoverData(share, share.ResourceID, ogCoverSize)
	if !ok {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return img
}

// listenCoverImage fetches artwork for the party's current track. The
// session track allowlist gates which ids resolve, same as the member
// cover endpoint.
func (h *Handler) listenCoverImage(session realtime.ListenSession, coverID string) image.Image {
	coverID = strings.TrimSpace(coverID)
	if coverID == "" || !h.devices.SessionAllowsTrack(session.ID, coverID) {
		return nil
	}
	data, _, ok := h.listenCoverData(session, coverID, ogCoverSize)
	if !ok {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return img
}

func (h *Handler) listenCoverData(session realtime.ListenSession, coverID string, size int) ([]byte, string, bool) {
	if strings.HasPrefix(coverID, "trk_") || strings.HasPrefix(coverID, "alb_") {
		catalog, err := h.library.LocalCatalogForUser(session.HostUserID)
		if err == nil {
			if data, mime, ok := h.partyLocalCover(session.HostUserID, catalog, coverID); ok {
				return data, mime, true
			}
		}
	}
	client := h.subsonicClientForPartyHost(session.HostUserID)
	if !client.Enabled() {
		return nil, "", false
	}
	body, contentType, err := client.CoverArt(coverID, size)
	if err != nil {
		return nil, "", false
	}
	defer func() { _ = body.Close() }()
	data, err := io.ReadAll(io.LimitReader(body, maxCoverBytes))
	if err != nil {
		return nil, "", false
	}
	return data, contentType, true
}

// shareCoverData returns the cover bytes and mime type for a share.
// coverID must already have passed ShareCoverAllowed.
func (h *Handler) shareCoverData(share store.Share, coverID string, size int) ([]byte, string, bool) {
	if strings.HasPrefix(coverID, "trk_") || strings.HasPrefix(coverID, "alb_") || strings.HasPrefix(share.ResourceID, "pl_") {
		catalog, err := h.localCatalogForShare(share)
		if err == nil {
			if data, mime, ok := h.localCoverDataForOwner(share, catalog, coverID); ok {
				return data, mime, true
			}
		}
	}
	client := h.subsonicClientForShare(share)
	if !client.Enabled() {
		return nil, "", false
	}
	body, contentType, err := client.CoverArt(coverID, size)
	if err != nil {
		return nil, "", false
	}
	defer func() { _ = body.Close() }()
	data, err := io.ReadAll(io.LimitReader(body, maxCoverBytes))
	if err != nil {
		return nil, "", false
	}
	return data, contentType, true
}
