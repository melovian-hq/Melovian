// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package navidrome

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func FuzzIsNavidromeServer(f *testing.F) {
	f.Add("Navidrome", "0.58.0")
	f.Add("navidrome", "0.54.0")
	f.Add("nd", "0.58.0")
	f.Add("navidrome/home", "0.58.0")
	f.Add("Subsonic", "1.16.1")
	f.Add("Airsonic", "10.8.0")
	f.Add("Home", "0.58.0")
	f.Add("", "")
	f.Add("My Music", "navidrome-0.54.5")

	f.Fuzz(func(t *testing.T, serverName, version string) {
		a := IsNavidromeServer(serverName, version)
		b := IsNavidromeServer(serverName, version)
		if a != b {
			t.Fatal("IsNavidromeServer must be deterministic")
		}
		haystack := strings.ToLower(strings.TrimSpace(serverName) + " " + strings.TrimSpace(version))
		if strings.Contains(haystack, "navidrome") && !a {
			t.Fatalf("expected navidrome for %q %q", serverName, version)
		}
		name := strings.ToLower(strings.TrimSpace(serverName))
		if (name == "nd" || strings.HasPrefix(name, "navidrome/")) && !a {
			t.Fatalf("expected navidrome alias for %q", serverName)
		}
	})
}

func TestClientConcurrentLoginRace(t *testing.T) {
	var logins atomicCounter
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/login":
			logins.Add(1)
			_, _ = w.Write([]byte(`{"token":"jwt-token"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/playlist":
			auth := r.Header.Get(authHeader)
			if !strings.Contains(auth, "Bearer jwt-token") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			body, _ := io.ReadAll(r.Body)
			var payload map[string]any
			_ = json.Unmarshal(body, &payload)
			_, _ = w.Write([]byte(`{"id":"pl-1","name":"mix"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "user", "pass")
	var wg sync.WaitGroup
	errCh := make(chan error, 32)
	for range 32 {
		wg.Go(func() {
			_, err := client.CreateSmartPlaylist(t.Context(), CreateSmartPlaylistRequest{
				Name:  "Jazz",
				Rules: map[string]any{"all": []any{}},
			})
			errCh <- err
		})
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent CreateSmartPlaylist: %v", err)
		}
	}
	if logins.Load() < 1 {
		t.Fatal("expected at least one login")
	}
}

type atomicCounter struct {
	mu sync.Mutex
	n  int
}

func (c *atomicCounter) Add(delta int) {
	c.mu.Lock()
	c.n += delta
	c.mu.Unlock()
}

func (c *atomicCounter) Load() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}
