// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { Page } from "@playwright/test";
import { test, expect, waitForAppShell } from "./fixtures";

/**
 * Frontend performance budgets measured against the demo server.
 *
 * The @smoke describe covers the routes that have regressed layout stability
 * in the past and runs in CI. The untagged sweep runs in the full e2e suite
 * and adds a heap/DOM growth check across repeated SPA navigations.
 *
 * Timing budgets are deliberately generous for shared CI runners. They catch
 * order-of-magnitude regressions, not small drift. Tight numbers belong to
 * the Lighthouse job (lighthouserc.cjs) and scripts/perf-audit.mjs.
 */

interface ProbeVitals {
  fcp: number;
  lcp: number;
  cls: number;
  worstShift: number;
  longtaskCount: number;
  worstLongtask: number;
}

const BUDGET = {
  cls: 0.1,
  fcpMs: 4000,
  lcpMs: 6000,
  worstLongtaskMs: 1000,
  longtaskCount: 25,
  domNodes: 2500,
  // Demo server routes issue ~230-350 requests, mostly per-card cover art.
  requests: 420,
  heapCeilingMB: 64,
};

const SMOKE_ROUTES = ["/music", "/music/playlists", "/music/artist/ar-006"];

const SWEEP_ROUTES = [
  { path: "/music", bare: false },
  { path: "/music/search", bare: false },
  { path: "/music/albums", bare: false },
  { path: "/music/album/al-006-01", bare: false },
  { path: "/music/artist/ar-006", bare: false },
  { path: "/music/playlists", bare: false },
  { path: "/music/playlist/pl-hits", bare: false },
  { path: "/music/favorites", bare: false },
  { path: "/music/history", bare: false },
  { path: "/music/now-playing", bare: false },
  { path: "/setup", bare: true },
];

const PROBE_INIT = `
window.__perfProbe = {
  fcp: 0, lcp: 0, cls: 0, worstShift: 0, longtaskCount: 0, worstLongtask: 0,
};
try {
  new PerformanceObserver((l) => {
    for (const e of l.getEntries()) {
      if (e.name === "first-contentful-paint") window.__perfProbe.fcp = e.startTime;
    }
  }).observe({ type: "paint", buffered: true });
  new PerformanceObserver((l) => {
    for (const e of l.getEntries()) window.__perfProbe.lcp = e.startTime;
  }).observe({ type: "largest-contentful-paint", buffered: true });
  new PerformanceObserver((l) => {
    for (const e of l.getEntries()) {
      if (e.hadRecentInput) continue;
      window.__perfProbe.cls += e.value;
      window.__perfProbe.worstShift = Math.max(window.__perfProbe.worstShift, e.value);
    }
  }).observe({ type: "layout-shift", buffered: true });
  new PerformanceObserver((l) => {
    for (const e of l.getEntries()) {
      window.__perfProbe.longtaskCount += 1;
      window.__perfProbe.worstLongtask = Math.max(window.__perfProbe.worstLongtask, e.duration);
    }
  }).observe({ type: "longtask", buffered: true });
} catch { /* observer types unsupported */ }
`;

interface RouteMeasurement {
  vitals: ProbeVitals;
  domNodes: number;
  requests: number;
  failed: string[];
}

async function measureRoute(
  page: Page,
  path: string,
  bare: boolean,
): Promise<RouteMeasurement> {
  await page.addInitScript(PROBE_INIT);

  const failed: string[] = [];
  page.on("response", (res) => {
    if (res.request().isNavigationRequest()) return;
    if (res.status() < 400) return;
    try {
      const url = res.url();
      if (new URL(url).origin !== new URL(page.url()).origin) return;
      // cache-ops probes playlist details and tolerates 404. Demo catalog
      // entries like pl-hits are server playlists that miss the local endpoint.
      if (res.status() === 404 && url.includes("/api/music/playlists/")) return;
      failed.push(`${res.status()} ${url}`);
    } catch {
      /* unparsable url */
    }
  });
  const requestCount = { n: 0 };
  page.on("request", () => {
    requestCount.n += 1;
  });

  await page.goto(path);
  if (bare) {
    await page.waitForLoadState("load");
  } else {
    await waitForAppShell(page);
  }
  await page.waitForTimeout(2500);

  const vitals = await page.evaluate(
    () => (window as never as { __perfProbe: ProbeVitals }).__perfProbe,
  );
  const domNodes = await page.evaluate(
    () => document.getElementsByTagName("*").length,
  );
  return { vitals, domNodes, requests: requestCount.n, failed };
}

function budgetViolations(m: RouteMeasurement): string[] {
  const v = m.vitals;
  const out: string[] = [];
  if (v.cls > BUDGET.cls) out.push(`cls ${v.cls.toFixed(3)} > ${BUDGET.cls}`);
  if (v.fcp > BUDGET.fcpMs)
    out.push(`fcp ${Math.round(v.fcp)}ms > ${BUDGET.fcpMs}ms`);
  if (v.lcp > BUDGET.lcpMs)
    out.push(`lcp ${Math.round(v.lcp)}ms > ${BUDGET.lcpMs}ms`);
  if (v.worstLongtask > BUDGET.worstLongtaskMs)
    out.push(`longtask ${Math.round(v.worstLongtask)}ms`);
  if (v.longtaskCount > BUDGET.longtaskCount)
    out.push(`${v.longtaskCount} long tasks`);
  if (m.domNodes > BUDGET.domNodes)
    out.push(`${m.domNodes} DOM nodes > ${BUDGET.domNodes}`);
  if (m.requests > BUDGET.requests)
    out.push(`${m.requests} requests > ${BUDGET.requests}`);
  for (const f of m.failed.slice(0, 5)) out.push(`failed request: ${f}`);
  return out;
}

test.describe("performance budgets @smoke", () => {
  for (const path of SMOKE_ROUTES) {
    test(`${path} stays inside budgets`, async ({ page }) => {
      const m = await measureRoute(page, path, false);
      expect(budgetViolations(m)).toEqual([]);
    });
  }
});

async function dismissTelemetryPrompt(page: Page): Promise<void> {
  const decline = page.getByRole("button", { name: "No thanks" });
  if (await decline.isVisible().catch(() => false)) {
    await decline.click();
    await page.waitForTimeout(300);
  }
}

test.describe("performance budgets", () => {
  for (const route of SWEEP_ROUTES) {
    test(`${route.path} stays inside budgets`, async ({ page }) => {
      const m = await measureRoute(page, route.path, route.bare);
      expect(budgetViolations(m)).toEqual([]);
    });
  }

  test("repeated navigation does not grow heap or DOM", async ({ page }) => {
    await page.goto("/music");
    await waitForAppShell(page);
    // The consent prompt opens after its settings fetch resolves and its
    // backdrop intercepts sidebar clicks. Decline it before navigating.
    await page.waitForTimeout(1500);
    await dismissTelemetryPrompt(page);

    const navTargets = [
      "/music/albums",
      "/music/artists",
      "/music/playlists",
      "/music",
    ];
    for (let cycle = 0; cycle < 3; cycle += 1) {
      for (const href of navTargets) {
        const link = page.locator(`a[href="${href}"]`).first();
        if (!(await link.isVisible().catch(() => false))) continue;
        await link.click();
        await page.waitForTimeout(600);
      }
    }

    const domNodes = await page.evaluate(
      () => document.getElementsByTagName("*").length,
    );
    expect(domNodes).toBeLessThanOrEqual(BUDGET.domNodes);

    const heapMB = await page.evaluate(() => {
      const mem = (
        performance as Performance & {
          memory?: { usedJSHeapSize: number };
        }
      ).memory;
      return mem ? Math.round(mem.usedJSHeapSize / 1024 / 1024) : -1;
    });
    if (heapMB >= 0) {
      // Generous ceiling: the demo library is small and the app should idle
      // well under this after a dozen client-side navigations.
      expect(heapMB).toBeLessThanOrEqual(BUDGET.heapCeilingMB);
    }
  });
});
