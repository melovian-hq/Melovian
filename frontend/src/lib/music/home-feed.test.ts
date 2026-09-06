// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import type { SlimPersonalMix } from "$lib/music/mix-storage";
import type {
  ListenEntry,
  MusicPlaylist,
  ServerPlaylist,
  SubsonicAlbum,
  SubsonicArtist,
} from "$lib/subsonic/types";
import {
  albumsMatchingArtist,
  becauseYouListened,
  buildHomeShortcuts,
  greetingForHour,
  homeHasLibraryContent,
  homePlaylists,
  jumpBackInAlbums,
  takeUnusedAlbums,
} from "./home-feed";

function listen(
  partial: Partial<ListenEntry> & { trackId: string },
): ListenEntry {
  return {
    trackTitle: "Track",
    artistName: "Artist",
    albumId: "alb-1",
    albumTitle: "Album",
    positionMs: 0,
    durationMs: 180000,
    played: true,
    playCount: 1,
    listenedMs: 120000,
    lastPlayedAt: "2026-08-15T12:00:00Z",
    coverArtId: "cover-1",
    ...partial,
  };
}

function album(id: string, name: string, artist = "Artist"): SubsonicAlbum {
  return { id, name, artist, coverArt: `art-${id}` };
}

function mix(id: string, title: string): SlimPersonalMix {
  return {
    id,
    title,
    subtitle: "Made for you",
    tracks: [],
    trackIds: ["t1"],
    coverArtId: `mix-${id}`,
    gradient: "linear-gradient(#000, #111)",
  };
}

function playlist(id: string, name: string): MusicPlaylist {
  return {
    id,
    name,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
    trackCount: 4,
    coverArtIds: [`pl-${id}`],
  };
}

const emptyFeed = {
  resumeTracks: [] as ListenEntry[],
  listenHistory: [] as ListenEntry[],
  recentAlbums: [] as SubsonicAlbum[],
  frequentAlbums: [] as SubsonicAlbum[],
  recommendations: [] as SubsonicAlbum[],
  favoriteAlbums: [] as SubsonicAlbum[],
  favoriteArtists: [] as SubsonicArtist[],
  favoriteTrackCount: 0,
  personalMixes: [] as SlimPersonalMix[],
  playlists: [] as MusicPlaylist[],
  serverPlaylists: [] as ServerPlaylist[],
};

describe("greetingForHour", () => {
  it("uses morning, afternoon, and evening windows", () => {
    expect(greetingForHour(5)).toBe("Good morning");
    expect(greetingForHour(11)).toBe("Good morning");
    expect(greetingForHour(12)).toBe("Good afternoon");
    expect(greetingForHour(17)).toBe("Good afternoon");
    expect(greetingForHour(18)).toBe("Good evening");
    expect(greetingForHour(4)).toBe("Good evening");
  });
});

describe("jumpBackInAlbums", () => {
  it("keeps unique albums in resume-then-history order", () => {
    const albums = jumpBackInAlbums(
      [
        listen({ trackId: "1", albumId: "a", albumTitle: "First" }),
        listen({ trackId: "2", albumId: "a", albumTitle: "First" }),
      ],
      [
        listen({
          trackId: "3",
          albumId: "b",
          albumTitle: "Second",
          artistName: "Other",
        }),
      ],
      8,
    );
    expect(albums.map((item) => item.id)).toEqual(["a", "b"]);
    expect(albums[0].name).toBe("First");
  });

  it("skips listens without an album id", () => {
    const albums = jumpBackInAlbums(
      [listen({ trackId: "1", albumId: "", albumTitle: "Loose" })],
      [],
    );
    expect(albums).toEqual([]);
  });
});

describe("becauseYouListened", () => {
  it("picks the latest named artist with at least two matching albums", () => {
    const result = becauseYouListened(
      [
        listen({ trackId: "1", artistName: "Unknown" }),
        listen({ trackId: "2", artistName: "Slowdive", albumId: "s1" }),
      ],
      [
        album("s1", "Souvlaki", "Slowdive"),
        album("s2", "Just for a Day", "Slowdive"),
        album("m1", "Loveless", "My Bloody Valentine"),
      ],
    );
    expect(result?.artist).toBe("Slowdive");
    expect(result?.albums.map((item) => item.id)).toEqual(["s1", "s2"]);
  });

  it("skips bracketed unknown artists", () => {
    expect(
      becauseYouListened(
        [
          listen({
            trackId: "1",
            artistName: "[Unknown Artist]",
            albumId: "u1",
          }),
        ],
        [
          album("u1", "Afscheid", "[Unknown Artist]"),
          album("u2", "Aikaintaite", "[Unknown Artist]"),
        ],
      ),
    ).toBeNull();
  });
});

describe("albumsMatchingArtist", () => {
  it("matches artist names without regard to case", () => {
    const matched = albumsMatchingArtist(
      [
        album("1", "A", "Slowdive"),
        album("2", "B", "slowdive "),
        album("3", "C", "Other"),
      ],
      "SLOWDIVE",
    );
    expect(matched.map((item) => item.id)).toEqual(["1", "2"]);
  });
});

describe("takeUnusedAlbums", () => {
  it("skips ids already shown in earlier shelves", () => {
    const used = new Set(["a"]);
    const taken = takeUnusedAlbums(
      [album("a", "Used"), album("b", "Fresh"), album("c", "Also")],
      used,
      12,
    );
    expect(taken.map((item) => item.id)).toEqual(["b", "c"]);
    expect(used.has("b")).toBe(true);
  });
});

describe("homePlaylists", () => {
  it("lists local playlists before server playlists", () => {
    const items = homePlaylists(
      [playlist("l1", "Local mix")],
      [{ id: "s1", name: "Server mix", songCount: 9, coverArt: "srv" }],
      12,
    );
    expect(items.map((item) => item.href)).toEqual([
      "/music/playlist/l1",
      "/music/server-playlist/s1",
    ]);
    expect(items[1].trackCount).toBe(9);
  });
});

describe("buildHomeShortcuts", () => {
  it("fills a Spotify-style shortcut row from library surfaces", () => {
    const shortcuts = buildHomeShortcuts({
      ...emptyFeed,
      favoriteTrackCount: 12,
      personalMixes: [
        mix("discover", "Discover Weekly"),
        mix("daily", "Daily Mix"),
      ],
      resumeTracks: [
        listen({ trackId: "t1", albumId: "alb-9", albumTitle: "Souvlaki" }),
      ],
      playlists: [playlist("pl1", "Night drives")],
      favoriteArtists: [{ id: "art-1", name: "Slowdive", coverArt: "c" }],
      recentAlbums: [album("new-1", "Newest")],
    });
    expect(shortcuts.map((item) => item.kind)).toEqual([
      "favorites",
      "mix",
      "album",
      "playlist",
      "artist",
      "history",
      "album",
      "mix",
    ]);
    expect(shortcuts[0].href).toBe("/music/favorites");
    expect(shortcuts[1].href).toBe("/music/mix/discover");
  });

  it("omits empty surfaces instead of padding with blanks", () => {
    const shortcuts = buildHomeShortcuts({
      ...emptyFeed,
      personalMixes: [mix("daily", "Daily Mix")],
    });
    expect(shortcuts).toHaveLength(1);
    expect(shortcuts[0].title).toBe("Daily Mix");
  });
});

describe("homeHasLibraryContent", () => {
  it("is false for a connected but empty library", () => {
    expect(homeHasLibraryContent(emptyFeed)).toBe(false);
  });

  it("is true when any shelf has items", () => {
    expect(
      homeHasLibraryContent({ ...emptyFeed, recentAlbums: [album("1", "A")] }),
    ).toBe(true);
  });
});
