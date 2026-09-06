// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  coverStackVariant,
  playlistHref,
  playlistMetaParts,
  playlistTrackCount,
} from "./playlist-display";
import type { MusicPlaylist, ServerPlaylist } from "$lib/subsonic";

describe("playlist-display", () => {
  it("builds hrefs for local and server playlists", () => {
    expect(playlistHref("server", "abc")).toBe("/music/server-playlist/abc");
    expect(playlistHref("local", "xyz")).toBe("/music/playlist/xyz");
  });

  it("maps view modes to cover stack sizes", () => {
    expect(coverStackVariant("list")).toBe("compact");
    expect(coverStackVariant("grid")).toBe("card");
    expect(coverStackVariant("card")).toBe("card");
  });

  it("formats playlist metadata", () => {
    const server: ServerPlaylist = {
      id: "pl-1",
      name: "Road mix",
      songCount: 12,
      duration: 3600,
      owner: "alice",
      public: true,
    };
    expect(playlistTrackCount(server, "server")).toBe(12);
    expect(playlistMetaParts(server, "server")).toEqual([
      "12 tracks",
      "1 hr",
      "alice",
      "Public",
    ]);

    const local: MusicPlaylist = {
      id: "local-1",
      name: "Downloads",
      trackCount: 3,
      createdAt: "",
      updatedAt: "",
    };
    expect(playlistTrackCount(local, "local")).toBe(3);
  });
});
