// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from "vitest";
import { runExtensionScript } from "$lib/extensions/sandbox";
import { isSafeRuleRegex, sanitizeDecoration } from "$lib/extensions/registry";

function latestIframe(): HTMLIFrameElement {
  const frames = document.querySelectorAll("iframe");
  expect(frames.length).toBeGreaterThan(0);
  return frames[frames.length - 1] as HTMLIFrameElement;
}

function postFrom(frame: HTMLIFrameElement, data: unknown) {
  window.dispatchEvent(
    new MessageEvent("message", { data, source: frame.contentWindow }),
  );
}

afterEach(() => {
  for (const frame of document.querySelectorAll("iframe")) frame.remove();
});

describe("extension sandbox iframe protocol", () => {
  it("runs scripts in an opaque-origin iframe", () => {
    void runExtensionScript("function register() {}", {});
    const frame = latestIframe();
    // allow-scripts without allow-same-origin forces a null origin:
    // no storage, no cookies, no credentialed fetches to the app API.
    expect(frame.getAttribute("sandbox")).toBe("allow-scripts");
    expect(frame.src).toContain("ext-sandbox.html");
  });

  it("waits for ready before sending the source", async () => {
    const postSpy = vi.fn();
    const promise = runExtensionScript("function register() {}", {
      accent: "red",
    });
    const frame = latestIframe();
    const contentWindow = frame.contentWindow!;
    contentWindow.postMessage = postSpy;

    // A done message before the script ever ran must be ignored until
    // the handshake completes, and the run payload carries the source.
    postFrom(frame, { type: "ready" });
    expect(postSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "run",
        source: "function register() {}",
        settings: { accent: "red" },
      }),
      "*",
    );
    postFrom(frame, { type: "done", rules: [] });
    await expect(promise).resolves.toEqual({ rules: [] });
  });

  it("collects schema-valid rules from the sandbox", async () => {
    const promise = runExtensionScript("function register() {}", {});
    const frame = latestIframe();
    postFrom(frame, { type: "ready" });
    postFrom(frame, {
      type: "done",
      rules: [
        {
          match: { genreContains: "rock" },
          decoration: { icon: "mdi:guitar" },
        },
        { nonsense: true },
        "not a rule",
      ],
    });
    const result = await promise;
    expect(result.rules).toEqual([
      { match: { genreContains: "rock" }, decoration: { icon: "mdi:guitar" } },
    ]);
  });

  it("ignores messages that do not come from the sandbox window", async () => {
    const promise = runExtensionScript("function register() {}", {}, [], {
      timeoutMs: 50,
    });
    const frame = latestIframe();
    // A same-shaped message from the parent window itself must not
    // resolve the run.
    window.dispatchEvent(
      new MessageEvent("message", {
        data: { type: "done", rules: [{ match: {}, decoration: {} }] },
        source: window,
      }),
    );
    postFrom(frame, {
      type: "done",
      rules: [{ match: {}, decoration: {} }],
    });
    const result = await promise;
    expect(result.rules).toHaveLength(1);
  });

  it("times out when the sandbox never answers", async () => {
    const result = await runExtensionScript("function register() {}", {}, [], {
      timeoutMs: 25,
    });
    expect(result.rules).toEqual([]);
    expect(result.error).toBeTruthy();
    expect(document.querySelectorAll("iframe")).toHaveLength(0);
  });

  it("propagates script errors without registering rules", async () => {
    const promise = runExtensionScript("throw new Error('boom')", {});
    const frame = latestIframe();
    postFrom(frame, { type: "ready" });
    postFrom(frame, { type: "done", rules: [], error: "Error: boom" });
    const result = await promise;
    expect(result.error).toBe("Error: boom");
    expect(result.rules).toEqual([]);
  });
});

describe("extension rule regex guard", () => {
  it("rejects nested quantifiers that cause catastrophic backtracking", () => {
    expect(isSafeRuleRegex("(a+)+$")).toBe(false);
    expect(isSafeRuleRegex("([a-z]+)*b")).toBe(false);
    expect(isSafeRuleRegex("(x|y)+")).toBe(true);
    expect(isSafeRuleRegex("^live at .+ hall$")).toBe(true);
  });

  it("rejects oversized patterns", () => {
    expect(isSafeRuleRegex("a".repeat(201))).toBe(false);
    expect(isSafeRuleRegex("a".repeat(200))).toBe(true);
  });
});

describe("decoration URL sanitization", () => {
  it("keeps same-origin paths and drops remote egress", () => {
    const clean = sanitizeDecoration({
      progressThumbUrl: "/api/extensions/demo/assets/thumb.png",
      progressParticleUrl: "https://evil.example/pixel.png",
      iconUrl: "//evil.example/icon.png",
      progressColor: "#fff",
    });
    expect(clean.progressThumbUrl).toBe(
      "/api/extensions/demo/assets/thumb.png",
    );
    expect(clean.progressParticleUrl).toBeUndefined();
    expect(clean.iconUrl).toBeUndefined();
    expect(clean.progressColor).toBe("#fff");
  });
});
