// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { selectTracksForOfflineDownload } from "./track-cache";

describe("offline batch acceptance", () => {
  it("accepts album download plan skipping locals and cached tracks", () => {
    const albumSongs = [
      { id: "trk_1", title: "Local" },
      { id: "remote-1", title: "One" },
      { id: "remote-2", title: "Two" },
      { id: "remote-1", title: "One Dup" },
    ];
    const pending = selectTracksForOfflineDownload(
      albumSongs,
      new Set(["remote-2"]),
    );
    expect(pending.map((t) => t.id)).toEqual(["remote-1"]);
    expect(pending).toHaveLength(1);
  });
});
