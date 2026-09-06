// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { isLocalMusicId } from "$lib/music/library-adapter";
import type { SubsonicAlbum } from "$lib/subsonic/types";

function normalizeAlbumText(text: string): string {
  return text
    .toLowerCase()
    .normalize("NFKD")
    .replace(/\p{M}/gu, "")
    .replace(/^the\s+/, "")
    .replace(/[^\p{L}\p{N}]+/gu, " ")
    .trim();
}

function stripAlbumQualityHints(name: string): string {
  return name
    .replace(
      /\s*[[(]?(?:flac|mp3|aac|opus|wav|alac|lossless|hi[- ]?res|24.?bit|cd|vinyl|v0|v2|128|192|256|320)[\])]?\s*/gi,
      " ",
    )
    .replace(/\s+/g, " ")
    .trim();
}

export function albumIdentityKey(
  album: Pick<SubsonicAlbum, "name" | "artist">,
): string {
  const artist = normalizeAlbumText(album.artist ?? "");
  const name = stripAlbumQualityHints(normalizeAlbumText(album.name ?? ""));
  return `${artist}\0${name}`;
}

export function albumQualityScore(album: SubsonicAlbum): number {
  let score = 0;
  const name = (album.name ?? "").toLowerCase();
  if (/\bflac\b|lossless|hi[- ]?res|24.?bit/.test(name)) score += 10_000;
  if (/\bmp3\b|\b128\b|\b192\b|\bv0\b|\bv2\b/.test(name)) score -= 3_000;
  if (isLocalMusicId(album.id)) score += 2_000;
  score += (album.songCount ?? 0) * 10;
  score += Math.min(album.duration ?? 0, 99_999);
  return score;
}

export function dedupeAlbums(albums: SubsonicAlbum[]): SubsonicAlbum[] {
  const best = new Map<string, SubsonicAlbum>();
  const order: string[] = [];
  for (const album of albums) {
    if (!album.id) continue;
    const key = albumIdentityKey(album);
    if (!best.has(key)) order.push(key);
    const prev = best.get(key);
    if (!prev || albumQualityScore(album) > albumQualityScore(prev)) {
      best.set(key, album);
    }
  }
  return order.map((key) => best.get(key)!);
}
