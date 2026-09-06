// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"fmt"
	"testing"
)

func TestListenEventsInsertAndList(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)
	userID := "instance:test"

	if err := listen.Upsert(userID, ListenUpsertInput{
		TrackID:       "t1",
		IncrementPlay: true,
		TrackTitle:    "Track One",
		ArtistName:    "Artist",
		AlbumTitle:    "Album",
		DurationMs:    180000,
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	events, err := listen.ListListenEvents(userID, ListenEventsQuery{Limit: 10})
	if err != nil {
		t.Fatalf("ListListenEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].TrackTitle != "Track One" {
		t.Fatalf("unexpected title %q", events[0].TrackTitle)
	}
}

func TestListenEventsPaginationAndSearch(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)
	userID := "instance:test"

	for i, title := range []string{"Alpha Song", "Beta Song", "Gamma Song"} {
		if err := listen.InsertListenEvent(userID, ListenUpsertInput{
			TrackID:    fmt.Sprintf("t%d", i+1),
			TrackTitle: title,
			ArtistName: "Search Artist",
		}, int64(1000+i)); err != nil {
			t.Fatalf("InsertListenEvent: %v", err)
		}
	}

	page, err := listen.ListListenEvents(userID, ListenEventsQuery{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("expected 2 items, got %d", len(page))
	}

	filtered, err := listen.ListListenEvents(userID, ListenEventsQuery{
		Limit:  10,
		Search: "beta",
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(filtered) != 1 || filtered[0].TrackTitle != "Beta Song" {
		t.Fatalf("unexpected search results %+v", filtered)
	}
}

func TestListenEventsScopedByUser(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)

	if err := listen.InsertListenEvent("user:a", ListenUpsertInput{
		TrackID:    "t1",
		TrackTitle: "A",
	}, 1000); err != nil {
		t.Fatalf("InsertListenEvent A: %v", err)
	}
	if err := listen.InsertListenEvent("user:b", ListenUpsertInput{
		TrackID:    "t1",
		TrackTitle: "B",
	}, 2000); err != nil {
		t.Fatalf("InsertListenEvent B: %v", err)
	}

	eventsA, err := listen.ListListenEvents("user:a", ListenEventsQuery{Limit: 10})
	if err != nil {
		t.Fatalf("ListListenEvents A: %v", err)
	}
	if len(eventsA) != 1 || eventsA[0].TrackTitle != "A" {
		t.Fatalf("unexpected events for user A %+v", eventsA)
	}
}

func TestListenEventsClearHistory(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)

	if err := listen.InsertListenEvent("user:a", ListenUpsertInput{
		TrackID:    "t1",
		TrackTitle: "Keep me",
	}, 1000); err != nil {
		t.Fatalf("InsertListenEvent A: %v", err)
	}
	if err := listen.Upsert("user:b", ListenUpsertInput{
		TrackID:       "t2",
		IncrementPlay: true,
		TrackTitle:    "Clear me",
	}); err != nil {
		t.Fatalf("Upsert B: %v", err)
	}

	if err := listen.ClearHistory("user:b"); err != nil {
		t.Fatalf("ClearHistory: %v", err)
	}

	eventsB, err := listen.ListListenEvents("user:b", ListenEventsQuery{Limit: 10})
	if err != nil {
		t.Fatalf("ListListenEvents B: %v", err)
	}
	if len(eventsB) != 0 {
		t.Fatalf("expected user B events cleared, got %+v", eventsB)
	}
	historyB, err := listen.History("user:b", 10)
	if err != nil {
		t.Fatalf("History B: %v", err)
	}
	if len(historyB) != 0 {
		t.Fatalf("expected user B progress cleared, got %+v", historyB)
	}

	eventsA, err := listen.ListListenEvents("user:a", ListenEventsQuery{Limit: 10})
	if err != nil {
		t.Fatalf("ListListenEvents A: %v", err)
	}
	if len(eventsA) != 1 || eventsA[0].TrackTitle != "Keep me" {
		t.Fatalf("expected user A events kept, got %+v", eventsA)
	}
}

func TestListenEventsBackfill(t *testing.T) {
	db := OpenTestDB(t)
	listen := NewListenStore(db)
	userID := "instance:test"

	if err := listen.Upsert(userID, ListenUpsertInput{
		TrackID:       "legacy",
		IncrementPlay: true,
		TrackTitle:    "Legacy Track",
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	_, err := db.exec(`DELETE FROM listen_events`)
	if err != nil {
		t.Fatalf("clear events: %v", err)
	}
	_, err = db.exec(`DELETE FROM app_settings WHERE key = ?`, listenEventsBackfillKey)
	if err != nil {
		t.Fatalf("clear backfill flag: %v", err)
	}

	if err := listen.BackfillListenEvents(); err != nil {
		t.Fatalf("BackfillListenEvents: %v", err)
	}

	events, err := listen.ListListenEvents(userID, ListenEventsQuery{Limit: 10})
	if err != nil {
		t.Fatalf("ListListenEvents: %v", err)
	}
	if len(events) != 1 || events[0].TrackTitle != "Legacy Track" {
		t.Fatalf("expected backfilled event, got %+v", events)
	}

	if err := listen.BackfillListenEvents(); err != nil {
		t.Fatalf("BackfillListenEvents second run: %v", err)
	}
	events, err = listen.ListListenEvents(userID, ListenEventsQuery{Limit: 10})
	if err != nil {
		t.Fatalf("ListListenEvents after second backfill: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected backfill to run once, got %d events", len(events))
	}
}
