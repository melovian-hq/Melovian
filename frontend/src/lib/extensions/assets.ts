// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { resolveApiUrl } from "$lib/config/remote-server";
import { ApiPaths } from "$lib/core/http/api-paths";

/** True when rel is a safe path under an extension directory (no traversal). */
export function isSafeExtensionAssetPath(rel: string): boolean {
  const cleaned = rel.trim().replace(/\\/g, "/").replace(/^\/+/, "");
  if (!cleaned) return false;
  if (cleaned.includes("..")) return false;
  if (cleaned.startsWith("/")) return false;
  return true;
}

/** Absolute URL for an installed extension asset served by the API. */
export function extensionAssetUrl(extensionId: string, rel: string): string {
  const cleaned = rel.trim().replace(/\\/g, "/").replace(/^\/+/, "");
  if (!extensionId.trim() || !isSafeExtensionAssetPath(cleaned)) {
    return "";
  }
  const encoded = cleaned
    .split("/")
    .map((part) => encodeURIComponent(part))
    .join("/");
  return resolveApiUrl(ApiPaths.extensionAsset(extensionId.trim(), encoded));
}

const preloaded = new Set<string>();

/** Warm the browser image cache for decoration assets (once per URL). */
export function preloadDecorationImages(urls: Array<string | undefined>) {
  if (typeof Image === "undefined") return;
  for (const url of urls) {
    const href = String(url ?? "").trim();
    if (!href || preloaded.has(href)) continue;
    preloaded.add(href);
    const img = new Image();
    img.decoding = "async";
    img.src = href;
  }
}
