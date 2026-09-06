// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  buildPlaylistReorderParams,
  movePlaylistSong,
  playlistOrderChanged,
} from "./server-playlist-editing";

describe("server playlist editing helpers", () => {
  it("detects unchanged playlist order", () => {
    expect(playlistOrderChanged(["a", "b"], ["a", "b"])).toBe(false);
    expect(
      buildPlaylistReorderParams(["a", "b", "c"], ["a", "b", "c"]),
    ).toBeNull();
  });

  it("builds remove-all then add-all params for reorder", () => {
    expect(
      buildPlaylistReorderParams(["a", "b", "c"], ["c", "a", "b"]),
    ).toEqual({
      songIndexToRemove: [2, 1, 0],
      songIdToAdd: ["c", "a", "b"],
    });
  });

  it("rejects reorder when song sets differ", () => {
    expect(() => buildPlaylistReorderParams(["a", "b"], ["a", "c"])).toThrow(
      /same songs/i,
    );
  });

  it("moves a song between indices", () => {
    expect(movePlaylistSong(["a", "b", "c"], 2, 0)).toEqual(["c", "a", "b"]);
    expect(movePlaylistSong(["a", "b", "c"], 0, 2)).toEqual(["b", "c", "a"]);
    expect(movePlaylistSong(["a", "b", "c"], 1, 1)).toEqual(["a", "b", "c"]);
  });
});
