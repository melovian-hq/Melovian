// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Stale chunk recovery. Hashed build output means a deploy leaves old
// clients pointing at files that no longer exist, so a lazy route import
// fails with "Failed to fetch dynamically imported module" and friends.
// The fix is always a reload: the new document references the new
// hashes. Reloading mid-track would cut audio, so while playback is
// active the caller gets "deferred" and can surface a refresh prompt
// instead.

import { APP_NAME, storageKey } from "$lib/brand";
import { toast } from "$lib/ui/toast.svelte";

const RELOAD_FLAG = storageKey("stale-reload");
const RELOAD_GUARD_MS = 60_000;
const RELOAD_DELAY_MS = 250;

export type StaleReloadResult = "reloading" | "deferred" | "limited";

export interface StaleChunkGuards {
  /** When true, defer instead of reloading (audio is playing). */
  isPlaying?: () => boolean;
  /** Test seam for the page reload itself. */
  reload?: () => void;
}

let guards: StaleChunkGuards = {};
let reloadScheduled = false;
let deferredNotified = false;

export function configureStaleChunkHandling(next: StaleChunkGuards) {
  guards = next;
}

/** True when the error is a missing-chunk failure from a stale build. */
export function isStaleChunkError(error: unknown): boolean {
  const name = error instanceof Error ? error.name.toLowerCase() : "";
  const message = (
    error instanceof Error ? error.message : String(error ?? "")
  ).toLowerCase();
  if (name === "chunkloaderror") return true;
  return (
    message.includes("dynamically imported module") ||
    message.includes("importing a module script") ||
    message.includes("module script failed") ||
    message.includes("unable to preload") ||
    message.includes("loading chunk") ||
    message.includes("chunkloaderror")
  );
}

function readReloadFlag(): number {
  try {
    return Number(sessionStorage.getItem(RELOAD_FLAG)) || 0;
  } catch {
    return 0;
  }
}

function writeReloadFlag(ts: number) {
  try {
    sessionStorage.setItem(RELOAD_FLAG, String(ts));
  } catch {
    /* storage unavailable, the in-memory flag still bounds this view */
  }
}

/**
 * Attempts a guarded reload for a stale build. "reloading" means the page
 * is about to reload, "deferred" means playback blocked it for now, and
 * "limited" means a reload already ran within the guard window so the
 * caller should show an error instead of looping.
 */
export function attemptStaleReload(): StaleReloadResult {
  if (reloadScheduled) return "reloading";
  if (guards.isPlaying?.()) return "deferred";
  const now = Date.now();
  if (now - readReloadFlag() < RELOAD_GUARD_MS) return "limited";
  writeReloadFlag(now);
  reloadScheduled = true;
  // A new service worker may already be waiting behind the deferred
  // update path. update() fetches it so its precache lands before the
  // reload, and register-sw.ts activates it when playback allows.
  void navigator.serviceWorker
    ?.getRegistration?.()
    .then((reg) => reg?.update())
    .catch(() => {});
  const go = guards.reload ?? (() => window.location.reload());
  window.setTimeout(go, RELOAD_DELAY_MS);
  return "reloading";
}

function notifyDeferredUpdate() {
  if (deferredNotified) return;
  deferredNotified = true;
  toast.info(
    `${APP_NAME} was updated in the background. Refresh for the latest version.`,
  );
}

/**
 * Vite wraps lazy imports and modulepreload links; failures emit
 * vite:preloadError before the import promise rejects. Preventing default
 * keeps the original error for the route boundary to classify, and the
 * guarded reload runs for chunk failures anywhere, not just routes.
 */
export function installStaleChunkHandlers() {
  if (typeof window === "undefined") return;
  window.addEventListener("vite:preloadError", (event) => {
    event.preventDefault();
    if (attemptStaleReload() === "deferred") notifyDeferredUpdate();
  });
}

/** Test helper. Clears module state between tests. */
export function resetStaleChunkForTests() {
  guards = {};
  reloadScheduled = false;
  deferredNotified = false;
  try {
    sessionStorage.removeItem(RELOAD_FLAG);
  } catch {
    /* ignore */
  }
}
