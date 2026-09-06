// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  gazeFromPointer,
  generatePixelSmileSpec,
  renderPixelSmile,
} from "./pixel-smile";

describe("pixel-smile", () => {
  it("renders all cells for any seed", () => {
    for (const seed of ["melovian", "a", "instance-1", ""]) {
      const spec = generatePixelSmileSpec(seed || "fallback");
      const smile = renderPixelSmile(spec, { x: 1, y: -1 });
      expect(smile.cells).toHaveLength(64);
      expect(smile.cells.every(Boolean)).toBe(true);
    }
  });

  it("computes gaze from pointer", () => {
    const rect = new DOMRect(100, 100, 32, 32);
    expect(gazeFromPointer(rect, 200, 200)).toEqual({ x: 1, y: 1 });
    expect(gazeFromPointer(rect, 116, 116)).toEqual({ x: 0, y: 0 });
  });
});
