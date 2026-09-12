// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { StorageKeys } from "$lib/brand";

const STORAGE_KEY = StorageKeys.desktopIntegration;

/** Window event fired when desktop integration settings change. */
export const DESKTOP_INTEGRATION_CHANGED_EVENT = `${STORAGE_KEY}-changed`;

export type CloseBehavior = "ask" | "quit" | "background";

export interface DesktopIntegrationSettings {
  taskbarEnabled: boolean;
  closeBehavior: CloseBehavior;
  nativeTitleBar: boolean;
}

export function defaultDesktopIntegrationSettings(): DesktopIntegrationSettings {
  return {
    taskbarEnabled: true,
    closeBehavior: "ask",
    nativeTitleBar: false,
  };
}

export function mergeDesktopIntegrationSettings(
  partial: Partial<DesktopIntegrationSettings> | null | undefined,
): DesktopIntegrationSettings {
  const defaults = defaultDesktopIntegrationSettings();
  if (!partial) return defaults;
  const closeBehavior =
    partial.closeBehavior === "ask" ||
    partial.closeBehavior === "quit" ||
    partial.closeBehavior === "background"
      ? partial.closeBehavior
      : defaults.closeBehavior;
  return {
    taskbarEnabled:
      typeof partial.taskbarEnabled === "boolean"
        ? partial.taskbarEnabled
        : defaults.taskbarEnabled,
    closeBehavior,
    nativeTitleBar:
      typeof partial.nativeTitleBar === "boolean"
        ? partial.nativeTitleBar
        : defaults.nativeTitleBar,
  };
}

export function loadDesktopIntegrationSettings(): DesktopIntegrationSettings {
  if (typeof localStorage === "undefined") {
    return defaultDesktopIntegrationSettings();
  }
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return defaultDesktopIntegrationSettings();
    return mergeDesktopIntegrationSettings(
      JSON.parse(raw) as Partial<DesktopIntegrationSettings>,
    );
  } catch {
    return defaultDesktopIntegrationSettings();
  }
}

export function saveDesktopIntegrationSettings(
  settings: DesktopIntegrationSettings,
): void {
  if (typeof localStorage === "undefined") return;
  localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
}

export type ResolvedCloseAction = "quit" | "background" | "minimize";

export function resolveCloseAction(
  settings: DesktopIntegrationSettings,
): ResolvedCloseAction | "ask" {
  if (settings.closeBehavior === "ask") return "ask";
  if (settings.closeBehavior === "quit") return "quit";
  if (!settings.taskbarEnabled) return "minimize";
  return "background";
}
