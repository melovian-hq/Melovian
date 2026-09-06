// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { defineConfig } from "vitest/config";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import path from "node:path";

export default defineConfig({
  plugins: [svelte()],
  resolve: {
    conditions: ["browser"],
    alias: {
      $lib: path.resolve("src/lib"),
      "@bindings": path.resolve("bindings"),
    },
    extensionAlias: {
      ".js": [".ts", ".js"],
    },
  },
  test: {
    environment: "jsdom",
    include: [
      "src/**/*.test.ts",
      "src/**/*.property.test.ts",
      "src/**/*.contract.test.ts",
      "src/**/*.leak.test.ts",
      "src/**/*.perf.test.ts",
      "src/**/*.memory.test.ts",
      "src/**/*.regression.test.ts",
      "src/**/*.crash.test.ts",
      "src/**/*.exploratory.test.ts",
      "src/**/*.oracle.test.ts",
      "src/**/*.acceptance.test.ts",
      "bindings/**/*.contract.test.ts",
    ],
    setupFiles: ["./vitest.setup.ts"],
    server: {
      deps: {
        inline: ["svelte"],
      },
    },
    coverage: {
      provider: "v8",
      reporter: ["text", "html", "lcov"],
      reportsDirectory: "./coverage",
      include: [
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
      thresholds: {
        statements: 80,
        branches: 70,
        functions: 90,
        lines: 80,
      },
    },
  },
});
