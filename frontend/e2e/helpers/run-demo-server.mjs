// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/**
 * Long-lived demo server process for Playwright webServer.
 * Keep this process alive until Playwright tears it down.
 */

import { spawn } from "node:child_process";
import { createWriteStream, existsSync } from "node:fs";
import { mkdir, rm } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { setTimeout as sleep } from "node:timers/promises";

const __dirname = dirname(fileURLToPath(import.meta.url));
const FRONTEND_ROOT = resolve(__dirname, "../..");
const REPO_ROOT = resolve(FRONTEND_ROOT, "..");
const LISTEN = process.env.E2E_LISTEN || "127.0.0.1:17348";
const BASE = process.env.E2E_BASE_URL || `http://${LISTEN}`;
const TMP_ROOT = join(FRONTEND_ROOT, "e2e", ".tmp");

function resolveBin() {
  if (process.env.E2E_BIN) return process.env.E2E_BIN;
  const name =
    process.platform === "win32" ? "melovian-server.exe" : "melovian-server";
  return join(REPO_ROOT, "bin", name);
}

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
  throw new Error(`demo server did not become healthy at ${url}`);
}

const bin = resolveBin();
if (!existsSync(bin)) {
  console.error(
    `melovian-server binary not found at ${bin}. Run \`task build:server\` first, or set E2E_BIN / E2E_BASE_URL.`,
  );
  process.exit(1);
}

await mkdir(TMP_ROOT, { recursive: true });
const dataDir = join(TMP_ROOT, `demo-data-${process.pid}`);
await rm(dataDir, { recursive: true, force: true });
await mkdir(dataDir, { recursive: true });

const logPath = join(TMP_ROOT, `server-${process.pid}.log`);
const log = createWriteStream(logPath, { flags: "w" });

const child = spawn(
  bin,
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
    cwd: REPO_ROOT,
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

function shutdown(signal) {
  if (!child.killed) {
    child.kill(signal);
  }
}

process.on("SIGINT", () => shutdown("SIGTERM"));
process.on("SIGTERM", () => shutdown("SIGTERM"));
child.on("exit", (code) => {
  process.exit(code && code !== 0 ? code : 0);
});

try {
  await waitForHealth(BASE);
  console.log(`e2e demo server ready at ${BASE}`);
} catch (err) {
  console.error(err);
  shutdown("SIGKILL");
  process.exit(1);
}

// Stay alive while Playwright runs.
await new Promise(() => {});
