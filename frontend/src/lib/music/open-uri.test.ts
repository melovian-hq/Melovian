// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  absoluteMediaUri,
  parseOpenMediaUri,
  trackFromOpenStream,
} from "$lib/music/open-uri";

describe("parseOpenMediaUri", () => {
  it("extracts local track ids", () => {
    expect(parseOpenMediaUri("/api/local-music/tracks/trk_abc/stream")).toEqual(
      { kind: "trackId", trackId: "trk_abc" },
    );
  });

  it("extracts subsonic stream ids", () => {
    expect(
      parseOpenMediaUri(
        "https://music.example/rest/stream.view?id=song-1&u=user&v=1.16.1&c=melovian",
      ),
    ).toEqual({ kind: "trackId", trackId: "song-1" });
  });

  it("accepts raw http streams", () => {
    const parsed = parseOpenMediaUri("https://radio.example/live.mp3");
    expect(parsed?.kind).toBe("stream");
    if (parsed?.kind === "stream") {
      expect(parsed.streamUrl).toBe("https://radio.example/live.mp3");
    }
  });

  it("builds external stream tracks", () => {
    const track = trackFromOpenStream(
      "https://radio.example/live.mp3",
      "live.mp3",
    );
    expect(track.isInternetRadio).toBe(true);
    expect(track.streamUrl).toContain("radio.example");
  });

  it("resolves relative api paths", () => {
    const uri = absoluteMediaUri("/api/downloads/trk_1/stream");
    expect(uri).toContain("/api/downloads/trk_1/stream");
  });
});
