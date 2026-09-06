// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/** @type {import('@lhci/cli').Config} */
const { urls } = require("./lighthouse-urls.cjs");

module.exports = {
  ci: {
    collect: {
      // One run per URL. Coverage is breadth (all demo-reachable pages), not median stability.
      numberOfRuns: 1,
      startServerCommand: "pnpm preview --host 127.0.0.1 --port 4173",
      startServerReadyPattern: "Local:",
      startServerReadyTimeout: 120000,
      url: urls,
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
            // CI runners are slower than a laptop. Floor is intentionally modest.
            "categories:performance": ["error", { minScore: 0.5 }],
            "categories:accessibility": ["error", { minScore: 0.9 }],
            "categories:best-practices": ["error", { minScore: 0.85 }],
            // SPA shell often lacks per-route meta. Warn until titles/descriptions land.
            "categories:seo": ["warn", { minScore: 0.8 }],
            "first-contentful-paint": ["warn", { maxNumericValue: 4000 }],
            interactive: ["warn", { maxNumericValue: 7000 }],
            "total-blocking-time": ["warn", { maxNumericValue: 600 }],
            "cumulative-layout-shift": ["error", { maxNumericValue: 0.25 }],
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
