// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/**
 * Capture a small set of showcase screenshots against demo mode.
 *
 * Usage (from repo root, after `task build:server`):
 *   node scripts/showcase/capture.mjs
 */

import { spawn } from "node:child_process";
import { createWriteStream } from "node:fs";
import { mkdir, readdir, rm, unlink, writeFile } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { setTimeout as sleep } from "node:timers/promises";
import { chromium } from "playwright";

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, "../..");
const OUT = resolve(process.env.SHOWCASE_OUT || join(ROOT, "showcase"));
const BIN = process.env.SHOWCASE_BIN || join(ROOT, "bin", "melovian-server");
const LISTEN = process.env.SHOWCASE_LISTEN || "127.0.0.1:17347";
const BASE = process.env.SHOWCASE_BASE_URL || `http://${LISTEN}`;

/** Curated shots only. Keep this list short. */
const SHOTS = [
  {
    id: "desktop-dark-home",
    device: "desktop",
    theme: "dark",
    path: "/?theme=dark",
    viewport: { width: 1440, height: 900 },
    withPlayer: true,
  },
  {
    id: "desktop-light-home",
    device: "desktop",
    theme: "light",
    path: "/?theme=light",
    viewport: { width: 1440, height: 900 },
    withPlayer: true,
  },
  {
    id: "desktop-dark-album",
    device: "desktop",
    theme: "dark",
    path: "ALBUM",
    viewport: { width: 1440, height: 900 },
  },
  {
    id: "desktop-dark-playlists",
    device: "desktop",
    theme: "dark",
    path: "/music/playlists?theme=dark",
    viewport: { width: 1440, height: 900 },
  },
  {
    id: "mobile-dark-home",
    device: "mobile",
    theme: "dark",
    path: "/?theme=dark",
    viewport: { width: 390, height: 844 },
    withPlayer: true,
  },
  {
    id: "mobile-light-album",
    device: "mobile",
    theme: "light",
    path: "ALBUM",
    viewport: { width: 390, height: 844 },
  },
];

async function waitForHealth(url, attempts = 60) {
  for (let i = 0; i < attempts; i++) {
    try {
      const res = await fetch(`${url}/health`);
      if (res.ok) return;
    } catch {
      // retry
    }
    await sleep(500);
  }
  throw new Error(`server did not become healthy at ${url}`);
}

async function waitForWarmCovers(base, ids, attempts = 60) {
  for (let i = 0; i < attempts; i++) {
    let ready = 0;
    for (const id of ids) {
      try {
        const res = await fetch(
          `${base}/api/subsonic/rest/getCoverArt.view?id=${encodeURIComponent(id)}&size=300`,
        );
        const ct = res.headers.get("content-type") || "";
        if (res.ok && ct.includes("image/") && !ct.includes("svg")) {
          ready += 1;
        }
      } catch {
        // retry
      }
    }
    if (ready >= ids.length) return;
    await sleep(500);
  }
  console.warn("cover warm wait timed out; continuing with available art");
}

async function startServer() {
  if (process.env.SHOWCASE_BASE_URL) {
    return null;
  }
  const dataDir = join(OUT, ".demo-data");
  await rm(dataDir, { recursive: true, force: true });
  await mkdir(dataDir, { recursive: true });

  const logPath = join(OUT, "server.log");
  const log = createWriteStream(logPath, { flags: "w" });
  const child = spawn(
    BIN,
    [
      "--env-file=",
      "--listen",
      LISTEN,
      "--data",
      dataDir,
      "--demo",
      "--navidrome-server",
      "fake://melovian-demo",
      "--navidrome-user",
      "demo",
      "--navidrome-password",
      "demo",
    ],
    {
      cwd: ROOT,
      env: {
        ...process.env,
        MELOVIAN_DEMO_MODE: "true",
        MELOVIAN_LISTEN: LISTEN,
        MELOVIAN_DATA: dataDir,
        NAVIDROME_SERVER: "fake://melovian-demo",
        NAVIDROME_USER: "demo",
        NAVIDROME_PASSWORD: "demo",
      },
      stdio: ["ignore", "pipe", "pipe"],
    },
  );
  child.stdout.pipe(log);
  child.stderr.pipe(log);
  child.on("exit", (code) => {
    if (code && code !== 0) {
      console.error(`demo server exited with code ${code}; see ${logPath}`);
    }
  });
  await waitForHealth(BASE);
  // Wait until iTunes-backed covers are JPEG, not SVG fallbacks.
  await waitForWarmCovers(BASE, [
    "al-006-01",
    "al-011-01",
    "al-013-01",
    "al-014-01",
    "al-002-01",
    "ar-006",
  ]);
  return child;
}

async function settle(page) {
  await page.waitForLoadState("networkidle", { timeout: 20000 }).catch(() => {});
  // Wait for real cover art (iTunes-backed) instead of tiny pixel fallbacks.
  await page
    .waitForFunction(
      () => {
        const imgs = Array.from(document.querySelectorAll("img.cover-art"));
        const ready = imgs.filter(
          (img) =>
            img.complete &&
            img.naturalWidth >= 64 &&
            !img.classList.contains("cover-art--pixelated"),
        );
        return ready.length >= Math.min(3, imgs.length);
      },
      { timeout: 45000 },
    )
    .catch(() => {});
  await sleep(1200);
}

async function clearOldShots() {
  await mkdir(OUT, { recursive: true });
  const entries = await readdir(OUT);
  for (const name of entries) {
    if (name.endsWith(".png") || name === "index.json" || name === "README.md") {
      await unlink(join(OUT, name)).catch(() => {});
    }
  }
}

async function resolveAlbumPath(page, theme) {
  await page.goto(`${BASE}/music/albums?theme=${theme}`, {
    waitUntil: "domcontentloaded",
  });
  await settle(page);
  const href = await page.evaluate(() => {
    const el = document.querySelector('a[href^="/music/album/"]');
    return el?.getAttribute("href") || null;
  });
  if (!href) {
    throw new Error("no album link found for showcase shot");
  }
  const joiner = href.includes("?") ? "&" : "?";
  return `${href}${joiner}theme=${theme}`;
}

/**
 * Start playback on Bad Bunny DtMF and seek to ~45% so home shots
 * show the bottom player mid-track.
 */
async function seedPlayerProgress(page, theme) {
  // al-006-01 = Debí Tirar Más Fotos (Bad Bunny)
  const albumPath = `/music/album/al-006-01?theme=${theme}`;
  await page.goto(`${BASE}${albumPath}`, { waitUntil: "domcontentloaded" });
  await settle(page);

  // Prefer the Hot 100 hit over the album's first track.
  const dtmfRow = page.locator('[class*="track-row"], [role="row"], button, a').filter({ hasText: /^DtMF$/ }).first();
  if (await dtmfRow.count()) {
    await dtmfRow.click({ timeout: 10000 }).catch(() => {});
  } else {
    const play = page.getByRole("button", { name: /^Play$/i }).first();
    await play.click({ timeout: 15000 }).catch(() => {});
  }

  await page
    .waitForSelector(".player, .mini-player", { timeout: 20000 })
    .catch(() => {});

  await page.evaluate(() => {
    document.querySelectorAll("audio").forEach((el) => {
      el.muted = true;
      el.volume = 0;
    });
  });

  // Wait until duration is known, then seek to 45%.
  await page
    .waitForFunction(() => {
      const input = document.querySelector(
        ".player__progress input, .mini-player__progress input",
      );
      return input && Number(input.max) > 1;
    }, { timeout: 15000 })
    .catch(() => {});

  await page.evaluate((pct) => {
    const input = document.querySelector(
      ".player__progress input, .mini-player__progress input",
    );
    if (!(input instanceof HTMLInputElement)) return;
    const max = Number(input.max) || 200;
    const value = Math.max(1, max * pct);
    input.value = String(value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
    input.dispatchEvent(new Event("change", { bubbles: true }));
  }, 0.45);

  // Pause so the bar stays near 45% for the screenshot.
  const pauseBtn = page
    .getByRole("button", { name: /pause/i })
    .or(page.locator('button[aria-label*="Pause" i]'))
    .first();
  await pauseBtn.click({ timeout: 5000 }).catch(() => {});
  await sleep(400);

  await page.evaluate((pct) => {
    const input = document.querySelector(
      ".player__progress input, .mini-player__progress input",
    );
    if (input instanceof HTMLInputElement) {
      const max = Number(input.max) || 200;
      input.value = String(max * pct);
      input.dispatchEvent(new Event("input", { bubbles: true }));
    }
    const fill = document.querySelector(
      ".player__progress-fill, .mini-player__progress-fill",
    );
    if (fill instanceof HTMLElement) {
      fill.style.transform = `scaleX(${pct})`;
    }
    document.querySelectorAll("audio").forEach((el) => {
      el.pause();
      if (el.duration && Number.isFinite(el.duration)) {
        el.currentTime = el.duration * pct;
      }
    });
  }, 0.45);

  await sleep(500);
}

async function capture() {
  await clearOldShots();
  const child = await startServer();
  const browser = await chromium.launch({ headless: true });
  const index = [];

  try {
    let albumPathDark = null;
    let albumPathLight = null;

    for (const shot of SHOTS) {
      const context = await browser.newContext({
        viewport: shot.viewport,
        deviceScaleFactor: shot.device === "mobile" ? 2 : 1,
        colorScheme: shot.theme,
      });
      const page = await context.newPage();
      await page.addInitScript((mode) => {
        localStorage.setItem("melovian-theme", mode);
        localStorage.setItem("mel-home-tips-dismissed", "1");
      }, shot.theme);

      if (shot.withPlayer) {
        await seedPlayerProgress(page, shot.theme);
        const homeLink = page
          .locator('a[href="/"], a[href="/music"], a[href^="/?"]')
          .filter({ hasText: /^Home$/i })
          .first();
        if (await homeLink.count()) {
          await homeLink.click();
        } else {
          await page.goto(`${BASE}/?theme=${shot.theme}`, {
            waitUntil: "domcontentloaded",
          });
        }
        await settle(page);
        await page
          .waitForSelector(".player, .mini-player", { timeout: 10000 })
          .catch(() => {});
        await page.evaluate(() => {
          document.querySelectorAll("audio").forEach((el) => {
            el.muted = true;
            el.volume = 0;
          });
        });
        await sleep(600);
      } else {
        let path = shot.path;
        if (path === "ALBUM") {
          if (shot.theme === "dark") {
            albumPathDark =
              albumPathDark || (await resolveAlbumPath(page, "dark"));
            path = albumPathDark;
          } else {
            albumPathLight =
              albumPathLight || (await resolveAlbumPath(page, "light"));
            path = albumPathLight;
          }
        }

        const url = path.startsWith("http") ? path : `${BASE}${path}`;
        await page.goto(url, { waitUntil: "domcontentloaded" });
        await settle(page);
      }

      const file = `${shot.id}.png`;
      await page.screenshot({ path: join(OUT, file), fullPage: false });
      index.push({
        file,
        device: shot.device,
        theme: shot.theme,
        page: shot.id.replace(/^(desktop|mobile)-(dark|light)-/, ""),
      });
      console.log(`wrote ${file}`);
      await context.close();
    }

    await writeFile(
      join(OUT, "index.json"),
      JSON.stringify(
        { generatedAt: new Date().toISOString(), shots: index },
        null,
        2,
      ) + "\n",
    );
    await writeFile(
      join(OUT, "README.md"),
      [
        "# Showcase screenshots",
        "",
        "Generated by `scripts/showcase/capture.mjs` against demo mode.",
        "Catalog names mirror real releases. Covers are fetched via iTunes lookup.",
        "",
        "| File | Device | Theme | Page |",
        "|------|--------|-------|------|",
        ...index.map(
          (s) => `| \`${s.file}\` | ${s.device} | ${s.theme} | ${s.page} |`,
        ),
        "",
      ].join("\n"),
    );
  } finally {
    await browser.close();
    if (child) {
      child.kill("SIGTERM");
      await sleep(500);
      if (!child.killed) child.kill("SIGKILL");
    }
  }
}

capture().catch((err) => {
  console.error(err);
  process.exit(1);
});
