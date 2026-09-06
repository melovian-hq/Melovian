// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { PlayerLayout } from "$lib/config/music.svelte";

/** Extra gap between toasts and the player chrome. */
const TOAST_GAP = "var(--jb-space-4)";

export type PlayerBottomInsetOptions = {
  /** Mobile bottom tab bar is present (stacks under slim/full player). */
  mobileNav?: boolean;
};

export function playerBottomInset(
  playerVisible: boolean,
  playerLayout: PlayerLayout,
  options?: PlayerBottomInsetOptions,
): string {
  const safe = "env(safe-area-inset-bottom, 0px)";

  if (options?.mobileNav) {
    return `calc(var(--jb-bottom-chrome-height, 3.5rem) + ${TOAST_GAP} + ${safe})`;
  }

  if (!playerVisible) {
    return `calc(var(--jb-space-6) + ${safe})`;
  }
  if (playerLayout === "mini") {
    return `calc(var(--jb-player-bar-height, 7rem) + ${TOAST_GAP} + ${safe})`;
  }
  return `calc(var(--jb-player-bar-height, 5.5rem) + ${TOAST_GAP} + ${safe})`;
}
