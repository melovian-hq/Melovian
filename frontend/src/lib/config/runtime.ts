// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { StorageKeys } from "$lib/brand";
import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import { ApiPaths } from "$lib/core/http/api-paths";
import {
  applyRuntimeSentryConfig,
  disableClientSentry,
} from "$lib/core/sentry";
import { setConnectionDefaults } from "$lib/music/connection-defaults";
import {
  applyCompatFromConfig,
  blockForClientTooOld,
  CLIENT_VERSION,
  versionsEqual,
  versionScheme,
} from "$lib/compat";
import { getRemoteServerUrl, isRemoteClient } from "$lib/config/remote-server";
import { normalizePublicBaseUrl } from "$lib/config/runtime-url";
import { parseJson, parsePayload } from "$lib/core/http/parse";
import { runtimeConfigSchema, upgradeRequiredSchema } from "./schemas";

export { normalizePublicBaseUrl } from "$lib/config/runtime-url";

/** One-shot soft reload when an embedded SPA is stale vs the server build. */
function maybeReloadStaleEmbeddedClient(serverVersion: string): boolean {
  if (typeof window === "undefined") return false;
  if (isRemoteClient() || isStaticDemo()) return false;
  if (!serverVersion || versionsEqual(serverVersion, CLIENT_VERSION)) {
    return false;
  }
  if (versionScheme(serverVersion) !== versionScheme(CLIENT_VERSION)) {
    return false;
  }

  const key = `${StorageKeys.spaReloadPrefix}${CLIENT_VERSION}->${serverVersion}`;
  try {
    if (sessionStorage.getItem(key) === "1") return false;
    sessionStorage.setItem(key, "1");
  } catch {
    // Still attempt one reload when storage is unavailable.
  }

  const next = new URL(window.location.href);
  next.searchParams.set("_jjv", serverVersion);
  window.location.replace(next.toString());
  return true;
}

let mediaBaseUrl = "";
let serverMode = typeof window !== "undefined" ? !hasWailsHost() : false;
let demoMode = false;
let fakeCatalog = false;
let staticDemo = false;

type WailsWindow = Window & {
  chrome?: { webview?: { postMessage?: (message: unknown) => void } };
  webkit?: {
    messageHandlers?: {
      external?: { postMessage?: (message: unknown) => void };
      melovianMedia?: { postMessage?: (message: unknown) => void };
    };
  };
  wails?: {
    invokeAsync?: (id: string, payload: string) => void;
    setMediaPlaying?: (playing: boolean) => void;
    setMediaState?: (state: string) => void;
    setMediaQueue?: (queue: string) => void;
  };
  _wails?: { environment?: { OS?: string } };
};

export function hasWailsHost(): boolean {
  if (typeof window === "undefined") return false;
  const w = window as WailsWindow;
  if (window.location?.protocol === "wails:") return true;
  if (w.chrome?.webview?.postMessage) return true;
  if (w.webkit?.messageHandlers?.external?.postMessage) return true;
  if (typeof w.wails?.invokeAsync === "function") return true;
  return false;
}

function wailsEnvironmentOS(): string {
  if (typeof window === "undefined") return "";
  return String(
    (window as WailsWindow)._wails?.environment?.OS ?? "",
  ).toLowerCase();
}

export function isWailsMobile(): boolean {
  const os = wailsEnvironmentOS();
  if (os === "android" || os === "ios") return true;
  return (
    typeof window !== "undefined" &&
    window.location?.hostname === "wails.localhost"
  );
}

export function isWailsDesktop(): boolean {
  return hasWailsHost() && !isWailsMobile();
}

export function isServerMode(): boolean {
  return serverMode;
}

export function isDemoMode(): boolean {
  return demoMode;
}

export function isFakeCatalog(): boolean {
  return fakeCatalog;
}

export function isStaticDemo(): boolean {
  return (
    staticDemo ||
    (typeof import.meta !== "undefined" &&
      import.meta.env?.VITE_STATIC_DEMO === "true")
  );
}

export function setStaticDemo(value: boolean): void {
  staticDemo = value;
  if (value) {
    demoMode = true;
    fakeCatalog = true;
    serverMode = true;
  }
}

export function nativeDesktopAvailable(): boolean {
  return isWailsDesktop() && !serverMode;
}

export function setMobileMediaPlaying(playing: boolean): void {
  if (typeof window === "undefined" || !isWailsMobile()) return;
  const w = window as WailsWindow;
  w.wails?.setMediaPlaying?.(playing);
}

export function setMobileMediaState(
  playing: boolean,
  track?: {
    title?: string;
    artist?: string;
    album?: string;
    artwork?: string;
    duration?: number;
    position?: number;
  },
): void {
  if (typeof window === "undefined" || !isWailsMobile()) return;
  const w = window as WailsWindow;
  const payload = {
    active: Boolean(track),
    playing,
    title: track?.title ?? "",
    artist: track?.artist ?? "",
    album: track?.album ?? "",
    artwork: track?.artwork ?? "",
    duration: track?.duration ?? 0,
    position: track?.position ?? 0,
  };
  if (w.wails?.setMediaState) {
    w.wails.setMediaState(JSON.stringify(payload));
    return;
  }
  // iOS: WKScriptMessageHandler registered as melovianMedia
  const handler = w.webkit?.messageHandlers?.melovianMedia;
  if (handler?.postMessage) {
    handler.postMessage({ type: "setMediaState", ...payload });
    return;
  }
  w.wails?.setMediaPlaying?.(playing);
}

export type MobileQueueTrack = {
  id?: string;
  title?: string;
  artist?: string;
  album?: string;
  artwork?: string;
};

/** Push the current queue to native (Android Auto browse / iOS optional). */
export function setMobileMediaQueue(
  tracks: MobileQueueTrack[],
  index: number,
): void {
  if (typeof window === "undefined" || !isWailsMobile()) return;
  const w = window as WailsWindow;
  const payload = JSON.stringify({
    index,
    tracks: tracks.slice(0, 80).map((t) => ({
      id: t.id ?? "",
      title: t.title ?? "",
      artist: t.artist ?? "",
      album: t.album ?? "",
      artwork: t.artwork ?? "",
    })),
  });
  if (w.wails?.setMediaQueue) {
    w.wails.setMediaQueue(payload);
    return;
  }
  const handler = w.webkit?.messageHandlers?.melovianMedia;
  if (handler?.postMessage) {
    handler.postMessage({ type: "setMediaQueue", ...JSON.parse(payload) });
  }
}

export function setMediaBaseUrl(url: string): void {
  mediaBaseUrl = normalizePublicBaseUrl(url);
}

export function getMediaBaseUrl(): string {
  return mediaBaseUrl;
}

/**
 * resolveMediaUrl prefixes a relative API path with the HTTP origin used for
 * streams and covers. Desktop uses the embedded localhost API. Mobile stays
 * same-origin unless a remote server host is configured.
 */
export function resolveMediaUrl(path: string): string {
  if (/^https?:\/\//i.test(path)) return path;
  if (isWailsMobile() && isRemoteClient()) {
    const base = getRemoteServerUrl() || mediaBaseUrl;
    if (!base) {
      return path.startsWith("/") ? path : `/${path}`;
    }
    return `${base}${path.startsWith("/") ? "" : "/"}${path}`;
  }
  if (isWailsMobile()) {
    return path.startsWith("/") ? path : `/${path}`;
  }
  if (!mediaBaseUrl) return path;
  return `${mediaBaseUrl}${path.startsWith("/") ? "" : "/"}${path}`;
}

export function mediaBaseFromListenAddr(listenAddr: string): string {
  if (typeof window !== "undefined" && window.location?.origin) {
    const host = listenAddr.trim();
    if (
      host.startsWith("0.0.0.0:") ||
      host.startsWith("[::]:") ||
      host === "0.0.0.0" ||
      host === "[::]" ||
      host === ":8080" ||
      host.endsWith(":8080")
    ) {
      return normalizePublicBaseUrl(window.location.origin);
    }
  }

  let host = listenAddr.trim();
  if (host === "") return "";
  if (host.startsWith(":")) {
    host = `127.0.0.1${host}`;
  }
  host = host.replace(/^0\.0\.0\.0$/, "127.0.0.1");
  host = host.replace(/^0\.0\.0\.0:/, "127.0.0.1:");
  host = host.replace(/^\[::\]$/, "127.0.0.1");
  host = host.replace(/^\[::\]:/, "127.0.0.1:");
  return `http://${host}`;
}

export async function loadRuntimeConfig(): Promise<void> {
  if (isStaticDemo()) {
    demoMode = true;
    fakeCatalog = true;
    serverMode = true;
    if (typeof window !== "undefined") {
      setMediaBaseUrl(window.location.origin);
    }
    applyCompatFromConfig({
      version: "0.1.0",
      apiVersion: 1,
      minClientVersion: "0.1.0",
      minServerVersion: "0.1.0",
      capabilities: [
        "browse",
        "playback",
        "playlists",
        "favorites",
        "history",
        "mixes",
        "personal_radio",
        "lyrics",
        "eq",
      ],
    });
    return;
  }
  try {
    const response = await fetchWithRetry(ApiPaths.config, {
      headers: apiHeaders(),
    });
    if (response.status === 426) {
      let message = "";
      try {
        const body = parsePayload(
          upgradeRequiredSchema,
          await response.json(),
          "upgrade required response",
        );
        message = body.message ?? "";
      } catch {
        /* ignore */
      }
      blockForClientTooOld(message);
      return;
    }
    if (!response.ok) return;
    const cfg = await parseJson(
      runtimeConfigSchema,
      response,
      "runtime config",
    );
    applyCompatFromConfig(cfg);
    if (
      typeof cfg.version === "string" &&
      maybeReloadStaleEmbeddedClient(cfg.version)
    ) {
      return;
    }
    if (isWailsMobile() && isRemoteClient()) {
      setMediaBaseUrl(getRemoteServerUrl());
    } else if (isWailsMobile()) {
      setMediaBaseUrl(
        typeof window !== "undefined" ? window.location.origin : "",
      );
    } else if (cfg.publicUrl) {
      setMediaBaseUrl(cfg.publicUrl);
    } else if (cfg.listenAddr) {
      setMediaBaseUrl(mediaBaseFromListenAddr(cfg.listenAddr));
    } else if (typeof window !== "undefined") {
      setMediaBaseUrl(window.location.origin);
    }
    if (cfg.connectionDefaults) {
      setConnectionDefaults(cfg.connectionDefaults);
    }
    if (cfg.sentry?.clientReporting) {
      applyRuntimeSentryConfig(cfg.sentry);
    } else {
      disableClientSentry();
    }
    serverMode = cfg.serverMode === true;
    demoMode = cfg.demoMode === true;
    fakeCatalog = cfg.fakeCatalog === true;
  } catch {
    if (typeof window !== "undefined") {
      setMediaBaseUrl(window.location.origin);
    }
  }
}
