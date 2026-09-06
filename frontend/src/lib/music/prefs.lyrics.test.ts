// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, beforeEach } from "vitest";
import {
  clampLyricsPanelPosition,
  clampLyricsPanelSize,
  loadLyricsPanelPosition,
  loadLyricsPanelSize,
  saveLyricsPanelPosition,
  saveLyricsPanelSize,
} from "./prefs";

describe("lyrics panel prefs", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("persists position and size", () => {
    saveLyricsPanelPosition({ x: 120, y: 80 });
    saveLyricsPanelSize({ width: 400, height: 500 });

    expect(loadLyricsPanelPosition()).toEqual({ x: 120, y: 80 });
    expect(loadLyricsPanelSize()).toEqual({ width: 400, height: 500 });
  });

  it("clamps position and size to viewport", () => {
    const size = clampLyricsPanelSize({ width: 4000, height: 4000 }, 800, 600);
    expect(size.width).toBeLessThanOrEqual(600);
    expect(size.height).toBeLessThanOrEqual(510);

    const position = clampLyricsPanelPosition(
      { x: 900, y: 900 },
      300,
      200,
      800,
      600,
    );
    expect(position.x).toBeLessThanOrEqual(500);
    expect(position.y).toBeLessThanOrEqual(400);
  });

  it("rejects corrupt stored panel geometry", () => {
    localStorage.setItem("mel-lyrics-panel-position", '{"x":"nope"}');
    localStorage.setItem("mel-lyrics-panel-size", '{"width":null}');
    expect(loadLyricsPanelPosition()).toBeNull();
    expect(loadLyricsPanelSize()).toBeNull();
  });
});
