// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, beforeEach, vi } from "vitest";
import {
  getMediaBaseUrl,
  hasWailsHost,
  isServerMode,
  isWailsDesktop,
  isWailsMobile,
  loadRuntimeConfig,
  mediaBaseFromListenAddr,
  nativeDesktopAvailable,
  normalizePublicBaseUrl,
  resolveMediaUrl,
  setMediaBaseUrl,
} from "./runtime";

describe("media base url", () => {
  beforeEach(() => {
    setMediaBaseUrl("");
  });

  it("builds a localhost base from a listen address", () => {
    expect(mediaBaseFromListenAddr("127.0.0.1:17337")).toBe(
      "http://127.0.0.1:17337",
    );
  });

  it("uses browser origin for wildcard listen addrs in web mode", () => {
    expect(mediaBaseFromListenAddr("0.0.0.0:8080")).toBe(
      normalizePublicBaseUrl(window.location.origin),
    );
    expect(mediaBaseFromListenAddr("0.0.0.0:17337")).toBe(
      normalizePublicBaseUrl(window.location.origin),
    );
  });

  it("rewrites wildcard hosts in public base urls", () => {
    expect(normalizePublicBaseUrl("http://0.0.0.0:8080/")).toBe(
      "http://127.0.0.1:8080",
    );
    expect(normalizePublicBaseUrl("http://[::]:8080")).toBe(
      "http://127.0.0.1:8080",
    );
  });

  it("fills in the loopback host when only a port is given", () => {
    expect(mediaBaseFromListenAddr(":17337")).toBe("http://127.0.0.1:17337");
  });

  it("returns empty for empty input", () => {
    expect(mediaBaseFromListenAddr("")).toBe("");
  });

  it("trims a trailing slash when stored", () => {
    setMediaBaseUrl("http://127.0.0.1:17337/");
    expect(getMediaBaseUrl()).toBe("http://127.0.0.1:17337");
  });

  it("prefixes relative paths once a base is set", () => {
    setMediaBaseUrl("http://127.0.0.1:17337");
    expect(resolveMediaUrl("/api/subsonic")).toBe(
      "http://127.0.0.1:17337/api/subsonic",
    );
  });

  it("leaves relative paths untouched without a base", () => {
    expect(resolveMediaUrl("/api/subsonic")).toBe("/api/subsonic");
  });

  it("leaves absolute urls untouched", () => {
    setMediaBaseUrl("http://127.0.0.1:17337");
    expect(resolveMediaUrl("https://example.com/stream")).toBe(
      "https://example.com/stream",
    );
  });
});

describe("platform detection", () => {
  it("detects non-wails browser mode", () => {
    expect(hasWailsHost()).toBe(false);
    expect(isWailsDesktop()).toBe(false);
    expect(isWailsMobile()).toBe(false);
    expect(nativeDesktopAvailable()).toBe(false);
  });

  it("treats injected android environment as mobile, not desktop", () => {
    const w = window as Window & {
      _wails?: { environment?: { OS?: string } };
      wails?: { invokeAsync?: (id: string, payload: string) => void };
    };
    w.wails = { invokeAsync: () => {} };
    w._wails = { environment: { OS: "android" } };
    expect(hasWailsHost()).toBe(true);
    expect(isWailsMobile()).toBe(true);
    expect(isWailsDesktop()).toBe(false);
    expect(nativeDesktopAvailable()).toBe(false);
    delete w.wails;
    delete w._wails;
  });

  it("treats injected ios environment as mobile, not desktop", () => {
    const w = window as Window & {
      _wails?: { environment?: { OS?: string } };
      wails?: { invokeAsync?: (id: string, payload: string) => void };
    };
    w.wails = { invokeAsync: () => {} };
    w._wails = { environment: { OS: "ios" } };
    expect(hasWailsHost()).toBe(true);
    expect(isWailsMobile()).toBe(true);
    expect(isWailsDesktop()).toBe(false);
    expect(nativeDesktopAvailable()).toBe(false);
    delete w.wails;
    delete w._wails;
  });

  it("keeps media urls same-origin on mobile without a remote host", () => {
    const w = window as Window & {
      _wails?: { environment?: { OS?: string } };
    };
    w._wails = { environment: { OS: "android" } };
    setMediaBaseUrl("http://127.0.0.1:17337");
    expect(resolveMediaUrl("/api/subsonic")).toBe("/api/subsonic");
    delete w._wails;
  });

  it("prefixes media urls on mobile when a remote Melovian host is set", async () => {
    const { setRemoteServerUrl, clearRemoteServerUrl } =
      await import("./remote-server");
    const w = window as Window & {
      _wails?: { environment?: { OS?: string } };
    };
    w._wails = { environment: { OS: "android" } };
    setRemoteServerUrl("https://melovian.example.com");
    expect(resolveMediaUrl("/api/subsonic")).toBe(
      "https://melovian.example.com/api/subsonic",
    );
    clearRemoteServerUrl();
    delete w._wails;
  });

  it("defaults to server mode in browser before config loads", () => {
    expect(isServerMode()).toBe(true);
  });

  it("clears server mode after config reports desktop build", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      headers: { get: () => null },
      json: async () => ({
        listenAddr: "127.0.0.1:17337",
        serverMode: false,
      }),
    });
    vi.stubGlobal("fetch", fetchMock);

    await loadRuntimeConfig();

    expect(isServerMode()).toBe(false);
    vi.unstubAllGlobals();
  });
});
