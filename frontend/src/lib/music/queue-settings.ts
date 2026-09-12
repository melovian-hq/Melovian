// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { StorageKeys } from "$lib/brand";

export interface QueueSettings {
  maxQueueSize: number;
}

const STORAGE_KEY = StorageKeys.queueSettings;

export const QUEUE_SIZE_OPTIONS = [
  { value: 0, label: "Unlimited" },
  { value: 100, label: "100 tracks" },
  { value: 250, label: "250 tracks" },
  { value: 500, label: "500 tracks" },
  { value: 1000, label: "1,000 tracks" },
  { value: 2000, label: "2,000 tracks" },
] as const;

export function defaultQueueSettings(): QueueSettings {
  return { maxQueueSize: 500 };
}

function clampInt(
  value: unknown,
  min: number,
  max: number,
  fallback: number,
): number {
  if (typeof value !== "number" || !Number.isFinite(value)) return fallback;
  return Math.max(min, Math.min(max, Math.round(value)));
}

export function mergeQueueSettings(
  partial: Partial<QueueSettings> | null | undefined,
): QueueSettings {
  const defaults = defaultQueueSettings();
  if (!partial || typeof partial !== "object") return defaults;

  return {
    maxQueueSize: clampInt(
      partial.maxQueueSize,
      0,
      5000,
      defaults.maxQueueSize,
    ),
  };
}

export function loadQueueSettings(): QueueSettings {
  if (typeof localStorage === "undefined") return defaultQueueSettings();
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return defaultQueueSettings();
    return mergeQueueSettings(JSON.parse(raw) as Partial<QueueSettings>);
  } catch {
    return defaultQueueSettings();
  }
}

export function saveQueueSettings(settings: QueueSettings): void {
  if (typeof localStorage === "undefined") return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
  } catch {
    /* storage full or unavailable */
  }
}

export function isQueueUnlimited(maxQueueSize: number): boolean {
  return maxQueueSize <= 0;
}

export function remainingQueueSlots(
  currentLength: number,
  maxQueueSize: number,
): number {
  if (isQueueUnlimited(maxQueueSize)) return Number.POSITIVE_INFINITY;
  return Math.max(0, maxQueueSize - currentLength);
}

export function truncateTrackList<T>(
  tracks: readonly T[],
  maxQueueSize: number,
): T[] {
  if (isQueueUnlimited(maxQueueSize) || tracks.length <= maxQueueSize) {
    return [...tracks];
  }
  return tracks.slice(0, maxQueueSize);
}

export function enforceQueueLimit<T>(
  queue: readonly T[],
  queueIndex: number,
  maxQueueSize: number,
): { queue: T[]; queueIndex: number } {
  if (isQueueUnlimited(maxQueueSize) || queue.length <= maxQueueSize) {
    return { queue: [...queue], queueIndex };
  }

  let next = [...queue];
  let idx = Math.max(0, Math.min(queueIndex, next.length - 1));

  while (next.length > maxQueueSize) {
    if (idx < next.length - 1) {
      next = next.slice(0, -1);
    } else if (idx > 0) {
      next = next.slice(1);
      idx -= 1;
    } else {
      next = next.slice(0, -1);
    }
  }

  return { queue: next, queueIndex: idx };
}
