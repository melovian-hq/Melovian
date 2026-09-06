// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { music } from "$lib/config/music.svelte";
import { nativeDesktopAvailable } from "$lib/config/runtime";
import { closePrompt } from "$lib/desktop/close-prompt.svelte";
import {
  loadDesktopIntegrationSettings,
  resolveCloseAction,
  saveDesktopIntegrationSettings,
  type CloseBehavior,
  type ResolvedCloseAction,
} from "$lib/desktop/desktop-integration-settings";
import { logger } from "$lib/core/logger";
import { APP_SLUG } from "$lib/brand";

type MediaServiceModule = typeof import("@bindings/melovian/services/index.js");

export const WINDOW_CLOSE_REQUESTED_EVENT = `${APP_SLUG}:window-close-requested`;
export const APP_QUIT_REQUESTED_EVENT = `${APP_SLUG}:app-quit-requested`;

export function bindWindowCloseHandler(): () => void {
  if (!nativeDesktopAvailable()) return () => {};
  let disposed = false;
  let unbindClose = () => {};
  let unbindQuit = () => {};

  void import("@wailsio/runtime")
    .then(({ Events }) => {
      if (disposed) return;
      unbindClose = Events.On(WINDOW_CLOSE_REQUESTED_EVENT, () => {
        void handleWindowCloseRequest();
      });
      unbindQuit = Events.On(APP_QUIT_REQUESTED_EVENT, () => {
        void executeCloseAction("quit");
      });
    })
    .catch((err) => {
      logger.warn(
        "Wails events unavailable",
        err instanceof Error ? { err: err.message } : { err: String(err) },
        "desktop.window-close",
      );
    });

  return () => {
    disposed = true;
    unbindClose();
    unbindQuit();
  };
}

export async function handleWindowCloseRequest(): Promise<void> {
  if (!nativeDesktopAvailable()) return;

  const settings = loadDesktopIntegrationSettings();
  const action = resolveCloseAction(settings);
  if (action === "ask") {
    closePrompt.show();
    return;
  }
  await executeCloseAction(action);
}

export async function executeCloseAction(
  action: ResolvedCloseAction,
): Promise<void> {
  if (!nativeDesktopAvailable()) return;

  music.persistPlaybackSnapshot();
  await music.flushPlaybackState();

  const { MediaService } = await import("@bindings/melovian/services/index.js");
  switch (action) {
    case "quit":
      await MediaService.QuitApp();
      break;
    case "background":
      await MediaService.HideMainWindow();
      break;
    case "minimize":
      await MediaService.MinimizeMainWindow();
      break;
  }
}

export async function confirmCloseChoice(options: {
  action: ResolvedCloseAction;
  remember: boolean;
}): Promise<void> {
  closePrompt.hide();

  if (options.remember) {
    const settings = loadDesktopIntegrationSettings();
    const closeBehavior: CloseBehavior =
      options.action === "quit" ? "quit" : "background";
    saveDesktopIntegrationSettings({
      ...settings,
      closeBehavior,
    });
  }

  await executeCloseAction(options.action);
}

export function bindDesktopIntegrationSettings(): () => void {
  if (!nativeDesktopAvailable()) return () => {};
  void syncTaskbarIntegration();
  return () => {};
}

async function syncTaskbarIntegration(
  enabled = loadDesktopIntegrationSettings().taskbarEnabled,
): Promise<void> {
  const { MediaService }: MediaServiceModule =
    await import("@bindings/melovian/services/index.js");
  await MediaService.SetTaskbarIntegrationEnabled(enabled);
}

export function setTaskbarIntegrationEnabled(enabled: boolean): void {
  const settings = loadDesktopIntegrationSettings();
  saveDesktopIntegrationSettings({ ...settings, taskbarEnabled: enabled });
  void syncTaskbarIntegration(enabled);
}

export function setCloseBehavior(behavior: CloseBehavior): void {
  const settings = loadDesktopIntegrationSettings();
  saveDesktopIntegrationSettings({ ...settings, closeBehavior: behavior });
}
