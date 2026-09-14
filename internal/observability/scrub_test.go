// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package observability

import (
	"strings"
	"testing"

	"github.com/getsentry/sentry-go"
)

func TestScrubURLRedactsSubsonicAuthParams(t *testing.T) {
	raw := "https://music.example/rest/getCoverArt?u=alice&p=hunter2&t=tok&s=salt&v=1.16.1&c=melovian&id=song-9"
	got := ScrubURL(raw)
	for _, secret := range []string{"alice", "hunter2", "tok", "salt"} {
		if strings.Contains(got, secret) {
			t.Fatalf("url still contains %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, "id=song-9") {
		t.Fatalf("safe param dropped: %s", got)
	}
	if !strings.Contains(got, "u=%5Bredacted%5D") {
		t.Fatalf("param not redacted: %s", got)
	}
}

func TestScrubURLDropsUserinfoAndFragment(t *testing.T) {
	got := ScrubURL("https://user:pass@example.com/path?x=1#frag")
	if strings.Contains(got, "user") || strings.Contains(got, "pass") {
		t.Fatalf("userinfo leaked: %s", got)
	}
	if strings.Contains(got, "frag") {
		t.Fatalf("fragment leaked: %s", got)
	}
}

func TestScrubURLHandlesEmptyAndBroken(t *testing.T) {
	if ScrubURL("") != "" {
		t.Fatal("expected empty for empty input")
	}
	if ScrubURL("http://[::1") != "" {
		t.Fatal("expected empty for unparseable url")
	}
}

func TestScrubEventStripsRequestSecrets(t *testing.T) {
	event := &sentry.Event{
		Request: &sentry.Request{
			URL:         "https://music.example/rest/stream?u=alice&t=tok&id=1",
			QueryString: "u=alice&t=tok&id=1",
			Cookies:     "session=abc",
			Data:        `{"password":"hunter2"}`,
			Headers: map[string]string{
				"Authorization": "Bearer abc",
				"Cookie":        "session=abc",
				"Content-Type":  "application/json",
			},
		},
		Breadcrumbs: []*sentry.Breadcrumb{
			{Data: map[string]any{"url": "https://x.example/rest/ping?u=alice&p=hunter2"}},
		},
		User: sentry.User{Username: "alice", Email: "a@b.c"},
	}
	got := ScrubEvent(event, nil)
	if strings.Contains(got.Request.URL, "alice") || strings.Contains(got.Request.URL, "tok") {
		t.Fatalf("request url leaked: %s", got.Request.URL)
	}
	if got.Request.QueryString != "" || got.Request.Cookies != "" || got.Request.Data != "" {
		t.Fatal("request secrets not cleared")
	}
	if _, ok := got.Request.Headers["Authorization"]; ok {
		t.Fatal("authorization header leaked")
	}
	if _, ok := got.Request.Headers["Cookie"]; ok {
		t.Fatal("cookie header leaked")
	}
	if got.Request.Headers["Content-Type"] != "application/json" {
		t.Fatal("safe header dropped")
	}
	crumbURL, _ := got.Breadcrumbs[0].Data["url"].(string)
	if strings.Contains(crumbURL, "alice") {
		t.Fatalf("breadcrumb url leaked: %s", crumbURL)
	}
	if got.User.Username != "" || got.User.Email != "" {
		t.Fatal("user pii leaked")
	}
}

func TestScrubBreadcrumbRedactsURLData(t *testing.T) {
	crumb := &sentry.Breadcrumb{
		Data: map[string]any{"url": "https://x.example/a?t=tok&keep=1"},
	}
	got := ScrubBreadcrumb(crumb, nil)
	url, _ := got.Data["url"].(string)
	if strings.Contains(url, "tok") || !strings.Contains(url, "keep=1") {
		t.Fatalf("breadcrumb url not scrubbed: %s", url)
	}
}
