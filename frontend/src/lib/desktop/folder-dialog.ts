// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { nativeDesktopAvailable } from "$lib/config/runtime";

export function folderDialogAvailable(): boolean {
  return nativeDesktopAvailable();
}

export async function pickFolder(options?: {
  title?: string;
  directory?: string;
}): Promise<string | null> {
  if (!folderDialogAvailable()) return null;

  try {
    const { Dialogs } = await import("@wailsio/runtime");
    const selected = await Dialogs.OpenFile({
      Title: options?.title ?? "Select music folder",
      CanChooseDirectories: true,
      CanChooseFiles: false,
      Directory: options?.directory?.trim() || undefined,
    });
    const path = (Array.isArray(selected) ? selected[0] : selected)?.trim();
    return path || null;
  } catch {
    return null;
  }
}

export function folderNameFromPath(path: string): string {
  const trimmed = path.replace(/\/+$/, "");
  const parts = trimmed.split("/").filter(Boolean);
  return parts.at(-1) ?? trimmed;
}
