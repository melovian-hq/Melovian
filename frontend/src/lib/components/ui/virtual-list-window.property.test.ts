// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import { queuePanelListHeight, virtualWindow } from "./virtual-list-window";

describe("virtual-list-window property", () => {
  it("never produces an empty window while items and viewport exist", () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 1, max: 500 }),
        fc.integer({ min: 20, max: 80 }),
        fc.integer({ min: 0, max: 50_000 }),
        fc.integer({ min: 40, max: 900 }),
        fc.integer({ min: 0, max: 12 }),
        (itemCount, itemHeight, scrollTop, viewportHeight, overscan) => {
          const win = virtualWindow({
            itemCount,
            itemHeight,
            scrollTop,
            viewportHeight,
            overscan,
          });
          expect(win.startIndex).toBeGreaterThanOrEqual(0);
          expect(win.startIndex).toBeLessThan(itemCount);
          expect(win.endIndex).toBeGreaterThan(win.startIndex);
          expect(win.endIndex).toBeLessThanOrEqual(itemCount);
          expect(win.scrollTop).toBeLessThanOrEqual(
            Math.max(0, win.totalHeight - viewportHeight),
          );
          expect(win.offsetY).toBe(win.startIndex * itemHeight);
        },
      ),
      { numRuns: 200 },
    );
  });

  it("clamps absurd scroll positions after the list shrinks", () => {
    const before = virtualWindow({
      itemCount: 100,
      itemHeight: 52,
      scrollTop: 4800,
      viewportHeight: 360,
    });
    expect(before.startIndex).toBeGreaterThan(50);

    const after = virtualWindow({
      itemCount: 5,
      itemHeight: 52,
      scrollTop: before.scrollTop,
      viewportHeight: 360,
    });
    expect(after.startIndex).toBeLessThan(5);
    expect(after.endIndex).toBeGreaterThan(after.startIndex);
    expect(after.endIndex - after.startIndex).toBeGreaterThan(0);
  });
});

describe("queuePanelListHeight property", () => {
  it("stays within usable bounds", () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 0, max: 400 }),
        fc.integer({ min: 200, max: 2000 }),
        (queueLength, viewportHeight) => {
          const height = queuePanelListHeight(queueLength, viewportHeight);
          expect(height).toBeGreaterThanOrEqual(220);
          expect(height).toBeLessThanOrEqual(
            Math.max(Math.floor(viewportHeight * 0.55), 280),
          );
        },
      ),
      { numRuns: 100 },
    );
  });
});
