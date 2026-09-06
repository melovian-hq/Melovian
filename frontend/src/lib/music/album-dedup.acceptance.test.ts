// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { albumIdentityKey, dedupeAlbums } from "./album-dedup";
import type { SubsonicAlbum } from "$lib/subsonic/types";

describe("album-dedup acceptance", () => {
  it("accepts merging remote and local duplicates into one shelf entry", () => {
    const remote: SubsonicAlbum = {
      id: "remote-1",
      name: "Random Access Memories",
      artist: "Daft Punk",
      songCount: 13,
      duration: 4400,
    };
    const local: SubsonicAlbum = {
      id: "alb_42",
      name: "Random Access Memories (FLAC)",
      artist: "Daft Punk",
      songCount: 13,
      duration: 4500,
    };
    const other: SubsonicAlbum = {
      id: "remote-2",
      name: "Discovery",
      artist: "Daft Punk",
      songCount: 14,
      duration: 3600,
    };

    expect(albumIdentityKey(remote)).toBe(albumIdentityKey(local));
    const shelf = dedupeAlbums([remote, other, local]);
    expect(shelf).toHaveLength(2);
    expect(shelf.map((a) => a.name)).toContain("Discovery");
    expect(shelf.some((a) => a.id === "alb_42")).toBe(true);
  });
});
