// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  buildM3UFromTracks,
  matchM3UEntryToSong,
  parseM3U,
} from "./playlist-m3u";
import type { SubsonicSong } from "$lib/subsonic";

const sample = `#EXTM3U
#PLAYLIST:Road Trip
#EXTINF:213,Queen - Bohemian Rhapsody
https://music.example/rest/stream.view?id=song-1
#EXTINF:180,Unknown Track
/path/to/file.flac
`;

describe("playlist-m3u", () => {
  it("parses standard m3u playlists", () => {
    const playlist = parseM3U(sample);
    expect(playlist.name).toBe("Road Trip");
    expect(playlist.entries).toHaveLength(2);
    expect(playlist.entries[0]?.trackId).toBe("song-1");
    expect(playlist.entries[0]?.artist).toBe("Queen");
    expect(playlist.entries[0]?.title).toBe("Bohemian Rhapsody");
  });

  it("builds m3u from tracks", () => {
    const tracks: SubsonicSong[] = [
      {
        id: "song-1",
        title: "Song",
        artist: "Band",
        duration: 200,
      },
    ];
    const body = buildM3UFromTracks(tracks, {
      serverUrl: "https://music.example",
      version: "1.16.1",
      clientName: "Melovian",
    });
    expect(body).toContain("#EXTM3U");
    expect(body).toContain("#EXTINF:200,Band - Song");
    expect(body).toContain("song-1");
  });

  it("matches entries to library songs", () => {
    const library: SubsonicSong[] = [
      { id: "song-1", title: "Bohemian Rhapsody", artist: "Queen" },
    ];
    const playlist = parseM3U(sample);
    const match = matchM3UEntryToSong(playlist.entries[0]!, library);
    expect(match?.id).toBe("song-1");
  });
});
