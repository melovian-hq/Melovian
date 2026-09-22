// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/**
 * Per-route performance audit against the built frontend.
 *
 * Measures FCP, LCP, CLS, long tasks, request counts, DOM size, and heap for
 * every route in lighthouse-urls.cjs, then compares each against the budgets
 * below. Exits 1 when any route violates a budget.
 *
 * Usage:
 *   node scripts/perf-audit.mjs [--scope=core|full] [--base=URL]
 *                               [--json=path] [--settle=ms]
 *
 * Without --base it expects `dist/` to be a static demo build and starts
 * `vite preview` on 127.0.0.1:4173, or reuses a server already listening.
 */

import { spawn } from "node:child_process";
import { existsSync } from "node:fs";
import { writeFile } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { setTimeout as sleep } from "node:timers/promises";
import { chromium } from "@playwright/test";

const __dirname = dirname(fileURLToPath(import.meta.url));
const FRONTEND_ROOT = resolve(__dirname, "..");
const {
  PATHS,
  CORE_PATHS,
  BASE: DEFAULT_BASE,
} = await import(join(FRONTEND_ROOT, "lighthouse-urls.cjs"));

const args = Object.fromEntries(
  process.argv.slice(2).map((a) => {
    const [k, v] = a.replace(/^--/, "").split("=");
    return [k, v ?? true];
  }),
);
const scope = args.scope === "full" ? PATHS : CORE_PATHS;
const SETTLE_MS = Number(args.settle || 2500);
const BASE = args.base || DEFAULT_BASE;
const JSON_OUT = args.json || null;

// Hard gates. Derived from measured medians with roughly 3x headroom so the
// audit fails on regressions, not on machine noise. The Lighthouse job owns
// the tighter CWV thresholds under throttling.
const BUDGET = {
  fcpMs: 2000,
  lcpMs: 3000,
  cls: 0.1,
  worstLongtaskMs: 500,
  domNodes: 2500,
  requests: 180,
  heapMB: 64,
};

const PROBE_INIT = `
window.__probe = {
  fcp: 0, lcp: 0, lcpEl: "", cls: 0, worstShift: 0, longtasks: [],
};
try {
  new PerformanceObserver((l) => {
    for (const e of l.getEntries()) {
      if (e.name === "first-contentful-paint") window.__probe.fcp = e.startTime;
    }
  }).observe({ type: "paint", buffered: true });
  new PerformanceObserver((l) => {
    for (const e of l.getEntries()) {
      window.__probe.lcp = e.startTime;
      const el = e.element;
      window.__probe.lcpEl = el
        ? el.tagName.toLowerCase() + "." + String(el.className).slice(0, 40)
        : "";
    }
  }).observe({ type: "largest-contentful-paint", buffered: true });
  new PerformanceObserver((l) => {
    for (const e of l.getEntries()) {
      if (e.hadRecentInput) continue;
      window.__probe.cls += e.value;
      window.__probe.worstShift = Math.max(window.__probe.worstShift, e.value);
    }
  }).observe({ type: "layout-shift", buffered: true });
  new PerformanceObserver((l) => {
    for (const e of l.getEntries()) window.__probe.longtasks.push(Math.round(e.duration));
  }).observe({ type: "longtask", buffered: true });
} catch { /* observer types unsupported */ }
`;

async function isUp(url) {
  try {
    return (await fetch(`${url}/index.html`, { method: "HEAD" })).ok;
  } catch {
    return false;
  }
}

async function ensureServer() {
  if (await isUp(BASE)) return null;
  if (!existsSync(join(FRONTEND_ROOT, "dist", "index.html"))) {
    console.error(
      "dist/ is missing. Build the demo bundle first: " +
        "VITE_STATIC_DEMO=true pnpm build",
    );
    process.exit(2);
  }
  const port = new URL(BASE).port || "4173";
  const proc = spawn(
    "pnpm",
    ["exec", "vite", "preview", "--host", "127.0.0.1", "--port", port],
    { cwd: FRONTEND_ROOT, stdio: "ignore" },
  );
  for (let i = 0; i < 60; i += 1) {
    if (await isUp(BASE)) return proc;
    await sleep(500);
  }
  proc.kill();
  throw new Error("vite preview did not come up");
}

async function measureRoute(browser, path) {
  const context = await browser.newContext();
  const page = await context.newPage();
  await page.addInitScript(PROBE_INIT);

  const failed = [];
  page.on("response", (res) => {
    if (res.request().isNavigationRequest() || res.status() < 400) return;
    try {
      if (new URL(res.url()).origin === new URL(BASE).origin) {
        failed.push(`${res.status()} ${res.url()}`);
      }
    } catch {
      /* unparsable url */
    }
  });

  await page.goto(`${BASE}${path}`, { waitUntil: "load" });
  await page.waitForTimeout(SETTLE_MS);

  const vitals = await page.evaluate(() => window.__probe);
  const res = await page.evaluate(() =>
    performance.getEntriesByType("resource").map((e) => ({
      n: e.name,
      t: e.initiatorType,
      s: Math.round(e.startTime),
      e: Math.round(e.responseEnd),
      sz: e.transferSize || 0,
    })),
  );
  const domNodes = await page.evaluate(
    () => document.getElementsByTagName("*").length,
  );
  const heap = await page.evaluate(() =>
    performance.memory ? performance.memory.usedJSHeapSize : -1,
  );
  await context.close();

  const js = res.filter((r) => r.n.endsWith(".js"));
  const ext = res.filter((r) => /^https?:/.test(r.n) && !r.n.startsWith(BASE));
  return {
    path,
    vitals: {
      fcp: Math.round(vitals.fcp),
      lcp: Math.round(vitals.lcp),
      lcpEl: vitals.lcpEl,
      cls: +vitals.cls.toFixed(4),
      worstShift: +vitals.worstShift.toFixed(4),
      longtasks: vitals.longtasks,
    },
    requests: res.length,
    jsBytes: js.reduce((a, r) => a + r.sz, 0),
    external: ext.length,
    failed,
    domNodes,
    heapMB: heap > 0 ? Math.round(heap / 1024 / 1024) : -1,
    res,
  };
}

function violations(m) {
  const v = m.vitals;
  const out = [];
  if (v.cls > BUDGET.cls) out.push(`cls ${v.cls} > ${BUDGET.cls}`);
  if (v.fcp > BUDGET.fcpMs) out.push(`fcp ${v.fcp}ms > ${BUDGET.fcpMs}ms`);
  if (v.lcp > BUDGET.lcpMs) out.push(`lcp ${v.lcp}ms > ${BUDGET.lcpMs}ms`);
  for (const t of v.longtasks) {
    if (t > BUDGET.worstLongtaskMs) {
      out.push(`longtask ${t}ms`);
      break;
    }
  }
  if (m.domNodes > BUDGET.domNodes)
    out.push(`${m.domNodes} DOM nodes > ${BUDGET.domNodes}`);
  if (m.requests > BUDGET.requests)
    out.push(`${m.requests} requests > ${BUDGET.requests}`);
  if (m.heapMB > BUDGET.heapMB)
    out.push(`heap ${m.heapMB}MB > ${BUDGET.heapMB}MB`);
  for (const f of m.failed.slice(0, 3)) out.push(`failed request: ${f}`);
  return out;
}

const server = await ensureServer();
const browser = await chromium.launch();
const results = [];
let failedRoutes = 0;

try {
  for (const path of scope) {
    try {
      const m = await measureRoute(browser, path);
      const v = violations(m);
      results.push(m);
      const flag = v.length ? "FAIL" : "ok";
      if (v.length) failedRoutes += 1;
      console.log(
        `${flag.padEnd(4)} ${m.path.padEnd(38)} ` +
          `fcp ${String(m.vitals.fcp).padStart(5)} ` +
          `lcp ${String(m.vitals.lcp).padStart(5)} ` +
          `cls ${m.vitals.cls.toFixed(4)} ` +
          `reqs ${String(m.requests).padStart(3)} ` +
          `ext ${String(m.external).padStart(3)} ` +
          `jsKB ${String(Math.round(m.jsBytes / 1024)).padStart(4)} ` +
          `heap ${String(m.heapMB).padStart(3)}MB ` +
          `dom ${String(m.domNodes).padStart(4)}`,
      );
      for (const line of v) console.log(`     ! ${line}`);
    } catch (err) {
      failedRoutes += 1;
      console.log(`FAIL ${path.padEnd(38)} ${err.message.split("\n")[0]}`);
    }
  }
} finally {
  await browser.close();
  server?.kill();
}

if (JSON_OUT) {
  await writeFile(JSON_OUT, JSON.stringify(results, null, 2));
  console.log(`wrote ${JSON_OUT}`);
}
console.log(
  failedRoutes
    ? `\n${failedRoutes} route(s) over budget`
    : `\nall ${results.length} routes within budget`,
);
process.exit(failedRoutes ? 1 : 0);
