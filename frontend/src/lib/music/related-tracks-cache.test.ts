// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from "vitest";
import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import type { SubsonicSong } from "$lib/subsonic";
import {
  clearRelatedTracksCache,
  getRelatedTracksCached,
  peekRelatedTracks,
} from "./related-tracks-cache";

const track: SubsonicSong = {
  id: "trk_1",
  title: "Song",
  artist: "Artist",
};

function createLibrary(similar: SubsonicSong[] = []): MusicLibraryAdapter {
  return {
    getSimilarSongs: vi.fn().mockResolvedValue(similar),
    getAlbum: vi.fn(),
    getArtist: vi.fn(),
    search3: vi.fn(),
  } as unknown as MusicLibraryAdapter;
}

describe("related-tracks-cache", () => {
  beforeEach(() => {
    clearRelatedTracksCache();
  });

  it("returns cached results on subsequent calls", async () => {
    const related: SubsonicSong[] = [{ id: "trk_2", title: "Other" }];
    const library = createLibrary(related);

    const first = await getRelatedTracksCached(library, track, 24);
    const second = await getRelatedTracksCached(library, track, 24);

    expect(first).toEqual(related);
    expect(second).toEqual(related);
    expect(library.getSimilarSongs).toHaveBeenCalledTimes(1);
  });

  it("peeks cached related tracks", async () => {
    const related: SubsonicSong[] = [{ id: "trk_2", title: "Other" }];
    const library = createLibrary(related);

    expect(peekRelatedTracks(track, 24)).toBeNull();
    await getRelatedTracksCached(library, track, 24);
    expect(peekRelatedTracks(track, 24)).toEqual(related);
  });

  it("clears cached related tracks", async () => {
    const library = createLibrary([{ id: "rel-1", title: "Related" }]);
    await getRelatedTracksCached(library, track, 24);
    expect(peekRelatedTracks(track, 24)?.length).toBe(1);

    clearRelatedTracksCache();
    expect(peekRelatedTracks(track, 24)).toBeNull();
  });
});
