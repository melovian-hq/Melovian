// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, beforeEach, afterEach, vi } from "vitest";

describe("static-api", () => {
  beforeEach(() => {
    vi.resetModules();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("installs and serves /api/config in demo mode", async () => {
    const catalog = {
      artists: [
        {
          ID: "ar1",
          Name: "Artist",
          CoverArt: "ar1",
          AlbumIDs: ["al1"],
        },
      ],
      albums: [
        {
          ID: "al1",
          Name: "Album",
          ArtistID: "ar1",
          Artist: "Artist",
          Year: 2020,
          Genre: "Rock",
          CoverArt: "al1",
          SongIDs: ["s1"],
          CreatedAt: "2025-01-01T00:00:00Z",
        },
      ],
      songs: [
        {
          ID: "s1",
          Title: "Song",
          AlbumID: "al1",
          Album: "Album",
          ArtistID: "ar1",
          Artist: "Artist",
          Track: 1,
          Duration: 120,
          Year: 2020,
          Genre: "Rock",
          CoverArt: "al1",
          BitRate: 128,
          Starred: true,
          PlayCount: 1,
          ContentType: "audio/wav",
          Suffix: "wav",
        },
      ],
      playlists: [],
      genres: [{ Name: "Rock", SongCount: 1, AlbumCount: 1 }],
    };

    const realFetch = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("catalog.json")) {
        return new Response(JSON.stringify(catalog), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }
      if (url.includes("silent.wav")) {
        return new Response(new Uint8Array([0, 0, 0, 0]), {
          status: 200,
          headers: { "Content-Type": "audio/wav" },
        });
      }
      return new Response("missing", { status: 404 });
    });

    vi.stubGlobal("fetch", realFetch);
    vi.stubGlobal("window", {
      location: { origin: "http://localhost" },
      fetch: realFetch,
    });

    const { installStaticDemoApi } = await import("./static-api");
    await installStaticDemoApi();

    const configRes = await fetch("/api/config");
    expect(configRes.ok).toBe(true);
    const config = await configRes.json();
    expect(config.demoMode).toBe(true);
    expect(config.fakeCatalog).toBe(true);
    expect(config.capabilities).toContain("browse");

    const mut = await fetch("/api/instances", { method: "POST" });
    expect(mut.status).toBe(403);

    const artists = await fetch("/api/subsonic/rest/getArtists.view");
    expect(artists.ok).toBe(true);
    const body = await artists.json();
    expect(body["subsonic-response"].artists.index.length).toBeGreaterThan(0);
  });
});
