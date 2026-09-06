// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/**
 * Pure virtual-list window math. Kept free of DOM so property tests can
 * assert scroll/index clamping without mounting Svelte.
 */
export function virtualWindow(input: {
  itemCount: number;
  itemHeight: number;
  scrollTop: number;
  viewportHeight: number;
  overscan?: number;
}): {
  startIndex: number;
  endIndex: number;
  offsetY: number;
  totalHeight: number;
  scrollTop: number;
} {
  const itemCount = Math.max(0, Math.floor(input.itemCount));
  const itemHeight = Math.max(1, input.itemHeight);
  const overscan = Math.max(0, Math.floor(input.overscan ?? 6));
  const totalHeight = itemCount * itemHeight;
  const viewportHeight = Math.max(0, input.viewportHeight);
  const maxScroll = Math.max(0, totalHeight - viewportHeight);
  const scrollTop = Math.min(Math.max(0, input.scrollTop), maxScroll);

  if (itemCount === 0) {
    return {
      startIndex: 0,
      endIndex: 0,
      offsetY: 0,
      totalHeight: 0,
      scrollTop: 0,
    };
  }

  const startIndex = Math.max(
    0,
    Math.min(itemCount - 1, Math.floor(scrollTop / itemHeight) - overscan),
  );
  const endIndex = Math.min(
    itemCount,
    Math.ceil((scrollTop + viewportHeight) / itemHeight) + overscan,
  );

  return {
    startIndex,
    endIndex: Math.max(startIndex, endIndex),
    offsetY: startIndex * itemHeight,
    totalHeight,
    scrollTop,
  };
}

export function queuePanelListHeight(
  queueLength: number,
  viewportHeight: number,
  itemHeight = 52,
): number {
  const rows = Math.min(Math.max(queueLength, 4), 12);
  const ideal = rows * itemHeight + 8;
  const viewportCap = Math.floor(Math.max(viewportHeight, 0) * 0.55);
  return Math.min(Math.max(ideal, 220), Math.max(viewportCap, 280));
}
