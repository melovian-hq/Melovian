// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import type { SmartPlaylistDraft } from "./types";
import type { MusicPlaylist, PlaylistTrack } from "$lib/subsonic";

describe("smart playlist persistence acceptance", () => {
  it("accepts create then refresh membership replacement shape", () => {
    const draft: SmartPlaylistDraft = {
      name: "Late Night",
      comment: "",
      public: false,
      sort: "+random",
      limit: 50,
      limitPercent: null,
      root: {
        id: "root",
        logic: "all",
        rules: [
          {
            id: "r1",
            field: "genre",
            operator: "contains",
            value: "Ambient",
          },
        ],
        groups: [],
      },
    };

    const created: MusicPlaylist = {
      id: "pl-1",
      name: draft.name,
      kind: "smart",
      rulesJson: JSON.stringify(draft),
      createdAt: "2026-07-18T00:00:00Z",
      updatedAt: "2026-07-18T00:00:00Z",
      trackCount: 1,
      tracks: [
        {
          trackId: "old-1",
          trackTitle: "Old",
          artistName: "A",
          albumId: "al1",
          albumTitle: "Album",
          durationMs: 1000,
          coverArtId: "c1",
        },
      ],
    };

    const refreshedTracks: PlaylistTrack[] = [
      {
        trackId: "new-1",
        trackTitle: "New Match",
        artistName: "B",
        albumId: "al2",
        albumTitle: "Other",
        durationMs: 2000,
        coverArtId: "c2",
      },
    ];
    const restored = JSON.parse(created.rulesJson!) as SmartPlaylistDraft;
    expect(restored.root.rules[0]?.value).toBe("Ambient");

    const afterRefresh: MusicPlaylist = {
      ...created,
      trackCount: refreshedTracks.length,
      tracks: refreshedTracks,
      updatedAt: "2026-07-18T01:00:00Z",
    };
    expect(afterRefresh.kind).toBe("smart");
    expect(afterRefresh.tracks?.map((t) => t.trackId)).toEqual(["new-1"]);
    expect(afterRefresh.tracks?.map((t) => t.trackId)).not.toContain("old-1");
  });
});
