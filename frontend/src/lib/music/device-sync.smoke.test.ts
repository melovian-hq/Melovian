// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";

/**
 * Mock smoke tests for device-sync command surface.
 * Avoids booting MusicStore; validates message shapes and local helpers.
 */

const sendMock = vi.fn((..._args: unknown[]) => true);
const onMock = vi.fn((..._args: unknown[]) => () => {});
const onOpenMock = vi.fn((handler: () => void) => {
  handler();
  return () => {};
});
const connectMock = vi.fn();

vi.mock("$lib/core/events/ws.svelte", () => ({
  eventSocket: {
    connected: true,
    connecting: false,
    failed: false,
    send: (...args: unknown[]) => sendMock(...args),
    on: (...args: unknown[]) => onMock(...args),
    onOpen: (...args: unknown[]) => onOpenMock(args[0] as () => void),
    connect: () => connectMock(),
  },
}));

vi.mock("$lib/ui/toast.svelte", () => ({
  toast: {
    info: vi.fn(),
    error: vi.fn(),
    success: vi.fn(),
  },
}));

const musicMock = vi.hoisted(() => ({
  currentTrack: null as {
    id: string;
    title: string;
    artist?: string;
    coverArt?: string;
    albumId?: string;
  } | null,
  currentTime: 0,
  duration: 0,
  playing: false,
  queue: [] as Array<{ id: string }>,
  queueIndex: -1,
  togglePlay: vi.fn(),
  next: vi.fn(),
  previous: vi.fn(),
  seek: vi.fn(),
  playQueueIndex: vi.fn(),
  playTrackById: vi.fn(),
  applyRemoteQueue: vi.fn(),
  armStartPosition: vi.fn(),
}));

vi.mock("$lib/config/music.svelte", () => ({
  music: musicMock,
}));

describe("device-sync mock smoke", () => {
  beforeEach(() => {
    sendMock.mockClear();
    onMock.mockClear();
    onOpenMock.mockClear();
    connectMock.mockClear();
    musicMock.currentTrack = null;
    musicMock.currentTime = 0;
    musicMock.duration = 0;
    musicMock.playing = false;
    musicMock.queue = [];
    musicMock.queueIndex = -1;
    localStorage.removeItem("mel-device-id");
    localStorage.removeItem("mel-device-name");
  });

  afterEach(() => {
    vi.resetModules();
  });

  it("registers device on start and publishes playback", async () => {
    const { deviceSync } = await import("./device-sync.svelte");
    deviceSync.start();
    expect(connectMock).toHaveBeenCalled();
    expect(sendMock).toHaveBeenCalledWith(
      "device.register",
      expect.objectContaining({
        deviceId: expect.any(String),
        name: expect.any(String),
      }),
    );
    expect(sendMock).toHaveBeenCalledWith(
      "playback.state",
      expect.objectContaining({
        paused: true,
        positionMs: expect.any(Number),
        coverArt: "",
      }),
    );
  });

  it("publishes coverArt from the current track", async () => {
    musicMock.currentTrack = {
      id: "t1",
      title: "Song",
      artist: "Artist",
      coverArt: "cover-9",
      albumId: "alb-1",
    };
    musicMock.playing = true;
    musicMock.currentTime = 12.5;
    musicMock.duration = 180;
    musicMock.queue = [{ id: "t1" }];
    musicMock.queueIndex = 0;

    const { deviceSync } = await import("./device-sync.svelte");
    deviceSync.start();

    expect(sendMock).toHaveBeenCalledWith(
      "playback.state",
      expect.objectContaining({
        trackId: "t1",
        trackTitle: "Song",
        artistName: "Artist",
        coverArt: "cover-9",
        paused: false,
      }),
    );
  });

  it("falls back coverArt to albumId then track id", async () => {
    musicMock.currentTrack = {
      id: "t2",
      title: "Bare",
      albumId: "alb-2",
    };
    const { deviceSync } = await import("./device-sync.svelte");
    deviceSync.start();
    expect(sendMock).toHaveBeenCalledWith(
      "playback.state",
      expect.objectContaining({ coverArt: "alb-2" }),
    );

    sendMock.mockClear();
    vi.resetModules();
    musicMock.currentTrack = { id: "t3", title: "NoArt" };
    const again = await import("./device-sync.svelte");
    again.deviceSync.start();
    expect(sendMock).toHaveBeenCalledWith(
      "playback.state",
      expect.objectContaining({ coverArt: "t3" }),
    );
  });

  it("sends handoff and session commands", async () => {
    const { deviceSync } = await import("./device-sync.svelte");
    deviceSync.start();
    sendMock.mockClear();

    deviceSync.takeOver();
    expect(sendMock).toHaveBeenCalledWith("playback.takeOver", {});

    deviceSync.transferTo("peer-1");
    expect(sendMock).toHaveBeenCalledWith("playback.handoffTo", {
      targetId: "peer-1",
    });

    deviceSync.remoteCommand("pause");
    expect(sendMock).toHaveBeenCalledWith("playback.command", {
      action: "pause",
    });

    deviceSync.createListenTogether();
    expect(sendMock).toHaveBeenCalledWith("session.create", {});

    deviceSync.joinListenTogether("lt-host");
    expect(sendMock).toHaveBeenCalledWith("session.join", {
      sessionId: "lt-host",
    });

    deviceSync.leaveListenTogether();
    expect(sendMock).toHaveBeenCalledWith("session.leave", {});
  });

  it("renames device over the wire", async () => {
    const { deviceSync } = await import("./device-sync.svelte");
    deviceSync.start();
    sendMock.mockClear();
    deviceSync.rename("Office Tablet");
    expect(deviceSync.deviceName).toBe("Office Tablet");
    expect(sendMock).toHaveBeenCalledWith("device.rename", {
      name: "Office Tablet",
    });
  });
});
