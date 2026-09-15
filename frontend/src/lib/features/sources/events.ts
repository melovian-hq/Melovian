// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { eventSocket } from "$lib/core/events/ws.svelte";
import { bustLibraryCache } from "$lib/music/api";
import { music } from "$lib/config/music.svelte";
import { sources } from "./store.svelte";

interface NavidromeScanStatus {
  scanning?: boolean;
  count?: number;
}

const REFRESH_DEBOUNCE_MS = 2000;

// bindSourceEvents mirrors upstream source events (Navidrome SSE bridged
// over the app websocket as source.<id>.<event>) into cache invalidation
// and home data refreshes. Returns an unsubscribe for teardown.
export function bindSourceEvents(): () => void {
  let scanning = false;
  let refreshTimer = 0;

  const scheduleRefresh = () => {
    // Remote events are irrelevant while the user browses a local-only view.
    if (sources.mode === "local") return;
    if (refreshTimer) clearTimeout(refreshTimer);
    refreshTimer = window.setTimeout(() => {
      refreshTimer = 0;
      void bustLibraryCache()
        .then(() => music.refreshHome())
        .catch(() => {});
    }, REFRESH_DEBOUNCE_MS);
  };

  const unsubs = [
    eventSocket.on("source.navidrome.scanStatus", (event) => {
      const payload = event.payload as NavidromeScanStatus;
      const now = payload?.scanning === true;
      // Refresh once when a scan finishes, not on every progress tick.
      if (scanning && !now) scheduleRefresh();
      scanning = now;
    }),
    eventSocket.on("source.navidrome.refreshResource", () => {
      scheduleRefresh();
    }),
    eventSocket.on("source.navidrome.serverStart", () => {
      scheduleRefresh();
    }),
  ];

  return () => {
    for (const unsub of unsubs) unsub();
    if (refreshTimer) {
      clearTimeout(refreshTimer);
      refreshTimer = 0;
    }
  };
}
