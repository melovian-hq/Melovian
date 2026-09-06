// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import {
  buildPersonalProfile,
  createPersonalRadioState,
  seedPersonalRadioTracks,
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
});
