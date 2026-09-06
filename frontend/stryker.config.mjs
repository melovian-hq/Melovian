// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/** @type {import('@stryker-mutator/api/core').PartialStrykerOptions} */
const config = {
  $schema: "./node_modules/@stryker-mutator/core/schema/stryker-schema.json",
  packageManager: "pnpm",
  reporters: ["clear-text", "progress", "html"],
  testRunner: "vitest",
  plugins: ["@stryker-mutator/vitest-runner"],
  coverageAnalysis: "perTest",
  mutate: [
    "src/lib/music/search.ts",
    "src/lib/music/playback-queue.ts",
    "src/lib/music/album-dedup.ts",
    "src/lib/music/smart-playlist/evaluate.ts",
    "src/lib/music/smart-playlist/validate.ts",
    "src/lib/core/bounded-cache.ts",
    "src/lib/core/collection.ts",
    "src/lib/utils/local-search.ts",
    "src/lib/router/match.ts",
  ],
  vitest: {
    configFile: "vitest.config.ts",
  },
  thresholds: {
    high: 80,
    low: 60,
    break: null,
  },
  timeoutMS: 60_000,
  concurrency: 2,
  htmlReporter: {
    fileName: "reports/mutation/mutation.html",
  },
};

export default config;
