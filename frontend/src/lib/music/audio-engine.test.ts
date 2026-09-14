// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AudioEngine } from "./audio-engine";
import { defaultBandParams, defaultEqSettings } from "./eq";

function fakeAudioParam() {
  return {
    value: 0,
    cancelScheduledValues: vi.fn(),
    setValueAtTime: vi.fn(),
    linearRampToValueAtTime: vi.fn(),
  };
}

function fakeAudioNode(extra: Record<string, unknown> = {}) {
  return { connect: vi.fn(), disconnect: vi.fn(), ...extra };
}

/** Minimal Web Audio stand-in so the crossfade path runs under jsdom. */
class FakeAudioContext {
  currentTime = 0;
  state = "running";
  destination = fakeAudioNode();
  createGain() {
    return fakeAudioNode({ gain: fakeAudioParam() });
  }
  createBiquadFilter() {
    return fakeAudioNode({
      type: "",
      frequency: fakeAudioParam(),
      Q: fakeAudioParam(),
      gain: fakeAudioParam(),
    });
  }
  createMediaElementSource() {
    return fakeAudioNode();
  }
  createChannelSplitter() {
    return fakeAudioNode();
  }
  createChannelMerger() {
    return fakeAudioNode();
  }
  createDelay() {
    return fakeAudioNode({ delayTime: fakeAudioParam() });
  }
  resume() {
    return Promise.resolve();
  }
  close() {
    return Promise.resolve();
  }
}

function preparedEngine(url = "http://x/next") {
  const engine = new AudioEngine();
  const internals = engine as unknown as { standby: HTMLAudioElement };
  // jsdom never buffers media, so hasPrepared needs a stubbed readyState.
  Object.defineProperty(internals.standby, "readyState", {
    configurable: true,
    get: () => 4,
  });
  engine.prepareNext(url);
  return engine;
}

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

describe("AudioEngine crossfade abort", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.stubGlobal("AudioContext", FakeAudioContext);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("does not commit the swap when pause cancels the fade", async () => {
    const engine = preparedEngine();
    const outgoing = engine.audio;

    const fade = engine.activatePrepared("http://x/next", 1);
    await vi.advanceTimersByTimeAsync(0);
    engine.pause();
    await vi.advanceTimersByTimeAsync(2000);

    await expect(fade).resolves.toBe(false);
    expect(engine.audio).toBe(outgoing);
    engine.destroy();
  });

  it("does not commit the swap when a new loadSource supersedes the fade", async () => {
    const engine = preparedEngine();
    const outgoing = engine.audio;

    const fade = engine.activatePrepared("http://x/next", 1);
    await vi.advanceTimersByTimeAsync(0);
    const load = engine.loadSource("http://x/other");
    // silenceForSwap waits ~23ms before the new load attaches listeners.
    await vi.advanceTimersByTimeAsync(100);
    outgoing.dispatchEvent(new Event("canplay"));
    await load;
    await vi.advanceTimersByTimeAsync(2000);

    await expect(fade).resolves.toBe(false);
    expect(engine.audio).toBe(outgoing);
    engine.destroy();
  });
});
