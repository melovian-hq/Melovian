// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { mergeMixSettings } from "./mix-settings";

describe("mix-settings radio tuning", () => {
  it("merges personal radio tuning fields", () => {
    const merged = mergeMixSettings({
      personalRadioRecencyHours: 3,
      personalRadioColdStartPlays: 8,
      radioExploreBonus: 1.1,
      flowAlbumLookback: 5,
    });

    expect(merged.personalRadioRecencyHours).toBe(3);
    expect(merged.personalRadioColdStartPlays).toBe(8);
    expect(merged.radioExploreBonus).toBe(1.1);
    expect(merged.flowAlbumLookback).toBe(5);
  });

  it("clamps invalid radio tuning values", () => {
    const merged = mergeMixSettings({
      personalRadioRecencyHours: 99,
      radioExploreBonus: 5,
      flowAlbumLookback: 1,
    });

    expect(merged.personalRadioRecencyHours).toBe(24);
    expect(merged.radioExploreBonus).toBe(2);
    expect(merged.flowAlbumLookback).toBe(2);
  });
});
