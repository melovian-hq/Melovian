// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { defineConfig, devices } from "@playwright/test";

const listen = process.env.E2E_LISTEN || "127.0.0.1:17348";
const baseURL = process.env.E2E_BASE_URL || `http://${listen}`;
const reuseExisting = Boolean(process.env.E2E_BASE_URL) || !process.env.CI;

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 2 : undefined,
  reporter: process.env.CI
    ? [["github"], ["html", { open: "never" }]]
    : [["list"], ["html", { open: "never" }]],
  timeout: 60_000,
  expect: {
    timeout: 15_000,
  },
  use: {
    baseURL,
    ...devices["Desktop Chrome"],
    colorScheme: "dark",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "off",
  },
  webServer: process.env.E2E_BASE_URL
    ? undefined
    : {
        command: "node e2e/helpers/run-demo-server.mjs",
        url: `${baseURL}/health`,
        reuseExistingServer: reuseExisting,
        timeout: 120_000,
        stdout: "pipe",
        stderr: "pipe",
      },
  projects: [
    {
      name: "smoke",
      grep: /@smoke/,
    },
    {
      name: "chromium",
      grepInvert: /@smoke/,
    },
  ],
});
