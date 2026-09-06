// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  mediaAlbumDownloadZipUrl,
  mediaPlaylistDownloadZipUrl,
  mediaServerPlaylistDownloadZipUrl,
  mediaTrackDownloadUrl,
  publicShareDownloadUrl,
  publicShareStreamUrl,
} from "./api";

describe("share and media download urls", () => {
  it("builds public share stream and download urls", () => {
    expect(publicShareStreamUrl("abc", "trk_1")).toContain(
      "/s/abc/tracks/trk_1/stream",
    );
    expect(publicShareDownloadUrl("abc", "trk_1")).toContain(
      "/s/abc/tracks/trk_1/download",
    );
  });

  it("builds media download urls", () => {
    expect(
      mediaTrackDownloadUrl("trk_1", { title: "A", artist: "B" }),
    ).toContain("/api/media/tracks/trk_1/download");
    expect(mediaAlbumDownloadZipUrl("alb_1")).toContain(
      "/api/media/albums/alb_1/download.zip",
    );
    expect(mediaPlaylistDownloadZipUrl("pl1")).toContain(
      "/api/media/playlists/pl1/download.zip",
    );
    expect(mediaServerPlaylistDownloadZipUrl("srv1")).toContain(
      "/api/media/server-playlists/srv1/download.zip",
    );
  });
});
