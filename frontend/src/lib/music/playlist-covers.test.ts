// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from "vitest";

const { fetchServerPlaylist } = vi.hoisted(() => ({
  fetchServerPlaylist: vi.fn(),
}));

vi.mock("$lib/config/music.svelte", () => ({
  music: {
    fetchServerPlaylist,
  },
}));

import {
  clearServerPlaylistCoverCache,
  enrichServerPlaylistCovers,
  loadServerPlaylistCoverIds,
} from "./playlist-covers";

describe("loadServerPlaylistCoverIds", () => {
  beforeEach(() => {
    clearServerPlaylistCoverCache();
    fetchServerPlaylist.mockReset();
  });

  it("uses server coverArt without fetching the full playlist", async () => {
    const ids = await loadServerPlaylistCoverIds("pl-1", "cover-1");
    expect(ids).toEqual(["cover-1"]);
    expect(fetchServerPlaylist).not.toHaveBeenCalled();
  });

  it("fetches playlist tracks only when coverArt is missing", async () => {
    fetchServerPlaylist.mockResolvedValue({
      playlist: { id: "pl-2", name: "Mix" },
      songs: [
        { id: "s1", coverArt: "art-1" },
        { id: "s2", albumId: "alb-2" },
      ],
    });

    const ids = await loadServerPlaylistCoverIds("pl-2");
    expect(ids).toEqual(["art-1", "alb-2"]);
    expect(fetchServerPlaylist).toHaveBeenCalledTimes(1);
  });

  it("enriches a single server cover with more track art", async () => {
    fetchServerPlaylist.mockResolvedValue({
      playlist: { id: "pl-3", name: "Wide" },
      songs: [
        { id: "s1", coverArt: "cover-1" },
        { id: "s2", coverArt: "cover-2" },
        { id: "s3", coverArt: "cover-3" },
      ],
    });

    const ids = await enrichServerPlaylistCovers("pl-3", ["cover-1"]);
    expect(ids).toEqual(["cover-1", "cover-2", "cover-3"]);
    expect(fetchServerPlaylist).toHaveBeenCalledTimes(1);
  });
});
