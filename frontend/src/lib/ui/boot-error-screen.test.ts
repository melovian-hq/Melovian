// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import {
  bootErrorPayloadFromUnknown,
  buildBootErrorDetail,
  buildBootErrorLog,
  closeApplication,
  copyTextToClipboard,
  isNativeDesktopRuntime,
  QUIT_APP_CALL_ID,
} from "./boot-error-screen";

vi.mock("@wailsio/runtime", () => ({
  Call: {
    ByID: vi.fn().mockResolvedValue(undefined),
  },
}));

describe("boot-error-screen", () => {
  it("builds boot error detail and full log text", () => {
    const payload = {
      title: "Melovian crashed",
      message: "boom",
      source: "main",
      stack: "Error: boom\n    at test",
      url: "http://localhost:5173/",
      time: "2026-01-01T00:00:00.000Z",
    };

    const detail = buildBootErrorDetail(payload);
    expect(detail).toContain("Source: main");
    expect(detail).toContain("URL: http://localhost:5173/");
    expect(detail).toContain("Time: 2026-01-01T00:00:00.000Z");
    expect(detail).toContain("Error: boom");

    const log = buildBootErrorLog(payload.title, payload.message, detail);
    expect(log).toContain("Melovian crashed");
    expect(log).toContain("boom");
    expect(log).toContain("Source: main");
  });

  it("formats unknown boot errors", () => {
    const payload = bootErrorPayloadFromUnknown(
      new Error("load failed"),
      "main",
    );
    expect(payload.message).toBe("load failed");
    expect(payload.source).toBe("main");
    expect(payload.stack).toContain("load failed");
  });

  describe("copyTextToClipboard", () => {
    beforeEach(() => {
      document.body.innerHTML = "";
    });

    it("uses the clipboard API when available", async () => {
      const writeText = vi.fn().mockResolvedValue(undefined);
      vi.stubGlobal("navigator", { clipboard: { writeText } });

      await expect(copyTextToClipboard("hello")).resolves.toBe(true);
      expect(writeText).toHaveBeenCalledWith("hello");
    });

    it("returns false for empty text", async () => {
      await expect(copyTextToClipboard("")).resolves.toBe(false);
    });
  });

  describe("closeApplication", () => {
    afterEach(() => {
      vi.unstubAllGlobals();
      vi.resetModules();
    });

    it("detects native desktop runtime", () => {
      vi.stubGlobal("window", {
        location: { protocol: "wails:" },
        close: vi.fn(),
      });
      expect(isNativeDesktopRuntime()).toBe(true);
    });

    it("falls back to window.close when quit is unavailable", async () => {
      const close = vi.fn();
      vi.stubGlobal("window", {
        location: { protocol: "http:" },
        close,
      });

      await closeApplication();
      expect(close).toHaveBeenCalledTimes(1);
    });

    it("calls QuitApp on native desktop runtime", async () => {
      const close = vi.fn();
      vi.stubGlobal("window", {
        location: { protocol: "wails:" },
        close,
      });

      const { Call } = await import("@wailsio/runtime");
      await closeApplication();
      expect(Call.ByID).toHaveBeenCalledWith(QUIT_APP_CALL_ID);
      expect(close).not.toHaveBeenCalled();
    });
  });
});
