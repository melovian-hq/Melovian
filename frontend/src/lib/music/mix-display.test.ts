// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it } from "vitest";
import {
  loadMixDisplay,
  MIX_DISPLAY_KEY,
  saveMixDisplay,
  subscribeMixDisplay,
  type MixDisplayStyle,
} from "./mix-display";

describe("mix-display", () => {
  afterEach(() => {
    localStorage.removeItem(MIX_DISPLAY_KEY);
  });

  it("defaults to cards", () => {
    expect(loadMixDisplay()).toBe("cards");
  });

  it("persists discs preference", () => {
    saveMixDisplay("discs");
    expect(loadMixDisplay()).toBe("discs");
    expect(localStorage.getItem(MIX_DISPLAY_KEY)).toBe("discs");
  });

  it("rejects adversarial invalid values", () => {
    localStorage.setItem(MIX_DISPLAY_KEY, "hexagons");
    expect(loadMixDisplay()).toBe("cards");
    localStorage.setItem(MIX_DISPLAY_KEY, "");
    expect(loadMixDisplay()).toBe("cards");
    localStorage.setItem(MIX_DISPLAY_KEY, JSON.stringify({ evil: true }));
    expect(loadMixDisplay()).toBe("cards");
  });

  it("round-trips both styles", () => {
    const styles: MixDisplayStyle[] = ["cards", "discs"];
    for (const style of styles) {
      saveMixDisplay(style);
      expect(loadMixDisplay()).toBe(style);
    }
  });

  it("notifies subscribers when display style changes", () => {
    const seen: MixDisplayStyle[] = [];
    const stop = subscribeMixDisplay((style) => {
      seen.push(style);
    });
    saveMixDisplay("discs");
    saveMixDisplay("cards");
    stop();
    expect(seen).toEqual(["discs", "cards"]);
  });
});
