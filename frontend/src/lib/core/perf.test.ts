// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  initPerf,
  perfEntries,
  perfRecord,
  perfSummary,
  perfTime,
  perfVitals,
} from "./perf";

describe("perfRecord", () => {
  it("stores entries with a duration", () => {
    const before = perfEntries().length;
    perfRecord("test-op", 12.5, "detail");
    const entries = perfEntries();
    expect(entries.length).toBe(before + 1);
    expect(entries[entries.length - 1]).toMatchObject({
      name: "test-op",
      detail: "detail",
    });
  });

  it("keeps the buffer bounded", () => {
    for (let i = 0; i < 300; i += 1) {
      perfRecord("flood", i);
    }
    expect(perfEntries().length).toBeLessThanOrEqual(200);
  });
});

describe("perfTime", () => {
  it("records the wrapped function duration and returns its value", () => {
    const result = perfTime("measured", () => 42);
    expect(result).toBe(42);
    const last = perfEntries()[perfEntries().length - 1];
    expect(last.name).toBe("measured");
    expect(last.durationMs).toBeGreaterThanOrEqual(0);
  });

  it("records even when the wrapped function throws", () => {
    expect(() =>
      perfTime("throwing", () => {
        throw new Error("boom");
      }),
    ).toThrow("boom");
    const last = perfEntries()[perfEntries().length - 1];
    expect(last.name).toBe("throwing");
  });
});

describe("perfSummary", () => {
  it("reports slowest entries in descending order", () => {
    perfRecord("slow-a", 500);
    perfRecord("slow-b", 900);
    const summary = perfSummary();
    expect(summary.slowest[0].name).toBe("slow-b");
    expect(summary.slowest[1].name).toBe("slow-a");
    expect(summary.recent.length).toBeLessThanOrEqual(25);
  });
});

describe("initPerf", () => {
  it("exposes a console handle and starts cls at zero", () => {
    initPerf();
    expect(perfVitals().cls).toBe(0);
    expect(window.__melPerf).toBeDefined();
    expect(window.__melPerf?.vitals.cls).toBe(0);
    expect(typeof window.__melPerf?.summary).toBe("function");
  });

  it("is idempotent", () => {
    initPerf();
    initPerf();
    expect(window.__melPerf).toBeDefined();
  });
});
