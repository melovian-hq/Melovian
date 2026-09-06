// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  buildStreamOptions,
  defaultTranscodingSettings,
  mergeTranscodingSettings,
} from "./transcoding-settings";

describe("transcoding settings", () => {
  it("builds stream options when always transcoding", () => {
    const settings = mergeTranscodingSettings({
      alwaysTranscode: true,
      maxBitRate: 192,
      format: "mp3",
    });
    expect(buildStreamOptions(settings)).toEqual({
      maxBitRate: 192,
      format: "mp3",
    });
  });

  it("defaults to 320 kbps when forcing transcode without a bitrate", () => {
    const settings = defaultTranscodingSettings();
    settings.alwaysTranscode = true;
    expect(buildStreamOptions(settings)).toEqual({ maxBitRate: 320 });
  });

  it("applies max bitrate without always transcode", () => {
    const settings = mergeTranscodingSettings({
      maxBitRate: 128,
      format: "opus",
    });
    expect(buildStreamOptions(settings)).toEqual({
      maxBitRate: 128,
      format: "opus",
    });
  });
});
