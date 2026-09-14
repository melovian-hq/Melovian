// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { defineConfig, type Plugin } from "vite";
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
// Absolute origin for og:image tags. Set to the deploy origin on Pages.
process.env.VITE_APP_ORIGIN ??= "";

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

/**
 * Emits the service worker and web app manifest at build time. The service
 * worker gets the hashed precache list injected so each release is a distinct
 * cache generation. No runtime plugin dependency, the SW is a static file.
 */
function pwaPlugin(): Plugin {
  return {
    name: "melovian-pwa",
    apply: "build",
    generateBundle(_options, bundle) {
      const precache: string[] = [];
      for (const [fileName, item] of Object.entries(bundle)) {
        if (fileName === "index.html") {
          precache.push(`${base}index.html`);
          continue;
        }
        if (item.type !== "chunk" && item.type !== "asset") continue;
        // Precache entry chunks and the global stylesheet only. Per-route
        // chunks and styles stay lazy. Everything else under assets/ is
        // cached by the service worker on first fetch.
        const isEntry = item.type === "chunk" && item.isEntry;
        const isGlobalCss = /(^|\/)index-[^/]*\.css$/.test(fileName);
        if (isEntry || isGlobalCss) {
          precache.push(`${base}${fileName}`);
        }
      }

      const swTemplate = readFileSync(path.resolve("pwa/sw.js"), "utf8");
      const swSource = swTemplate
        .replaceAll("__VERSION__", appVersion)
        .replaceAll("__PRECACHE__", JSON.stringify(precache))
        .replaceAll("__BASE_URL__", base);
      this.emitFile({
        type: "asset",
        fileName: "sw.js",
        source: swSource,
      });

      const name = process.env.VITE_APP_NAME || "Melovian";
      const manifest = {
        name,
        short_name: name,
        description:
          process.env.VITE_APP_DESCRIPTION ||
          "Music player for local libraries and Subsonic-compatible servers",
        id: base,
        start_url: base,
        scope: base,
        display: "standalone",
        orientation: "any",
        background_color: "#0a0a0f",
        theme_color: "#0a0a0f",
        categories: ["music", "entertainment"],
        icons: [
          {
            src: `${base}favicon-192.png`,
            sizes: "192x192",
            type: "image/png",
          },
          {
            src: `${base}icon-512.png`,
            sizes: "512x512",
            type: "image/png",
            purpose: "any",
          },
          {
            src: `${base}icon-512.png`,
            sizes: "512x512",
            type: "image/png",
            purpose: "maskable",
          },
          {
            src: `${base}favicon.svg`,
            sizes: "any",
            type: "image/svg+xml",
          },
        ],
      };
      this.emitFile({
        type: "asset",
        fileName: "manifest.webmanifest",
        source: JSON.stringify(manifest, null, 2),
      });
    },
  };
}

export default defineConfig({
  base,
  plugins: [svelte(), tailwindcss(), wails("./bindings"), pwaPlugin()],
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
