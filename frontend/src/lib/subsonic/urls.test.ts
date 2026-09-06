// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, beforeEach, afterEach } from "vitest";
import { streamUrl, coverArtImageUrl, coverArtUrl } from "./urls";
import { setMediaBaseUrl } from "$lib/config/runtime";
import { setActiveInstanceId } from "$lib/features/instances/context";
import { setMusicSourceKind } from "$lib/music/source.svelte";
import type { SubsonicConfig } from "./types";

const config: SubsonicConfig = {
  serverUrl: "/api/subsonic",
  clientName: "melovian",
  version: "1.16.1",
};

describe("stream urls", () => {
  beforeEach(() => {
    setMediaBaseUrl("");
    setActiveInstanceId(null);
    setMusicSourceKind("subsonic");
  });

  afterEach(() => {
    setMediaBaseUrl("");
    setActiveInstanceId(null);
    setMusicSourceKind("subsonic");
  });

  it("targets the real http server when a media base is set", () => {
    setMediaBaseUrl("http://127.0.0.1:17337");
    const url = streamUrl(config, "track-1");
    expect(
      url.startsWith("http://127.0.0.1:17337/api/subsonic/rest/stream.view"),
    ).toBe(true);
  });

  it("includes id, client, and version", () => {
    const url = new URL(streamUrl(config, "track-1"), "http://localhost");
    expect(url.searchParams.get("id")).toBe("track-1");
    expect(url.searchParams.get("c")).toBe("melovian");
    expect(url.searchParams.get("v")).toBe("1.16.1");
  });

  it("appends the active instance for the audio element", () => {
    setActiveInstanceId("inst-42");
    const url = new URL(streamUrl(config, "track-1"), "http://localhost");
    expect(url.searchParams.get("_instance")).toBe("inst-42");
  });

  it("omits the instance param when none is active", () => {
    const url = new URL(streamUrl(config, "track-1"), "http://localhost");
    expect(url.searchParams.get("_instance")).toBeNull();
  });

  it("requests transcoding when maxBitRate is set", () => {
    const url = new URL(
      streamUrl(config, "track-1", { maxBitRate: 320 }),
      "http://localhost",
    );
    expect(url.searchParams.get("maxBitRate")).toBe("320");
  });

  it("requests a stream format when format is set", () => {
    const url = new URL(
      streamUrl(config, "track-1", { format: "mp3", maxBitRate: 192 }),
      "http://localhost",
    );
    expect(url.searchParams.get("format")).toBe("mp3");
    expect(url.searchParams.get("maxBitRate")).toBe("192");
  });

  it("builds cover art urls scoped to the active instance", () => {
    setActiveInstanceId("inst-42");
    const raw = coverArtImageUrl(config, "cover-1");
    expect(raw).not.toBeNull();
    const url = new URL(raw as string, "http://localhost");
    expect(url.searchParams.get("id")).toBe("cover-1");
    expect(url.searchParams.get("_instance")).toBe("inst-42");
    expect(url.searchParams.get("f")).toBeNull();
  });

  it("uses local stream urls when the local source is active", () => {
    setMusicSourceKind("local");
    setMediaBaseUrl("http://127.0.0.1:17337");
    expect(streamUrl(config, "trk_abc")).toBe(
      "http://127.0.0.1:17337/api/local-music/tracks/trk_abc/stream",
    );
  });

  it("uses local cover urls when the local source is active", () => {
    setMusicSourceKind("local");
    setMediaBaseUrl("http://127.0.0.1:17337");
    expect(coverArtUrl(config, "alb_123", 300)).toBe(
      "http://127.0.0.1:17337/api/local-music/cover/alb_123?size=300",
    );
  });
});
