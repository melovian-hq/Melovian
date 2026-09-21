// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/**
 * Tracks artwork URLs that already failed to load. Servers report
 * artistImageUrl and coverArt ids even when no image exists upstream, so
 * every render of a missing artist fires a request that 404s and lands in
 * the console as an error. Remembering the misses keeps repeat renders
 * silent and lets the pixel fallback show immediately.
 *
 * The set is bounded and mirrored into sessionStorage so a reload does not
 * re-probe every known-missing image. It clears when the connection store
 * reports a recovered outage, since failures recorded while the server was
 * unreachable are not real 404s.
 */

import { StorageKeys } from "$lib/brand";
import { createBoundedSet } from "$lib/core/bounded-cache";

const BROKEN_MAX_ENTRIES = 800;

const broken = createBoundedSet<string>(BROKEN_MAX_ENTRIES);
let restored = false;

function restore(): void {
  if (restored) return;
  restored = true;
  try {
    const raw = sessionStorage.getItem(StorageKeys.brokenArtwork);
    if (!raw) return;
    const urls: unknown = JSON.parse(raw);
    if (!Array.isArray(urls)) return;
    for (const url of urls.slice(0, BROKEN_MAX_ENTRIES)) {
      if (typeof url === "string" && url) broken.add(url);
    }
  } catch {
    /* sessionStorage unavailable or corrupt payload */
  }
}

function persist(): void {
  try {
    sessionStorage.setItem(
      StorageKeys.brokenArtwork,
      JSON.stringify([...broken]),
    );
  } catch {
    /* storage full or unavailable */
  }
}

function trackable(url: string): boolean {
  return !url.startsWith("data:") && !url.startsWith("blob:");
}

export function isArtworkBroken(url: string | null | undefined): boolean {
  if (!url || !trackable(url)) return false;
  restore();
  return broken.has(url);
}

export function markArtworkBroken(url: string | null | undefined): void {
  if (!url || !trackable(url)) return;
  restore();
  if (broken.has(url)) return;
  broken.add(url);
  persist();
}

export function clearBrokenArtwork(): void {
  restored = true;
  broken.clear();
  try {
    sessionStorage.removeItem(StorageKeys.brokenArtwork);
  } catch {
    /* storage unavailable */
  }
}
