// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  buildMixContextInput,
  createSeededRandom,
  dedupeTracks,
  interleaveByArtist,
  orderForFlow,
  shuffleWithSeed,
  sortMixesByPriority,
  upsertMix,
} from "./mix-generator";
import { defaultMixSettings } from "./mix-settings";
import type { GeneratedMix } from "./mix-generator";
import type { SubsonicSong } from "$lib/subsonic/types";

const song = (id: string, artist: string): SubsonicSong => ({
  id,
  title: `Track ${id}`,
  artist,
  albumId: `album-${artist}`,
});

describe("mix-generator acceptance", () => {
  it("accepts daily mix rebuild with stable seed", () => {
    const tracks = [
      song("1", "Alpha"),
      song("2", "Beta"),
      song("3", "Gamma"),
      song("4", "Alpha"),
      song("5", "Beta"),
    ];
    const seed = "2026-07-18";
    const first = orderForFlow(dedupeTracks(tracks), seed);
    const second = orderForFlow(dedupeTracks(tracks), seed);
    expect(first.map((t) => t.id)).toEqual(second.map((t) => t.id));
    expect(interleaveByArtist(first).length).toBe(first.length);
  });

  it("accepts mix upsert and priority sort for shelf display", () => {
    const morning: GeneratedMix = {
      id: "on-repeat",
      title: "Morning",
      subtitle: "Ease in",
      tracks: [song("1", "A")],
      gradient: "linear-gradient(red, blue)",
    };
    const discovery: GeneratedMix = {
      id: "discover",
      title: "Discovery",
      subtitle: "New finds",
      tracks: [song("2", "B")],
      gradient: "linear-gradient(green, yellow)",
    };
    let mixes = upsertMix([], morning);
    mixes = upsertMix(mixes, discovery);
    mixes = upsertMix(mixes, { ...morning, title: "Morning Updated" });
    expect(mixes).toHaveLength(2);
    expect(mixes.find((m) => m.id === "on-repeat")?.title).toBe(
      "Morning Updated",
    );
    expect(sortMixesByPriority(mixes).map((m) => m.id)[0]).toBe("on-repeat");
  });

  it("accepts seeded random used by mix context day seed", () => {
    const ctx = buildMixContextInput(
      null,
      [],
      [],
      "day-1",
      defaultMixSettings(),
      () => song("x", "X"),
    );
    expect(ctx.daySeed).toBe("day-1");
    const rand = createSeededRandom(ctx.daySeed);
    expect(rand()).toBe(createSeededRandom("day-1")());
    expect(shuffleWithSeed(["a", "b", "c"], ctx.daySeed)).toEqual(
      shuffleWithSeed(["a", "b", "c"], "day-1"),
    );
  });
});
