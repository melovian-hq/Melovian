// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from "vitest";
import { AudioEngine } from "./audio-engine";
import { defaultBandParams, defaultEqSettings } from "./eq";

describe("AudioEngine", () => {
  it("buffers prefetch elements and releases stale ones without recreating", () => {
    const spy = vi.spyOn(globalThis, "Audio");
    const engine = new AudioEngine();
    spy.mockClear();

    engine.prefetch(["http://x/a", "http://x/b"]);
    expect(spy).toHaveBeenCalledTimes(2);

    spy.mockClear();
    engine.prefetch(["http://x/a", "http://x/b"]);
    expect(spy).toHaveBeenCalledTimes(0);

    engine.prefetch(["http://x/a"]);
    expect(spy).toHaveBeenCalledTimes(0);

    engine.prefetch([]);
    expect(spy).toHaveBeenCalledTimes(0);

    engine.destroy();
    spy.mockRestore();
  });

  it("does not report a prepared track until it has buffered", async () => {
    const engine = new AudioEngine();
    engine.prepareNext("http://x/next");
    expect(engine.hasPrepared("http://x/next")).toBe(false);
    await expect(engine.activatePrepared("http://x/next")).resolves.toBe(false);
    engine.destroy();
  });

  it("applies volume to the active element", () => {
    const engine = new AudioEngine();
    engine.setVolume(0.42);
    expect(engine.audio.volume).toBeCloseTo(0.42);
    engine.destroy();
  });

  it("accepts enabled EQ settings without breaking volume control", () => {
    const engine = new AudioEngine();
    engine.applyEq({
      presetId: "flat",
      enabled: true,
      bands: defaultBandParams(),
    });
    engine.setVolume(0.42);
    expect(engine.audio.volume).toBeGreaterThan(0);
    engine.destroy();
  });

  it("keeps direct element output when EQ is disabled", () => {
    const engine = new AudioEngine();
    engine.applyEq(defaultEqSettings());
    engine.setVolume(0.42);
    expect(engine.audio.volume).toBeCloseTo(0.42);
    engine.destroy();
  });

  it("dispatches ended events to registered handlers and supports unsubscribe", () => {
    const engine = new AudioEngine();
    const handler = vi.fn();
    const off = engine.onEnded(handler);

    engine.audio.dispatchEvent(new Event("ended"));
    expect(handler).toHaveBeenCalledTimes(1);

    off();
    engine.audio.dispatchEvent(new Event("ended"));
    expect(handler).toHaveBeenCalledTimes(1);

    engine.destroy();
  });
});
