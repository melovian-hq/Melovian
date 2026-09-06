// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export function downloadBlob(filename: string, blob: Blob): void {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.rel = "noopener";
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

export function downloadTextFile(
  filename: string,
  content: string,
  mimeType = "text/plain;charset=utf-8",
): void {
  downloadBlob(filename, new Blob([content], { type: mimeType }));
}

export function sanitizeFilename(name: string, fallback = "download"): string {
  const trimmed = name
    .trim()
    .replace(/[\\/:*?"<>|]/g, "_")
    .replace(/\s+/g, " ");
  const collapsed = trimmed.replace(/_+/g, "_").replace(/^\.+/, "");
  if (!collapsed || /^_+$/.test(collapsed)) return fallback;
  return collapsed.slice(0, 180);
}

export async function downloadFromUrl(
  url: string,
  filename: string,
  init?: RequestInit,
): Promise<void> {
  const response = await fetch(url, init);
  if (!response.ok) {
    throw new Error(`Download failed (${response.status})`);
  }
  const blob = await response.blob();
  downloadBlob(filename, blob);
}
