// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// IndexedDB media cache. Holds the current and next track as full blobs so
// playback survives flaky or dead networks. Policy gates prefetching on
// bandwidth (saveData, effectiveType) so it never competes with an active
// stream on a constrained link.
//
// URL pinning: the audio engine compares stream URL strings to decide whether
// the standby element already holds a track, so resolvePlaybackUrl returns a
// stable URL per track for the session. A cached blob is only picked when it
// was stored before the track's first resolution. Network-failure paths can
// still reach the blob through cachedTrackUrl.

import { isInternetRadioTrack, type QueueTrack } from "$lib/subsonic";
import { isLocalTrackId } from "$lib/music/source.svelte";
import { getActiveInstanceId } from "$lib/features/instances/context";
import { logger } from "$lib/core/logger";

const DB_NAME = "melovian-media-cache";
const DB_VERSION = 1;
const STORE = "tracks";
const MAX_ENTRIES = 16;
const MAX_TOTAL_BYTES = 256 * 1024 * 1024;
const MAX_ENTRY_BYTES = 80 * 1024 * 1024;
const QUOTA_FRACTION = 0.2;

type CacheEntry = {
  key: string;
  trackId: string;
  blob: Blob;
  size: number;
  lastUsed: number;
};

let dbPromise: Promise<IDBDatabase> | null = null;
let dbHandle: IDBDatabase | null = null;
const pins = new Map<string, string>();
const objectUrls = new Map<string, string>();
const knownKeys = new Set<string>();
const inflight = new Map<string, Promise<void>>();
let keysWarmed = false;

function cacheKey(track: QueueTrack): string {
  const instance = getActiveInstanceId() || "default";
  return `${instance}:${track.id}`;
}

export function isCacheableTrack(
  track: QueueTrack | null | undefined,
): boolean {
  if (!track?.id) return false;
  if (isInternetRadioTrack(track)) return false;
  if (isLocalTrackId(track.id)) return false;
  return true;
}

/** True when a background fetch will not hurt an active stream. */
export function bandwidthAllowsPrefetch(): boolean {
  if (typeof navigator === "undefined") return false;
  if (navigator.onLine === false) return false;
  const conn = (
    navigator as Navigator & {
      connection?: { saveData?: boolean; effectiveType?: string };
    }
  ).connection;
  if (conn?.saveData) return false;
  const type = conn?.effectiveType;
  if (type === "slow-2g" || type === "2g") return false;
  return true;
}

function openDb(): Promise<IDBDatabase> {
  if (dbPromise) return dbPromise;
  dbPromise = new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, DB_VERSION);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(STORE)) {
        db.createObjectStore(STORE, { keyPath: "key" });
      }
    };
    req.onsuccess = () => {
      dbHandle = req.result;
      resolve(req.result);
    };
    req.onerror = () => {
      dbPromise = null;
      reject(req.error ?? new Error("indexedDB open failed"));
    };
  });
  return dbPromise;
}

function txDone(tx: IDBTransaction): Promise<void> {
  return new Promise((resolve, reject) => {
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
    tx.onabort = () => reject(tx.error);
  });
}

async function idbGet(key: string): Promise<CacheEntry | null> {
  const db = await openDb();
  return new Promise((resolve, reject) => {
    const req = db.transaction(STORE, "readonly").objectStore(STORE).get(key);
    req.onsuccess = () => resolve((req.result as CacheEntry) ?? null);
    req.onerror = () => reject(req.error);
  });
}

async function idbPut(entry: CacheEntry): Promise<void> {
  const db = await openDb();
  const tx = db.transaction(STORE, "readwrite");
  tx.objectStore(STORE).put(entry);
  await txDone(tx);
  knownKeys.add(entry.key);
}

async function idbAllKeys(): Promise<
  { key: string; size: number; lastUsed: number }[]
> {
  const db = await openDb();
  return new Promise((resolve, reject) => {
    const req = db.transaction(STORE, "readonly").objectStore(STORE).getAll();
    req.onsuccess = () =>
      resolve(
        ((req.result as CacheEntry[]) ?? []).map((e) => ({
          key: e.key,
          size: e.size,
          lastUsed: e.lastUsed,
        })),
      );
    req.onerror = () => reject(req.error);
  });
}

async function idbDelete(key: string): Promise<void> {
  const db = await openDb();
  const tx = db.transaction(STORE, "readwrite");
  tx.objectStore(STORE).delete(key);
  await txDone(tx);
  knownKeys.delete(key);
  const url = objectUrls.get(key);
  if (url) {
    URL.revokeObjectURL(url);
    objectUrls.delete(key);
  }
}

async function evictIfNeeded(keepKeys: Set<string>): Promise<void> {
  let quotaLimit = MAX_TOTAL_BYTES;
  try {
    const estimate = await navigator.storage?.estimate?.();
    if (estimate?.quota && estimate.quota > 0) {
      quotaLimit = Math.min(
        quotaLimit,
        Math.floor(estimate.quota * QUOTA_FRACTION),
      );
    }
  } catch {
    /* estimate is best-effort */
  }

  const all = (await idbAllKeys()).sort((a, b) => a.lastUsed - b.lastUsed);
  let total = all.reduce((sum, e) => sum + e.size, 0);
  for (const entry of all) {
    if (all.length <= MAX_ENTRIES && total <= quotaLimit) break;
    if (keepKeys.has(entry.key)) {
      // Never evict what playback currently depends on, but still count it
      // against the byte budget.
      continue;
    }
    await idbDelete(entry.key);
    total -= entry.size;
    all.splice(all.indexOf(entry), 1);
    if (all.length <= MAX_ENTRIES && total <= quotaLimit) break;
  }
}

function objectUrlFor(key: string, blob: Blob): string {
  const existing = objectUrls.get(key);
  if (existing) return existing;
  const url = URL.createObjectURL(blob);
  objectUrls.set(key, url);
  return url;
}

/**
 * resolvePlaybackUrl returns a session-stable playable URL for a track. When
 * the blob is already cached it pins a blob: URL, otherwise it pins the
 * network URL and kicks off a background fill so later plays hit the cache.
 */
export function resolvePlaybackUrl(
  track: QueueTrack,
  networkUrl: string,
): string {
  const key = cacheKey(track);
  const pinned = pins.get(key);
  if (pinned) return pinned;
  const existing = objectUrls.get(key);
  if (existing) {
    pins.set(key, existing);
    return existing;
  }
  pins.set(key, networkUrl);
  void ensureCached(track, networkUrl);
  return networkUrl;
}

/**
 * cachedTrackUrl returns a blob: URL when the track is in the cache. Failure
 * paths call this directly since the session pin may point at a dead network
 * URL.
 */
export async function cachedTrackUrl(
  track: QueueTrack,
): Promise<string | null> {
  if (!isCacheableTrack(track)) return null;
  const key = cacheKey(track);
  try {
    const entry = await idbGet(key);
    if (!entry) return null;
    entry.lastUsed = Date.now();
    void idbPut(entry).catch(() => {});
    return objectUrlFor(key, entry.blob);
  } catch {
    return null;
  }
}

/** ensureCached fetches the full track into IndexedDB when policy allows. */
export function ensureCached(track: QueueTrack, url: string): Promise<void> {
  if (!isCacheableTrack(track)) return Promise.resolve();
  if (!url || url.startsWith("blob:")) return Promise.resolve();
  const key = cacheKey(track);
  if (knownKeys.has(key)) return Promise.resolve();
  const pending = inflight.get(key);
  if (pending) return pending;
  if (!bandwidthAllowsPrefetch()) return Promise.resolve();

  const task = (async () => {
    try {
      if (!keysWarmed) {
        keysWarmed = true;
        const all = await idbAllKeys();
        for (const e of all) knownKeys.add(e.key);
        if (knownKeys.has(key)) return;
      }
      const response = await fetch(url);
      if (!response.ok) return;
      const blob = await response.blob();
      if (blob.size === 0 || blob.size > MAX_ENTRY_BYTES) return;
      const entry: CacheEntry = {
        key,
        trackId: track.id,
        blob,
        size: blob.size,
        lastUsed: Date.now(),
      };
      await idbPut(entry);
      await evictIfNeeded(new Set([key, ...pins.keys()]));
    } catch (err) {
      logger.debug(
        `media cache fill failed for ${track.id}`,
        { error: String(err) },
        "media-cache",
      );
    } finally {
      inflight.delete(key);
    }
  })();
  inflight.set(key, task);
  return task;
}

/** dropTrack removes a cached blob, used when a server or track is deleted. */
export async function dropCachedTrack(track: QueueTrack): Promise<void> {
  try {
    await idbDelete(cacheKey(track));
    pins.delete(cacheKey(track));
  } catch {
    /* best effort */
  }
}

/** For tests and diagnostics. */
export async function mediaCacheStats(): Promise<{
  entries: number;
  bytes: number;
}> {
  try {
    const all = await idbAllKeys();
    return {
      entries: all.length,
      bytes: all.reduce((s, e) => s + e.size, 0),
    };
  } catch {
    return { entries: 0, bytes: 0 };
  }
}

export function resetMediaCacheForTests(): void {
  for (const url of objectUrls.values()) URL.revokeObjectURL(url);
  objectUrls.clear();
  pins.clear();
  knownKeys.clear();
  inflight.clear();
  keysWarmed = false;
  dbHandle?.close();
  dbHandle = null;
  dbPromise = null;
}
