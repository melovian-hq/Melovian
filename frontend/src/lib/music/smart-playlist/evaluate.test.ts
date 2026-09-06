// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { songToSmartTrack } from "./context";
import { trackMatchesSmartPlaylist } from "./evaluate";
import { createEmptySmartPlaylistDraft } from "./compile";

describe("trackMatchesSmartPlaylist", () => {
  it("matches AND groups", () => {
    const draft = createEmptySmartPlaylistDraft();
    draft.root.rules = [
      { id: "1", field: "genre", operator: "contains", value: "Jazz" },
      { id: "2", field: "artist", operator: "contains", value: "Miles" },
    ];
    const track = songToSmartTrack(
      {
        id: "1",
        title: "So What",
        artist: "Miles Davis",
      },
      { genre: "Jazz" },
    );
    expect(trackMatchesSmartPlaylist(track, draft.root)).toBe(true);
  });

  it("matches OR nested groups", () => {
    const draft = createEmptySmartPlaylistDraft();
    draft.root.logic = "any";
    draft.root.rules = [
      { id: "1", field: "artist", operator: "contains", value: "Queen" },
    ];
    draft.root.groups = [
      {
        id: "g1",
        logic: "all",
        rules: [
          { id: "2", field: "title", operator: "contains", value: "live" },
        ],
        groups: [],
      },
    ];
    const track = songToSmartTrack({
      id: "1",
      title: "Studio Cut",
      artist: "Queen",
    });
    expect(trackMatchesSmartPlaylist(track, draft.root)).toBe(true);
  });

  it("supports regex contains rules", () => {
    const draft = createEmptySmartPlaylistDraft();
    draft.root.rules = [
      { id: "1", field: "title", operator: "contains", value: "/live$/i" },
    ];
    const live = songToSmartTrack({ id: "1", title: "Concert live" });
    const studio = songToSmartTrack({ id: "2", title: "Song (studio)" });
    expect(trackMatchesSmartPlaylist(live, draft.root)).toBe(true);
    expect(trackMatchesSmartPlaylist(studio, draft.root)).toBe(false);
  });
});
