// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { applyConnectionDefaults } from "./connection-defaults";
import { StorageKeys } from "$lib/brand";

const SETTINGS_KEY = StorageKeys.connectionSettings;
const HISTORY_KEY = StorageKeys.connectionHistory;

export interface ConnectionEvent {
  disconnectedAt: number;
  reconnectedAt?: number;
}

export interface ConnectionSettings {
  autoReconnect: boolean;
  minDelayMs: number;
  maxDelayMs: number;
  backoffMultiplier: number;
  healthCheckIntervalMs: number;
  offlinePollMs: number;
  rememberHistory: boolean;
  maxHistoryEntries: number;
  selfHealIntervalMs: number;
  retryOnOnline: boolean;
}

export const defaultConnectionSettings = (): ConnectionSettings => ({
  autoReconnect: true,
  minDelayMs: 2000,
  maxDelayMs: 120_000,
  backoffMultiplier: 1.6,
  healthCheckIntervalMs: 60_000,
  offlinePollMs: 5000,
  rememberHistory: true,
  maxHistoryEntries: 32,
  selfHealIntervalMs: 300_000,
  retryOnOnline: true,
});

export function loadConnectionSettings(): ConnectionSettings {
  try {
    const raw = localStorage.getItem(SETTINGS_KEY);
    if (!raw) return applyConnectionDefaults(defaultConnectionSettings());
    const parsed = JSON.parse(raw) as Partial<ConnectionSettings>;
    return applyConnectionDefaults({
      ...defaultConnectionSettings(),
      ...parsed,
    });
  } catch {
    return applyConnectionDefaults(defaultConnectionSettings());
  }
}

export function saveConnectionSettings(settings: ConnectionSettings): void {
  localStorage.setItem(SETTINGS_KEY, JSON.stringify(settings));
}

export function mergeConnectionSettings(
  partial: Partial<ConnectionSettings> | null | undefined,
): ConnectionSettings {
  if (!partial || typeof partial !== "object") {
    return applyConnectionDefaults(defaultConnectionSettings());
  }
  return applyConnectionDefaults({
    ...defaultConnectionSettings(),
    ...partial,
  });
}

export function loadConnectionHistory(): ConnectionEvent[] {
  try {
    const raw = localStorage.getItem(HISTORY_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as ConnectionEvent[];
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(
      (entry) =>
        typeof entry.disconnectedAt === "number" &&
        (entry.reconnectedAt === undefined ||
          typeof entry.reconnectedAt === "number"),
    );
  } catch {
    return [];
  }
}

export function saveConnectionHistory(history: ConnectionEvent[]): void {
  localStorage.setItem(HISTORY_KEY, JSON.stringify(history));
}

export function appendConnectionEvent(
  history: ConnectionEvent[],
  event: ConnectionEvent,
  maxEntries: number,
): ConnectionEvent[] {
  const next = [...history, event];
  if (next.length <= maxEntries) return next;
  return next.slice(next.length - maxEntries);
}

export function closeLatestConnectionEvent(
  history: ConnectionEvent[],
  reconnectedAt: number,
): ConnectionEvent[] {
  if (history.length === 0) return history;
  const last = history[history.length - 1];
  if (last.reconnectedAt !== undefined) return history;
  const updated = { ...last, reconnectedAt };
  return [...history.slice(0, -1), updated];
}
