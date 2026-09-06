// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  crossfadeActive,
  defaultPlaybackSettings,
  mergePlaybackSettings,
} from "./playback-settings";

describe("playback settings", () => {
  it("defaults crossfade to disabled and continue playback to disabled", () => {
    expect(defaultPlaybackSettings()).toEqual({
      crossfadeEnabled: false,
      crossfadeDurationSec: 3,
      continuePlaybackOnLaunch: false,
    });
  });

  it("clamps crossfade duration", () => {
    expect(
      mergePlaybackSettings({
        crossfadeEnabled: true,
        crossfadeDurationSec: 99,
      }),
    ).toEqual({
      crossfadeEnabled: true,
      crossfadeDurationSec: 12,
      continuePlaybackOnLaunch: false,
    });
  });

  it("enables crossfade whenever the setting is on", () => {
    expect(
      crossfadeActive(
        {
          crossfadeEnabled: true,
          crossfadeDurationSec: 5,
          continuePlaybackOnLaunch: true,
        },
        true,
      ),
    ).toBe(true);
    expect(
      crossfadeActive(
        {
          crossfadeEnabled: true,
          crossfadeDurationSec: 5,
          continuePlaybackOnLaunch: true,
        },
        false,
      ),
    ).toBe(true);
    expect(
      crossfadeActive(
        {
          crossfadeEnabled: false,
          crossfadeDurationSec: 5,
          continuePlaybackOnLaunch: true,
        },
        false,
      ),
    ).toBe(false);
  });
});
