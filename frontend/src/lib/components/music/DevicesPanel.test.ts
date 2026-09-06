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
  globalThis.matchMedia = ((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  })) as typeof window.matchMedia;
});

const {
  takeOver,
  transferTo,
  remoteCommand,
  joinListenTogether,
  createListenTogether,
  leaveListenTogether,
  inviteDevice,
  retryConnect,
  rename,
  deviceSyncState,
} = vi.hoisted(() => {
  const takeOver = vi.fn();
  const transferTo = vi.fn();
  const remoteCommand = vi.fn();
  const joinListenTogether = vi.fn();
  const createListenTogether = vi.fn();
  const leaveListenTogether = vi.fn();
  const inviteDevice = vi.fn();
  const retryConnect = vi.fn();
  const rename = vi.fn();
  const deviceSyncState = {
    panelOpen: false,
    deviceId: "self",
    deviceName: "This Laptop",
    devices: [] as SyncDevice[],
    peers: [] as SyncDevice[],
    sessionId: null as string | null,
    isHost: false,
    following: false,
    crossUser: false,
    lastError: null as string | null,
    pendingOp: null as string | null,
    linkState: "connected" as string,
    statusLabel: "Connected",
    sessionLabel: "",
    sessionMemberLabel: "",
    sessionMemberCount: 0,
    isSelfActive: false,
    get activePeer() {
      return (
        this.devices.find(
          (d: SyncDevice) => d.isActivePlayer && d.deviceId !== this.deviceId,
        ) ?? null
      );
    },
    get sessionMembers() {
      if (!this.sessionId) return [] as SyncDevice[];
      return this.devices.filter(
        (d: SyncDevice) => d.sessionId === this.sessionId,
      );
    },
  };
  return {
    takeOver,
    transferTo,
    remoteCommand,
    joinListenTogether,
    createListenTogether,
    leaveListenTogether,
    inviteDevice,
    retryConnect,
    rename,
    deviceSyncState,
  };
});

const selfDevice: SyncDevice = {
  deviceId: "self",
  name: "This Laptop",
  userAgent: "Mozilla/5.0 (X11; Linux x86_64)",
  connectedAt: 1,
  lastSeen: 1,
  isActivePlayer: false,
};

const idlePeer: SyncDevice = {
  deviceId: "peer-idle",
  name: "Idle Phone",
  userAgent: "Mozilla/5.0 (iPhone)",
  connectedAt: 1,
  lastSeen: 1,
  isActivePlayer: false,
};

const activePeer: SyncDevice = {
  deviceId: "peer-active",
  name: "Living Room",
  userAgent: "Mozilla/5.0 (Windows NT 10.0)",
  connectedAt: 1,
  lastSeen: 1,
  isActivePlayer: true,
  playback: {
    trackId: "t1",
    trackTitle: "Weathergirl",
    artistName: "Flavor Foley",
    coverArt: "cov_peer",
    positionMs: 1000,
    paused: false,
  },
};

vi.mock("$lib/music/device-sync.svelte", () => ({
  deviceSync: {
    get panelOpen() {
      return deviceSyncState.panelOpen;
    },
    set panelOpen(value: boolean) {
      deviceSyncState.panelOpen = value;
    },
    get deviceId() {
      return deviceSyncState.deviceId;
    },
    get deviceName() {
      return deviceSyncState.deviceName;
    },
    get devices() {
      return deviceSyncState.devices;
    },
    get peers() {
      return deviceSyncState.peers;
    },
    get sessionId() {
      return deviceSyncState.sessionId;
    },
    get isHost() {
      return deviceSyncState.isHost;
    },
    get following() {
      return deviceSyncState.following;
    },
    get crossUser() {
      return deviceSyncState.crossUser;
    },
    get lastError() {
      return deviceSyncState.lastError;
    },
    get pendingOp() {
      return deviceSyncState.pendingOp;
    },
    get linkState() {
      return deviceSyncState.linkState;
    },
    get statusLabel() {
      return deviceSyncState.statusLabel;
    },
    get sessionLabel() {
      return deviceSyncState.sessionLabel;
    },
    get sessionMemberLabel() {
      return deviceSyncState.sessionMemberLabel;
    },
    get sessionMemberCount() {
      return deviceSyncState.sessionMemberCount;
    },
    get sessionMembers() {
      return deviceSyncState.sessionMembers;
    },
    get inListenTogether() {
      return !!deviceSyncState.sessionId;
    },
    get isSelfActive() {
      return deviceSyncState.isSelfActive;
    },
    get activePeer() {
      return deviceSyncState.activePeer;
    },
    takeOver,
    transferTo,
    remoteCommand,
    joinListenTogether,
    createListenTogether,
    leaveListenTogether,
    inviteDevice,
    createInviteLink: vi.fn(async () => "/listen/test"),
    inviteByUsername: vi.fn(async () => true),
    retryConnect,
    rename,
  },
}));

vi.mock("$lib/features/auth/store.svelte", () => ({
  auth: {
    enabled: true,
    authenticated: true,
  },
}));

vi.mock("$lib/config/music.svelte", () => ({
  music: {
    config: {
      serverUrl: "http://localhost",
      version: "1.16.1",
      clientName: "melovian",
    },
    currentTrack: {
      id: "local-1",
      title: "Local Track",
      coverArt: "cov_local",
      albumId: "alb_local",
    },
  },
}));

vi.mock("$lib/components/ui/CoverArt.svelte", async () => ({
  default: (await import("../../../test-fixtures/Passthrough.svelte")).default,
}));

vi.mock("$lib/components/ui/Button.svelte", async () => ({
  default: (await import("../../../test-fixtures/Passthrough.svelte")).default,
}));

vi.mock("$lib/components/ui/Input.svelte", async () => ({
  default: (await import("../../../test-fixtures/Passthrough.svelte")).default,
}));

vi.mock("$lib/components/ui/MdiIcon.svelte", async () => ({
  default: (await import("../../../test-fixtures/Passthrough.svelte")).default,
}));

vi.mock("$lib/ui/toast.svelte", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    warning: vi.fn(),
  },
}));

import DevicesPanel from "./DevicesPanel.svelte";

function renderPanel() {
  const target = document.createElement("div");
  document.body.appendChild(target);
  const instance = mount(DevicesPanel, { target });
  flushSync();
  return {
    target,
    cleanup: () => {
      unmount(instance);
      target.remove();
    },
  };
}

describe("DevicesPanel", () => {
  beforeEach(() => {
    takeOver.mockClear();
    transferTo.mockClear();
    remoteCommand.mockClear();
    joinListenTogether.mockClear();
    createListenTogether.mockClear();
    leaveListenTogether.mockClear();
    inviteDevice.mockClear();
    retryConnect.mockClear();
    rename.mockClear();
    deviceSyncState.panelOpen = false;
    deviceSyncState.deviceId = "self";
    deviceSyncState.deviceName = "This Laptop";
    deviceSyncState.devices = [selfDevice];
    deviceSyncState.peers = [];
    deviceSyncState.sessionId = null;
    deviceSyncState.isHost = false;
    deviceSyncState.following = false;
    deviceSyncState.crossUser = false;
    deviceSyncState.lastError = null;
    deviceSyncState.pendingOp = null;
    deviceSyncState.linkState = "connected";
    deviceSyncState.statusLabel = "Connected";
    deviceSyncState.sessionLabel = "";
    deviceSyncState.sessionMemberLabel = "";
    deviceSyncState.sessionMemberCount = 0;
    deviceSyncState.isSelfActive = false;
  });

  it("renders nothing when the panel is closed", () => {
    const { target, cleanup } = renderPanel();
    expect(target.querySelector(".devices-panel")).toBeNull();
    cleanup();
  });

  it("shows glass shell, ambient wash, and empty peers copy", () => {
    deviceSyncState.panelOpen = true;
    const { target, cleanup } = renderPanel();

    expect(target.querySelector(".devices-panel")).toBeTruthy();
    expect(target.querySelector(".ambient-cover-backdrop")).toBeTruthy();
    expect(target.textContent).toContain("No other devices connected yet.");
    cleanup();
  });

  it("shows transfer and transport icons for an active peer only", () => {
    deviceSyncState.panelOpen = true;
    deviceSyncState.devices = [selfDevice, idlePeer, activePeer];
    deviceSyncState.peers = [idlePeer, activePeer];
    const { target, cleanup } = renderPanel();

    expect(target.textContent).toContain("Living Room");
    expect(target.textContent).toContain("Idle Phone");
    expect(target.textContent).toContain("Transfer");

    const playButtons = target.querySelectorAll(
      '[aria-label="Play on that device"]',
    );
    const pauseButtons = target.querySelectorAll(
      '[aria-label="Pause that device"]',
    );
    const nextButtons = target.querySelectorAll(
      '[aria-label="Next on that device"]',
    );
    expect(playButtons.length).toBe(1);
    expect(pauseButtons.length).toBe(1);
    expect(nextButtons.length).toBe(1);
    expect(target.querySelector(".device-avatar--active")).toBeTruthy();
    expect(target.querySelectorAll(".device-avatar").length).toBeGreaterThan(1);
    cleanup();
  });

  it("transfers playback to the peer, not this device", () => {
    deviceSyncState.panelOpen = true;
    deviceSyncState.isSelfActive = true;
    deviceSyncState.devices = [selfDevice, idlePeer];
    deviceSyncState.peers = [idlePeer];
    const { target, cleanup } = renderPanel();

    const transfer = [...target.querySelectorAll("button")].find(
      (btn) => btn.textContent?.trim() === "Transfer",
    );
    expect(transfer).toBeTruthy();
    transfer?.click();
    expect(transferTo).toHaveBeenCalledWith("peer-idle");
    expect(takeOver).not.toHaveBeenCalled();
    cleanup();
  });

  it("shows connection status and listen-together role", () => {
    deviceSyncState.panelOpen = true;
    deviceSyncState.linkState = "connecting";
    deviceSyncState.statusLabel = "Connecting to other devices…";
    deviceSyncState.sessionId = "lt-1";
    deviceSyncState.isHost = true;
    deviceSyncState.sessionLabel = "Hosting listen together";
    deviceSyncState.sessionMemberLabel = "This Laptop (you, host)";
    deviceSyncState.sessionMemberCount = 1;
    deviceSyncState.devices = [{ ...selfDevice, sessionId: "lt-1" }];
    const { target, cleanup } = renderPanel();

    expect(target.textContent).toContain("Connecting to other devices…");
    expect(target.textContent).toContain("Hosting listen together");
    expect(target.querySelector(".avatar-stack")).toBeTruthy();
    cleanup();
  });

  it("renders device avatars for peers and session stack", () => {
    deviceSyncState.panelOpen = true;
    deviceSyncState.sessionId = "lt-2";
    deviceSyncState.isHost = true;
    deviceSyncState.sessionLabel = "Listening together · 2";
    deviceSyncState.sessionMemberLabel = "This Laptop (you, host), Idle Phone";
    deviceSyncState.sessionMemberCount = 2;
    deviceSyncState.devices = [
      { ...selfDevice, sessionId: "lt-2" },
      { ...idlePeer, sessionId: "lt-2" },
      activePeer,
    ];
    deviceSyncState.peers = [{ ...idlePeer, sessionId: "lt-2" }, activePeer];
    const { target, cleanup } = renderPanel();

    expect(target.querySelectorAll(".device-avatar").length).toBeGreaterThan(1);
    expect(target.querySelector(".avatar-stack")).toBeTruthy();
    expect(target.querySelector(".devices-panel__together-stack")).toBeTruthy();
    cleanup();
  });

  it("does not show transport icons for idle peers alone", () => {
    deviceSyncState.panelOpen = true;
    deviceSyncState.devices = [selfDevice, idlePeer];
    deviceSyncState.peers = [idlePeer];
    const { target, cleanup } = renderPanel();

    expect(
      target.querySelector('[aria-label="Play on that device"]'),
    ).toBeNull();
    expect(target.querySelector(".device-avatar--active")).toBeNull();
    cleanup();
  });

  it("shows invite on peers when hosting listen together", () => {
    deviceSyncState.panelOpen = true;
    deviceSyncState.sessionId = "lt-host";
    deviceSyncState.isHost = true;
    deviceSyncState.sessionMemberCount = 1;
    deviceSyncState.devices = [
      { ...selfDevice, sessionId: "lt-host" },
      idlePeer,
    ];
    deviceSyncState.peers = [idlePeer];
    const { target, cleanup } = renderPanel();

    const invite = Array.from(target.querySelectorAll("button")).find((btn) =>
      btn.textContent?.includes("Invite"),
    );
    expect(invite).toBeTruthy();
    invite?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    expect(inviteDevice).toHaveBeenCalledWith("peer-idle");
    cleanup();
  });
});
