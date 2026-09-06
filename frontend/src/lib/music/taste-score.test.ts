// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  buildSessionMood,
  buildTasteProfile,
  pickQualitySeedIds,
  rankAlbumsByTaste,
  rankTracksByTaste,
  scoreTrackTaste,
  weightedSampleTracks,
} from "./taste-score";
import type { ListenEntry, SubsonicSong } from "$lib/subsonic/types";

const song = (id: string, artist: string, album?: string): SubsonicSong => ({
  id,
  title: id,
  artist,
  album,
});

const entry = (
  trackId: string,
  artistName: string,
  playCount: number,
): ListenEntry => ({
  trackId,
  trackTitle: trackId,
  artistName,
  albumId: "a1",
  albumTitle: "Album",
  positionMs: 0,
  durationMs: 180_000,
  played: true,
  playCount,
  listenedMs: playCount * 180_000,
  lastPlayedAt: new Date().toISOString(),
  coverArtId: "",
});

describe("taste-score", () => {
  it("scores favored artists higher than unknown artists", () => {
    const history = [entry("t1", "Alpha", 8), entry("t2", "Beta", 2)];
    const profile = buildTasteProfile(
      {
        totalPlays: 10,
        uniqueTracks: 2,
        totalListeningMs: 1_000_000,
        topArtists: [{ key: "Alpha", label: "Alpha", count: 8 }],
        topTracks: [],
        topAlbums: [],
      },
      history,
      new Map([
        ["t1", 8],
        ["t2", 2],
      ]),
      new Set(),
    );

    const alpha = scoreTrackTaste(song("x1", "Alpha"), profile);
    const gamma = scoreTrackTaste(song("x2", "Gamma"), profile);
    expect(alpha).toBeGreaterThan(gamma);
  });

  it("penalizes skipped tracks", () => {
    const profile = buildTasteProfile(
      null,
      [],
      new Map(),
      new Set(["skip-me"]),
    );
    expect(scoreTrackTaste(song("skip-me", "Alpha"), profile)).toBeLessThan(0);
  });

  it("ranks and samples without duplicates", () => {
    const tracks = [
      song("1", "Alpha"),
      song("2", "Beta"),
      song("3", "Alpha"),
      song("4", "Gamma"),
    ];
    const profile = buildTasteProfile(
      {
        totalPlays: 5,
        uniqueTracks: 2,
        totalListeningMs: 1000,
        topArtists: [{ key: "Alpha", label: "Alpha", count: 5 }],
        topTracks: [],
        topAlbums: [],
      },
      [],
      new Map(),
      new Set(),
    );
    const ranked = rankTracksByTaste(tracks, profile);
    expect(ranked[0].artist).toBe("Alpha");

    const picks = weightedSampleTracks(tracks, profile, 3, () => 0.1);
    expect(picks).toHaveLength(3);
    expect(new Set(picks.map((t) => t.id)).size).toBe(3);
  });

  it("boosts starred tracks and matching decades", () => {
    const profile = buildTasteProfile(null, [], new Map(), new Set(), [
      { genre: "Rock", year: 2014 },
    ]);
    const decadeHit = scoreTrackTaste(
      { id: "y1", title: "y1", artist: "X", year: 2016 },
      profile,
    );
    const decadeMiss = scoreTrackTaste(
      { id: "y2", title: "y2", artist: "X", year: 1972 },
      profile,
    );
    expect(decadeHit).toBeGreaterThan(decadeMiss);

    const starred = scoreTrackTaste(
      { id: "star", title: "star", artist: "X" },
      profile,
      { starredTrackIds: new Set(["star"]) },
    );
    const plain = scoreTrackTaste(
      { id: "plain", title: "plain", artist: "X" },
      profile,
    );
    expect(starred).toBeGreaterThan(plain);
  });

  it("ranks topArtistKeys by affinity weight not insertion order", () => {
    const history = [
      entry("t1", "Zed", 1),
      entry("t2", "Alpha", 12),
      entry("t3", "Beta", 4),
    ];
    const profile = buildTasteProfile(
      null,
      history,
      new Map([
        ["t1", 1],
        ["t2", 12],
        ["t3", 4],
      ]),
      new Set(),
    );
    expect(profile.topArtistKeys[0]).toBe("alpha");
    expect(profile.topArtistKeys.slice(0, 3)).toEqual(["alpha", "beta", "zed"]);
  });

  it("weights recent listens more than old ones", () => {
    const now = Date.now();
    const recent: ListenEntry = {
      ...entry("r1", "RecentAct", 3),
      lastPlayedAt: new Date(now - 2 * 86_400_000).toISOString(),
      listenedMs: 540_000,
    };
    const old: ListenEntry = {
      ...entry("o1", "OldAct", 3),
      lastPlayedAt: new Date(now - 400 * 86_400_000).toISOString(),
      listenedMs: 540_000,
    };
    const profile = buildTasteProfile(
      null,
      [recent, old],
      new Map([
        ["r1", 3],
        ["o1", 3],
      ]),
      new Set(),
      [],
      new Map(),
      now,
    );
    expect(profile.artistAffinity.get("recentact") ?? 0).toBeGreaterThan(
      profile.artistAffinity.get("oldact") ?? 0,
    );
  });

  it("boosts tracks matching the live session mood", () => {
    const profile = buildTasteProfile(null, [], new Map(), new Set());
    const mood = buildSessionMood(
      [
        { id: "now", artist: "MoodBand", genre: "Jazz", album: "Night" },
        { id: "prev", artist: "MoodBand", genre: "Jazz" },
      ],
      1,
    );
    const matching = scoreTrackTaste(song("a", "MoodBand"), profile, {
      sessionMood: mood,
    });
    const other = scoreTrackTaste(song("b", "OtherAct"), profile, {
      sessionMood: mood,
    });
    expect(matching).toBeGreaterThan(other);
  });

  it("learns artist follow-ons from sequential listens", () => {
    const now = Date.now();
    const history: ListenEntry[] = [
      {
        ...entry("t1", "Alpha", 2),
        lastPlayedAt: new Date(now - 40 * 60_000).toISOString(),
        listenedMs: 300_000,
      },
      {
        ...entry("t2", "Beta", 2),
        lastPlayedAt: new Date(now - 20 * 60_000).toISOString(),
        listenedMs: 280_000,
      },
      {
        ...entry("t3", "Alpha", 2),
        lastPlayedAt: new Date(now - 10 * 60_000).toISOString(),
        listenedMs: 290_000,
      },
      {
        ...entry("t4", "Beta", 2),
        lastPlayedAt: new Date(now - 2 * 60_000).toISOString(),
        listenedMs: 270_000,
      },
    ];
    const profile = buildTasteProfile(
      null,
      history,
      new Map(history.map((e) => [e.trackId, e.playCount])),
      new Set(),
      [],
      new Map(history.map((e) => [e.trackId, e.listenedMs])),
      now,
    );
    expect(profile.followArtistAffinity.get("alpha|beta") ?? 0).toBeGreaterThan(
      0,
    );

    const mood = buildSessionMood([{ artist: "Alpha" }], 1);
    const followOn = scoreTrackTaste(song("x", "Beta"), profile, {
      sessionMood: mood,
    });
    const unrelated = scoreTrackTaste(song("y", "Zeta"), profile, {
      sessionMood: mood,
    });
    expect(followOn).toBeGreaterThan(unrelated);
  });

  it("picks quality seeds over play-count spam and prefers last completed", () => {
    const history = [entry("spam", "X", 20), entry("deep", "Y", 2)];
    history[0] = { ...history[0], listenedMs: 8_000, playCount: 20 };
    history[1] = { ...history[1], listenedMs: 600_000, playCount: 2 };

    const seeds = pickQualitySeedIds(
      history,
      {
        totalPlays: 22,
        uniqueTracks: 2,
        totalListeningMs: 608_000,
        topArtists: [],
        topTracks: [
          { key: "spam", label: "spam", count: 20 },
          { key: "deep", label: "deep", count: 2 },
        ],
        topAlbums: [],
      },
      3,
      "deep",
    );
    expect(seeds[0]).toBe("deep");
    expect(seeds).toContain("deep");
  });

  it("ranks albums by artist affinity", () => {
    const profile = buildTasteProfile(
      {
        totalPlays: 10,
        uniqueTracks: 2,
        totalListeningMs: 1000,
        topArtists: [{ key: "Fav", label: "Fav", count: 10 }],
        topTracks: [],
        topAlbums: [],
      },
      [entry("t1", "Fav", 8)],
      new Map([["t1", 8]]),
      new Set(),
    );
    const ranked = rankAlbumsByTaste(
      [
        { id: "a1", name: "Cold", artist: "Unknown" },
        { id: "a2", name: "Warm", artist: "Fav" },
      ],
      profile,
    );
    expect(ranked[0].id).toBe("a2");
  });
});
