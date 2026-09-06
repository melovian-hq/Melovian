// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import type { SubsonicSong } from "$lib/subsonic";
import type { SmartPlaylistDraft } from "./types";
import { getSmartField } from "./fields";

const SWEEP_CHARS = "abcdefghijklmnopqrstuvwxyz0123456789".split("");
const SEARCH_LIMIT = 250;

function mergeSongs(
  target: Map<string, SubsonicSong>,
  songs: SubsonicSong[],
): void {
  for (const song of songs) {
    if (!song.id) continue;
    target.set(song.id, song);
  }
}

function collectSearchTerms(draft: SmartPlaylistDraft): string[] {
  const terms = new Set<string>();

  function walkRules(group: SmartPlaylistDraft["root"]): void {
    for (const rule of group.rules) {
      if (typeof rule.value !== "string") continue;
      const trimmed = rule.value.trim();
      if (!trimmed || trimmed.startsWith("/")) continue;
      const field = getSmartField(rule.field);
      if (!field) continue;
      if (
        field.id === "title" ||
        field.id === "artist" ||
        field.id === "album" ||
        field.id === "genre" ||
        field.id === "albumartist"
      ) {
        terms.add(trimmed);
      }
    }
    for (const child of group.groups) walkRules(child);
  }

  walkRules(draft.root);
  return [...terms];
}

function collectGenreTerms(draft: SmartPlaylistDraft): string[] {
  const genres = new Set<string>();

  function walk(group: SmartPlaylistDraft["root"]): void {
    for (const rule of group.rules) {
      if (rule.field !== "genre") continue;
      if (typeof rule.value !== "string") continue;
      const value = rule.value.trim();
      if (value) genres.add(value);
    }
    for (const child of group.groups) walk(child);
  }

  walk(draft.root);
  return [...genres];
}

export async function collectSmartPlaylistCandidates(
  library: MusicLibraryAdapter,
  draft: SmartPlaylistDraft,
): Promise<SubsonicSong[]> {
  const byId = new Map<string, SubsonicSong>();

  const terms = collectSearchTerms(draft);
  if (terms.length === 0) {
    terms.push("a", "e", "i", "o", "u");
  }

  await Promise.all(
    terms.map(async (term) => {
      const result = await library.search3(term, SEARCH_LIMIT).catch(() => ({
        artists: [],
        albums: [],
        songs: [],
      }));
      mergeSongs(byId, result.songs);
    }),
  );

  const genres = collectGenreTerms(draft);
  for (const genre of genres) {
    const songs = await library
      .getSongsByGenre(genre, SEARCH_LIMIT, 0)
      .catch(() => []);
    mergeSongs(byId, songs);
  }

  if (byId.size < 200) {
    const random = await library.getRandomSongs(500).catch(() => []);
    mergeSongs(byId, random);
  }

  if (byId.size < 400) {
    for (const char of SWEEP_CHARS) {
      const result = await library.search3(char, SEARCH_LIMIT).catch(() => ({
        artists: [],
        albums: [],
        songs: [],
      }));
      mergeSongs(byId, result.songs);
      if (byId.size >= 2500) break;
    }
  }

  return [...byId.values()];
}
