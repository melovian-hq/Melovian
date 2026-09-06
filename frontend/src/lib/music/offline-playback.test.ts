// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  isDownloadStreamUrl,
  isOfflinePlaybackUrl,
} from "$lib/music/offline-playback";

describe("offline playback urls", () => {
  it("detects download cache streams", () => {
    expect(isDownloadStreamUrl("/api/downloads/trk_1/stream")).toBe(true);
    expect(
      isDownloadStreamUrl("https://127.0.0.1:8080/api/downloads/a/stream"),
    ).toBe(true);
  });

  it("detects local library streams", () => {
    expect(isOfflinePlaybackUrl("/api/local-music/tracks/trk_abc/stream")).toBe(
      true,
    );
  });

  it("rejects remote subsonic streams", () => {
    expect(
      isOfflinePlaybackUrl(
        "https://music.example/rest/stream.view?id=song-1&u=user",
      ),
    ).toBe(false);
  });
});
