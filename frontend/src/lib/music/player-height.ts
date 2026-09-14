// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/**
 * Svelte action that measures a fixed player bar and writes the space it
 * actually covers at the bottom of the viewport into --jb-player-bar-height
 * on .app-root. The static token underestimates the rendered bar (progress
 * rail, borders, safe-area padding), which lets fill-layout pages slide
 * their last pixels under the player.
 *
 * withBottomOffset reports viewport distance below the bar too. Bars that
 * float above other chrome (the mini player) need it; bars whose offset is
 * already accounted elsewhere (the slim player sits above the bottom nav)
 * must not include it or the inset is counted twice.
 */
export function reportPlayerBarHeight(
  node: HTMLElement,
  options: { bottomOffset?: boolean } = {},
): { destroy(): void } {
  const apply = () => {
    const root = document.querySelector<HTMLElement>(".app-root");
    if (!root) return;
    const rect = node.getBoundingClientRect();
    const covered = options.bottomOffset
      ? window.innerHeight - rect.top
      : rect.height;
    if (covered <= 0) return;
    root.style.setProperty("--jb-player-bar-height", `${Math.ceil(covered)}px`);
  };

  apply();
  const observer = new ResizeObserver(apply);
  observer.observe(node);

  return {
    destroy() {
      observer.disconnect();
      document
        .querySelector<HTMLElement>(".app-root")
        ?.style.removeProperty("--jb-player-bar-height");
    },
  };
}
