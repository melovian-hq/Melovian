// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  vi,
  type Mock,
} from "vitest";
import { defaultMetadataEnhancementSettings } from "./metadata-enhancement-settings";
import {
  artworkNeedsEnhancement,
  clearMetadataEnhancementCache,
  clearMetadataEnhancementMemoryCache,
  enhanceAlbumArtwork,
  enhanceArtistArtwork,
  enhanceTrackArtwork,
  metadataAlbumNamesMatch,
  metadataNamesMatch,
  normalizeAlbumMetadataName,
  normalizeMetadataName,
  upscaleItunesArtwork,
} from "./metadata-enhancement";

const ART_URL =
  "https://is1-ssl.mzstatic.com/image/thumb/Features/v4/test/100x100bb.jpg";
const ART_URL_600 =
  "https://is1-ssl.mzstatic.com/image/thumb/Features/v4/test/600x600bb.jpg";

function itunesPayload(results: unknown[]) {
  return new Response(
    JSON.stringify({ resultCount: results.length, results }),
    {
      status: 200,
      headers: { "Content-Type": "application/json" },
    },
  );
}

function parseItunesUrl(url: string) {
  const parsed = new URL(url);
  return {
    term: parsed.searchParams.get("term"),
    entity: parsed.searchParams.get("entity"),
    limit: parsed.searchParams.get("limit"),
  };
}

describe("metadata-enhancement helpers", () => {
  it("normalizes artist names for matching", () => {
    expect(normalizeMetadataName("The Beatles")).toBe("beatles");
    expect(normalizeMetadataName("AC/DC")).toBe("ac dc");
    expect(normalizeMetadataName("Монеточка")).toBe("монеточка");
    expect(normalizeMetadataName("ATL")).toBe("atl");
  });

  it("matches cyrillic artist names", () => {
    expect(metadataNamesMatch("Монеточка", "монеточка")).toBe(true);
    expect(metadataNamesMatch("ATL", "atl")).toBe(true);
  });

  it("matches cyrillic library names to latin itunes names", () => {
    expect(metadataNamesMatch("Монеточка", "Monetochka")).toBe(true);
    expect(
      metadataAlbumNamesMatch(
        "Необратимые последствия",
        "Neobratimye posledstviya",
      ),
    ).toBe(true);
  });

  it("matches equivalent artist names", () => {
    expect(metadataNamesMatch("Metallica", "metallica")).toBe(true);
    expect(metadataNamesMatch("Pink Floyd", "Floyd, Pink")).toBe(false);
    expect(metadataNamesMatch("Radiohead", "Radio Head")).toBe(false);
  });

  it("matches names with punctuation differences", () => {
    expect(metadataNamesMatch("Guns N' Roses", "Guns N Roses")).toBe(true);
  });

  it("matches album names with edition suffixes", () => {
    expect(normalizeAlbumMetadataName("Abbey Road (2019 Mix)")).toBe(
      "abbey road",
    );
    expect(metadataAlbumNamesMatch("Abbey Road", "Abbey Road (2019 Mix)")).toBe(
      true,
    );
    expect(
      metadataAlbumNamesMatch(
        "The Dark Side of the Moon",
        "The Dark Side of the Moon",
      ),
    ).toBe(true);
  });

  it("detects missing resolved artwork urls", () => {
    expect(artworkNeedsEnhancement(undefined)).toBe(true);
    expect(artworkNeedsEnhancement("")).toBe(true);
    expect(artworkNeedsEnhancement("   ")).toBe(true);
    expect(artworkNeedsEnhancement("http://example.com/cover.jpg")).toBe(false);
    expect(artworkNeedsEnhancement("http://example.com/cover.jpg", true)).toBe(
      true,
    );
    expect(artworkNeedsEnhancement("data:image/svg+xml,...")).toBe(true);
    expect(artworkNeedsEnhancement("DATA:image/png;base64,abc")).toBe(true);
  });

  it("upscales iTunes artwork urls", () => {
    expect(upscaleItunesArtwork(ART_URL)).toBe(ART_URL_600);
  });
});

describe("metadata-enhancement iTunes lookup", () => {
  let fetchMock: Mock;
  let itunesQueue: Response[];
  let artworkApiResponse: Response | null;

  function queueItunes(...responses: Response[]) {
    itunesQueue.push(...responses);
  }

  function itunesCalls(): string[] {
    return fetchMock.mock.calls
      .map((call) => String(call[0]))
      .filter((url) => url.includes("itunes.apple.com"));
  }

  function apiCalls(): string[] {
    return fetchMock.mock.calls
      .map((call) => String(call[0]))
      .filter((url) => url.includes("/api/metadata/artwork"));
  }

  beforeEach(() => {
    clearMetadataEnhancementCache();
    itunesQueue = [];
    artworkApiResponse = null;
    fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/api/metadata/artwork")) {
        return artworkApiResponse ?? new Response("", { status: 404 });
      }
      return itunesQueue.shift() ?? new Response("", { status: 500 });
    });
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    clearMetadataEnhancementCache();
    vi.unstubAllGlobals();
  });

  const enabled = {
    ...defaultMetadataEnhancementSettings(),
    enabled: true,
  };

  it("returns null when enhancement is disabled", async () => {
    const url = await enhanceArtistArtwork(
      { id: "a1", name: "Metallica" },
      { ...enabled, enabled: false },
    );
    expect(url).toBeNull();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("returns null when a resolved artwork url is already available", async () => {
    const url = await enhanceArtistArtwork(
      { id: "a1", name: "Metallica", coverArt: "existing" },
      enabled,
      { resolvedSrc: "http://example.com/art.jpg" },
    );
    expect(url).toBeNull();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("uses the backend artwork lookup when it resolves a url", async () => {
    artworkApiResponse = new Response(JSON.stringify({ url: ART_URL_600 }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });

    const url = await enhanceArtistArtwork(
      { id: "a1", name: "Metallica" },
      enabled,
    );

    expect(url).toBe(ART_URL_600);
    expect(apiCalls()).toHaveLength(1);
    expect(itunesCalls()).toHaveLength(0);
  });

  it("falls back to direct iTunes search when the backend lookup fails", async () => {
    queueItunes(
      itunesPayload([
        {
          artistName: "Metallica",
          artworkUrl100: ART_URL,
        },
      ]),
      itunesPayload([]),
    );

    const url = await enhanceArtistArtwork(
      { id: "a1", name: "Metallica" },
      enabled,
    );

    expect(url).toBe(ART_URL_600);
    expect(apiCalls()).toHaveLength(1);
    expect(itunesCalls()).toHaveLength(2);
  });

  it("still looks up artwork when only a cover art id exists", async () => {
    queueItunes(
      itunesPayload([]),
      itunesPayload([
        {
          artistName: "Metallica",
          artworkUrl100: ART_URL,
        },
      ]),
    );

    const url = await enhanceArtistArtwork(
      { id: "a1", name: "Metallica", coverArt: "cover-123" },
      { ...enabled, preferServerArtistArt: false },
    );

    expect(url).toBe(ART_URL_600);
  });

  it("skips iTunes when preferServerArtistArt and Navidrome cover exists", async () => {
    const url = await enhanceArtistArtwork(
      { id: "a1", name: "Metallica", coverArt: "ar-1" },
      enabled,
      { resolvedSrc: "https://example.com/nav-cover.jpg" },
    );
    expect(url).toBeNull();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("enhances when server art exists but failed to resolve", async () => {
    artworkApiResponse = new Response(JSON.stringify({ url: ART_URL_600 }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });

    const url = await enhanceArtistArtwork(
      { id: "a1", name: "Metallica", coverArt: "ar-1" },
      enabled,
      { resolvedSrc: null },
    );

    expect(url).toBe(ART_URL_600);
  });

  it("fetches artist artwork from iTunes with name verification", async () => {
    queueItunes(
      itunesPayload([
        {
          artistName: "Metallica",
        },
      ]),
      itunesPayload([
        {
          artistName: "Wrong Artist",
          artworkUrl100: ART_URL,
        },
        {
          artistName: "Metallica",
          collectionName: "Master of Puppets",
          artworkUrl100: ART_URL,
        },
      ]),
    );

    const url = await enhanceArtistArtwork(
      { id: "artist-metallica", name: "Metallica" },
      enabled,
    );

    expect(url).toBe(ART_URL_600);
    expect(itunesCalls()).toHaveLength(2);
    expect(parseItunesUrl(itunesCalls()[0])).toEqual({
      term: "Metallica",
      entity: "musicArtist",
      limit: "8",
    });
    expect(parseItunesUrl(itunesCalls()[1])).toEqual({
      term: "Metallica",
      entity: "album",
      limit: "12",
    });
  });

  it("rejects artist album results that do not match the requested name", async () => {
    queueItunes(
      itunesPayload([]),
      itunesPayload([
        {
          artistName: "Metal Church",
          collectionName: "Master of Puppets",
          artworkUrl100: ART_URL,
        },
      ]),
    );

    const url = await enhanceArtistArtwork(
      { id: "artist-metallica", name: "Metallica" },
      enabled,
    );

    expect(url).toBeNull();
  });

  it("fetches album artwork when album and artist names match", async () => {
    queueItunes(
      itunesPayload([
        {
          artistName: "Pink Floyd",
          collectionName: "The Dark Side of the Moon (2011 Remaster)",
          artworkUrl100: ART_URL,
        },
      ]),
    );

    const url = await enhanceAlbumArtwork(
      {
        id: "album-dsotm",
        name: "The Dark Side of the Moon",
        artist: "Pink Floyd",
      },
      enabled,
    );

    expect(url).toBe(ART_URL_600);
    const requestUrl = itunesCalls()[0];
    expect(parseItunesUrl(requestUrl)).toEqual({
      term: "Pink Floyd The Dark Side of the Moon",
      entity: "album",
      limit: "10",
    });
  });

  it("rejects album results with a mismatched artist", async () => {
    queueItunes(
      itunesPayload([
        {
          artistName: "Various Artists",
          collectionName: "The Dark Side of the Moon",
          artworkUrl100: ART_URL,
        },
      ]),
    );

    const url = await enhanceAlbumArtwork(
      {
        id: "album-dsotm",
        name: "The Dark Side of the Moon",
        artist: "Pink Floyd",
      },
      enabled,
    );

    expect(url).toBeNull();
  });

  it("uses album artwork for tracks when album lookup succeeds", async () => {
    queueItunes(
      itunesPayload([
        {
          artistName: "Nirvana",
          collectionName: "Nevermind",
          artworkUrl100: ART_URL,
        },
      ]),
      itunesPayload([]),
    );

    const url = await enhanceTrackArtwork(
      {
        id: "track-1",
        title: "Smells Like Teen Spirit",
        artist: "Nirvana",
        album: "Nevermind",
      },
      enabled,
    );

    expect(url).toBe(ART_URL_600);
    expect(
      itunesCalls().some((u) => parseItunesUrl(u).entity === "album"),
    ).toBe(true);
  });

  it("falls back to song lookup when album lookup misses", async () => {
    queueItunes(
      itunesPayload([]),
      itunesPayload([
        {
          artistName: "Nirvana",
          trackName: "Smells Like Teen Spirit",
          artworkUrl100: ART_URL,
        },
      ]),
    );

    const url = await enhanceTrackArtwork(
      {
        id: "track-1",
        title: "Smells Like Teen Spirit",
        artist: "Nirvana",
        album: "Nevermind",
      },
      enabled,
    );

    expect(url).toBe(ART_URL_600);
    expect(itunesCalls()).toHaveLength(2);
    expect(parseItunesUrl(itunesCalls()[1])).toEqual({
      term: "Nirvana Smells Like Teen Spirit",
      entity: "song",
      limit: "8",
    });
  });

  it("caches successful lookups in memory", async () => {
    queueItunes(
      itunesPayload([]),
      itunesPayload([
        {
          artistName: "Daft Punk",
          collectionName: "Discovery",
          artworkUrl100: ART_URL,
        },
      ]),
      itunesPayload([
        {
          artistName: "Daft Punk",
          collectionName: "Discovery",
          artworkUrl100: ART_URL,
        },
      ]),
    );

    const artist = { id: "artist-daft", name: "Daft Punk" };
    const first = await enhanceArtistArtwork(artist, enabled);
    const second = await enhanceArtistArtwork(artist, enabled);

    expect(first).toBe(ART_URL_600);
    expect(second).toBe(ART_URL_600);
    expect(itunesCalls()).toHaveLength(2);
  });

  it("persists successful lookups to localStorage", async () => {
    queueItunes(
      itunesPayload([]),
      itunesPayload([
        {
          artistName: "Bjork",
          collectionName: "Homogenic",
          artworkUrl100: ART_URL,
        },
      ]),
    );

    await enhanceArtistArtwork({ id: "artist-bjork", name: "Bjork" }, enabled);
    clearMetadataEnhancementMemoryCache();
    fetchMock.mockClear();

    const url = await enhanceArtistArtwork(
      { id: "artist-bjork", name: "Bjork" },
      enabled,
    );

    expect(url).toBe(ART_URL_600);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("returns null when iTunes responds with an error", async () => {
    // The queue default is an HTTP 500 for any iTunes call.
    const url = await enhanceArtistArtwork(
      { id: "artist-error", name: "Test Artist" },
      enabled,
    );

    expect(url).toBeNull();
  });

  it("retries iTunes lookups after a transient error", async () => {
    queueItunes(
      new Response("", { status: 500 }),
      itunesPayload([
        {
          artistName: "Test Artist",
          collectionName: "Album",
          artworkUrl100: ART_URL,
        },
      ]),
      itunesPayload([
        {
          artistName: "Test Artist",
          artworkUrl100: ART_URL,
        },
      ]),
      itunesPayload([]),
    );

    const first = await enhanceArtistArtwork(
      { id: "artist-retry", name: "Test Artist" },
      enabled,
    );
    expect(first).toBeNull();

    const second = await enhanceArtistArtwork(
      { id: "artist-retry", name: "Test Artist" },
      enabled,
    );
    expect(second).toBe(ART_URL_600);
    expect(itunesCalls()).toHaveLength(4);
  });
});

describe("metadata-enhancement iTunes live", () => {
  const live = process.env.MELOVIAN_LIVE_ITUNES === "1";

  it.skipIf(!live)(
    "fetches real artwork for a well-known artist",
    async () => {
      clearMetadataEnhancementCache();
      const url = await enhanceArtistArtwork(
        { id: "live-beatles", name: "The Beatles" },
        { ...defaultMetadataEnhancementSettings(), enabled: true },
      );

      expect(typeof url).toBe("string");
      expect(url).toMatch(/^https:\/\//);
      expect(url).toMatch(/bb\.(jpg|png)/);
    },
    15_000,
  );

  it.skipIf(!live)(
    "fetches real album artwork",
    async () => {
      clearMetadataEnhancementCache();
      const url = await enhanceAlbumArtwork(
        {
          id: "live-abbey-road",
          name: "Abbey Road",
          artist: "The Beatles",
        },
        { ...defaultMetadataEnhancementSettings(), enabled: true },
      );

      expect(typeof url).toBe("string");
      expect(url).toMatch(/^https:\/\//);
    },
    15_000,
  );
});
