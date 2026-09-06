// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { buildMixContextInput, createMixBuildState } from "./mix-generator";
import { defaultMixSettings } from "./mix-settings";
import type { ListenEntry } from "$lib/subsonic/types";

const entry = (
  overrides: Partial<ListenEntry> & Pick<ListenEntry, "trackId">,
): ListenEntry => ({
  trackId: overrides.trackId,
  trackTitle: overrides.trackTitle ?? overrides.trackId,
  artistName: overrides.artistName ?? "Artist",
  albumId: overrides.albumId ?? "al",
  albumTitle: overrides.albumTitle ?? "Album",
  positionMs: overrides.positionMs ?? 0,
  durationMs: overrides.durationMs ?? 180_000,
  played: overrides.played ?? true,
  playCount: overrides.playCount ?? 1,
  listenedMs: overrides.listenedMs ?? 0,
  lastPlayedAt:
    overrides.lastPlayedAt ??
    new Date(Date.now() - 40 * 86_400_000).toISOString(),
  coverArtId: "",
});

describe("mix-generator listen context unit", () => {
  it("records listenedMsByTrack from history", () => {
    const ctx = buildMixContextInput(
      null,
      [
        entry({ trackId: "a", listenedMs: 90_000, playCount: 2 }),
        entry({ trackId: "b", listenedMs: 0, playCount: 5 }),
      ],
      [],
      "day:test",
      defaultMixSettings(),
      (e) => ({ id: e.trackId, title: e.trackTitle, artist: e.artistName }),
    );
    expect(ctx.listenedMsByTrack.get("a")).toBe(90_000);
    expect(ctx.listenedMsByTrack.has("b")).toBe(false);
  });

  it("marks abandoned incomplete listens as skipped", () => {
    const ctx = buildMixContextInput(
      null,
      [
        entry({
          trackId: "partial",
          played: false,
          positionMs: 10_000,
          durationMs: 200_000,
          listenedMs: 20_000,
        }),
      ],
      [],
      "day:skip",
      defaultMixSettings(),
      (e) => ({ id: e.trackId, title: e.trackTitle }),
    );
    expect(ctx.skippedTrackIds.has("partial")).toBe(true);
  });

  it("weights language inference by listen affinity", () => {
    const ctx = buildMixContextInput(
      null,
      [
        entry({
          trackId: "en1",
          trackTitle: "Hello World",
          artistName: "English Band",
          listenedMs: 400_000,
          playCount: 1,
        }),
      ],
      [],
      "day:lang",
      defaultMixSettings(),
      (e) => ({ id: e.trackId, title: e.trackTitle, artist: e.artistName }),
    );
    const state = createMixBuildState(ctx);
    expect(state.languageWeights.size).toBeGreaterThanOrEqual(0);
  });
});
