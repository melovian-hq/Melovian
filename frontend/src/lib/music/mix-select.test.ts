// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { buildTasteProfile } from "./taste-score";
import { defaultMixSettings } from "./mix-settings";
import {
  clusterArtistFeatures,
  defaultMixPolicy,
  emptyMixSeed,
  genresRelated,
  mixFitScore,
  orderForFlow,
  selectMixTracks,
  type ArtistFeature,
  type MixSelectContext,
} from "./mix-select";
import type { SubsonicSong } from "$lib/subsonic/types";

const song = (
  id: string,
  artist: string,
  extras: Partial<SubsonicSong> = {},
): SubsonicSong => ({
  id,
  title: extras.title ?? `Track ${id}`,
  artist,
  albumId: extras.albumId ?? `album-${artist}`,
  genre: extras.genre,
  year: extras.year,
  duration: extras.duration,
});

function features(
  rows: Array<[string, string[], number[], number]>,
): ArtistFeature[] {
  return rows.map(([label, genres, decades, weight]) => ({
    label,
    key: label.toLowerCase(),
    weight,
    genres,
    decades,
  }));
}

function selectCtx(extras: Partial<MixSelectContext> = {}): MixSelectContext {
  const playCountByTrack = extras.playCountByTrack ?? new Map();
  const skippedTrackIds = extras.skippedTrackIds ?? new Set();
  return {
    playCountByTrack,
    listenedMsByTrack: extras.listenedMsByTrack ?? new Map(),
    skippedTrackIds,
    recentlyPlayedIds: extras.recentlyPlayedIds ?? new Set(),
    recentMixTrackIds: extras.recentMixTrackIds ?? new Set(),
    settings: extras.settings ?? defaultMixSettings(),
    tasteProfile:
      extras.tasteProfile ??
      buildTasteProfile(null, [], playCountByTrack, skippedTrackIds),
    starredTrackIds: extras.starredTrackIds ?? new Set(),
  };
}

describe("genresRelated", () => {
  it("matches exact, containment, and shared tokens", () => {
    expect(genresRelated("Rock", "rock")).toBe(true);
    expect(genresRelated("Indie Rock", "Rock")).toBe(true);
    expect(genresRelated("Alternative Rock", "Indie Rock")).toBe(true);
    expect(genresRelated("Jazz", "Metal")).toBe(false);
  });
});

describe("clusterArtistFeatures", () => {
  it("groups artists that share a genre and keeps clusters stable", () => {
    const input = features([
      ["Rock One", ["Rock"], [2010], 20],
      ["Rock Two", ["Rock"], [2010], 18],
      ["Rock Three", ["Rock"], [2000], 12],
      ["Jazz One", ["Jazz"], [1960], 16],
      ["Jazz Two", ["Jazz"], [1960], 10],
      ["Jazz Three", ["Jazz"], [1970], 8],
    ]);
    const first = clusterArtistFeatures(input, "day:clusters", 3, 3);
    const second = clusterArtistFeatures(input, "day:clusters", 3, 3);
    expect(first.map((cluster) => cluster.artists.map((a) => a.label))).toEqual(
      second.map((cluster) => cluster.artists.map((a) => a.label)),
    );
    expect(first.length).toBeGreaterThanOrEqual(2);
    const rockCluster = first.find((cluster) =>
      cluster.artists.some((artist) => artist.label === "Rock One"),
    );
    expect(
      rockCluster?.artists.every((artist) => artist.genres.includes("Rock")),
    ).toBe(true);
    const jazzCluster = first.find((cluster) =>
      cluster.artists.some((artist) => artist.label === "Jazz One"),
    );
    expect(
      jazzCluster?.artists.every((artist) => artist.genres.includes("Jazz")),
    ).toBe(true);
  });

  it("returns no clusters for an empty library", () => {
    expect(clusterArtistFeatures([], "seed", 3, 4)).toEqual([]);
  });
});

describe("selectMixTracks", () => {
  it("excludes skipped tracks", () => {
    const tracks = [
      song("keep-1", "Alpha"),
      song("skip-me", "Beta"),
      song("keep-2", "Gamma"),
      song("keep-3", "Delta"),
      song("keep-4", "Epsilon"),
    ];
    const picked = selectMixTracks(
      tracks,
      selectCtx({ skippedTrackIds: new Set(["skip-me"]) }),
      emptyMixSeed(),
      defaultMixPolicy(4),
      "skip-seed",
    );
    expect(picked.map((track) => track.id)).not.toContain("skip-me");
    expect(picked.length).toBe(4);
  });

  it("keeps tracks that are missing genre and year", () => {
    const tracks = [
      song("a", "Alpha"),
      song("b", "Beta"),
      song("c", "Gamma"),
      song("d", "Delta"),
      song("e", "Epsilon"),
    ];
    const picked = selectMixTracks(
      tracks,
      selectCtx(),
      {
        artistKeys: new Set(["alpha"]),
        genreKeys: new Set(["rock"]),
        decades: new Set([2010]),
        artistFocused: false,
      },
      defaultMixPolicy(5),
      "meta-seed",
    );
    expect(picked.map((track) => track.id).sort()).toEqual([
      "a",
      "b",
      "c",
      "d",
      "e",
    ]);
  });

  it("returns an empty list for an empty library", () => {
    expect(
      selectMixTracks(
        [],
        selectCtx(),
        emptyMixSeed(),
        defaultMixPolicy(30),
        "empty",
      ),
    ).toEqual([]);
  });

  it("is stable for the same seed", () => {
    const tracks = Array.from({ length: 20 }, (_, i) =>
      song(`t${i}`, `Artist ${i % 6}`, { albumId: `al-${i % 8}` }),
    );
    const ctx = selectCtx();
    const seed = emptyMixSeed();
    const policy = defaultMixPolicy(10);
    expect(
      selectMixTracks(tracks, ctx, seed, policy, "stable").map((t) => t.id),
    ).toEqual(
      selectMixTracks(tracks, ctx, seed, policy, "stable").map((t) => t.id),
    );
  });

  it("spreads artists instead of filling from one act", () => {
    const tracks = [
      ...Array.from({ length: 8 }, (_, i) =>
        song(`a${i}`, "Alpha", { albumId: "al-a" }),
      ),
      ...Array.from({ length: 8 }, (_, i) =>
        song(`b${i}`, "Beta", { albumId: "al-b" }),
      ),
      ...Array.from({ length: 8 }, (_, i) =>
        song(`c${i}`, "Gamma", { albumId: "al-c" }),
      ),
    ];
    const picked = selectMixTracks(
      tracks,
      selectCtx(),
      emptyMixSeed(),
      { ...defaultMixPolicy(9), maxPerArtist: 3, maxPerAlbum: 2 },
      "spread",
    );
    const byArtist = new Map<string, number>();
    for (const track of picked) {
      byArtist.set(
        track.artist ?? "",
        (byArtist.get(track.artist ?? "") ?? 0) + 1,
      );
    }
    expect([...byArtist.values()].every((count) => count <= 3)).toBe(true);
    expect(byArtist.size).toBeGreaterThanOrEqual(3);
  });
});

describe("orderForFlow diversity", () => {
  it("does not play five tracks from the same artist in a row when others exist", () => {
    const tracks = [
      song("a1", "Alpha", { albumId: "a-1" }),
      song("a2", "Alpha", { albumId: "a-2" }),
      song("a3", "Alpha", { albumId: "a-3" }),
      song("a4", "Alpha", { albumId: "a-4" }),
      song("a5", "Alpha", { albumId: "a-5" }),
      song("b1", "Beta", { albumId: "b-1" }),
      song("b2", "Beta", { albumId: "b-2" }),
      song("c1", "Gamma", { albumId: "c-1" }),
      song("c2", "Gamma", { albumId: "c-2" }),
    ];
    const ordered = orderForFlow(tracks, "diversity-flow");
    let run = 1;
    let longest = 1;
    for (let i = 1; i < ordered.length; i++) {
      if (ordered[i].artist === ordered[i - 1].artist) run += 1;
      else run = 1;
      longest = Math.max(longest, run);
    }
    expect(longest).toBeLessThan(5);
  });
});

describe("mixFitScore", () => {
  it("prefers seed artists over unrelated tracks", () => {
    const ctx = selectCtx({
      tasteProfile: buildTasteProfile(
        {
          totalPlays: 10,
          uniqueTracks: 2,
          totalListeningMs: 1000,
          topArtists: [{ key: "Alpha", label: "Alpha", count: 10 }],
          topTracks: [],
          topAlbums: [],
        },
        [],
        new Map(),
        new Set(),
      ),
    });
    const seed = {
      artistKeys: new Set(["alpha"]),
      genreKeys: new Set(["rock"]),
      decades: new Set([2010]),
      artistFocused: false,
    };
    const policy = defaultMixPolicy(10);
    const inSeed = mixFitScore(
      song("1", "Alpha", { genre: "Rock", year: 2012 }),
      ctx,
      seed,
      policy,
    );
    const outside = mixFitScore(
      song("2", "Zed", { genre: "Polka", year: 1970 }),
      ctx,
      seed,
      policy,
    );
    expect(inSeed).toBeGreaterThan(outside);
  });

  it("soft-matches related seed genres instead of only exact labels", () => {
    const ctx = selectCtx();
    const seed = {
      artistKeys: new Set<string>(),
      genreKeys: new Set(["rock"]),
      decades: new Set<number>(),
      artistFocused: false,
    };
    const policy = defaultMixPolicy(10);
    const related = mixFitScore(
      song("1", "Alpha", { genre: "Indie Rock", year: 2012 }),
      ctx,
      seed,
      policy,
    );
    const unrelated = mixFitScore(
      song("2", "Beta", { genre: "Polka", year: 1970 }),
      ctx,
      seed,
      policy,
    );
    expect(related).toBeGreaterThan(unrelated);
  });
});
