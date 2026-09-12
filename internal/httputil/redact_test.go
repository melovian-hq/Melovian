// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"errors"
	"net/url"
	"strings"
	"testing"
)

func TestSanitizeErrorURLStripsQueryAndUserinfo(t *testing.T) {
	orig := &url.Error{
		Op:  "Get",
		URL: "https://user:pass@example.com/rest/ping.view?u=a&t=deadbeef&s=salt",
		Err: errors.New("connection refused"),
	}
	out := SanitizeErrorURL(orig)
	msg := out.Error()
	for _, secret := range []string{"deadbeef", "u=a", "user:pass", "salt"} {
		if strings.Contains(msg, secret) {
			t.Fatalf("sanitized error still contains %q: %s", secret, msg)
		}
	}
	if !strings.Contains(msg, "example.com") {
		t.Fatalf("expected host to remain for debugging, got %s", msg)
	}
	var uerr *url.Error
	if !errors.As(out, &uerr) {
		t.Fatal("sanitized error must remain a *url.Error")
	}
}

func TestSanitizeErrorURLPassthrough(t *testing.T) {
	orig := errors.New("plain failure")
	if got := SanitizeErrorURL(orig); got != orig {
		t.Fatal("non url errors must pass through unchanged")
	}
}

func TestRedactURLUserinfo(t *testing.T) {
	got := RedactURLUserinfo("https://user:pass@host.example:4533/app")
	if strings.Contains(got, "pass") {
		t.Fatalf("userinfo still present: %s", got)
	}
	if got != "https://host.example:4533/app" {
		t.Fatalf("unexpected redaction: %s", got)
	}
	if got := RedactURLUserinfo("https://host.example"); got != "https://host.example" {
		t.Fatalf("clean url changed: %s", got)
	}
	if got := RedactURLUserinfo("not a url :::"); got == "" {
		t.Fatal("unparseable input must pass through")
	}
}
