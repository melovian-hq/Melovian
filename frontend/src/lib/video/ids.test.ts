// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  buildEmbedUrl,
  displayVideoTitle,
  invidiousIdFromPlayId,
  isInvidiousPlayId,
  isYouTubePlayId,
  looksLikeYouTubeVideoId,
  mergeVideoSettings,
  musicVideoSearchQuery,
  playIdForInvidious,
  playIdForYouTube,
  searchReady,
  videoPlayerPath,
  youtubeIdFromPlayId,
} from "./ids";

describe("video ids", () => {
  it("builds and parses invidious play ids", () => {
    const playId = playIdForInvidious("abc123");
    expect(isInvidiousPlayId(playId)).toBe(true);
    expect(invidiousIdFromPlayId(playId)).toBe("abc123");
    expect(videoPlayerPath(playId)).toBe("/play/ext%3Ainvidious%3Aabc123");
  });

  it("builds and parses youtube play ids", () => {
    const playId = playIdForYouTube("yt456");
    expect(isYouTubePlayId(playId)).toBe(true);
    expect(youtubeIdFromPlayId(playId)).toBe("yt456");
  });

  it("rejects local ids as external", () => {
    expect(isInvidiousPlayId("trk_local")).toBe(false);
    expect(invidiousIdFromPlayId("trk_local")).toBeNull();
  });

  it("keeps videos disabled by default", () => {
    const settings = mergeVideoSettings(null);
    expect(settings.enabled).toBe(false);
    expect(searchReady(settings)).toBe(false);
  });

  it("never displays a bare video id as the title", () => {
    expect(looksLikeYouTubeVideoId("Vu0nRz_bQ0Y")).toBe(true);
    expect(displayVideoTitle("Vu0nRz_bQ0Y", "Vu0nRz_bQ0Y")).toBe("Music video");
    expect(displayVideoTitle(undefined, "Vu0nRz_bQ0Y")).toBe("Music video");
    expect(displayVideoTitle("Official MV", "Vu0nRz_bQ0Y")).toBe("Official MV");
  });

  it("puts a real title on the player path", () => {
    const playId = playIdForInvidious("Vu0nRz_bQ0Y");
    expect(videoPlayerPath(playId, { title: "Official MV" })).toBe(
      "/play/ext%3Ainvidious%3AVu0nRz_bQ0Y?title=Official%20MV",
    );
    expect(videoPlayerPath(playId, { title: "Vu0nRz_bQ0Y" })).toBe(
      "/play/ext%3Ainvidious%3AVu0nRz_bQ0Y",
    );
  });

  it("builds a music video search query from track metadata", () => {
    expect(
      musicVideoSearchQuery({ artist: "XS Project", title: "Hangout" }),
    ).toBe("XS Project Hangout official music video");
  });

  it("builds embed urls from settings without a resolve call", () => {
    expect(
      buildEmbedUrl(
        {
          enabled: true,
          searchProvider: "invidious",
          invidiousBaseUrl: "https://invidious.example.com/",
          youtubeApiKey: "",
        },
        "invidious",
        "Vu0nRz_bQ0Y",
      ),
    ).toBe("https://invidious.example.com/embed/Vu0nRz_bQ0Y");
    expect(
      buildEmbedUrl(
        {
          enabled: true,
          searchProvider: "youtube",
          invidiousBaseUrl: "",
          youtubeApiKey: "key",
        },
        "youtube",
        "Vu0nRz_bQ0Y",
      ),
    ).toBe("https://www.youtube.com/embed/Vu0nRz_bQ0Y");
  });
});
