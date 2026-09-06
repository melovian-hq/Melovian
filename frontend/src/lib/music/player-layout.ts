// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { PlayerLayout } from "$lib/config/music.svelte";

export function isVideoPlaybackPath(pathname: string): boolean {
  return pathname.startsWith("/play/");
}

export function isNowPlayingPath(pathname: string): boolean {
  return pathname === "/music/now-playing";
}

export function playerLayoutForPath(
  _pathname: string,
  currentLayout: PlayerLayout,
): PlayerLayout {
  if (currentLayout === "dismissed") return "dismissed";
  return "full";
}
