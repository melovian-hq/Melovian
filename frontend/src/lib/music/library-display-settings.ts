// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

const STORAGE_KEY = "mel-hide-unknown-metadata";

export function loadHideUnknownMetadata(): boolean {
  if (typeof localStorage === "undefined") return false;
  try {
    return localStorage.getItem(STORAGE_KEY) === "true";
  } catch {
    return false;
  }
}

export function saveHideUnknownMetadata(hide: boolean): void {
  if (typeof localStorage === "undefined") return;
  try {
    localStorage.setItem(STORAGE_KEY, hide ? "true" : "false");
  } catch {
    /* storage full or unavailable */
  }
}
