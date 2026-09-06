// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from "vitest";
import { SubsonicClient } from "./client";
import {
  getAlbum,
  getArtist,
  getArtistInfo,
  getGenres,
  getLyricsForSong,
  getSong,
  search3,
  createServerPlaylist,
  updateServerPlaylist,
  deleteServerPlaylist,
  addSongsToServerPlaylist,
  removeSongFromServerPlaylist,
  setServerPlaylistSongOrder,
} from "./api";

function mockSubsonicResponse(body: Record<string, unknown>) {
  return {
    ok: true,
    status: 200,
    text: async () =>
      JSON.stringify({
        "subsonic-response": {
          status: "ok",
          ...body,
        },
      }),
  };
}

describe("subsonic api mappers", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("maps album songs and transcoded flags", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        mockSubsonicResponse({
          album: {
            id: "a1",
            name: "Album",
            song: [
              { id: "s1", title: "Lossless", suffix: "flac" },
              {
                id: "s2",
                title: "Transcoded",
                suffix: "mp3",
                isTranscoded: true,
              },
            ],
          },
        }),
      ),
    );

    const client = new SubsonicClient();
    const { album, songs } = await getAlbum(client, "a1");

    expect(album.id).toBe("a1");
    expect(songs).toHaveLength(2);
    expect(songs[0]?.transcoded).toBe(false);
    expect(songs[1]?.transcoded).toBe(true);
  });

  it("unwraps a single song object", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        mockSubsonicResponse({
          song: { id: "s1", title: "Only Song", transcoded: true },
        }),
      ),
    );

    const client = new SubsonicClient();
    const song = await getSong(client, "s1");
    expect(song?.id).toBe("s1");
    expect(song?.transcoded).toBe(true);
  });

  it("maps structured song artists for collaborations", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        mockSubsonicResponse({
          song: {
            id: "s1",
            title: "Collab",
            artist: "OG Buda & Artist2",
            artistId: "ar-collab",
            artists: [
              { id: "ar1", name: "OG Buda" },
              { id: "ar2", name: "Artist2" },
            ],
          },
        }),
      ),
    );

    const client = new SubsonicClient();
    const song = await getSong(client, "s1");
    expect(song?.artists).toEqual([
      { id: "ar1", name: "OG Buda" },
      { id: "ar2", name: "Artist2" },
    ]);
  });

  it("aggregates search3 results", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        mockSubsonicResponse({
          searchResult3: {
            artist: { id: "ar1", name: "Artist" },
            album: [{ id: "al1", name: "Album" }],
            song: { id: "s1", title: "Song" },
          },
        }),
      ),
    );

    const client = new SubsonicClient();
    const result = await search3(client, "query", 5);
    expect(result.artists).toHaveLength(1);
    expect(result.albums).toHaveLength(1);
    expect(result.songs).toHaveLength(1);
  });

  it("filters empty genres and sorts by song count", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        mockSubsonicResponse({
          genres: {
            genre: [
              { name: "", songCount: 99 },
              { value: "Rock", songCount: 10 },
              { value: "Jazz", songCount: 25 },
            ],
          },
        }),
      ),
    );

    const client = new SubsonicClient();
    const genres = await getGenres(client);
    expect(genres.map((genre) => genre.name)).toEqual(["Jazz", "Rock"]);
  });

  it("maps artist details and artist image URL", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        mockSubsonicResponse({
          artist: {
            id: "ar1",
            name: "Test Artist",
            coverArt: "cover-1",
            artistImageUrl: "https://example.com/artist.jpg",
          },
          album: [],
        }),
      ),
    );

    const client = new SubsonicClient();
    const result = await getArtist(client, "ar1");
    expect(result.artist.id).toBe("ar1");
    expect(result.artist.coverArt).toBe("cover-1");
    expect(result.artist.artistImageUrl).toBe("https://example.com/artist.jpg");
  });

  it("maps artist info and similar artists", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        mockSubsonicResponse({
          artistInfo: {
            biography: "Bio text",
            lastFmUrl: "https://last.fm/music/Test",
            similarArtist: [
              { id: "ar2", name: "Similar One" },
              { name: "Name Only" },
            ],
          },
        }),
      ),
    );

    const client = new SubsonicClient();
    const info = await getArtistInfo(client, "ar1");
    expect(info.biography).toBe("Bio text");
    expect(info.similarArtists).toHaveLength(2);
    expect(info.similarArtists[0]?.id).toBe("ar2");
  });

  it("dedupes similar artists by id or name", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        mockSubsonicResponse({
          artistInfo: {
            similarArtist: [
              { id: "ar2", name: "Duplicate" },
              { id: "ar2", name: "Duplicate" },
              { name: "Name Only" },
              { name: "Name Only" },
            ],
          },
        }),
      ),
    );

    const client = new SubsonicClient();
    const info = await getArtistInfo(client, "ar1");
    expect(info.similarArtists).toHaveLength(2);
    expect(info.similarArtists.map((artist) => artist.name)).toEqual([
      "Duplicate",
      "Name Only",
    ]);
  });

  it("loads structured lyrics by song id before falling back to artist/title", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(
      mockSubsonicResponse({
        lyricsList: {
          structuredLyrics: {
            synced: true,
            offset: 0,
            displayArtist: "Artist",
            displayTitle: "Song",
            line: [
              { start: 0, value: "Line one" },
              { start: 2500, value: "Line two" },
            ],
          },
        },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const client = new SubsonicClient();
    const lyrics = await getLyricsForSong(client, {
      id: "s1",
      artist: "Artist",
      title: "Song",
    });

    expect(lyrics?.synced).toBe(true);
    expect(lyrics?.lines[1]?.text).toBe("Line two");
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});

describe("server playlist mutations", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("creates a server playlist with POST and repeated songId params", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      mockSubsonicResponse({
        playlist: { id: "pl-1", name: "Road mix", songCount: 2 },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const client = new SubsonicClient();
    const playlist = await createServerPlaylist(client, "Road mix", [
      "s1",
      "s2",
    ]);

    expect(playlist.id).toBe("pl-1");
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(init.method).toBe("POST");
    expect(url).toContain("createPlaylist.view");
    expect(url).toContain("name=Road+mix");
    expect(url).toContain("songId=s1");
    expect(url).toContain("songId=s2");
  });

  it("updates and deletes server playlists", async () => {
    const fetchMock = vi.fn().mockResolvedValue(mockSubsonicResponse({}));
    vi.stubGlobal("fetch", fetchMock);

    const client = new SubsonicClient();
    await updateServerPlaylist(client, {
      playlistId: "pl-1",
      name: "Renamed",
      songIdToAdd: ["s3"],
      songIndexToRemove: [0],
    });
    await deleteServerPlaylist(client, "pl-1");

    expect(fetchMock).toHaveBeenCalledTimes(2);
    const updateUrl = String((fetchMock.mock.calls[0] as [string])[0]);
    expect(updateUrl).toContain("updatePlaylist.view");
    expect(updateUrl).toContain("playlistId=pl-1");
    expect(updateUrl).toContain("name=Renamed");
    expect(updateUrl).toContain("songIdToAdd=s3");
    expect(updateUrl).toContain("songIndexToRemove=0");
    const deleteUrl = String((fetchMock.mock.calls[1] as [string])[0]);
    expect(deleteUrl).toContain("deletePlaylist.view");
    expect(deleteUrl).toContain("id=pl-1");
  });

  it("adds and removes individual tracks", async () => {
    const fetchMock = vi.fn().mockResolvedValue(mockSubsonicResponse({}));
    vi.stubGlobal("fetch", fetchMock);

    const client = new SubsonicClient();
    await addSongsToServerPlaylist(client, "pl-1", ["s1", "s1", "s2"]);
    await removeSongFromServerPlaylist(client, "pl-1", 3);

    const addUrl = String((fetchMock.mock.calls[0] as [string])[0]);
    expect(addUrl).toContain("songIdToAdd=s1");
    expect(addUrl).toContain("songIdToAdd=s2");
    expect(addUrl.match(/songIdToAdd=s1/g)).toHaveLength(1);
    const removeUrl = String((fetchMock.mock.calls[1] as [string])[0]);
    expect(removeUrl).toContain("songIndexToRemove=3");
  });

  it("reorders playlists by clearing and re-adding tracks", async () => {
    const fetchMock = vi.fn().mockResolvedValue(mockSubsonicResponse({}));
    vi.stubGlobal("fetch", fetchMock);

    const client = new SubsonicClient();
    await setServerPlaylistSongOrder(
      client,
      "pl-1",
      ["a", "b", "c"],
      ["c", "a", "b"],
    );

    const url = String((fetchMock.mock.calls[0] as [string])[0]);
    expect(url).toContain("songIndexToRemove=2");
    expect(url).toContain("songIndexToRemove=1");
    expect(url).toContain("songIndexToRemove=0");
    expect(url).toContain("songIdToAdd=c");
    expect(url).toContain("songIdToAdd=a");
    expect(url).toContain("songIdToAdd=b");
  });
});
