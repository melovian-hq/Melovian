// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { spawn, type ChildProcess } from "node:child_process";
import { createWriteStream, existsSync } from "node:fs";
import { mkdir, rm } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { setTimeout as sleep } from "node:timers/promises";

const __dirname = dirname(fileURLToPath(import.meta.url));
const FRONTEND_ROOT = resolve(__dirname, "../..");
const REPO_ROOT = resolve(FRONTEND_ROOT, "..");

export const DEFAULT_E2E_LISTEN = process.env.E2E_LISTEN || "127.0.0.1:17348";
export const DEFAULT_E2E_BASE_URL =
  process.env.E2E_BASE_URL || `http://${DEFAULT_E2E_LISTEN}`;

const TMP_ROOT = join(FRONTEND_ROOT, "e2e", ".tmp");

export function resolveServerBinary(): string {
  if (process.env.E2E_BIN) {
    return process.env.E2E_BIN;
  }
  const name =
    process.platform === "win32" ? "melovian-server.exe" : "melovian-server";
  return join(REPO_ROOT, "bin", name);
}

export async function waitForHealth(
  baseUrl: string,
  attempts = 60,
): Promise<void> {
  for (let i = 0; i < attempts; i++) {
    try {
      const res = await fetch(`${baseUrl}/health`);
      if (res.ok) return;
    } catch {
      // retry until timeout
    }
    await sleep(500);
  }
  throw new Error(`demo server did not become healthy at ${baseUrl}`);
}

export type DemoServerHandle = {
  child: ChildProcess;
  baseUrl: string;
  dataDir: string;
  logPath: string;
};

/**
 * Start melovian-server in demo mode with the built-in fake catalog.
 * Caller must stop the returned child when finished.
 */
export async function startDemoServer(options?: {
  listen?: string;
  baseUrl?: string;
  bin?: string;
}): Promise<DemoServerHandle> {
  const listen = options?.listen || DEFAULT_E2E_LISTEN;
  const baseUrl = options?.baseUrl || `http://${listen}`;
  const bin = options?.bin || resolveServerBinary();

  if (!existsSync(bin)) {
    throw new Error(
      `melovian-server binary not found at ${bin}. Run \`task build:server\` first, or set E2E_BIN / E2E_BASE_URL.`,
    );
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
      listen,
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
        MELOVIAN_LISTEN: listen,
        MELOVIAN_DATA: dataDir,
        NAVIDROME_SERVER: "fake://melovian-demo",
        NAVIDROME_USER: "demo",
        NAVIDROME_PASSWORD: "demo",
      },
      stdio: ["ignore", "pipe", "pipe"],
    },
  );

  child.stdout?.pipe(log);
  child.stderr?.pipe(log);
  child.on("exit", (code) => {
    if (code && code !== 0) {
      console.error(`demo server exited with code ${code}; see ${logPath}`);
    }
  });

  try {
    await waitForHealth(baseUrl);
  } catch (err) {
    child.kill("SIGTERM");
    throw err;
  }

  return { child, baseUrl, dataDir, logPath };
}

export async function stopDemoServer(
  handle: DemoServerHandle | null,
): Promise<void> {
  if (!handle) return;
  const { child, dataDir } = handle;
  if (!child.killed) {
    child.kill("SIGTERM");
    await sleep(500);
    if (!child.killed) {
      child.kill("SIGKILL");
    }
  }
  await rm(dataDir, { recursive: true, force: true }).catch(() => {});
}
