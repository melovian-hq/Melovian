// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  continuousQueueTarget,
  continuousRefillCount,
  createLibraryPoolState,
  filterUniqueTracks,
  migrateContinuousMode,
  pullLibraryTracks,
  resetLibraryPool,
} from "./continuous-pool";
import type { SubsonicAlbum, SubsonicSong } from "$lib/subsonic/types";

const song = (id: string): SubsonicSong => ({ id, title: id });
const album = (id: string): SubsonicAlbum => ({ id, name: id });

describe("continuous-pool", () => {
  it("migrates legacy randomRadio boolean", () => {
    expect(migrateContinuousMode({ randomRadio: true })).toBe("random");
    expect(migrateContinuousMode({ continuousMode: "personal" })).toBe(
      "personal",
    );
    expect(migrateContinuousMode({})).toBe("off");
  });

  it("targets queue size from settings with unlimited soft cap", () => {
    expect(continuousQueueTarget(250)).toBe(250);
    expect(continuousQueueTarget(0)).toBe(500);
  });

  it("computes refill batches within remaining slots", () => {
    expect(continuousRefillCount(20, 500, "random")).toBeGreaterThan(0);
    expect(continuousRefillCount(500, 500, "library")).toBe(0);
    expect(continuousRefillCount(0, 100, "library")).toBe(100);
  });

  it("targets upcoming tracks rather than total queue length", () => {
    // A queue of 25 with only 4 tracks still upcoming has drained below the
    // soft target and must top back up.
    expect(continuousRefillCount(25, 500, "random", 4)).toBe(21);
    // Upcoming already at the soft target means no refill.
    expect(continuousRefillCount(40, 500, "random", 25)).toBe(0);
    // Refills never grow the queue past the configured cap.
    expect(continuousRefillCount(490, 500, "random", 4)).toBe(10);
  });

  it("filters unique tracks against existing ids", () => {
    const unique = filterUniqueTracks(
      [song("a"), song("b"), song("a")],
      new Set(["b"]),
    );
    expect(unique.map((t) => t.id)).toEqual(["a"]);
  });

  it("pulls library tracks from album pages and resets when exhausted", async () => {
    const state = createLibraryPoolState();
    let calls = 0;
    const tracks = await pullLibraryTracks(
      state,
      {
        getAlbumList2: async () => {
          calls += 1;
          if (calls > 2) return [];
          return [album(`al-${calls}`)];
        },
        getAlbumSongs: async (albumId) => [
          song(`${albumId}-1`),
          song(`${albumId}-2`),
        ],
      },
      3,
      new Set(),
      () => 0.5,
    );
    expect(tracks.length).toBe(3);
    expect(state.seenTrackIds.size).toBe(3);

    resetLibraryPool(state);
    expect(state.seenTrackIds.size).toBe(0);
    expect(state.exhaustedPasses).toBe(1);
  });
});
