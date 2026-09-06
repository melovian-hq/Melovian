// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { ConnectionSettings } from "./connection-settings";

export interface ConnectionDefaults {
  minDelayMs?: number;
  maxDelayMs?: number;
  backoffMultiplier?: number;
  healthCheckIntervalMs?: number;
  offlinePollMs?: number;
  selfHealIntervalMs?: number;
  maxHistoryEntries?: number;
}

let serverDefaults: ConnectionDefaults = {};

export function setConnectionDefaults(defaults: ConnectionDefaults): void {
  serverDefaults = defaults ?? {};
}

export function applyConnectionDefaults(
  settings: ConnectionSettings,
): ConnectionSettings {
  const next = { ...settings };
  if (typeof serverDefaults.minDelayMs === "number") {
    next.minDelayMs = serverDefaults.minDelayMs;
  }
  if (typeof serverDefaults.maxDelayMs === "number") {
    next.maxDelayMs = serverDefaults.maxDelayMs;
  }
  if (typeof serverDefaults.backoffMultiplier === "number") {
    next.backoffMultiplier = serverDefaults.backoffMultiplier;
  }
  if (typeof serverDefaults.healthCheckIntervalMs === "number") {
    next.healthCheckIntervalMs = serverDefaults.healthCheckIntervalMs;
  }
  if (typeof serverDefaults.offlinePollMs === "number") {
    next.offlinePollMs = serverDefaults.offlinePollMs;
  }
  if (typeof serverDefaults.selfHealIntervalMs === "number") {
    next.selfHealIntervalMs = serverDefaults.selfHealIntervalMs;
  }
  if (typeof serverDefaults.maxHistoryEntries === "number") {
    next.maxHistoryEntries = serverDefaults.maxHistoryEntries;
  }
  return next;
}
