// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { bottomNavActiveTab } from "./bottom-nav-active";

describe("bottomNavActiveTab", () => {
  it("marks More when the sidebar drawer is open", () => {
    expect(bottomNavActiveTab("/music", true)).toBe("more");
  });

  it("matches Home for the music root", () => {
    expect(bottomNavActiveTab("/music", false)).toBe("home");
  });

  it("matches Search", () => {
    expect(bottomNavActiveTab("/music/search", false)).toBe("search");
  });

  it("matches Library for albums artists and genres", () => {
    expect(bottomNavActiveTab("/music/albums", false)).toBe("library");
    expect(bottomNavActiveTab("/music/album/abc", false)).toBe("library");
    expect(bottomNavActiveTab("/music/artists", false)).toBe("library");
    expect(bottomNavActiveTab("/music/artist/xyz", false)).toBe("library");
    expect(bottomNavActiveTab("/music/genres", false)).toBe("library");
    expect(bottomNavActiveTab("/music/genre/rock", false)).toBe("library");
  });

  it("matches Playlists for playlist routes", () => {
    expect(bottomNavActiveTab("/music/playlists", false)).toBe("playlists");
    expect(bottomNavActiveTab("/music/playlist/1", false)).toBe("playlists");
    expect(bottomNavActiveTab("/music/server-playlist/2", false)).toBe(
      "playlists",
    );
  });

  it("matches More for overflow destinations", () => {
    expect(bottomNavActiveTab("/settings/profile", false)).toBe("more");
    expect(bottomNavActiveTab("/music/favorites", false)).toBe("more");
    expect(bottomNavActiveTab("/music/history", false)).toBe("more");
    expect(bottomNavActiveTab("/music/now-playing", false)).toBe("more");
    expect(bottomNavActiveTab("/music/mix/daily", false)).toBe("more");
  });
});
