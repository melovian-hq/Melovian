// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package lastfm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestErrorDoesNotLeakSignedURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{`))
	}))
	t.Cleanup(ts.Close)

	c := NewClient(ts.Client())
	c.BaseURL = ts.URL

	err := c.Scrobble(context.Background(), "apikey123", "secret", "sessionkey456",
		Track{Title: "t", Artist: "a"}, time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
	for _, leaked := range []string{"apikey123", "sessionkey456", "api_sig"} {
		if strings.Contains(err.Error(), leaked) {
			t.Fatalf("error leaks signed URL material %q: %v", leaked, err)
		}
	}
}

func TestValidateTokenErrorDoesNotLeakURL(t *testing.T) {
	// A failing transport produces a *url.Error whose message embeds the
	// signed request URL. It must be scrubbed before it reaches callers.
	c := NewClient(&http.Client{Transport: failingTransport{}})
	c.BaseURL = "https://ws.audioscrobbler.test"

	_, err := c.ValidateToken(context.Background(), "apikey123", "secret", "sessionkey456")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, leaked := range []string{"apikey123", "sessionkey456", "api_sig"} {
		if strings.Contains(err.Error(), leaked) {
			t.Fatalf("error leaks signed URL material %q: %v", leaked, err)
		}
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("dial failed")
}
