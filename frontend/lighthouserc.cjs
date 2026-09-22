// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/** @type {import('@lhci/cli').Config} */
const { urls, coreUrls } = require("./lighthouse-urls.cjs");

// Push/PR runs audit the core layouts to keep CI fast. workflow_dispatch sets
// LHCI_SCOPE=full to audit every route. LHCI_RUNS overrides the run count.
const scope = process.env.LHCI_SCOPE === "full" ? "full" : "core";
const numberOfRuns = Number(
  process.env.LHCI_RUNS || (scope === "full" ? 3 : 2),
);

module.exports = {
  ci: {
    collect: {
      // Two runs per URL on the core scope. Single-run numeric metrics
      // (notably CLS) are flaky on shared CI runners; the assert matrix
      // aggregates by median.
      numberOfRuns,
      url: scope === "full" ? urls : coreUrls,
      startServerCommand: "pnpm preview --host 127.0.0.1 --port 4173",
      startServerReadyPattern: "Local:",
      startServerReadyTimeout: 120000,
      settings: {
        ...(process.env.CHROME_PATH
          ? { chromePath: process.env.CHROME_PATH }
          : {}),
        chromeFlags: "--no-sandbox --disable-dev-shm-usage --headless=new",
        preset: "desktop",
        // Give the Svelte SPA time to mount and fetch the demo catalog.
        maxWaitForLoad: 60000,
        pauseAfterLoadMs: 2000,
      },
    },
    assert: {
      assertMatrix: [
        {
          matchingUrlPattern: ".*",
          assertions: {
            // Hard gates. CLS uses the CWV good boundary since every audited
            // route now measures under 0.04. Timing floors keep roughly 3x
            // headroom over measured medians on a laptop (FCP ~880ms,
            // LCP ~1350ms, TBT ~16ms) so shared CI runners have room.
            "categories:performance": ["error", { minScore: 0.5 }],
            "categories:accessibility": ["error", { minScore: 0.9 }],
            "categories:best-practices": ["error", { minScore: 0.85 }],
            "first-contentful-paint": ["error", { maxNumericValue: 4000 }],
            "largest-contentful-paint": ["error", { maxNumericValue: 4000 }],
            interactive: ["error", { maxNumericValue: 10000 }],
            "total-blocking-time": ["error", { maxNumericValue: 1000 }],
            "cumulative-layout-shift": ["error", { maxNumericValue: 0.1 }],
          },
        },
        {
          matchingUrlPattern: ".*",
          assertions: {
            // Warnings sit closer to measured medians so drift shows in the
            // report before it becomes a failure. Script bytes guard against
            // eager-bundle creep (~297KB transferred today).
            "categories:performance": ["warn", { minScore: 0.85 }],
            // SPA shell often lacks per-route meta. Warn until titles/descriptions land.
            "categories:seo": ["warn", { minScore: 0.8 }],
            "first-contentful-paint": ["warn", { maxNumericValue: 2000 }],
            "largest-contentful-paint": ["warn", { maxNumericValue: 2500 }],
            interactive: ["warn", { maxNumericValue: 7000 }],
            "total-blocking-time": ["warn", { maxNumericValue: 300 }],
            "cumulative-layout-shift": ["warn", { maxNumericValue: 0.05 }],
            "resource-summary:script": [
              "warn",
              { maxNumericValue: 450 * 1024 },
            ],
            "errors-in-console": ["warn", { minScore: 1 }],
          },
        },
      ],
    },
    upload: {
      target: "filesystem",
      outputDir: ".lighthouseci",
    },
  },
};
