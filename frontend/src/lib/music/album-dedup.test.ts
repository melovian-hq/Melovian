// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  albumIdentityKey,
  albumQualityScore,
  dedupeAlbums,
} from "./album-dedup";
import type { SubsonicAlbum } from "$lib/subsonic/types";

function album(
  id: string,
  name: string,
  artist: string,
  extra: Partial<SubsonicAlbum> = {},
): SubsonicAlbum {
  return { id, name, artist, ...extra };
}

describe("albumIdentityKey", () => {
  it("matches albums with the same artist and title", () => {
    const a = album("1", "OK Computer", "Radiohead");
    const b = album("2", "ok computer", "RADIOHEAD");
    expect(albumIdentityKey(a)).toBe(albumIdentityKey(b));
  });

  it("ignores quality hints in the album title", () => {
    const a = album("1", "OK Computer", "Radiohead");
    const b = album("2", "OK Computer (FLAC)", "Radiohead");
    const c = album("3", "OK Computer [MP3 128]", "Radiohead");
    expect(albumIdentityKey(a)).toBe(albumIdentityKey(b));
    expect(albumIdentityKey(a)).toBe(albumIdentityKey(c));
  });
});

describe("dedupeAlbums", () => {
  it("keeps the higher-quality duplicate album", () => {
    const mp3 = album("sub-1", "OK Computer [MP3 128]", "Radiohead", {
      songCount: 12,
      duration: 3200,
    });
    const flac = album("alb_2", "OK Computer (FLAC)", "Radiohead", {
      songCount: 12,
      duration: 3200,
    });
    const result = dedupeAlbums([mp3, flac]);
    expect(result).toHaveLength(1);
    expect(result[0]?.id).toBe("alb_2");
    expect(albumQualityScore(flac)).toBeGreaterThan(albumQualityScore(mp3));
  });

  it("preserves the first-seen order of unique albums", () => {
    const first = album("1", "Album A", "Artist");
    const second = album("2", "Album B", "Artist");
    const dupe = album("3", "album a", "artist");
    const result = dedupeAlbums([first, second, dupe]);
    expect(result.map((item) => item.id)).toEqual(["1", "2"]);
  });

  it("keeps distinct cyrillic albums from the same artist", () => {
    const radio = album("1", "Радио Апокалипсис", "ATL", { songCount: 10 });
    const consequences = album("2", "Необратимые последствия", "ATL", {
      songCount: 12,
    });
    const result = dedupeAlbums([radio, consequences]);
    expect(result).toHaveLength(2);
    expect(result.map((item) => item.id)).toEqual(["1", "2"]);
  });
});
