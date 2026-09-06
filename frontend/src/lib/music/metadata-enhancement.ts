// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { createBoundedMap } from "$lib/core/bounded-cache";
import type { MetadataEnhancementSettings } from "./metadata-enhancement-settings";

interface ITunesItem {
  artistName?: string;
  collectionName?: string;
  trackName?: string;
  artworkUrl100?: string;
}

const MEMORY_CACHE_MAX_ENTRIES = 400;

const memoryCache = createBoundedMap<string, string | null>(
  MEMORY_CACHE_MAX_ENTRIES,
);
const inflight = new Map<string, Promise<string | null>>();
const CACHE_PREFIX = "mel-meta-art:";
const CACHE_TTL_MS = 7 * 24 * 60 * 60 * 1000;

const CYRILLIC = /[\u0400-\u04FF]/;

const RU_TO_LAT: Record<string, string> = {
  а: "a",
  б: "b",
  в: "v",
  г: "g",
  д: "d",
  е: "e",
  ё: "e",
  ж: "zh",
  з: "z",
  и: "i",
  й: "y",
  к: "k",
  л: "l",
  м: "m",
  н: "n",
  о: "o",
  п: "p",
  р: "r",
  с: "s",
  т: "t",
  у: "u",
  ф: "f",
  х: "kh",
  ц: "ts",
  ч: "ch",
  ш: "sh",
  щ: "shch",
  ъ: "",
  ы: "y",
  ь: "",
  э: "e",
  ю: "yu",
  я: "ya",
};

export function normalizeMetadataName(value: string): string {
  return value
    .toLowerCase()
    .normalize("NFD")
    .replace(/\p{M}/gu, "")
    .replace(/^the\s+/, "")
    .replace(/[^\p{L}\p{N}]+/gu, " ")
    .trim();
}

function transliterateRussian(value: string): string {
  let out = "";
  for (const char of value) {
    out += RU_TO_LAT[char] ?? char;
  }
  return out.replace(/\s+/g, " ").trim();
}

function metadataComparable(value: string): string[] {
  const normalized = normalizeMetadataName(value);
  if (!normalized) return [];
  const variants = new Set<string>([normalized]);
  if (CYRILLIC.test(normalized)) {
    const latin = transliterateRussian(normalized);
    if (latin) variants.add(latin);
  }
  return [...variants];
}

function comparableNamesMatch(left: string, right: string): boolean {
  if (!left || !right) return false;
  if (left === right) return true;

  const shorter = left.length <= right.length ? left : right;
  const longer = left.length > right.length ? left : right;
  if (!longer.includes(shorter)) return false;

  return shorter.length / longer.length >= 0.85;
}

export function normalizeAlbumMetadataName(value: string): string {
  const stripped = value
    .replace(/\s*[-–—]\s*.+$/u, "")
    .replace(/\s*\([^)]*\)\s*$/g, "")
    .replace(/\s*\[[^\]]*\]\s*$/g, "")
    .trim();
  return normalizeMetadataName(stripped);
}

export function metadataAlbumNamesMatch(
  expected: string,
  candidate: string,
): boolean {
  return metadataNamesMatch(
    normalizeAlbumMetadataName(expected),
    normalizeAlbumMetadataName(candidate),
  );
}

export function metadataNamesMatch(
  expected: string,
  candidate: string,
): boolean {
  const leftVariants = metadataComparable(expected);
  const rightVariants = metadataComparable(candidate);
  if (leftVariants.length === 0 || rightVariants.length === 0) return false;

  for (const left of leftVariants) {
    for (const right of rightVariants) {
      if (comparableNamesMatch(left, right)) return true;
    }
  }
  return false;
}

const DATA_URL_PATTERN = /^data:/i;

export function artworkNeedsEnhancement(
  resolvedSrc?: string | null,
  primaryLoadFailed = false,
): boolean {
  const src = resolvedSrc?.trim();
  // Treat generated fallback data URLs (blank/random pixel) as missing so the
  // component still attempts to fetch enhanced artwork instead of leaving the
  // placeholder in place.
  if (src && !primaryLoadFailed && !DATA_URL_PATTERN.test(src)) return false;
  return true;
}

export function upscaleItunesArtwork(url: string, size = 600): string {
  return url.replace(/\d+x\d+bb/, `${size}x${size}bb`);
}

function cacheKey(kind: string, id: string): string {
  return `${kind}:${id}`;
}

function readPersistentCache(key: string): string | null | undefined {
  if (typeof localStorage === "undefined") return undefined;
  try {
    const raw = localStorage.getItem(`${CACHE_PREFIX}${key}`);
    if (!raw) return undefined;
    const parsed = JSON.parse(raw) as { url: string | null; at: number };
    if (Date.now() - parsed.at > CACHE_TTL_MS) {
      localStorage.removeItem(`${CACHE_PREFIX}${key}`);
      return undefined;
    }
    return parsed.url;
  } catch {
    return undefined;
  }
}

function writePersistentCache(key: string, url: string | null): void {
  if (typeof localStorage === "undefined") return;
  try {
    localStorage.setItem(
      `${CACHE_PREFIX}${key}`,
      JSON.stringify({ url, at: Date.now() }),
    );
  } catch {
    /* storage full or unavailable */
  }
}

async function searchItunes(
  params: Record<string, string>,
): Promise<ITunesItem[]> {
  const url = new URL("https://itunes.apple.com/search");
  for (const [key, value] of Object.entries(params)) {
    url.searchParams.set(key, value);
  }
  if (CYRILLIC.test(params.term ?? "")) {
    url.searchParams.set("country", "RU");
  }

  const response = await fetch(url.toString());
  if (!response.ok) {
    throw new Error(`iTunes search failed: ${response.status}`);
  }
  const payload = (await response.json()) as { results?: ITunesItem[] };
  return payload.results ?? [];
}

async function resolveCached(
  key: string,
  loader: () => Promise<string | null>,
): Promise<string | null> {
  if (memoryCache.has(key)) return memoryCache.get(key) ?? null;

  const persisted = readPersistentCache(key);
  if (persisted !== undefined) {
    memoryCache.set(key, persisted);
    return persisted;
  }

  const pending = inflight.get(key);
  if (pending) return pending;

  const promise = loader()
    .then((url) => {
      memoryCache.set(key, url);
      writePersistentCache(key, url);
      inflight.delete(key);
      return url;
    })
    .catch(() => {
      // Do not cache failures (network errors, rate limits, etc.) so flaky
      // conditions in server/web deployments can recover on the next attempt.
      inflight.delete(key);
      return null;
    });

  inflight.set(key, promise);
  return promise;
}

export function clearMetadataEnhancementMemoryCache(): void {
  memoryCache.clear();
  inflight.clear();
}

export function clearMetadataEnhancementCache(): void {
  clearMetadataEnhancementMemoryCache();
  if (typeof localStorage === "undefined") return;
  const keys: string[] = [];
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i);
    if (key?.startsWith(CACHE_PREFIX)) keys.push(key);
  }
  for (const key of keys) localStorage.removeItem(key);
}

export async function enhanceArtistArtwork(
  artist: {
    id: string;
    name: string;
    coverArt?: string;
    artistImageUrl?: string;
  },
  settings: MetadataEnhancementSettings,
  options: {
    resolvedSrc?: string | null;
    primaryLoadFailed?: boolean;
  } = {},
): Promise<string | null> {
  if (!settings.enabled || !settings.artists) return null;
  if (
    settings.preferServerArtistArt &&
    (artist.artistImageUrl?.trim() || artist.coverArt?.trim()) &&
    !options.primaryLoadFailed
  ) {
    return null;
  }
  if (
    !artworkNeedsEnhancement(
      options.resolvedSrc,
      options.primaryLoadFailed ?? false,
    )
  ) {
    return null;
  }

  const key = cacheKey("artist", artist.id);
  return resolveCached(key, async () => {
    const [artistResults, albumResults] = await Promise.all([
      searchItunes({
        term: artist.name,
        entity: "musicArtist",
        limit: "8",
      }),
      searchItunes({
        term: artist.name,
        entity: "album",
        limit: "12",
      }),
    ]);

    const directMatch = artistResults.find(
      (item) =>
        item.artworkUrl100 &&
        metadataNamesMatch(artist.name, item.artistName ?? ""),
    );
    if (directMatch?.artworkUrl100) {
      return upscaleItunesArtwork(directMatch.artworkUrl100);
    }

    const albumMatch = albumResults.find(
      (item) =>
        item.artworkUrl100 &&
        metadataNamesMatch(artist.name, item.artistName ?? ""),
    );

    return albumMatch?.artworkUrl100
      ? upscaleItunesArtwork(albumMatch.artworkUrl100)
      : null;
  });
}

export async function enhanceAlbumArtwork(
  album: {
    id: string;
    name: string;
    artist?: string;
    coverArt?: string;
  },
  settings: MetadataEnhancementSettings,
  options: {
    resolvedSrc?: string | null;
    primaryLoadFailed?: boolean;
  } = {},
): Promise<string | null> {
  if (!settings.enabled || !settings.albums) return null;
  if (
    !artworkNeedsEnhancement(
      options.resolvedSrc,
      options.primaryLoadFailed ?? false,
    )
  ) {
    return null;
  }

  const key = cacheKey("album", album.id);
  return resolveCached(key, async () => {
    const term = [album.artist, album.name].filter(Boolean).join(" ");
    const results = await searchItunes({
      term,
      entity: "album",
      limit: "10",
    });

    const match = results.find((item) => {
      if (!item.artworkUrl100) return false;
      if (!metadataAlbumNamesMatch(album.name, item.collectionName ?? "")) {
        return false;
      }
      if (album.artist?.trim()) {
        return metadataNamesMatch(album.artist, item.artistName ?? "");
      }
      return true;
    });

    return match?.artworkUrl100
      ? upscaleItunesArtwork(match.artworkUrl100)
      : null;
  });
}

export async function enhanceTrackArtwork(
  track: {
    id: string;
    title: string;
    artist?: string;
    album?: string;
    coverArt?: string;
    albumId?: string;
  },
  settings: MetadataEnhancementSettings,
  options: {
    resolvedSrc?: string | null;
    primaryLoadFailed?: boolean;
  } = {},
): Promise<string | null> {
  if (!settings.enabled || !settings.tracks) return null;
  if (
    !artworkNeedsEnhancement(
      options.resolvedSrc,
      options.primaryLoadFailed ?? false,
    )
  ) {
    return null;
  }

  const key = cacheKey("track", track.id);
  return resolveCached(key, async () => {
    const albumPromise = track.album?.trim()
      ? enhanceAlbumArtwork(
          {
            id: track.albumId ?? `track-album:${track.id}`,
            name: track.album,
            artist: track.artist,
          },
          settings,
          options,
        )
      : Promise.resolve(null);

    const songPromise = (async () => {
      const term = [track.artist, track.title].filter(Boolean).join(" ");
      const results = await searchItunes({
        term,
        entity: "song",
        limit: "8",
      });

      const match = results.find((item) => {
        if (!item.artworkUrl100) return false;
        if (!metadataNamesMatch(track.title, item.trackName ?? ""))
          return false;
        if (track.artist?.trim()) {
          return metadataNamesMatch(track.artist, item.artistName ?? "");
        }
        return true;
      });

      return match?.artworkUrl100
        ? upscaleItunesArtwork(match.artworkUrl100)
        : null;
    })();

    const [albumArt, songArt] = await Promise.all([albumPromise, songPromise]);
    return albumArt ?? songArt;
  });
}
