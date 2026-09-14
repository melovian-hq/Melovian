// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { QueueTrack } from "$lib/subsonic";
import { connection } from "$lib/music/connection.svelte";
import { OFFLINE_SUSPECT_SKIP_THRESHOLD } from "$lib/music/offline-gate";
import {
  disconnect,
  markPendingReconnectResume,
  resumeAfterReconnect,
  suspendForReconnect,
  type MusicConnectContext,
} from "./music/connect-ops";
import {
  skipFailedTrack,
  type MusicTrackBoundaryContext,
} from "./music/track-boundary-ops";

function track(id: string): QueueTrack {
  return {
    id,
    title: id,
    artist: "a",
    album: "b",
    duration: 200,
  } as QueueTrack;
}

/**
 * The acceptance harness shares one mutable state bag between the connect
 * context and the track-boundary context, the same way MusicStore fields back
 * both in production. Mutable fields delegate through getters and setters so
 * ops writes stay visible to assertions and to the sibling context.
 */
function makeHarness(queueLength = 6) {
  const queue = Array.from({ length: queueLength }, (_, i) => track(`t${i}`));
  const engine = {
    pause: vi.fn(),
    play: vi.fn(async () => {}),
    currentTime: 0,
  };
  const state = {
    queue,
    queueIndex: 2,
    playing: true,
    currentTime: 42.5,
    reconnectResumePending: false,
    reconnectPositionMs: 0,
    reconnectResumeTrackId: null as string | null,
    failedTrackSkips: 0,
    error: null as string | null,
  };
  const calls = {
    playCurrent: vi.fn(async () => {}),
    seek: vi.fn(),
    stopSmoothProgress: vi.fn(),
    syncMediaSession: vi.fn(),
    persistPlaybackState: vi.fn(),
    stopLibraryWatch: vi.fn(),
    advanceTrack: vi.fn(() => {
      if (state.queueIndex >= state.queue.length - 1) return false;
      state.queueIndex += 1;
      return true;
    }),
    stopAtQueueEnd: vi.fn(() => {
      state.playing = false;
    }),
  };

  const sharedFields = {
    get queue() {
      return state.queue;
    },
    get queueIndex() {
      return state.queueIndex;
    },
    set queueIndex(v: number) {
      state.queueIndex = v;
    },
    get playing() {
      return state.playing;
    },
    set playing(v: boolean) {
      state.playing = v;
    },
    get currentTime() {
      return state.currentTime;
    },
    set currentTime(v: number) {
      state.currentTime = v;
    },
    get reconnectResumePending() {
      return state.reconnectResumePending;
    },
    set reconnectResumePending(v: boolean) {
      state.reconnectResumePending = v;
    },
    get reconnectPositionMs() {
      return state.reconnectPositionMs;
    },
    set reconnectPositionMs(v: number) {
      state.reconnectPositionMs = v;
    },
    get reconnectResumeTrackId() {
      return state.reconnectResumeTrackId;
    },
    set reconnectResumeTrackId(v: string | null) {
      state.reconnectResumeTrackId = v;
    },
    get failedTrackSkips() {
      return state.failedTrackSkips;
    },
    set failedTrackSkips(v: number) {
      state.failedTrackSkips = v;
    },
    get error() {
      return state.error;
    },
    set error(v: string | null) {
      state.error = v;
    },
    get currentTrack() {
      return state.queue[state.queueIndex] ?? null;
    },
  };

  // Spread would evaluate the accessors into plain values, so copy the
  // descriptors to keep the delegation live.
  const descriptors = Object.getOwnPropertyDescriptors(sharedFields);

  const connectCtx = Object.defineProperties(
    { engine, ...calls },
    descriptors,
  ) as unknown as MusicConnectContext;

  const boundaryCtx = Object.defineProperties(
    {
      playbackEpoch: 3,
      engine,
      suspendForReconnect: () => suspendForReconnect(connectCtx),
      markPendingReconnectResume: () => markPendingReconnectResume(connectCtx),
      ...calls,
    },
    descriptors,
  ) as unknown as MusicTrackBoundaryContext;

  return { state, calls, connectCtx, boundaryCtx, engine };
}

function resetConnection() {
  connection.dispose();
  connection.managed = false;
  connection.serverOnline = false;
  connection.browserOnline = true;
  connection.phase = "offline";
  connection.reconnectAttempt = 0;
  connection.lastDisconnectedAt = null;
  connection.lastConnectedAt = null;
  connection.nextRetryAt = null;
  connection.history = [];
}

describe("offline playback hold (acceptance)", () => {
  beforeEach(resetConnection);
  afterEach(() => connection.dispose());

  it("marks a resume even when the failure lands mid-skip while paused", () => {
    const { state, connectCtx, engine } = makeHarness();
    // The cascade sets playing=false before skipFailedTrack runs, so suspend
    // must still record the resume intent or reconnect finds nothing to do.
    state.playing = false;

    suspendForReconnect(connectCtx);

    expect(state.reconnectResumePending).toBe(true);
    expect(state.reconnectPositionMs).toBe(42500);
    expect(engine.pause).toHaveBeenCalledTimes(1);
    expect(state.playing).toBe(false);
  });

  it("does nothing without a current track", () => {
    const { state, connectCtx } = makeHarness();
    state.queueIndex = -1;

    suspendForReconnect(connectCtx);

    expect(state.reconnectResumePending).toBe(false);
  });

  it("stops advancing the queue once the server is unreachable", () => {
    const { state, calls, boundaryCtx } = makeHarness();
    connection.managed = true;
    connection.onServerConnected();
    connection.onServerDisconnected("stream failed");
    state.playing = false;

    skipFailedTrack(boundaryCtx, 3, "stream failed");

    expect(calls.advanceTrack).not.toHaveBeenCalled();
    expect(state.queueIndex).toBe(2);
    expect(state.reconnectResumePending).toBe(true);
    expect(calls.persistPlaybackState).toHaveBeenCalledTimes(1);
  });

  it("burns at most a few tracks before repeated failures hold the queue", () => {
    const { state, calls, boundaryCtx } = makeHarness();
    connection.managed = true;
    connection.serverOnline = true;
    state.playing = false;

    for (let i = 0; i < 12; i += 1) {
      skipFailedTrack(boundaryCtx, 3, `failure ${i}`);
    }

    expect(calls.advanceTrack).toHaveBeenCalledTimes(
      OFFLINE_SUSPECT_SKIP_THRESHOLD - 1,
    );
    expect(state.queueIndex).toBe(2 + OFFLINE_SUSPECT_SKIP_THRESHOLD - 1);
    expect(connection.serverOnline).toBe(false);
    expect(state.reconnectResumePending).toBe(true);
  });

  it("resumes the held track at the saved position after reconnect", async () => {
    const { state, calls, connectCtx } = makeHarness();
    state.playing = false;
    suspendForReconnect(connectCtx);

    connection.managed = true;
    connection.onServerConnected();
    await resumeAfterReconnect(connectCtx);

    expect(state.reconnectResumePending).toBe(false);
    expect(state.reconnectPositionMs).toBe(0);
    expect(calls.playCurrent).toHaveBeenCalledTimes(1);
    expect(calls.seek).toHaveBeenCalledWith(42.5);
  });

  it("does not seek a different track to the parked position", async () => {
    const { state, calls, connectCtx } = makeHarness();
    state.playing = false;
    suspendForReconnect(connectCtx);
    expect(state.reconnectResumeTrackId).toBe("t2");

    // The user picks another track while suspended and it fails too, so the
    // pending resume now belongs to the newer track at its own position, not
    // the 42.5s parked against t2.
    state.queueIndex = 4;
    state.currentTime = 5;
    suspendForReconnect(connectCtx);
    expect(state.reconnectResumeTrackId).toBe("t4");
    expect(state.reconnectPositionMs).toBe(5000);

    connection.managed = true;
    connection.onServerConnected();
    await resumeAfterReconnect(connectCtx);

    expect(calls.playCurrent).toHaveBeenCalledTimes(1);
    expect(calls.seek).toHaveBeenCalledWith(5);
    expect(state.reconnectResumeTrackId).toBeNull();
  });

  it("does not resume when nothing was suspended", async () => {
    const { calls, connectCtx } = makeHarness();

    await resumeAfterReconnect(connectCtx);

    expect(calls.playCurrent).not.toHaveBeenCalled();
    expect(calls.seek).not.toHaveBeenCalled();
  });

  it("consumes the parked resume when the queue was torn down", async () => {
    const { state, calls, connectCtx } = makeHarness();
    state.playing = false;
    suspendForReconnect(connectCtx);
    expect(state.reconnectResumePending).toBe(true);

    // The user cleared the queue while suspended. The flag must be consumed
    // anyway so a later unrelated reconnect cannot playCurrent into the
    // rebuilt queue.
    state.queue = [];
    state.queueIndex = -1;
    connection.managed = true;
    connection.onServerConnected();
    await resumeAfterReconnect(connectCtx);

    expect(state.reconnectResumePending).toBe(false);
    expect(state.reconnectResumeTrackId).toBeNull();
    expect(state.reconnectPositionMs).toBe(0);
    expect(calls.playCurrent).not.toHaveBeenCalled();
    expect(calls.seek).not.toHaveBeenCalled();
  });

  it("disconnect clears the parked resume and tears down silently", async () => {
    const { state, calls, connectCtx } = makeHarness();
    const onDisconnected = vi.fn();
    connection.init(
      vi.fn(async () => true),
      vi.fn(async () => true),
      vi.fn(async () => {}),
      { onDisconnected },
    );
    // Let init's first tryConnect land so the store is online before the
    // deliberate disconnect.
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(connection.serverOnline).toBe(true);
    expect(connection.managed).toBe(true);

    state.playing = false;
    suspendForReconnect(connectCtx);
    expect(state.reconnectResumePending).toBe(true);

    disconnect(connectCtx);

    // Deliberate sign-out is not an outage: no disconnect callback, no
    // history entry, and no scheduled reconnect.
    expect(onDisconnected).not.toHaveBeenCalled();
    expect(connection.history).toEqual([]);
    expect(connection.managed).toBe(false);
    expect(connection.serverOnline).toBe(false);
    expect(connection.nextRetryAt).toBeNull();
    expect(connection.lastDisconnectedAt).toBeNull();
    expect(state.reconnectResumePending).toBe(false);
    expect(state.reconnectResumeTrackId).toBeNull();
    expect(calls.playCurrent).not.toHaveBeenCalled();
    expect(calls.stopLibraryWatch).toHaveBeenCalledTimes(1);
  });
});
