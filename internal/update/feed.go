// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package update

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// feed is the minimal slice of an Atom document needed to enumerate
// release entries. GitHub publishes releases newest-first.
type feed struct {
	Entries []feedEntry `xml:"entry"`
}

type feedEntry struct {
	Title   string     `xml:"title"`
	ID      string     `xml:"id"`
	Updated time.Time  `xml:"updated"`
	Links   []feedLink `xml:"link"`
}

type feedLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

// tag extracts the release tag from an Atom entry. GitHub entry IDs look
// like "tag:github.com,2008:Repository/12345" and the alternate link ends
// with "/releases/tag/vX.Y.Z". The link is authoritative.
func (e feedEntry) tag() string {
	for _, l := range e.Links {
		if l.Rel != "" && l.Rel != "alternate" {
			continue
		}
		if i := strings.Index(l.Href, "/releases/tag/"); i >= 0 {
			return strings.Trim(l.Href[i+len("/releases/tag/"):], "/ \t")
		}
	}
	// Fall back to the title, which is the release name. Releases created
	// without a custom title carry the bare tag.
	t := strings.TrimSpace(e.Title)
	if _, err := ParseSemver(t); err == nil {
		return t
	}
	return ""
}

func (e feedEntry) notesURL() string {
	for _, l := range e.Links {
		if l.Rel == "" || l.Rel == "alternate" {
			return l.Href
		}
	}
	return ""
}

// fetchFeed downloads and parses the releases Atom feed.
func fetchFeed(ctx context.Context, client *http.Client, feedURL string) (*feed, error) {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/atom+xml")
	req.Header.Set("User-Agent", userAgent())
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch update feed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch update feed: %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read update feed: %w", err)
	}
	var f feed
	if err := xml.Unmarshal(body, &f); err != nil {
		return nil, fmt.Errorf("parse update feed: %w", err)
	}
	return &f, nil
}

// entriesToReleases converts feed entries to ReleaseInfo, dropping entries
// without a parseable tag.
func entriesToReleases(f *feed) []ReleaseInfo {
	out := make([]ReleaseInfo, 0, len(f.Entries))
	for _, e := range f.Entries {
		tag := e.tag()
		if tag == "" {
			continue
		}
		v, err := ParseSemver(tag)
		if err != nil {
			continue
		}
		out = append(out, ReleaseInfo{
			Version:     strings.TrimPrefix(tag, "v"),
			Tag:         tag,
			NotesURL:    e.notesURL(),
			PublishedAt: e.Updated,
			Prerelease:  v.IsPrerelease(),
		})
	}
	return out
}

// latestFromFeed returns the newest release acceptable for the channel.
// Feed order is newest-first but prerelease filtering may reorder, so the
// list is compared rather than truncated.
func latestFromFeed(f *feed, channel Channel) *ReleaseInfo {
	rels := entriesToReleases(f)
	var best *ReleaseInfo
	for i := range rels {
		r := rels[i]
		if channel != ChannelPrerelease && r.Prerelease {
			continue
		}
		if best == nil || Compare(r.Version, best.Version) > 0 {
			c := r
			best = &c
		}
	}
	return best
}

// findInFeed locates a specific version in the feed. The version may be
// given with or without the leading "v".
func findInFeed(f *feed, version string) *ReleaseInfo {
	want := strings.TrimPrefix(strings.TrimSpace(version), "v")
	for _, r := range entriesToReleases(f) {
		if r.Version == want {
			c := r
			return &c
		}
	}
	return nil
}

// Check queries the update feed and reports whether a newer release exists.
func Check(ctx context.Context, currentVersion string, opts Options) (*CheckResult, error) {
	report(opts.OnProgress, Progress{Stage: StageCheck, Message: "Checking for updates"})
	f, err := fetchFeed(ctx, opts.HTTPClient, FeedURL)
	if err != nil {
		return nil, err
	}
	var latest *ReleaseInfo
	if opts.TargetVersion != "" {
		latest = findInFeed(f, opts.TargetVersion)
		if latest == nil {
			return nil, fmt.Errorf("release %q not found in update feed", opts.TargetVersion)
		}
	} else {
		latest = latestFromFeed(f, opts.Channel)
	}
	res := &CheckResult{Current: currentVersion, CheckedAt: time.Now()}
	if latest == nil || !IsNewer(currentVersion, latest.Version) {
		res.UpToDate = true
		return res, nil
	}
	res.Latest = latest
	return res, nil
}

func userAgent() string {
	return "melovian-updater"
}
