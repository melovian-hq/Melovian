// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { nativeDesktopAvailable } from "$lib/config/runtime";
import {
  DESKTOP_INTEGRATION_CHANGED_EVENT,
  loadDesktopIntegrationSettings,
  saveDesktopIntegrationSettings,
  type DesktopIntegrationSettings,
} from "$lib/desktop/desktop-integration-settings";

type MediaServiceModule = typeof import("@bindings/melovian/services/index.js");

export function showCustomWindowControls(
  settings: DesktopIntegrationSettings = loadDesktopIntegrationSettings(),
): boolean {
  return nativeDesktopAvailable() && !settings.nativeTitleBar;
}

async function syncNativeTitleBar(
  enabled = loadDesktopIntegrationSettings().nativeTitleBar,
): Promise<void> {
  const { MediaService }: MediaServiceModule =
    await import("@bindings/melovian/services/index.js");
  await MediaService.SetNativeTitleBarEnabled(enabled);
}

export function bindNativeTitleBarSettings(): () => void {
  if (!nativeDesktopAvailable()) return () => {};
  applyWindowChromeDocumentState();
  void syncNativeTitleBar();
  return () => {};
}

export function setNativeTitleBarEnabled(enabled: boolean): void {
  const settings = loadDesktopIntegrationSettings();
  saveDesktopIntegrationSettings({ ...settings, nativeTitleBar: enabled });
  void syncNativeTitleBar(enabled);
  applyWindowChromeDocumentState({ ...settings, nativeTitleBar: enabled });
  if (typeof window !== "undefined") {
    window.dispatchEvent(new Event(DESKTOP_INTEGRATION_CHANGED_EVENT));
  }
}

export function applyWindowChromeDocumentState(
  settings: DesktopIntegrationSettings = loadDesktopIntegrationSettings(),
): void {
  if (typeof document === "undefined" || !nativeDesktopAvailable()) return;
  const custom = showCustomWindowControls(settings);
  document.documentElement.dataset.nativeTitleBar = settings.nativeTitleBar
    ? "true"
    : "false";
  document.documentElement.dataset.customWindowChrome = custom
    ? "true"
    : "false";
  document.documentElement.style.setProperty(
    "--jb-window-chrome-offset",
    custom ? "var(--jb-window-chrome-height, 2rem)" : "0px",
  );
  document.documentElement.style.setProperty(
    "--jb-window-controls-inset",
    custom ? "var(--jb-window-controls-width, 8.25rem)" : "0px",
  );
}

export async function handleTitleBarDoubleClick(): Promise<void> {
  if (!nativeDesktopAvailable()) return;
  const { MediaService }: MediaServiceModule =
    await import("@bindings/melovian/services/index.js");
  await MediaService.HandleTitleBarDoubleClick();
}
