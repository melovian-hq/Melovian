// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  formatImmersiveBadge,
  isAtmosCapableTrack,
  isImmersiveCodecTrack,
  shouldPreserveImmersiveStream,
} from "./immersive-audio";
import {
  defaultImmersiveAudioSettings,
  mergeImmersiveAudioSettings,
} from "./immersive-audio-settings";
import { buildStreamOptions } from "./transcoding-settings";

describe("immersive audio detection", () => {
  it("detects Atmos from title and E-AC-3 suffix", () => {
    expect(
      isAtmosCapableTrack({ title: "Song (Dolby Atmos)", suffix: "flac" }),
    ).toBe(true);
    expect(isAtmosCapableTrack({ suffix: "eac3" })).toBe(true);
    expect(isAtmosCapableTrack({ suffix: "mp3" })).toBe(false);
  });

  it("detects multichannel and surround codecs", () => {
    expect(isImmersiveCodecTrack({ channels: 6 })).toBe(true);
    expect(isImmersiveCodecTrack({ suffix: "dts" })).toBe(true);
    expect(isImmersiveCodecTrack({ suffix: "mp3", channels: 2 })).toBe(false);
  });

  it("builds immersive badges", () => {
    expect(formatImmersiveBadge({ title: "Track Atmos Mix" })).toBe("Atmos");
    expect(formatImmersiveBadge({ channels: 6 })).toBe("5.1");
    expect(formatImmersiveBadge({ channels: 8 })).toBe("7.1");
    expect(formatImmersiveBadge({ suffix: "mp3" })).toBeNull();
  });

  it("preserves immersive streams when enabled", () => {
    expect(shouldPreserveImmersiveStream({ channels: 6 }, true)).toBe(true);
    expect(shouldPreserveImmersiveStream({ channels: 6 }, false)).toBe(false);
  });
});

describe("immersive audio settings", () => {
  it("merges unknown modes to auto", () => {
    expect(
      mergeImmersiveAudioSettings({
        mode: "nope" as never,
        exclusiveOutput: true,
      }),
    ).toEqual({
      ...defaultImmersiveAudioSettings(),
      exclusiveOutput: true,
    });
  });
});

describe("buildStreamOptions immersive preserve", () => {
  it("skips always-transcode for immersive tracks when preserve is on", () => {
    expect(
      buildStreamOptions(
        { alwaysTranscode: true, maxBitRate: 192, format: "mp3" },
        false,
        {
          track: { channels: 6, suffix: "flac" },
          preserveImmersiveStreams: true,
        },
      ),
    ).toBeUndefined();
  });

  it("still forces transcode after decode failure", () => {
    expect(
      buildStreamOptions(
        { alwaysTranscode: false, maxBitRate: 0, format: "server-default" },
        true,
        {
          track: { channels: 6 },
          preserveImmersiveStreams: true,
        },
      ),
    ).toEqual({ maxBitRate: 320 });
  });
});
