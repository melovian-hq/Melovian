// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { music } from "$lib/config/music.svelte";

export function bindPlaybackLifecycle(): () => void {
  if (typeof window === "undefined") return () => {};

  const flush = () => {
    music.persistPlaybackSnapshot();
    void music.flushPlaybackState();
  };

  const onVisibilityChange = () => {
    if (document.visibilityState === "hidden") flush();
  };

  window.addEventListener("pagehide", flush);
  window.addEventListener("beforeunload", flush);
  document.addEventListener("visibilitychange", onVisibilityChange);

  return () => {
    window.removeEventListener("pagehide", flush);
    window.removeEventListener("beforeunload", flush);
    document.removeEventListener("visibilitychange", onVisibilityChange);
  };
}
