// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { formatTrackQuality, isLosslessQuality } from "./quality";

describe("formatTrackQuality", () => {
  it("labels transcoded tracks", () => {
    expect(formatTrackQuality({ transcoded: true, bitRate: 320 })).toBe(
      "MP3 320",
    );
    expect(formatTrackQuality({ transcoded: true, bitRate: 0 })).toBe(
      "Transcoded",
    );
  });

  it("labels lossless formats", () => {
    expect(
      formatTrackQuality({ suffix: "flac", contentType: "audio/flac" }),
    ).toBe("FLAC");
    expect(
      formatTrackQuality({ suffix: "alac", contentType: "audio/alac" }),
    ).toBe("ALAC");
    expect(formatTrackQuality({ suffix: "wav" })).toBe("WAV");
  });

  it("labels mp3 bitrates", () => {
    expect(formatTrackQuality({ suffix: "mp3", bitRate: 320 })).toBe("MP3 320");
    expect(formatTrackQuality({ suffix: "mp3", bitRate: 192 })).toBe("MP3 192");
    expect(formatTrackQuality({ suffix: "mp3", bitRate: 128 })).toBe("MP3 128");
  });

  it("falls back to suffix or bitrate", () => {
    expect(formatTrackQuality({ suffix: "opus", bitRate: 128 })).toBe(
      "Opus 128",
    );
    expect(formatTrackQuality({ bitRate: 256 })).toBe("256 kbps");
    expect(formatTrackQuality({})).toBeNull();
  });
});

describe("isLosslessQuality", () => {
  it("detects lossless labels", () => {
    expect(isLosslessQuality("FLAC")).toBe(true);
    expect(isLosslessQuality("ALAC 24-bit")).toBe(true);
    expect(isLosslessQuality("MP3 320")).toBe(false);
    expect(isLosslessQuality(null)).toBe(false);
  });
});
