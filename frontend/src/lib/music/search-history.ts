// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

const STORAGE_KEY = "mel-search-history";
const MAX_ITEMS = 6;

export function loadSearchHistory(): string[] {
  if (typeof localStorage === "undefined") return [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((entry): entry is string => typeof entry === "string");
  } catch {
    return [];
  }
}

export function saveSearchHistory(history: string[]): void {
  if (typeof localStorage === "undefined") return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(history));
  } catch {
    /* storage full or unavailable */
  }
}

export function addSearchHistory(term: string, history: string[]): string[] {
  const trimmed = term.trim();
  if (!trimmed) return history;
  const needle = trimmed.toLowerCase();
  const next = [
    trimmed,
    ...history.filter((entry) => entry.toLowerCase() !== needle),
  ];
  return next.slice(0, MAX_ITEMS);
}
