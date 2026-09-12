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
    // Dynamic imports in tests stall past 5s under parallel
    // transform load.
    testTimeout: 15_000,
    include: ["src/**/*.exploratory.test.ts"],
    setupFiles: ["./vitest.setup.ts"],
    server: {
      deps: {
        inline: ["svelte"],
      },
    },
  },
});
