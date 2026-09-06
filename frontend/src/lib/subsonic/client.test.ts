// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from "vitest";
import { SubsonicClient, SubsonicApiError } from "./client";

describe("SubsonicClient", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("builds rest urls with client metadata and query params", () => {
    const client = new SubsonicClient({
      serverUrl: "http://127.0.0.1:17337/api/subsonic",
      clientName: "melovian",
      version: "1.16.1",
    });

    const url = new URL(
      client.buildUrl("getAlbum.view", { id: "album-1", empty: "" }),
      "http://localhost",
    );

    expect(url.pathname).toBe("/api/subsonic/rest/getAlbum.view");
    expect(url.searchParams.get("f")).toBe("json");
    expect(url.searchParams.get("c")).toBe("melovian");
    expect(url.searchParams.get("v")).toBe("1.16.1");
    expect(url.searchParams.get("id")).toBe("album-1");
    expect(url.searchParams.has("empty")).toBe(false);
  });

  it("throws SubsonicApiError on http failure", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 503,
        text: async () => "service unavailable",
      }),
    );

    const client = new SubsonicClient();
    await expect(client.request("ping.view")).rejects.toMatchObject({
      name: "SubsonicApiError",
      status: 503,
      body: "service unavailable",
    });
  });

  it("throws SubsonicApiError when subsonic status is failed", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        text: async () =>
          JSON.stringify({
            "subsonic-response": {
              status: "failed",
              error: { code: 40, message: "Wrong username or password" },
            },
          }),
      }),
    );

    const client = new SubsonicClient();
    await expect(client.request("ping.view")).rejects.toSatisfy(
      (err: unknown) =>
        err instanceof SubsonicApiError &&
        err.message === "Wrong username or password",
    );
  });

  it("returns parsed subsonic payload on success", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        text: async () =>
          JSON.stringify({
            "subsonic-response": {
              status: "ok",
              song: { id: "t1", title: "Track One" },
            },
          }),
      }),
    );

    const client = new SubsonicClient();
    const payload = await client.request<{ song?: { id: string } }>(
      "getSong.view",
      { id: "t1" },
    );
    expect(payload.song?.id).toBe("t1");
  });
});
