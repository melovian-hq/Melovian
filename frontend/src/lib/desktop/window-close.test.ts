// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { closePrompt } from "./close-prompt.svelte";
import { saveDesktopIntegrationSettings } from "./desktop-integration-settings";

const {
  persistPlaybackSnapshot,
  flushPlaybackState,
  nativeDesktopAvailable,
  QuitApp,
  HideMainWindow,
  MinimizeMainWindow,
  SetTaskbarIntegrationEnabled,
  eventHandlers,
  unbindFns,
} = vi.hoisted(() => ({
  persistPlaybackSnapshot: vi.fn(),
  flushPlaybackState: vi.fn().mockResolvedValue(undefined),
  nativeDesktopAvailable: vi.fn(() => true),
  QuitApp: vi.fn().mockResolvedValue(undefined),
  HideMainWindow: vi.fn().mockResolvedValue(undefined),
  MinimizeMainWindow: vi.fn().mockResolvedValue(undefined),
  SetTaskbarIntegrationEnabled: vi.fn().mockResolvedValue(undefined),
  eventHandlers: new Map<string, () => void>(),
  unbindFns: [] as Array<ReturnType<typeof vi.fn>>,
}));

vi.mock("$lib/config/music.svelte", () => ({
  music: {
    persistPlaybackSnapshot,
    flushPlaybackState,
  },
}));

vi.mock("@wailsio/runtime", () => ({
  Events: {
    On: vi.fn((event: string, handler: () => void) => {
      eventHandlers.set(event, handler);
      const unbind = vi.fn();
      unbindFns.push(unbind);
      return unbind;
    }),
  },
}));

vi.mock("@bindings/melovian/services/index.js", () => ({
  MediaService: {
    QuitApp,
    HideMainWindow,
    MinimizeMainWindow,
    SetTaskbarIntegrationEnabled,
  },
}));

vi.mock("$lib/config/runtime", () => ({
  nativeDesktopAvailable,
}));

import {
  APP_QUIT_REQUESTED_EVENT,
  bindDesktopIntegrationSettings,
  bindWindowCloseHandler,
  confirmCloseChoice,
  executeCloseAction,
  handleWindowCloseRequest,
  setCloseBehavior,
  setTaskbarIntegrationEnabled,
  WINDOW_CLOSE_REQUESTED_EVENT,
} from "./window-close";

describe("window-close", () => {
  beforeEach(() => {
    nativeDesktopAvailable.mockReturnValue(true);
    eventHandlers.clear();
    unbindFns.length = 0;
    closePrompt.hide();
    vi.clearAllMocks();
  });

  afterEach(() => {
    localStorage.clear();
    closePrompt.hide();
  });

  it("exports event names that match the Go desktop contract", () => {
    expect(WINDOW_CLOSE_REQUESTED_EVENT).toBe(
      "melovian:window-close-requested",
    );
    expect(APP_QUIT_REQUESTED_EVENT).toBe("melovian:app-quit-requested");
  });

  it("does not bind handlers when native desktop is unavailable", () => {
    nativeDesktopAvailable.mockReturnValue(false);
    const unbind = bindWindowCloseHandler();
    expect(eventHandlers.size).toBe(0);
    unbind();
  });

  it("binds close and quit handlers and unbinds on cleanup", async () => {
    const unbind = bindWindowCloseHandler();
    await vi.waitFor(() => {
      expect(eventHandlers.has(WINDOW_CLOSE_REQUESTED_EVENT)).toBe(true);
      expect(eventHandlers.has(APP_QUIT_REQUESTED_EVENT)).toBe(true);
    });
    unbind();
    expect(unbindFns).toHaveLength(2);
    for (const fn of unbindFns) {
      expect(fn).toHaveBeenCalledOnce();
    }
  });

  it("shows the close prompt when close behavior is ask", async () => {
    saveDesktopIntegrationSettings({
      taskbarEnabled: true,
      closeBehavior: "ask",
      nativeTitleBar: false,
    });

    await handleWindowCloseRequest();

    expect(closePrompt.open).toBe(true);
    expect(QuitApp).not.toHaveBeenCalled();
    expect(HideMainWindow).not.toHaveBeenCalled();
  });

  it("executes quit when close behavior is quit", async () => {
    saveDesktopIntegrationSettings({
      taskbarEnabled: true,
      closeBehavior: "quit",
      nativeTitleBar: false,
    });

    await handleWindowCloseRequest();

    expect(closePrompt.open).toBe(false);
    expect(persistPlaybackSnapshot).toHaveBeenCalledOnce();
    expect(flushPlaybackState).toHaveBeenCalledOnce();
    expect(QuitApp).toHaveBeenCalledOnce();
  });

  it("hides to tray when close behavior is background and tray is enabled", async () => {
    saveDesktopIntegrationSettings({
      taskbarEnabled: true,
      closeBehavior: "background",
      nativeTitleBar: false,
    });

    await executeCloseAction("background");

    expect(HideMainWindow).toHaveBeenCalledOnce();
    expect(MinimizeMainWindow).not.toHaveBeenCalled();
    expect(QuitApp).not.toHaveBeenCalled();
  });

  it("minimizes when close behavior resolves to minimize", async () => {
    await executeCloseAction("minimize");

    expect(MinimizeMainWindow).toHaveBeenCalledOnce();
    expect(HideMainWindow).not.toHaveBeenCalled();
  });

  it("persists playback before every close action", async () => {
    await executeCloseAction("quit");

    expect(persistPlaybackSnapshot.mock.invocationCallOrder[0]).toBeLessThan(
      flushPlaybackState.mock.invocationCallOrder[0],
    );
    expect(flushPlaybackState.mock.invocationCallOrder[0]).toBeLessThan(
      QuitApp.mock.invocationCallOrder[0],
    );
  });

  it("remembers close choice and maps minimize to background storage", async () => {
    closePrompt.show();

    await confirmCloseChoice({ action: "minimize", remember: true });

    expect(closePrompt.open).toBe(false);
    expect(
      JSON.parse(localStorage.getItem("mel-desktop-integration")!),
    ).toEqual({
      taskbarEnabled: true,
      closeBehavior: "background",
      nativeTitleBar: false,
    });
    expect(MinimizeMainWindow).toHaveBeenCalledOnce();
  });

  it("syncs taskbar integration on desktop settings bind", async () => {
    saveDesktopIntegrationSettings({
      taskbarEnabled: false,
      closeBehavior: "ask",
      nativeTitleBar: false,
    });

    bindDesktopIntegrationSettings();
    await vi.waitFor(() => {
      expect(SetTaskbarIntegrationEnabled).toHaveBeenCalledWith(false);
    });
  });

  it("updates taskbar integration when toggled from settings", async () => {
    setTaskbarIntegrationEnabled(false);

    expect(
      JSON.parse(localStorage.getItem("mel-desktop-integration")!),
    ).toEqual({
      taskbarEnabled: false,
      closeBehavior: "ask",
      nativeTitleBar: false,
    });
    await vi.waitFor(() => {
      expect(SetTaskbarIntegrationEnabled).toHaveBeenCalledWith(false);
    });
  });

  it("persists close behavior without touching tray settings", () => {
    setCloseBehavior("quit");

    expect(
      JSON.parse(localStorage.getItem("mel-desktop-integration")!),
    ).toEqual({
      taskbarEnabled: true,
      closeBehavior: "quit",
      nativeTitleBar: false,
    });
  });

  it("routes tray quit events to executeCloseAction quit", async () => {
    bindWindowCloseHandler();
    await vi.waitFor(() => {
      expect(eventHandlers.get(APP_QUIT_REQUESTED_EVENT)).toBeDefined();
    });
    const quitHandler = eventHandlers.get(APP_QUIT_REQUESTED_EVENT);
    expect(quitHandler).toBeDefined();

    quitHandler!();
    await vi.waitFor(() => {
      expect(QuitApp).toHaveBeenCalledOnce();
    });
  });
});
