// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type {
  ConnectionEvent,
  ConnectionSettings,
} from "./connection-settings";

export type ConnectionPhase =
  "online" | "offline" | "reconnecting" | "retrying" | "self-healing";

export function averageOfflineDurationMs(
  history: ConnectionEvent[],
): number | null {
  const completed = history.filter(
    (entry) =>
      entry.reconnectedAt !== undefined &&
      entry.reconnectedAt > entry.disconnectedAt,
  );
  if (completed.length === 0) return null;

  let total = 0;
  for (const entry of completed) {
    total += entry.reconnectedAt! - entry.disconnectedAt;
  }
  return total / completed.length;
}

export function recentOfflineDurationMs(
  history: ConnectionEvent[],
): number | null {
  for (let i = history.length - 1; i >= 0; i--) {
    const entry = history[i];
    if (
      entry.reconnectedAt !== undefined &&
      entry.reconnectedAt > entry.disconnectedAt
    ) {
      return entry.reconnectedAt - entry.disconnectedAt;
    }
  }
  return null;
}

export function computeReconnectDelayMs(
  settings: ConnectionSettings,
  attempt: number,
  history: ConnectionEvent[],
  browserOnline: boolean,
): number {
  if (!browserOnline) {
    return settings.offlinePollMs;
  }

  const base =
    settings.minDelayMs * settings.backoffMultiplier ** Math.max(0, attempt);
  let delay = Math.min(base, settings.maxDelayMs);

  if (settings.rememberHistory && history.length > 0) {
    const recent = recentOfflineDurationMs(history);
    const average = averageOfflineDurationMs(history);

    if (recent !== null && recent < 30_000) {
      delay = Math.min(delay, settings.minDelayMs);
    } else if (average !== null && average > 60_000) {
      const hint = Math.min(average * 0.25, settings.maxDelayMs);
      delay = Math.max(delay, hint);
    }
  }

  return Math.round(Math.max(settings.minDelayMs, delay));
}

export function phaseForAttempt(
  attempt: number,
  selfHealing: boolean,
): ConnectionPhase {
  if (selfHealing) return "self-healing";
  if (attempt <= 1) return "reconnecting";
  return "retrying";
}
