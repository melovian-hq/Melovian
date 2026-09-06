// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"melovian/internal/metaloader"
	"melovian/internal/store"
)

func TestConcurrentLocalLibraryScanAndBrowse(t *testing.T) {
	srv, db := newTestServerWithLocalLibrary(t)
	libraries := store.NewLocalLibraryStore(db)
	tracks := store.NewLocalTrackStore(db)
	scanner := metaloader.NewScanner(libraries, tracks)

	root := t.TempDir()
	for i := range 5 {
		name := "track" + string(rune('a'+i)) + ".mp3"
		if err := os.WriteFile(filepath.Join(root, name), []byte("data-"+name), 0o644); err != nil {
			t.Fatalf("write track: %v", err)
		}
	}

	lib, err := libraries.Create(store.CreateLocalLibraryInput{Name: "Local", Path: root})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := scanner.Scan(lib.ID); err != nil {
		t.Fatalf("initial scan: %v", err)
	}
	if err := libraries.SetActive(lib.ID); err != nil {
		t.Fatalf("SetActive: %v", err)
	}

	done := make(chan struct{})
	var wg sync.WaitGroup

	for range 4 {
		wg.Go(func() {
			for range 20 {
				req := httptest.NewRequest(http.MethodPost, "/api/local-libraries/"+lib.ID+"/scan", nil)
				rec := httptest.NewRecorder()
				srv.Handler().ServeHTTP(rec, req)
			}
		})
		wg.Go(func() {
			for range 40 {
				req := httptest.NewRequest(http.MethodGet, "/api/local-music/artists", nil)
				rec := httptest.NewRecorder()
				srv.Handler().ServeHTTP(rec, req)
			}
		})
		wg.Go(func() {
			for range 40 {
				req := httptest.NewRequest(http.MethodGet, "/api/music/library-stats", nil)
				rec := httptest.NewRecorder()
				srv.Handler().ServeHTTP(rec, req)
			}
		})
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("concurrent local library scan and browse did not complete within timeout")
	}

	loaded, err := libraries.Get(lib.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if loaded.ScanStatus != "idle" {
		t.Fatalf("expected scan status idle after concurrent scans, got %q (error: %q)", loaded.ScanStatus, loaded.ScanError)
	}
	if loaded.TrackCount != 5 {
		t.Fatalf("expected 5 tracks, got %d", loaded.TrackCount)
	}
	_, err = tracks.ListPresent(lib.ID)
	if err != nil {
		t.Fatalf("ListPresent: %v", err)
	}
}

func TestConcurrentLocalAndInstanceActivate(t *testing.T) {
	srv, db := newTestServerWithLocalLibrary(t)
	instances := store.NewInstanceStore(db)
	libraries := store.NewLocalLibraryStore(db)

	inst, err := instances.Create(store.CreateInstanceInput{
		Name: "Sub", ServerURL: "http://sub.example.com", Username: "u", Password: "p",
	})
	if err != nil {
		t.Fatalf("Create instance: %v", err)
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "song.mp3"), []byte("song"), 0o644); err != nil {
		t.Fatalf("write track: %v", err)
	}
	lib, err := libraries.Create(store.CreateLocalLibraryInput{Name: "Local", Path: root})
	if err != nil {
		t.Fatalf("Create library: %v", err)
	}

	done := make(chan struct{})
	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			for range 30 {
				req := httptest.NewRequest(http.MethodPost, "/api/instances/"+inst.ID+"/activate", nil)
				rec := httptest.NewRecorder()
				srv.Handler().ServeHTTP(rec, req)
			}
		})
		wg.Go(func() {
			for range 30 {
				req := httptest.NewRequest(http.MethodPost, "/api/local-libraries/"+lib.ID+"/activate", nil)
				rec := httptest.NewRecorder()
				srv.Handler().ServeHTTP(rec, req)
			}
		})
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("concurrent local and instance activate did not complete within timeout")
	}
}

func TestActivateLocalLibraryClearsActiveInstance(t *testing.T) {
	srv, db := newTestServerWithLocalLibrary(t)
	instances := store.NewInstanceStore(db)
	libraries := store.NewLocalLibraryStore(db)

	inst, err := instances.Create(store.CreateInstanceInput{
		Name: "Sub", ServerURL: "http://sub.example.com", Username: "u", Password: "p",
	})
	if err != nil {
		t.Fatalf("Create instance: %v", err)
	}
	if err := instances.SetActive(inst.ID); err != nil {
		t.Fatalf("SetActive instance: %v", err)
	}

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "song.mp3"), []byte("song"), 0o644); err != nil {
		t.Fatalf("write track: %v", err)
	}
	lib, err := libraries.Create(store.CreateLocalLibraryInput{Name: "Local", Path: root})
	if err != nil {
		t.Fatalf("Create library: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/local-libraries/"+lib.ID+"/activate", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("activate local library: %d %s", rec.Code, rec.Body.String())
	}

	activeInst, err := instances.GetActiveID()
	if err != nil {
		t.Fatalf("GetActiveID: %v", err)
	}
	if activeInst != "" {
		t.Fatalf("expected no active instance after local library activate, got %q", activeInst)
	}

	activeLib, err := libraries.GetActiveID()
	if err != nil {
		t.Fatalf("GetActiveID library: %v", err)
	}
	if activeLib != lib.ID {
		t.Fatalf("expected active library %q, got %q", lib.ID, activeLib)
	}
}

func TestActivateInstanceClearsActiveLocalLibrary(t *testing.T) {
	srv, db := newTestServerWithLocalLibrary(t)
	instances := store.NewInstanceStore(db)
	libraries := store.NewLocalLibraryStore(db)

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "song.mp3"), []byte("song"), 0o644); err != nil {
		t.Fatalf("write track: %v", err)
	}
	lib, err := libraries.Create(store.CreateLocalLibraryInput{Name: "Local", Path: root})
	if err != nil {
		t.Fatalf("Create library: %v", err)
	}
	if err := libraries.SetActive(lib.ID); err != nil {
		t.Fatalf("SetActive library: %v", err)
	}

	inst, err := instances.Create(store.CreateInstanceInput{
		Name: "Sub", ServerURL: "http://sub.example.com", Username: "u", Password: "p",
	})
	if err != nil {
		t.Fatalf("Create instance: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/instances/"+inst.ID+"/activate", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("activate instance: %d %s", rec.Code, rec.Body.String())
	}

	activeLib, err := libraries.GetActiveID()
	if err != nil {
		t.Fatalf("GetActiveID library: %v", err)
	}
	if activeLib != "" {
		t.Fatalf("expected no active local library after instance activate, got %q", activeLib)
	}

	activeInst, err := instances.GetActiveID()
	if err != nil {
		t.Fatalf("GetActiveID instance: %v", err)
	}
	if activeInst != inst.ID {
		t.Fatalf("expected active instance %q, got %q", inst.ID, activeInst)
	}
}
