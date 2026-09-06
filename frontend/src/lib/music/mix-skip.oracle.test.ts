// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { buildMixContextInput } from "./mix-generator";
import { defaultMixSettings } from "./mix-settings";
import type { ListenEntry } from "$lib/subsonic/types";

const entry = (
  overrides: Partial<ListenEntry> & Pick<ListenEntry, "trackId">,
): ListenEntry => ({
  trackId: overrides.trackId,
  trackTitle: overrides.trackTitle ?? overrides.trackId,
  artistName: overrides.artistName ?? "Artist",
  albumId: "al",
  albumTitle: "Album",
  positionMs: overrides.positionMs ?? 0,
  durationMs: overrides.durationMs ?? 180_000,
  played: overrides.played ?? false,
  playCount: overrides.playCount ?? 1,
  listenedMs: overrides.listenedMs ?? 0,
  lastPlayedAt:
    overrides.lastPlayedAt ??
    new Date(Date.now() - 40 * 86_400_000).toISOString(),
  coverArtId: "",
});

function ctx(history: ListenEntry[]) {
  return buildMixContextInput(
    null,
    history,
    [],
    "day:bug",
    defaultMixSettings(),
    (e) => ({ id: e.trackId, title: e.trackTitle }),
  );
}

describe("mix skip inference oracle (must catch regressions)", () => {
  it("does not treat completed plays as skips after position reset", () => {
    // MarkPlayed clears position_ms to 0 while played=true.
    const history = [
      entry({
        trackId: "finished",
        played: true,
        positionMs: 0,
        playCount: 4,
        listenedMs: 0,
      }),
    ];
    expect(ctx(history).skippedTrackIds.has("finished")).toBe(false);
  });

  it("does not skip completed tracks with partial listen telemetry", () => {
    const history = [
      entry({
        trackId: "migrated",
        played: true,
        positionMs: 0,
        playCount: 6,
        listenedMs: 20_000,
        durationMs: 200_000,
      }),
    ];
    expect(ctx(history).skippedTrackIds.has("migrated")).toBe(false);
  });

  it("still skips abandoned listens under 50%", () => {
    const history = [
      entry({
        trackId: "abandoned",
        played: false,
        positionMs: 20_000,
        durationMs: 200_000,
        listenedMs: 18_000,
      }),
    ];
    expect(ctx(history).skippedTrackIds.has("abandoned")).toBe(true);
  });

  it("skips played=false with zero progress", () => {
    const history = [
      entry({
        trackId: "barely",
        played: false,
        positionMs: 0,
        listenedMs: 0,
        durationMs: 180_000,
      }),
    ];
    expect(ctx(history).skippedTrackIds.has("barely")).toBe(true);
  });
});
