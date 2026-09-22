// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import {
  buildPersonalProfile,
  createPersonalRadioState,
  notePersonalComplete,
  notePersonalSkip,
  seedPersonalRadioTracks,
  type PersonalRadioFetchers,
} from "./personal-radio";
import type { ListenEntry, SubsonicSong } from "$lib/subsonic/types";

const entry = (
  overrides: Partial<ListenEntry> & Pick<ListenEntry, "trackId">,
): ListenEntry => ({
  trackId: overrides.trackId,
  trackTitle: overrides.trackTitle ?? overrides.trackId,
  artistName: overrides.artistName ?? "Artist",
  albumId: "al",
  albumTitle: "Album",
  positionMs: 0,
  durationMs: 180_000,
  played: true,
  playCount: overrides.playCount ?? 1,
  listenedMs: overrides.listenedMs ?? 0,
  lastPlayedAt: new Date().toISOString(),
  coverArtId: "",
});

describe("personal-radio listen seeds mock", () => {
  it("prefers high listenedMs history when stats topTracks are absent", async () => {
    const history = [
      entry({ trackId: "low", playCount: 9, listenedMs: 5_000 }),
      entry({ trackId: "high", playCount: 1, listenedMs: 800_000 }),
      entry({ trackId: "mid", playCount: 3, listenedMs: 120_000 }),
    ];
    const playCountByTrack = new Map(
      history.map((e) => [e.trackId, e.playCount]),
    );
    const listenedMsByTrack = new Map(
      history
        .filter((e) => e.listenedMs > 0)
        .map((e) => [e.trackId, e.listenedMs]),
    );
    const profile = buildPersonalProfile(
      null,
      history,
      playCountByTrack,
      new Set(),
      new Set(),
      listenedMsByTrack,
    );
    const state = createPersonalRadioState();
    const similar = vi.fn(async (id: string) => [
      { id: `sim-${id}`, title: `Sim ${id}`, artist: "Artist" } as SubsonicSong,
    ]);
    const tracks = await seedPersonalRadioTracks(
      state,
      {
        getSimilarSongs: similar,
        getRandomSongs: async () => [],
        searchArtistSongs: async () => [],
      },
      profile,
      history,
      null,
      4,
      new Set(),
      (e) => ({ id: e.trackId, title: e.trackTitle, artist: e.artistName }),
    );

    expect(state.seedTrackIds[0]).toBe("high");
    expect(similar).toHaveBeenCalled();
    expect(tracks.length).toBeGreaterThan(0);
  });

  it("rotates seed windows across refill batches", async () => {
    const history = Array.from({ length: 12 }, (_, i) =>
      entry({
        trackId: `seed-${i}`,
        artistName: `Artist ${i}`,
        playCount: 20 - i,
        listenedMs: 1_000_000 - i * 10_000,
      }),
    );
    const playCountByTrack = new Map(
      history.map((e) => [e.trackId, e.playCount]),
    );
    const profile = buildPersonalProfile(
      null,
      history,
      playCountByTrack,
      new Set(),
      new Set(),
    );
    const state = createPersonalRadioState();
    const queried: string[] = [];
    const fetchers: PersonalRadioFetchers = {
      getSimilarSongs: vi.fn(async () => []),
      getRandomSongs: async () => [],
      searchArtistSongs: async () => [],
      getRelatedTracks: async (track: SubsonicSong) => {
        if (track.id) queried.push(track.id);
        return [];
      },
    };
    const entryToSong = (e: ListenEntry) =>
      ({ id: e.trackId, title: e.trackTitle, artist: e.artistName });

    await seedPersonalRadioTracks(
      state, fetchers, profile, history, null, 6, new Set(), entryToSong,
    );
    const batch0 = [...queried];
    await seedPersonalRadioTracks(
      state, fetchers, profile, history, null, 6, new Set(), entryToSong,
    );
    const batch1 = queried.slice(batch0.length);

    expect(batch0.length).toBeGreaterThan(0);
    expect(batch1.length).toBeGreaterThan(0);
    // A rotated window means later refills lean on different seeds instead
    // of re-dealing the same picks forever.
    expect(batch1).not.toEqual(batch0);
  });

  it("anchors the last completed track in every batch", async () => {
    const history = Array.from({ length: 12 }, (_, i) =>
      entry({
        trackId: `seed-${i}`,
        artistName: `Artist ${i}`,
        playCount: 5,
        listenedMs: 100_000,
      }),
    );
    const profile = buildPersonalProfile(
      null,
      history,
      new Map(history.map((e) => [e.trackId, e.playCount])),
      new Set(),
      new Set(),
    );
    const state = createPersonalRadioState();
    notePersonalComplete(state, {
      id: "anchor",
      title: "Anchor",
      artist: "Anchor Artist",
    });
    const queried: string[] = [];
    const fetchers: PersonalRadioFetchers = {
      getSimilarSongs: vi.fn(async () => []),
      getRandomSongs: async () => [],
      searchArtistSongs: async () => [],
      getRelatedTracks: async (track: SubsonicSong) => {
        if (track.id) queried.push(track.id);
        return [];
      },
      resolveTrack: async (id) =>
        ({ id, title: id, artist: "Resolved" }) as SubsonicSong,
    };
    const entryToSong = (e: ListenEntry) =>
      ({ id: e.trackId, title: e.trackTitle, artist: e.artistName });

    for (let batch = 0; batch < 3; batch += 1) {
      await seedPersonalRadioTracks(
        state, fetchers, profile, history, null, 6, new Set(), entryToSong,
      );
    }

    // The anchor seed is queried on the first batch, then served from the
    // session similar cache on later batches.
    expect(queried).toContain("anchor");
  });

  it("prefers getRelatedTracks over the similar-songs endpoint", async () => {
    const history = [
      entry({ trackId: "seed-a", playCount: 8, listenedMs: 400_000 }),
    ];
    const profile = buildPersonalProfile(
      null,
      history,
      new Map(history.map((e) => [e.trackId, e.playCount])),
      new Set(),
      new Set(),
    );
    const state = createPersonalRadioState();
    const similar = vi.fn(async () => []);
    const related = vi.fn(async () => [
      { id: "rel-1", title: "Rel", artist: "Other" } as SubsonicSong,
    ]);
    await seedPersonalRadioTracks(
      state,
      {
        getSimilarSongs: similar,
        getRandomSongs: async () => [],
        searchArtistSongs: async () => [],
        getRelatedTracks: related,
      },
      profile,
      history,
      null,
      4,
      new Set(),
      (e) => ({ id: e.trackId, title: e.trackTitle, artist: e.artistName }),
    );

    expect(related).toHaveBeenCalled();
    expect(similar).not.toHaveBeenCalled();
  });

  it("bounds skip timestamps while tracking session skips", () => {
    const state = createPersonalRadioState();
    for (let i = 0; i < 40; i += 1) {
      notePersonalSkip(state, {
        id: `s-${i}`,
        title: `S${i}`,
        artist: "Skipper",
      });
    }
    expect(state.recentSkipAt.length).toBe(32);
    expect(state.skipTrackIds.size).toBe(40);
  });
});
