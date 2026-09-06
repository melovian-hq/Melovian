// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi, beforeEach } from "vitest";
import { flushSync, mount, unmount } from "svelte";
import type { SyncDevice } from "$lib/music/device-sync.svelte";

vi.hoisted(() => {
  const storage = new Map<string, string>();
  globalThis.localStorage = {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => {
      storage.set(key, value);
    },
    removeItem: (key: string) => {
      storage.delete(key);
    },
    clear: () => storage.clear(),
    key: () => null,
    length: 0,
  } as Storage;
});

const { deviceSyncState, togglePanel } = vi.hoisted(() => {
  const togglePanel = vi.fn();
  const deviceSyncState = {
    deviceId: "self",
    sessionId: null as string | null,
    sessionLabel: "",
    sessionMemberCount: 0,
    sessionMembers: [] as SyncDevice[],
    crossUser: false,
  };
  return { deviceSyncState, togglePanel };
});

vi.mock("$lib/music/device-sync.svelte", () => ({
  deviceSync: {
    get deviceId() {
      return deviceSyncState.deviceId;
    },
    get sessionId() {
      return deviceSyncState.sessionId;
    },
    get sessionLabel() {
      return deviceSyncState.sessionLabel;
    },
    get sessionMemberCount() {
      return deviceSyncState.sessionMemberCount;
    },
    get sessionMembers() {
      return deviceSyncState.sessionMembers;
    },
    get crossUser() {
      return deviceSyncState.crossUser;
    },
    get inListenTogether() {
      return !!deviceSyncState.sessionId;
    },
    togglePanel,
  },
}));

import ListenTogetherChip from "./ListenTogetherChip.svelte";

function renderChip(compact = false) {
  const target = document.createElement("div");
  document.body.appendChild(target);
  const instance = mount(ListenTogetherChip, {
    target,
    props: { compact },
  });
  flushSync();
  return {
    target,
    cleanup: () => {
      unmount(instance);
      target.remove();
    },
  };
}

describe("ListenTogetherChip", () => {
  beforeEach(() => {
    togglePanel.mockClear();
    deviceSyncState.deviceId = "self";
    deviceSyncState.sessionId = null;
    deviceSyncState.sessionLabel = "";
    deviceSyncState.sessionMemberCount = 0;
    deviceSyncState.sessionMembers = [];
  });

  it("renders nothing when not in a session", () => {
    const { target, cleanup } = renderChip();
    expect(target.querySelector(".together-chip")).toBeNull();
    cleanup();
  });

  it("shows text label for session without avatar stack", () => {
    deviceSyncState.sessionId = "lt-1";
    deviceSyncState.sessionLabel = "Listening together · 2";
    deviceSyncState.sessionMemberCount = 2;
    deviceSyncState.sessionMembers = [
      {
        deviceId: "self",
        name: "This Laptop",
        userAgent: "",
        connectedAt: 1,
        lastSeen: 1,
        isActivePlayer: true,
        sessionId: "lt-1",
      },
      {
        deviceId: "peer-1",
        name: "Phone",
        userAgent: "",
        connectedAt: 1,
        lastSeen: 1,
        isActivePlayer: false,
        sessionId: "lt-1",
      },
    ];
    const { target, cleanup } = renderChip();

    expect(target.querySelector(".together-chip")).toBeTruthy();
    expect(target.querySelector(".avatar-stack")).toBeNull();
    expect(target.querySelectorAll(".device-avatar").length).toBe(0);
    expect(target.textContent).toContain("Listening together · 2");
    cleanup();
  });
});
