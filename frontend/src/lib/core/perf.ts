// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { logger } from "./logger";

declare global {
  interface Window {
    __melPerf?: {
      entries: readonly PerfEntry[];
      vitals: PerfVitals;
      mark: (name: string) => void;
      measure: (name: string, startMark: string, detail?: string) => void;
      summary: () => PerfSummary;
    };
  }
}

export interface PerfEntry {
  name: string;
  durationMs: number;
  at: number;
  detail?: string;
}

export interface PerfVitals {
  fcp?: number;
  lcp?: number;
  domContentLoaded?: number;
  load?: number;
}

export interface PerfSummary {
  vitals: PerfVitals;
  longtaskCount: number;
  longtaskTotalMs: number;
  slowest: PerfEntry[];
  recent: PerfEntry[];
}

const BUFFER_MAX = 200;
const SLOW_OP_WARN_MS = 250;
const LONGTASK_WARN_MS = 200;

const entries: PerfEntry[] = [];
const vitals: PerfVitals = {};
let longtaskCount = 0;
let longtaskTotalMs = 0;
let started = false;

function timingNow(): number {
  try {
    return performance.now();
  } catch {
    return Date.now();
  }
}

export function perfRecord(
  name: string,
  durationMs: number,
  detail?: string,
): void {
  entries.push({ name, durationMs, at: Date.now(), detail });
  if (entries.length > BUFFER_MAX) {
    entries.splice(0, entries.length - BUFFER_MAX);
  }
  if (durationMs >= SLOW_OP_WARN_MS) {
    logger.debug(
      `slow operation ${name} took ${Math.round(durationMs)}ms`,
      detail ? { detail } : undefined,
      "perf",
    );
  }
}

export function perfMark(name: string): void {
  try {
    performance.mark(`mel:${name}`);
  } catch {
    /* performance API unavailable */
  }
}

export function perfMeasure(
  name: string,
  startMark: string,
  detail?: string,
): void {
  try {
    const measure = performance.measure(`mel:${name}`, `mel:${startMark}`);
    perfRecord(name, measure.duration, detail);
  } catch {
    /* missing mark or unsupported API */
  }
}

export function perfTime<T>(name: string, fn: () => T, detail?: string): T {
  const start = timingNow();
  try {
    return fn();
  } finally {
    perfRecord(name, timingNow() - start, detail);
  }
}

export async function perfTimeAsync<T>(
  name: string,
  fn: () => Promise<T> | T,
  detail?: string,
): Promise<T> {
  const start = timingNow();
  try {
    return await fn();
  } finally {
    perfRecord(name, timingNow() - start, detail);
  }
}

export function perfSummary(): PerfSummary {
  const slowest = [...entries]
    .sort((a, b) => b.durationMs - a.durationMs)
    .slice(0, 10);
  return {
    vitals: { ...vitals },
    longtaskCount,
    longtaskTotalMs: Math.round(longtaskTotalMs),
    slowest,
    recent: entries.slice(-25),
  };
}

export function perfEntries(): readonly PerfEntry[] {
  return entries;
}

export function perfVitals(): PerfVitals {
  return { ...vitals };
}

function captureNavigationVitals(): void {
  try {
    const nav = performance.getEntriesByType("navigation")[0] as
      PerformanceNavigationTiming | undefined;
    if (nav) {
      vitals.domContentLoaded = Math.round(nav.domContentLoadedEventEnd);
      vitals.load = Math.round(nav.loadEventEnd);
    }
    for (const entry of performance.getEntriesByType("paint")) {
      if (entry.name === "first-contentful-paint") {
        vitals.fcp = Math.round(entry.startTime);
      }
    }
  } catch {
    /* timing entries unavailable */
  }
}

function observe(
  type: string,
  callback: (entry: PerformanceEntry) => void,
): void {
  try {
    new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) callback(entry);
    }).observe({ type, buffered: true });
  } catch {
    /* entry type unsupported on this engine */
  }
}

/** Start vitals capture, the longtask observer, and the console handle. */
export function initPerf(): void {
  if (started || typeof window === "undefined") return;
  started = true;
  if (typeof performance === "undefined") return;

  captureNavigationVitals();
  observe("largest-contentful-paint", (entry) => {
    vitals.lcp = Math.round(entry.startTime);
  });
  observe("longtask", (entry) => {
    longtaskCount += 1;
    longtaskTotalMs += entry.duration;
    if (entry.duration >= LONGTASK_WARN_MS) {
      logger.debug(
        `long task blocked the main thread ${Math.round(entry.duration)}ms`,
        undefined,
        "perf",
      );
    }
  });

  window.__melPerf = {
    entries,
    vitals,
    mark: perfMark,
    measure: perfMeasure,
    summary: perfSummary,
  };
}
