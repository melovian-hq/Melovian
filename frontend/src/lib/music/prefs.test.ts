// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it } from "vitest";
import { setActiveInstanceId } from "$lib/features/instances/context";
import {
  clearSavedPlayback,
  clampQueuePanelPosition,
  loadQueuePanelPosition,
  loadSavedPlayback,
  loadVolume,
  savePlayback,
  saveQueuePanelPosition,
  saveVolume,
} from "./prefs";
import { migrateContinuousMode } from "./continuous-pool";

describe("music prefs", () => {
  afterEach(() => {
    clearSavedPlayback();
    localStorage.removeItem("mel-music-volume");
    localStorage.removeItem("mel-queue-panel-position");
    localStorage.removeItem("mel-music-playback");
    setActiveInstanceId(null);
  });

  it("persists and restores playback state", () => {
    setActiveInstanceId("server-a");
    savePlayback({
      trackIds: ["t1", "t2"],
      queueIndex: 1,
      positionMs: 45000,
      shuffle: true,
      autoplay: false,
    });

    const saved = loadSavedPlayback();
    expect(saved).toEqual({
      trackIds: ["t1", "t2"],
      queueIndex: 1,
      positionMs: 45000,
      shuffle: true,
      autoplay: false,
    });
  });

  it("persists and restores continuous mode with legacy randomRadio migration", () => {
    setActiveInstanceId("server-a");
    savePlayback({
      trackIds: ["t1"],
      queueIndex: 0,
      positionMs: 0,
      shuffle: true,
      autoplay: true,
      randomRadio: true,
    });

    expect(loadSavedPlayback()?.randomRadio).toBe(true);
    expect(migrateContinuousMode(loadSavedPlayback() ?? {})).toBe("random");

    savePlayback({
      trackIds: ["t1"],
      queueIndex: 0,
      positionMs: 0,
      shuffle: true,
      autoplay: true,
      continuousMode: "personal",
    });
    expect(loadSavedPlayback()?.continuousMode).toBe("personal");
  });

  it("scopes playback state per active instance", () => {
    setActiveInstanceId("server-a");
    savePlayback({
      trackIds: ["a1"],
      queueIndex: 0,
      positionMs: 1000,
      shuffle: false,
      autoplay: true,
    });

    setActiveInstanceId("server-b");
    savePlayback({
      trackIds: ["b1"],
      queueIndex: 0,
      positionMs: 2000,
      shuffle: false,
      autoplay: true,
    });

    expect(loadSavedPlayback()?.trackIds).toEqual(["b1"]);

    setActiveInstanceId("server-a");
    expect(loadSavedPlayback()?.trackIds).toEqual(["a1"]);
  });

  it("rejects invalid saved playback payloads", () => {
    localStorage.setItem("mel-music-playback", JSON.stringify({ trackIds: [] }));
    expect(loadSavedPlayback()).toBeNull();

    localStorage.setItem("mel-music-playback", "not-json");
    expect(loadSavedPlayback()).toBeNull();
  });

  it("persists volume within bounds", () => {
    saveVolume(0.42);
    expect(loadVolume()).toBe(0.42);

    saveVolume(2);
    expect(loadVolume()).toBe(1);

    saveVolume(-1);
    expect(loadVolume()).toBe(0);
  });

  it("persists and restores queue panel position", () => {
    saveQueuePanelPosition({ x: 120, y: 240 });
    expect(loadQueuePanelPosition()).toEqual({ x: 120, y: 240 });
  });

  it("rejects invalid queue panel position payloads", () => {
    localStorage.setItem(
      "mel-queue-panel-position",
      JSON.stringify({ x: "bad" }),
    );
    expect(loadQueuePanelPosition()).toBeNull();
  });

  it("clamps queue panel position to viewport bounds", () => {
    expect(
      clampQueuePanelPosition({ x: 900, y: -20 }, 200, 100, 800, 600),
    ).toEqual({ x: 600, y: 0 });
  });
});
