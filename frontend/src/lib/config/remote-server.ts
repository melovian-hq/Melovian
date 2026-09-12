// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { normalizePublicBaseUrl } from "./runtime-url";
import { parseJson } from "$lib/core/http/parse";
import { runtimeConfigSchema } from "./schemas";
import { APP_NAME, StorageKeys } from "$lib/brand";

const STORAGE_KEY = StorageKeys.remoteServerUrl;

let remoteServerUrl = "";
let hydrated = false;

function readStored(): string {
  if (typeof localStorage === "undefined") return "";
  try {
    return normalizePublicBaseUrl(localStorage.getItem(STORAGE_KEY) ?? "");
  } catch {
    return "";
  }
}

function hydrate(): void {
  if (hydrated) return;
  hydrated = true;
  remoteServerUrl = readStored();
}

/** True for hosts that are usually reached over a private/VPN overlay. */
export function looksLikePrivateOrOverlayHost(hostname: string): boolean {
  const host = hostname
    .trim()
    .toLowerCase()
    .replace(/^\[|\]$/g, "");
  if (!host) return false;
  if (
    host === "localhost" ||
    host.endsWith(".localhost") ||
    host.endsWith(".local") ||
    host.endsWith(".ts.net") ||
    host.endsWith(".nb.local") ||
    host.endsWith(".netbird.cloud")
  ) {
    return true;
  }
  if (/^\d{1,3}(?:\.\d{1,3}){3}$/.test(host)) {
    const parts = host.split(".").map((p) => Number(p));
    const [a, b] = parts;
    if (a === 10 || a === 127) return true;
    if (a === 192 && b === 168) return true;
    if (a === 172 && b >= 16 && b <= 31) return true;
    // Tailscale / CGNAT shared range
    if (a === 100 && b >= 64 && b <= 127) return true;
    return false;
  }
  return false;
}

/** Normalize a remote server host URL for storage (no trailing slash). */
export function normalizeRemoteServerUrl(raw: string): string {
  let url = raw.trim();
  if (!url) return "";
  if (!/^[a-z][a-z0-9+.-]*:\/\//i.test(url)) {
    const hostPart = url.split("/")[0] ?? url;
    const hostname = hostPart.replace(/:\d+$/, "");
    // Overlay/LAN hosts default to http. Public hostnames keep https.
    url = looksLikePrivateOrOverlayHost(hostname)
      ? `http://${url}`
      : `https://${url}`;
  }
  return normalizePublicBaseUrl(url);
}

export function getRemoteServerUrl(): string {
  hydrate();
  return remoteServerUrl;
}

export function isRemoteClient(): boolean {
  return Boolean(getRemoteServerUrl());
}

export function setRemoteServerUrl(raw: string): string {
  const normalized = normalizeRemoteServerUrl(raw);
  hydrate();
  remoteServerUrl = normalized;
  if (typeof localStorage !== "undefined") {
    try {
      if (normalized) {
        localStorage.setItem(STORAGE_KEY, normalized);
      } else {
        localStorage.removeItem(STORAGE_KEY);
      }
    } catch {
      /* ignore quota / private mode */
    }
  }
  return normalized;
}

export function clearRemoteServerUrl(): void {
  setRemoteServerUrl("");
}

/** Prefix a relative API or media path with the remote server origin when set. */
export function resolveApiUrl(path: string): string {
  if (/^https?:\/\//i.test(path) || /^wss?:\/\//i.test(path)) {
    return path;
  }
  const base = getRemoteServerUrl();
  if (!base) {
    return path.startsWith("/") || path === "" ? path : `/${path}`;
  }
  if (!path || path === "/") {
    return base;
  }
  return `${base}${path.startsWith("/") ? "" : "/"}${path}`;
}

/** Probe a remote server host without saving. Throws on failure. */
export async function probeRemoteServer(raw: string): Promise<{
  url: string;
  authEnabled: boolean;
  serverMode: boolean;
  demoMode: boolean;
}> {
  const url = normalizeRemoteServerUrl(raw);
  if (!url) {
    throw new Error(`Enter a ${APP_NAME} server URL`);
  }
  let response: Response;
  try {
    response = await fetch(`${url}/api/config`, {
      method: "GET",
      credentials: "omit",
      headers: { Accept: "application/json" },
    });
  } catch {
    throw new Error(
      `Could not reach ${url}. Check the URL and that this device is on the same Tailscale/Netbird network.`,
    );
  }
  if (!response.ok) {
    throw new Error(`Could not reach ${APP_NAME} (${response.status})`);
  }
  const cfg = await parseJson(
    runtimeConfigSchema,
    response,
    "remote server config",
  );
  return {
    url,
    authEnabled: cfg.authEnabled === true,
    serverMode: cfg.serverMode === true,
    demoMode: cfg.demoMode === true,
  };
}
