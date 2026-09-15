// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package navidrome

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStreamEventsParsesSSEFrames(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "jwt-1"})
		case "/api/events":
			if r.URL.Query().Get("jwt") != "jwt-1" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: scanStatus\ndata: {\"scanning\":true,\"count\":5}\n\n"))
			_, _ = w.Write([]byte("data: {\"name\":\"serverStart\",\"data\":{\"startTime\":1}}\n\n"))
			_, _ = w.Write([]byte(": comment\n\nevent: refreshResource\ndata: {\"resource\":\"song\",\"id\":\"abc\"}\n\n"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "u", "p")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var got []string
	err := client.StreamEvents(ctx, func(name string, data json.RawMessage) error {
		got = append(got, name+":"+string(data))
		if len(got) == 3 {
			return errors.New("stop")
		}
		return nil
	})
	if err == nil || err.Error() != "stop" {
		t.Fatalf("expected emit error to end stream, got %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 events, got %#v", got)
	}
	if got[0] != `scanStatus:{"scanning":true,"count":5}` {
		t.Fatalf("event frame: %q", got[0])
	}
	if got[1] != `serverStart:{"startTime":1}` {
		t.Fatalf("name-in-data frame should unwrap payload, got %q", got[1])
	}
	if got[2] != `refreshResource:{"resource":"song","id":"abc"}` {
		t.Fatalf("refresh frame: %q", got[2])
	}
}

func TestStreamEventsAuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "u", "p")
	err := client.StreamEvents(context.Background(), func(string, json.RawMessage) error { return nil })
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestTokenCachesLogin(t *testing.T) {
	logins := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logins++
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "jwt-x"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "u", "p")
	tok, err := client.Token(context.Background())
	if err != nil || tok != "jwt-x" {
		t.Fatalf("token = %q err %v", tok, err)
	}
	tok, err = client.Token(context.Background())
	if err != nil || tok != "jwt-x" {
		t.Fatalf("second token = %q err %v", tok, err)
	}
	if logins != 1 {
		t.Fatalf("expected 1 login, got %d", logins)
	}
}
