// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";
import wails from "@wailsio/runtime/plugins/vite";
import path from "node:path";
import { execSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";

const pkg = JSON.parse(readFileSync(path.resolve("package.json"), "utf8")) as {
  version?: string;
};

const base = process.env.VITE_BASE || "/";

// Brand identity. Set VITE_APP_NAME / VITE_APP_SLUG in the environment to
// rebrand. The defaults here also feed the %VITE_APP_*% tokens in index.html.
process.env.VITE_APP_NAME ??= "Melovian";
process.env.VITE_APP_SLUG ??= "melovian";
process.env.VITE_APP_DESCRIPTION ??=
  "Music player for local libraries and Subsonic-compatible servers";

/** Match Go/server stamps: env, live git, optional .build-version, then package.json. */
function resolveAppVersion(): string {
  const fromEnv = process.env.VITE_APP_VERSION?.trim();
  if (fromEnv) return fromEnv;

  // Prefer live git over frontend/.build-version. A stale stamp file is what
  // caused client/server mismatch banners after rebuilding only the Go binary.
  try {
    const described = execSync("git describe --tags --always --dirty", {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "ignore"],
      cwd: path.resolve(".."),
    }).trim();
    if (described) return described;
  } catch {
    // Not a git checkout or git missing.
  }

  const stampPath = path.resolve(".build-version");
  if (existsSync(stampPath)) {
    const stamped = readFileSync(stampPath, "utf8").trim();
    if (stamped) return stamped;
  }

  return pkg.version || "0.1.0";
}

const appVersion = resolveAppVersion();

export default defineConfig({
  base,
  plugins: [svelte(), tailwindcss(), wails("./bindings")],
  define: {
    "import.meta.env.VITE_APP_VERSION": JSON.stringify(appVersion),
    "import.meta.env.VITE_STATIC_DEMO": JSON.stringify(
      process.env.VITE_STATIC_DEMO || "false",
    ),
  },
  resolve: {
    alias: {
      $lib: path.resolve("src/lib"),
      "@bindings": path.resolve("bindings"),
    },
    extensionAlias: {
      ".js": [".ts", ".js"],
    },
  },
  build: {
    // The Wails WebView and server-mode browsers are modern.
    // Skipping downlevel transpilation shrinks output and speeds builds.
    target: "es2022",
    // No source maps in production builds.
    sourcemap: false,
    rollupOptions: {
      output: {
        advancedChunks: {
          groups: [
            {
              name: "vendor-svelte",
              test: /node_modules[\\/]svelte[\\/]/,
            },
            {
              name: "vendor-sentry",
              test: /node_modules[\\/]@sentry[\\/]/,
            },
            {
              name: "vendor-wails",
              test: /node_modules[\\/]@wailsio[\\/]/,
            },
            {
              name: "vendor-icons",
              test: /node_modules[\\/]@iconify[\\/]/,
            },
            {
              name: "bindings",
              test: /[\\/]bindings[\\/]/,
            },
          ],
        },
      },
    },
  },
  server: {
    host: "127.0.0.1",
    warmup: {
      clientFiles: ["./src/main.ts", "./src/App.svelte", "./src/routes.ts"],
    },
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
    proxy:
      process.env.VITE_STATIC_DEMO === "true"
        ? undefined
        : {
            "/api": {
              target: process.env.MELOVIAN_API_URL || "http://127.0.0.1:17337",
              changeOrigin: true,
              ws: true,
            },
            "/rest": {
              target: process.env.MELOVIAN_API_URL || "http://127.0.0.1:17337",
              changeOrigin: true,
            },
            "/health": {
              target: process.env.MELOVIAN_API_URL || "http://127.0.0.1:17337",
              changeOrigin: true,
            },
            "/metrics": {
              target: process.env.MELOVIAN_API_URL || "http://127.0.0.1:17337",
              changeOrigin: true,
            },
          },
  },
});
