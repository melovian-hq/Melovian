// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/** Capability and version contract mirrored from internal/compat. */

import { APP_NAME } from "$lib/brand";

export const CLIENT_VERSION =
  (typeof import.meta !== "undefined" &&
    (import.meta.env?.VITE_APP_VERSION as string | undefined)) ||
  "0.1.0";

export const API_VERSION = 1;

export const MIN_SERVER_VERSION = "0.1.0";

export const HEADER_CLIENT_VERSION = "X-Melovian-Client-Version";
export const HEADER_API_VERSION = "X-Melovian-API-Version";
export const HEADER_CAPABILITIES = "X-Melovian-Capabilities";
export const HEADER_SERVER_VERSION = "X-Melovian-Server-Version";

export const Cap = {
  browse: "browse",
  playback: "playback",
  auth: "auth",
  instances: "instances",
  playlists: "playlists",
  favorites: "favorites",
  history: "history",
  mixes: "mixes",
  personalRadio: "personal_radio",
  localLibrary: "local_library",
  downloads: "downloads",
  party: "party",
  extensions: "extensions",
  ws: "ws",
  lyrics: "lyrics",
  eq: "eq",
  devices: "devices",
  shared: "shared",
  videos: "videos",
  metadataEditor: "metadata_editor",
  settings: "settings",
} as const;

export type Capability = (typeof Cap)[keyof typeof Cap] | string;

export const CLIENT_CAPABILITIES: string[] = Object.values(Cap);

/** Assumed when the server omits capabilities (pre-compat builds). */
export const LEGACY_CAPABILITIES: string[] = [
  Cap.browse,
  Cap.playback,
  Cap.auth,
  Cap.instances,
  Cap.playlists,
  Cap.favorites,
  Cap.history,
  Cap.settings,
];

export type CompatState = {
  serverVersion: string;
  apiVersion: number;
  minClientVersion: string;
  minServerVersion: string;
  serverCapabilities: string[];
  supported: string[];
  legacy: boolean;
  serverTooOld: boolean;
  versionMismatch: boolean;
  blocked: boolean;
  blockReason: string;
};

let state: CompatState = {
  serverVersion: "",
  apiVersion: 0,
  minClientVersion: "",
  minServerVersion: MIN_SERVER_VERSION,
  serverCapabilities: [...LEGACY_CAPABILITIES],
  supported: intersect(CLIENT_CAPABILITIES, LEGACY_CAPABILITIES),
  legacy: true,
  serverTooOld: false,
  versionMismatch: false,
  blocked: false,
  blockReason: "",
};

export function getCompatState(): CompatState {
  return state;
}

export function supports(cap: Capability): boolean {
  if (state.blocked) return false;
  return state.supported.includes(cap);
}

export function compareSemver(a: string, b: string): number {
  const na = normalizeSemver(a);
  const nb = normalizeSemver(b);
  if (na < nb) return -1;
  if (na > nb) return 1;
  return 0;
}

/** Exact version match after trim and optional leading v. Used for mismatch banners. */
export function versionsEqual(a: string, b: string): boolean {
  return normalizeVersionLabel(a) === normalizeVersionLabel(b);
}

/** Rough scheme check so package.json 0.1.0 vs git 416368d does not warn in mixed stamps. */
export function versionScheme(v: string): "git" | "semver" | "other" {
  const n = normalizeVersionLabel(v);
  if (/^[0-9a-f]{4,40}(-dirty)?$/i.test(n)) return "git";
  if (/^\d+\.\d+/.test(n)) return "semver";
  return "other";
}

function normalizeVersionLabel(v: string): string {
  return String(v ?? "")
    .trim()
    .replace(/^[vV]/, "");
}

function normalizeSemver(v: string): string {
  let s = String(v ?? "")
    .trim()
    .replace(/^[vV]/, "");
  const cut = s.search(/[-+]/);
  if (cut >= 0) s = s.slice(0, cut);
  const parts = s.split(".");
  while (parts.length < 3) parts.push("0");
  return parts
    .slice(0, 3)
    .map((p) => {
      const n = String(parseInt(p.replace(/^0+(?=\d)/, "") || "0", 10));
      return n.padStart(8, "0");
    })
    .join(".");
}

function intersect(a: string[], b: string[]): string[] {
  const set = new Set(b);
  return a.filter((c) => set.has(c));
}

export type CompatConfigPayload = {
  version?: string;
  apiVersion?: number;
  minClientVersion?: string;
  minServerVersion?: string;
  capabilities?: string[];
};

/** Apply /api/config compat fields. Old servers omit them (legacy mode). */
export function applyCompatFromConfig(cfg: CompatConfigPayload): CompatState {
  const hasCaps = Array.isArray(cfg.capabilities);
  const hasVersion = typeof cfg.version === "string" && cfg.version.length > 0;
  const legacy = !hasCaps && !hasVersion;

  const serverVersion = hasVersion ? String(cfg.version) : "";
  const apiVersion =
    typeof cfg.apiVersion === "number" ? cfg.apiVersion : legacy ? 0 : 0;
  const minClientVersion =
    typeof cfg.minClientVersion === "string" ? cfg.minClientVersion : "";
  const minServerVersion =
    typeof cfg.minServerVersion === "string" && cfg.minServerVersion
      ? cfg.minServerVersion
      : MIN_SERVER_VERSION;

  const serverCapabilities = hasCaps
    ? [...(cfg.capabilities as string[])]
    : [...LEGACY_CAPABILITIES];

  const serverTooOld =
    Boolean(serverVersion) &&
    compareSemver(serverVersion, MIN_SERVER_VERSION) < 0;

  const versionMismatch =
    Boolean(serverVersion) &&
    !versionsEqual(serverVersion, CLIENT_VERSION) &&
    !serverTooOld &&
    versionScheme(serverVersion) === versionScheme(CLIENT_VERSION);

  const blocked = serverTooOld;
  const blockReason = blocked
    ? `This server (${serverVersion}) is older than the minimum this client supports (${MIN_SERVER_VERSION}). Update the server.`
    : "";

  state = {
    serverVersion,
    apiVersion,
    minClientVersion,
    minServerVersion,
    serverCapabilities,
    supported: intersect(CLIENT_CAPABILITIES, serverCapabilities),
    legacy,
    serverTooOld,
    versionMismatch,
    blocked,
    blockReason,
  };
  return state;
}

/** Mark the session blocked because the server refused this client (HTTP 426). */
export function blockForClientTooOld(message?: string): CompatState {
  state = {
    ...state,
    blocked: true,
    serverTooOld: false,
    blockReason:
      message ||
      `This ${APP_NAME} client is too old for the server. Update the client.`,
    supported: [],
  };
  return state;
}

export function resetCompatForTests(): void {
  state = {
    serverVersion: "",
    apiVersion: 0,
    minClientVersion: "",
    minServerVersion: MIN_SERVER_VERSION,
    serverCapabilities: [...LEGACY_CAPABILITIES],
    supported: intersect(CLIENT_CAPABILITIES, LEGACY_CAPABILITIES),
    legacy: true,
    serverTooOld: false,
    versionMismatch: false,
    blocked: false,
    blockReason: "",
  };
}

export function clientCompatHeaders(): Record<string, string> {
  return {
    [HEADER_CLIENT_VERSION]: CLIENT_VERSION,
    [HEADER_API_VERSION]: String(API_VERSION),
    [HEADER_CAPABILITIES]: CLIENT_CAPABILITIES.join(","),
  };
}
