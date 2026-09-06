// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { extensionAssetUrl, isSafeExtensionAssetPath } from "./assets";
import { APP_SLUG } from "$lib/brand";

const STYLE_ATTR = `data-${APP_SLUG}-extension-style`;

export type StyleManifest = {
  id: string;
  styles?: string[];
};

/** Inject or remove link tags for enabled extension stylesheets. */
export function syncExtensionStyles(manifests: StyleManifest[]) {
  if (typeof document === "undefined") return;

  const wanted = new Map<string, string>();
  for (const manifest of manifests) {
    const id = String(manifest.id ?? "").trim();
    if (!id) continue;
    for (const rel of manifest.styles ?? []) {
      if (!isSafeExtensionAssetPath(rel)) continue;
      const href = extensionAssetUrl(id, rel);
      if (!href) continue;
      wanted.set(
        `${id}:${rel.trim().replace(/\\/g, "/").replace(/^\/+/, "")}`,
        href,
      );
    }
  }

  for (const el of Array.from(
    document.querySelectorAll(`link[${STYLE_ATTR}]`),
  )) {
    const key = el.getAttribute(STYLE_ATTR);
    if (!key || !wanted.has(key)) {
      el.remove();
      continue;
    }
    wanted.delete(key);
  }

  for (const [key, href] of wanted) {
    const link = document.createElement("link");
    link.rel = "stylesheet";
    link.href = href;
    link.setAttribute(STYLE_ATTR, key);
    document.head.appendChild(link);
  }
}

/** Test helper. Removes all injected extension stylesheets. */
export function clearExtensionStylesForTests() {
  if (typeof document === "undefined") return;
  for (const el of Array.from(
    document.querySelectorAll(`link[${STYLE_ATTR}]`),
  )) {
    el.remove();
  }
}
