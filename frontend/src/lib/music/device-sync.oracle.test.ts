// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";

const sendMock = vi.fn((..._args: unknown[]) => true);
const handlers = new Map<
  string,
  Set<(event: { type: string; payload: unknown }) => void>
>();

vi.mock("$lib/core/events/ws.svelte", () => ({
  eventSocket: {
    connected: true,
    connecting: false,
    failed: false,
    send: (...args: unknown[]) => sendMock(...args),
    on: (
      type: string,
      handler: (event: { type: string; payload: unknown }) => void,
    ) => {
      let set = handlers.get(type);
      if (!set) {
        set = new Set();
        handlers.set(type, set);
      }
      set.add(handler);
      return () => set?.delete(handler);
    },
    onOpen: (handler: () => void) => {
      handler();
      return () => {};
    },
    connect: vi.fn(),
  },
}));

vi.mock("$lib/ui/toast.svelte", () => ({
  toast: {
    info: vi.fn(),
    error: vi.fn(),
    success: vi.fn(),
  },
}));

vi.mock("$lib/config/music.svelte", () => ({
  music: {
    currentTrack: null,
    currentTime: 0,
    duration: 0,
    playing: false,
    queue: [],
    queueIndex: -1,
    togglePlay: vi.fn(),
    next: vi.fn(),
    previous: vi.fn(),
    seek: vi.fn(),
    playQueueIndex: vi.fn(),
    playTrackById: vi.fn(),
    applyRemoteQueue: vi.fn(),
    armStartPosition: vi.fn(),
  },
}));

function emit(type: string, payload: unknown) {
  handlers.get(type)?.forEach((handler) => handler({ type, payload }));
}

describe("device-sync session isolation oracle", () => {
  beforeEach(() => {
    sendMock.mockClear();
    handlers.clear();
    localStorage.setItem("mel-device-id", "device-self");
    localStorage.setItem("mel-device-name", "Self");
  });

  afterEach(() => {
    vi.resetModules();
  });

  async function loadSync() {
    const mod = await import("./device-sync.svelte");
    const musicMod = await import("$lib/config/music.svelte");
    vi.mocked(musicMod.music.togglePlay).mockClear();
    vi.mocked(musicMod.music.seek).mockClear();
    vi.mocked(musicMod.music.applyRemoteQueue).mockClear();
    vi.mocked(musicMod.music.playTrackById).mockClear();
    vi.mocked(musicMod.music.armStartPosition).mockClear();
    mod.deviceSync.start();
    return { deviceSync: mod.deviceSync, music: musicMod.music };
  }

  it("does not join a listen-together session meant for another device", async () => {
    const { deviceSync } = await loadSync();

    emit("session.updated", {
      action: "joined",
      sessionId: "lt-other-host",
      hostId: "other-host",
      deviceId: "other-guest",
    });

    expect(deviceSync.sessionId).toBeNull();
    expect(deviceSync.following).toBe(false);
    expect(deviceSync.isHost).toBe(false);
  });

  it("only the joining device becomes a follower", async () => {
    const { deviceSync } = await loadSync();

    emit("session.updated", {
      action: "joined",
      sessionId: "lt-host",
      hostId: "host",
      deviceId: "device-self",
      state: {
        trackId: "t1",
        positionMs: 12_000,
        paused: false,
      },
    });

    expect(deviceSync.sessionId).toBe("lt-host");
    expect(deviceSync.following).toBe(true);
    expect(deviceSync.isHost).toBe(false);
  });

  it("ignores playback.state without together flag while following is false", async () => {
    const { deviceSync, music } = await loadSync();
    deviceSync.following = false;

    emit("playback.state", {
      deviceId: "host",
      together: false,
      state: { trackId: "x", positionMs: 99_000, paused: false },
    });

    expect(music.seek).not.toHaveBeenCalled();
    expect(music.applyRemoteQueue).not.toHaveBeenCalled();
  });

  it("ignores remote commands targeted at another device", async () => {
    const { music } = await loadSync();

    emit("playback.command", {
      action: "pause",
      targetId: "someone-else",
      fromId: "remote",
    });

    expect(music.togglePlay).not.toHaveBeenCalled();
  });

  it("does not publish playback while following (prevents steal + WS storm)", async () => {
    const { deviceSync, music } = await loadSync();
    deviceSync.following = true;
    music.playing = true;
    music.currentTrack = { id: "t1", title: "Song", artist: "A" } as never;
    sendMock.mockClear();

    deviceSync.publishPlayback(true);
    deviceSync.notifyPlaybackChanged();

    expect(sendMock).not.toHaveBeenCalledWith(
      "playback.state",
      expect.anything(),
    );
  });

  it("clears following on take_over so this device can become active", async () => {
    const { deviceSync } = await loadSync();
    deviceSync.following = true;
    sendMock.mockClear();

    emit("playback.command", {
      action: "take_over",
      targetId: "device-self",
      state: {
        trackId: "t9",
        positionMs: 1000,
        paused: true,
        queueIds: [],
        queueIndex: -1,
      },
    });

    expect(deviceSync.following).toBe(false);
  });

  it("sends session.create when listen together starts while connected", async () => {
    const { deviceSync } = await loadSync();
    sendMock.mockClear();
    deviceSync.createListenTogether();
    expect(sendMock).toHaveBeenCalledWith("session.create", {});
    expect(deviceSync.isHost).toBe(false);
    expect(deviceSync.pendingOp).toBe("create-session");
  });

  it("clears guest following when the host session ends", async () => {
    const { deviceSync } = await loadSync();
    emit("session.updated", {
      action: "joined",
      sessionId: "lt-host",
      hostId: "host",
      deviceId: "device-self",
    });
    expect(deviceSync.following).toBe(true);

    emit("session.updated", {
      action: "ended",
      sessionId: "lt-host",
      reason: "host_left",
    });
    expect(deviceSync.following).toBe(false);
    expect(deviceSync.sessionId).toBeNull();
    expect(deviceSync.isHost).toBe(false);
  });

  it("syncs guest role from devices.updated after a stale session", async () => {
    const { deviceSync } = await loadSync();
    deviceSync.following = true;
    deviceSync.sessionId = "lt-stale";

    emit("devices.updated", {
      devices: [
        {
          deviceId: "device-self",
          name: "Self",
          connectedAt: 1,
          lastSeen: 1,
          isActivePlayer: false,
        },
      ],
    });

    expect(deviceSync.following).toBe(false);
    expect(deviceSync.sessionId).toBeNull();
  });

  it("sends playback.handoffTo when transferring to a peer", async () => {
    const { deviceSync } = await loadSync();
    emit("devices.updated", {
      devices: [
        {
          deviceId: "device-self",
          name: "Self",
          connectedAt: 1,
          lastSeen: 1,
          isActivePlayer: true,
        },
        {
          deviceId: "peer-phone",
          name: "Phone",
          connectedAt: 1,
          lastSeen: 1,
          isActivePlayer: false,
        },
      ],
    });
    sendMock.mockClear();
    deviceSync.transferTo("peer-phone");
    expect(sendMock).toHaveBeenCalledWith("playback.handoffTo", {
      targetId: "peer-phone",
    });
    expect(deviceSync.pendingOp).toBe("transfer");
  });

  it("surfaces device.error instead of failing silently", async () => {
    const { deviceSync } = await loadSync();
    const toastMod = await import("$lib/ui/toast.svelte");
    emit("device.error", {
      code: "nothing_playing",
      message: "Nothing is playing to transfer.",
    });
    expect(deviceSync.lastError).toBe("Nothing is playing to transfer.");
    expect(toastMod.toast.error).toHaveBeenCalled();
  });

  it("seeks an already-loaded host track when drift is large", async () => {
    const { deviceSync, music } = await loadSync();
    deviceSync.following = true;
    music.currentTrack = { id: "t9", title: "Song" } as never;
    music.queue = [{ id: "t9", title: "Song" }];
    music.queueIndex = 0;
    music.currentTime = 1;
    music.playing = true;

    emit("playback.state", {
      deviceId: "host",
      together: true,
      state: {
        trackId: "t9",
        queueIds: ["t9"],
        queueIndex: 0,
        positionMs: 40_000,
        paused: false,
        serverTime: Date.now(),
      },
    });

    await vi.waitFor(() => {
      expect(music.seek).toHaveBeenCalled();
    });
    expect(music.playTrackById).not.toHaveBeenCalled();
    expect(music.armStartPosition).not.toHaveBeenCalled();
    const [seekSeconds] = vi.mocked(music.seek).mock.calls[0];
    expect(seekSeconds).toBeCloseTo(40, 1);
  });

  it("arms start position before loading a new host track", async () => {
    const { deviceSync, music } = await loadSync();
    deviceSync.following = true;
    music.currentTrack = null;
    music.queue = [];
    music.queueIndex = -1;
    const now = Date.now();

    emit("playback.state", {
      deviceId: "host",
      together: true,
      state: {
        trackId: "t9",
        positionMs: 12_000,
        durationMs: 180_000,
        paused: false,
        serverTime: now - 800,
      },
    });

    await vi.waitFor(() => {
      expect(music.armStartPosition).toHaveBeenCalled();
    });
    const [seconds, paused] = vi.mocked(music.armStartPosition).mock.calls[0];
    expect(seconds).toBeGreaterThan(12.5);
    expect(seconds).toBeLessThan(13.2);
    expect(paused).toBe(false);
    expect(music.playTrackById).toHaveBeenCalledWith("t9");
    expect(music.seek).not.toHaveBeenCalled();
  });

  it("routes guest transport to the host", async () => {
    const { deviceSync, music } = await loadSync();
    deviceSync.following = true;
    music.playing = true;
    sendMock.mockClear();

    expect(deviceSync.dispatchTransport("next")).toBe(true);
    expect(sendMock).toHaveBeenCalledWith("playback.command", {
      action: "next",
    });
    expect(deviceSync.dispatchTransport("toggle")).toBe(true);
    expect(sendMock).toHaveBeenCalledWith("playback.command", {
      action: "pause",
    });
    expect(deviceSync.dispatchTransport("seek", { positionSec: 44.2 })).toBe(
      true,
    );
    expect(sendMock).toHaveBeenCalledWith("playback.command", {
      action: "seek",
      positionMs: 44200,
    });
  });

  it("does not intercept transport when not following", async () => {
    const { deviceSync } = await loadSync();
    expect(deviceSync.dispatchTransport("next")).toBe(false);
  });

  it("host republishes playback when a guest joins", async () => {
    const { music } = await loadSync();
    emit("session.updated", {
      action: "created",
      sessionId: "lt-host",
      hostId: "device-self",
    });
    music.currentTrack = { id: "t1", title: "Song" } as never;
    music.playing = true;
    sendMock.mockClear();

    emit("session.updated", {
      action: "joined",
      sessionId: "lt-host",
      hostId: "device-self",
      deviceId: "guest-1",
    });

    expect(sendMock).toHaveBeenCalledWith(
      "playback.state",
      expect.objectContaining({ trackId: "t1" }),
    );
  });
});
