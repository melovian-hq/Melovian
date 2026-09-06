// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  isRemoteCacheableTrack,
  isRemoteCacheableTrackId,
  selectTracksForOfflineDownload,
} from "$lib/music/track-cache";

describe("track-cache helpers", () => {
  it("rejects local library tracks", () => {
    expect(isRemoteCacheableTrackId("trk_abc")).toBe(false);
    expect(isRemoteCacheableTrack({ id: "trk_abc" })).toBe(false);
  });

  it("rejects internet radio and open streams", () => {
    expect(isRemoteCacheableTrackId("open:https://radio.example/live")).toBe(
      false,
    );
    expect(
      isRemoteCacheableTrack({
        id: "open:https://radio.example/live",
        isInternetRadio: true,
      }),
    ).toBe(false);
  });

  it("allows remote subsonic tracks", () => {
    expect(isRemoteCacheableTrackId("song-123")).toBe(true);
    expect(isRemoteCacheableTrack({ id: "song-123" })).toBe(true);
  });

  it("selects only undownloaded remote tracks for offline batch", () => {
    const selected = selectTracksForOfflineDownload(
      [
        { id: "trk_local" },
        { id: "song-1" },
        { id: "song-1" },
        { id: "song-2" },
        { id: "song-3", isInternetRadio: true },
        { id: "song-4" },
      ],
      new Set(["song-2"]),
    );
    expect(selected.map((t) => t.id)).toEqual(["song-1", "song-4"]);
  });
});
