// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { createDetailCache, DETAIL_CACHE_TTL_MS } from "$lib/core/detail-cache";
import type { MusicLibraryAdapter } from "$lib/music/library-adapter";
import { invalidateArtistInfoCache } from "$lib/music/artist-info-cache";

type AlbumDetail = Awaited<ReturnType<MusicLibraryAdapter["getAlbum"]>>;
type ArtistDetail = Awaited<ReturnType<MusicLibraryAdapter["getArtist"]>>;

const albumCache = createDetailCache<AlbumDetail>(DETAIL_CACHE_TTL_MS, 64);
const artistCache = createDetailCache<ArtistDetail>(DETAIL_CACHE_TTL_MS, 96);

export async function fetchAlbumWithCache(
  library: MusicLibraryAdapter,
  id: string,
  onStale?: (data: AlbumDetail) => void,
): Promise<AlbumDetail> {
  const cached = albumCache.peek(id);
  if (cached && !cached.stale) return cached.value;
  if (cached?.stale) {
    onStale?.(cached.value);
    try {
      const fresh = await library.getAlbum(id);
      albumCache.set(id, fresh);
      return fresh;
    } catch {
      return cached.value;
    }
  }

  const fresh = await library.getAlbum(id);
  albumCache.set(id, fresh);
  return fresh;
}

export async function fetchArtistWithCache(
  library: MusicLibraryAdapter,
  id: string,
  onStale?: (data: ArtistDetail) => void,
): Promise<ArtistDetail> {
  const cached = artistCache.peek(id);
  if (cached && !cached.stale) return cached.value;
  if (cached?.stale) {
    onStale?.(cached.value);
    try {
      const fresh = await library.getArtist(id);
      artistCache.set(id, fresh);
      return fresh;
    } catch {
      return cached.value;
    }
  }

  const fresh = await library.getArtist(id);
  artistCache.set(id, fresh);
  return fresh;
}

export function invalidateAlbumDetailCache(id?: string) {
  if (id) {
    albumCache.delete(id);
    return;
  }
  albumCache.clear();
}

export function invalidateArtistDetailCache(id?: string) {
  if (id) {
    artistCache.delete(id);
    return;
  }
  artistCache.clear();
}

export function resetSubsonicDetailCaches() {
  albumCache.clear();
  artistCache.clear();
  invalidateArtistInfoCache();
}
