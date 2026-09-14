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

  it("matches the real API shapes the bootstrap chain parses", async () => {
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
          SongIDs: ["s1", "s2"],
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
          PlayCount: 5,
          ContentType: "audio/wav",
          Suffix: "wav",
        },
        {
          ID: "s2",
          Title: "Other",
          AlbumID: "al1",
          Album: "Album",
          ArtistID: "ar1",
          Artist: "Artist",
          Track: 2,
          Duration: 90,
          Year: 2020,
          Genre: "Rock",
          CoverArt: "al1",
          BitRate: 128,
          Starred: false,
          PlayCount: 2,
          ContentType: "audio/wav",
          Suffix: "wav",
        },
      ],
      playlists: [
        {
          ID: "pl1",
          Name: "Mix",
          Comment: "",
          SongIDs: ["s1", "s2"],
          Created: "2025-01-01T00:00:00Z",
          Changed: "2025-01-02T00:00:00Z",
          Public: true,
          Owner: "demo",
        },
      ],
      genres: [{ Name: "Rock", SongCount: 2, AlbumCount: 1 }],
    };

    const realFetch = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("catalog.json")) {
        return new Response(JSON.stringify(catalog), {
          status: 200,
          headers: { "Content-Type": "application/json" },
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

    // getActiveInstance() probes for a top-level id. A wrapped
    // { instance } payload leaves activeId null and forces /setup.
    const activeRes = await fetch("/api/instances/active");
    const active = await activeRes.json();
    expect(active.id).toBe("demo-instance");
    expect(active.serverUrl).toBe("fake://melovian-demo");

    const localActive = await (
      await fetch("/api/local-libraries/active")
    ).json();
    expect(localActive.id).toBeUndefined();

    const sources = await (await fetch("/api/sources/status")).json();
    expect(sources.mode).toBe("subsonic");
    expect(sources.activeInstanceId).toBe("demo-instance");

    const stats = await (await fetch("/api/music/library-stats")).json();
    expect(stats.songCount).toBe(2);
    expect(stats.albumCount).toBe(1);
    expect(stats.artistCount).toBe(1);

    const playlists = await (await fetch("/api/music/playlists")).json();
    expect(playlists.playlists).toHaveLength(1);
    expect(playlists.playlists[0].trackCount).toBe(2);
    expect(playlists.playlists[0].coverArtIds).toContain("al1");

    const detail = await (await fetch("/api/music/playlists/pl1")).json();
    expect(detail.id).toBe("pl1");
    expect(detail.tracks).toHaveLength(2);
    expect(detail.tracks[0].trackId).toBe("s1");

    const favorites = await (await fetch("/api/music/favorites")).json();
    expect(favorites.items).toHaveLength(1);
    expect(favorites.items[0].trackId).toBe("s1");

    const history = await (await fetch("/api/music/history")).json();
    expect(history.items.length).toBeGreaterThan(0);

    const resume = await (await fetch("/api/music/resume")).json();
    expect(resume.items.length).toBe(1);
    expect(resume.items[0].played).toBe(false);
    expect(resume.items[0].positionMs).toBeGreaterThan(0);

    const batch = await (await fetch("/api/music/batch?ids=s1")).json();
    expect(batch.favorites).toBeUndefined();
  });
});
