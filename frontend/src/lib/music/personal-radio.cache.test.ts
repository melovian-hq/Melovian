// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import {
  buildPersonalProfile,
  createPersonalRadioState,
  seedPersonalRadioTracks,
} from "./personal-radio";
import type { ListenEntry, SubsonicSong } from "$lib/subsonic/types";

describe("personal-radio session cache", () => {
  it("reuses similar song cache within a session", async () => {
    const history: ListenEntry[] = [
      {
        trackId: "seed-1",
        trackTitle: "Seed",
        artistName: "Artist",
        albumId: "al-1",
        albumTitle: "Album",
        positionMs: 0,
        durationMs: 180_000,
        played: true,
        playCount: 5,
        listenedMs: 120_000,
        lastPlayedAt: new Date().toISOString(),
        coverArtId: "",
      },
    ];
    const playCountByTrack = new Map([["seed-1", 5]]);
    const profile = buildPersonalProfile(
      null,
      history,
      playCountByTrack,
      new Set(),
      new Set(),
      new Map([["seed-1", 120_000]]),
    );
    const state = createPersonalRadioState();
    state.lastCompletedId = "seed-1";

    const similar = vi.fn(async (id: string) => [
      { id: `sim-${id}`, title: `Sim ${id}`, artist: "Artist" } as SubsonicSong,
    ]);
    const fetchers = {
      getSimilarSongs: similar,
      getRandomSongs: async () => [] as SubsonicSong[],
      searchArtistSongs: async () => [] as SubsonicSong[],
    };
    const entryToSong = (entry: ListenEntry) => ({
      id: entry.trackId,
      title: entry.trackTitle,
      artist: entry.artistName,
    });

    await seedPersonalRadioTracks(
      state,
      fetchers,
      profile,
      history,
      null,
      3,
      new Set(),
      entryToSong,
    );
    const callsAfterFirst = similar.mock.calls.length;
    expect(callsAfterFirst).toBeGreaterThan(0);

    await seedPersonalRadioTracks(
      state,
      fetchers,
      profile,
      history,
      null,
      3,
      new Set(),
      entryToSong,
    );
    expect(similar.mock.calls.length).toBe(callsAfterFirst);
    expect(state.similarCache.size).toBeGreaterThan(0);
  });

  it("keeps stable session seed across refills", () => {
    const state = createPersonalRadioState();
    const seed = state.sessionSeed;
    state.lastCompletedId = "track-2";
    expect(state.sessionSeed).toBe(seed);
  });
});
