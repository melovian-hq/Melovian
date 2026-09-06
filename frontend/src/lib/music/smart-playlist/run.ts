// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import type { SubsonicSong } from "$lib/subsonic";
import { collectSmartPlaylistCandidates } from "./collect-candidates";
import { songToSmartTrack } from "./context";
import { trackMatchesSmartPlaylist } from "./evaluate";
import { validateSmartPlaylistDraft } from "./validate";
import type { SmartPlaylistDraft } from "./types";

function sortTracks(tracks: SubsonicSong[], sort: string): SubsonicSong[] {
  const desc = sort.startsWith("-");
  const key = sort.replace(/^[+-]/, "");

  const copy = [...tracks];
  copy.sort((a, b) => {
    switch (key) {
      case "random":
        return Math.random() - 0.5;
      case "title":
        return a.title.localeCompare(b.title);
      case "artist":
        return (a.artist ?? "").localeCompare(b.artist ?? "");
      case "year":
        return (a.year ?? 0) - (b.year ?? 0);
      default:
        return a.title.localeCompare(b.title);
    }
  });

  return desc ? copy.reverse() : copy;
}

export async function runSmartPlaylist(
  draft: SmartPlaylistDraft,
  library: MusicLibraryAdapter,
): Promise<SubsonicSong[]> {
  const errors = validateSmartPlaylistDraft(draft);
  if (errors.length > 0) {
    throw new Error(errors[0]?.message ?? "Invalid smart playlist");
  }

  const candidates = await collectSmartPlaylistCandidates(library, draft);
  const matched = candidates.filter((song) =>
    trackMatchesSmartPlaylist(songToSmartTrack(song), draft.root),
  );

  let sorted = sortTracks(matched, draft.sort.trim() || "+random");

  if (draft.limitPercent !== null && draft.limitPercent > 0) {
    const limit = Math.max(
      1,
      Math.floor((sorted.length * draft.limitPercent) / 100),
    );
    sorted = sorted.slice(0, limit);
  } else if (draft.limit !== null && draft.limit > 0) {
    sorted = sorted.slice(0, draft.limit);
  }

  return sorted;
}
