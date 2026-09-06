// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it } from "vitest";
import { shouldRestorePlayback } from "./playback-session";
import { clearSavedPlayback, savePlayback } from "./prefs";

describe("shouldRestorePlayback", () => {
  afterEach(() => {
    clearSavedPlayback();
  });

  it("returns false when playback is already active", () => {
    savePlayback({
      trackIds: ["1"],
      queueIndex: 0,
      positionMs: 0,
      shuffle: false,
      autoplay: true,
    });

    expect(
      shouldRestorePlayback({
        playbackRestored: false,
        playing: true,
        queueLength: 1,
        queueIndex: 0,
      }),
    ).toBe(false);
  });

  it("returns false when queue already has tracks", () => {
    savePlayback({
      trackIds: ["1"],
      queueIndex: 0,
      positionMs: 0,
      shuffle: false,
      autoplay: true,
    });

    expect(
      shouldRestorePlayback({
        playbackRestored: false,
        playing: false,
        queueLength: 2,
        queueIndex: 1,
      }),
    ).toBe(false);
  });

  it("returns false when continue playback on launch is disabled", () => {
    savePlayback({
      trackIds: ["1", "2"],
      queueIndex: 1,
      positionMs: 12000,
      shuffle: true,
      autoplay: true,
    });
    localStorage.setItem(
      "mel-playback-settings",
      JSON.stringify({ continuePlaybackOnLaunch: false }),
    );

    expect(
      shouldRestorePlayback({
        playbackRestored: false,
        playing: false,
        queueLength: 0,
        queueIndex: -1,
      }),
    ).toBe(false);
  });

  it("returns true when saved playback exists and session is idle", () => {
    savePlayback({
      trackIds: ["1", "2"],
      queueIndex: 1,
      positionMs: 12000,
      shuffle: true,
      autoplay: true,
    });
    localStorage.setItem(
      "mel-playback-settings",
      JSON.stringify({ continuePlaybackOnLaunch: true }),
    );

    expect(
      shouldRestorePlayback({
        playbackRestored: false,
        playing: false,
        queueLength: 0,
        queueIndex: -1,
      }),
    ).toBe(true);
  });
});
