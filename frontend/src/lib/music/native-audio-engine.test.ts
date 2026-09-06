// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from "vitest";

const prepareURL = vi.fn().mockResolvedValue(undefined);
const hasPrepared = vi.fn().mockResolvedValue(true);
const activatePrepared = vi.fn().mockResolvedValue(undefined);
const clearPrepared = vi.fn().mockResolvedValue(undefined);
const play = vi.fn().mockResolvedValue(undefined);
const pause = vi.fn();
const getState = vi.fn().mockResolvedValue({
  currentTime: 0,
  duration: 10,
  playing: false,
  ended: false,
  error: "",
});
const loadURL = vi.fn().mockResolvedValue(undefined);
const seek = vi.fn().mockResolvedValue(undefined);
const setVolume = vi.fn();
const clearEnded = vi.fn();

vi.mock("@bindings/melovian/services/index.js", () => ({
  AudioService: {
    PrepareURL: prepareURL,
    HasPrepared: hasPrepared,
    ActivatePrepared: activatePrepared,
    ClearPrepared: clearPrepared,
    Play: play,
    Pause: pause,
    GetState: getState,
    LoadURL: loadURL,
    Seek: seek,
    SetVolume: setVolume,
    ClearEnded: clearEnded,
  },
}));

describe("NativeAudioEngine prepare/activate", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    hasPrepared.mockResolvedValue(true);
  });

  it("prepares then activates the standby URL", async () => {
    const { NativeAudioEngine } = await import("./native-audio-engine");
    const engine = new NativeAudioEngine();
    engine.prepareNext("https://example.com/a.flac");
    await vi.waitFor(() => expect(prepareURL).toHaveBeenCalled());
    expect(prepareURL).toHaveBeenCalledWith("https://example.com/a.flac");
    expect(engine.hasPrepared("https://example.com/a.flac")).toBe(true);

    const ok = await engine.activatePrepared("https://example.com/a.flac", 0);
    expect(ok).toBe(true);
    expect(activatePrepared).toHaveBeenCalledWith(0);
    expect(engine.hasPrepared("https://example.com/a.flac")).toBe(false);
    engine.destroy();
  });

  it("clears preparation when given an empty URL", async () => {
    const { NativeAudioEngine } = await import("./native-audio-engine");
    const engine = new NativeAudioEngine();
    engine.prepareNext("https://example.com/a.flac");
    await vi.waitFor(() => expect(prepareURL).toHaveBeenCalled());
    engine.prepareNext("");
    await vi.waitFor(() => expect(clearPrepared).toHaveBeenCalled());
    expect(engine.hasPrepared("https://example.com/a.flac")).toBe(false);
    engine.destroy();
  });
});
