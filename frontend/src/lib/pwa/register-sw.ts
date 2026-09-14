// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Service worker registration and auto-update. Scoped to real web contexts
// only. Wails desktop and mobile hosts never register a worker, the embedded
// WebView serves the bundle directly and a worker would only add failure
// modes there.

import { hasWailsHost, isWailsMobile } from "$lib/config/runtime";

const UPDATE_CHECK_MS = 30 * 60 * 1000;
const DEFERRED_POLL_MS = 15_000;
const RELOAD_FLAG_PREFIX = "melovian-sw-activated:";

type UpdateGuards = {
  /** When true a pending activation is deferred instead of reloading. */
  isPlaying?: () => boolean;
  onUpdateReady?: () => void;
};

export function pwaSupported(): boolean {
  if (typeof window === "undefined" || !("serviceWorker" in navigator)) {
    return false;
  }
  if (import.meta.env.DEV) return false;
  const protocol = window.location?.protocol ?? "";
  if (protocol !== "http:" && protocol !== "https:") return false;
  if (hasWailsHost() || isWailsMobile()) return false;
  return true;
}

export function registerServiceWorker(guards: UpdateGuards = {}): void {
  if (!pwaSupported()) return;
  const base = import.meta.env.BASE_URL || "/";
  const swUrl = `${base.endsWith("/") ? base : `${base}/`}sw.js`;

  let refreshing = false;
  let deferTimer: number | undefined;
  // controllerchange also fires on first install. Only reload when the page
  // itself asked a waiting worker to take over.
  let activationRequested = false;

  const applyOrDefer = (waiting: ServiceWorker | null) => {
    if (!waiting) return;
    if (guards.isPlaying?.()) {
      guards.onUpdateReady?.();
      // Poll so the update lands shortly after playback stops instead of
      // waiting for the next half-hour update check or a relaunch.
      if (deferTimer === undefined) {
        deferTimer = window.setInterval(() => {
          if (!registration.waiting) {
            window.clearInterval(deferTimer);
            deferTimer = undefined;
            return;
          }
          applyOrDefer(registration.waiting);
        }, DEFERRED_POLL_MS);
      }
      return;
    }
    activationRequested = true;
    waiting.postMessage({ type: "SKIP_WAITING" });
  };

  let registration: ServiceWorkerRegistration;

  const onControllerChange = () => {
    if (refreshing || !activationRequested) return;
    refreshing = true;
    // Guard against reload thrash if the new worker immediately finds yet
    // another update.
    try {
      const flag = `${RELOAD_FLAG_PREFIX}${swUrl}`;
      if (sessionStorage.getItem(flag) === "1") return;
      sessionStorage.setItem(flag, "1");
    } catch {
      /* storage unavailable, still reload */
    }
    window.location.reload();
  };

  navigator.serviceWorker.addEventListener("controllerchange", () => {
    onControllerChange();
  });

  void navigator.serviceWorker
    .register(swUrl, { scope: base })
    .then((reg) => {
      registration = reg;
      if (registration.waiting) applyOrDefer(registration.waiting);
      registration.addEventListener("updatefound", () => {
        const installing = registration.installing;
        if (!installing) return;
        installing.addEventListener("statechange", () => {
          if (
            installing.state === "installed" &&
            navigator.serviceWorker.controller
          ) {
            applyOrDefer(registration.waiting);
          }
        });
      });

      const check = () => {
        // A worker deferred while audio played gets another chance here, so
        // an update applied during a track lands as soon as playback stops.
        applyOrDefer(registration.waiting);
        void registration.update().catch(() => {});
      };
      check();
      window.addEventListener("online", check);
      window.setInterval(check, UPDATE_CHECK_MS);
    })
    .catch(() => {
      // Registration failures must never affect playback or the shell
    });
}
