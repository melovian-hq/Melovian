// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  clearRemoteServerUrl,
  getRemoteServerUrl,
  isRemoteClient,
  looksLikePrivateOrOverlayHost,
  normalizeRemoteServerUrl,
  resolveApiUrl,
  setRemoteServerUrl,
} from "./remote-server";

describe("remote-server", () => {
  beforeEach(() => {
    clearRemoteServerUrl();
    localStorage.clear();
  });

  it("defaults https for public hostnames and strips trailing slashes", () => {
    expect(normalizeRemoteServerUrl("melovian.example.com/")).toBe(
      "https://melovian.example.com",
    );
  });

  it("defaults http for Tailscale and LAN hosts without a scheme", () => {
    expect(normalizeRemoteServerUrl("100.64.1.2:8080")).toBe(
      "http://100.64.1.2:8080",
    );
    expect(normalizeRemoteServerUrl("192.168.1.10:8080")).toBe(
      "http://192.168.1.10:8080",
    );
    expect(normalizeRemoteServerUrl("box.ts.net")).toBe("http://box.ts.net");
  });

  it("detects private and overlay hostnames", () => {
    expect(looksLikePrivateOrOverlayHost("100.64.0.1")).toBe(true);
    expect(looksLikePrivateOrOverlayHost("10.0.0.5")).toBe(true);
    expect(looksLikePrivateOrOverlayHost("melovian.example.com")).toBe(false);
  });

  it("persists and clears the remote host", () => {
    expect(isRemoteClient()).toBe(false);
    setRemoteServerUrl("https://melovian.example.com/");
    expect(getRemoteServerUrl()).toBe("https://melovian.example.com");
    expect(isRemoteClient()).toBe(true);
    expect(localStorage.getItem("melovian.remoteServerUrl")).toBe(
      "https://melovian.example.com",
    );
    clearRemoteServerUrl();
    expect(getRemoteServerUrl()).toBe("");
    expect(isRemoteClient()).toBe(false);
  });

  it("prefixes relative api paths when remote is set", () => {
    setRemoteServerUrl("http://100.64.1.2:8080");
    expect(resolveApiUrl("/api/config")).toBe(
      "http://100.64.1.2:8080/api/config",
    );
    expect(resolveApiUrl("https://other.example/x")).toBe(
      "https://other.example/x",
    );
  });

  it("leaves relative paths alone without a remote host", () => {
    expect(resolveApiUrl("/api/config")).toBe("/api/config");
  });

  it("probes http and https hosts", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        authEnabled: false,
        serverMode: true,
        demoMode: false,
      }),
    });
    vi.stubGlobal("fetch", fetchMock);
    const { probeRemoteServer } = await import("./remote-server");
    const httpResult = await probeRemoteServer("http://100.64.1.2:8080");
    expect(httpResult.url).toBe("http://100.64.1.2:8080");
    expect(httpResult.authEnabled).toBe(false);
    expect(fetchMock).toHaveBeenCalledWith(
      "http://100.64.1.2:8080/api/config",
      expect.objectContaining({ credentials: "omit" }),
    );

    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ authEnabled: true, serverMode: true }),
    });
    const httpsResult = await probeRemoteServer("https://melovian.example.com");
    expect(httpsResult.url).toBe("https://melovian.example.com");
    expect(httpsResult.authEnabled).toBe(true);
    vi.unstubAllGlobals();
  });
});
