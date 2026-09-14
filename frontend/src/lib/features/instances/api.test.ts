// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from "vitest";
import { detectServers, readErrorMessage } from "./api";

function mockFetchResponse(init: {
  ok: boolean;
  status?: number;
  json?: () => Promise<unknown>;
  text?: () => Promise<string>;
}) {
  return {
    ok: init.ok,
    status: init.status ?? (init.ok ? 200 : 500),
    headers: new Headers(),
    json: init.json ?? (async () => ({})),
    text: init.text ?? (async () => ""),
  };
}

describe("readErrorMessage", () => {
  it("prefers JSON error field", async () => {
    const response = new Response(
      JSON.stringify({ error: "tls handshake timeout" }),
      {
        status: 400,
        headers: { "Content-Type": "application/json" },
      },
    );
    await expect(
      readErrorMessage(response, "Connection test failed"),
    ).resolves.toBe("tls handshake timeout");
  });

  it("falls back to plain text", async () => {
    const response = new Response("empty request body", { status: 400 });
    await expect(
      readErrorMessage(response, "Connection test failed"),
    ).resolves.toBe("empty request body");
  });

  it("uses fallback when body is empty", async () => {
    const response = new Response("", { status: 400 });
    await expect(
      readErrorMessage(response, "Connection test failed"),
    ).resolves.toBe("Connection test failed");
  });
});

describe("detectServers", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("returns reachable servers from the detect endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      mockFetchResponse({
        ok: true,
        json: async () => ({
          servers: [
            {
              url: "http://127.0.0.1:4533",
              serverName: "navidrome",
              version: "0.55.2",
              reachable: true,
            },
            {
              url: "http://127.0.0.1:4040",
              serverName: "Subsonic",
              version: "1.16.1",
              reachable: false,
            },
          ],
        }),
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const servers = await detectServers();
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/api/instances/detect"),
      expect.anything(),
    );
    expect(servers).toEqual([
      {
        url: "http://127.0.0.1:4533",
        serverName: "navidrome",
        version: "0.55.2",
        reachable: true,
      },
    ]);
  });

  it("returns an empty list when nothing is found", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        mockFetchResponse({
          ok: true,
          json: async () => ({ servers: [] }),
        }),
      ),
    );
    await expect(detectServers()).resolves.toEqual([]);
  });

  it("throws when the request fails", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        mockFetchResponse({
          ok: false,
          status: 401,
          text: async () => "unauthorized",
        }),
      ),
    );
    await expect(detectServers()).rejects.toThrow("unauthorized");
  });
});
