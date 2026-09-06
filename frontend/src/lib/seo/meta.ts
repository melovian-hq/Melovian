// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { APP_DESCRIPTION, APP_NAME, DEFAULT_OG_IMAGE } from "$lib/brand";
import { withBase } from "$lib/router/router.svelte";
import type { RouteDefinition } from "$lib/router/router.svelte";

export type OgType = "website" | "music.album" | "music.song" | "profile";

export interface PageMeta {
  /** Short page title without the app name suffix. */
  title: string;
  description?: string;
  /** Absolute or site-root path for og:image. */
  image?: string;
  /** Path only (no origin). Defaults to the current location. */
  path?: string;
  type?: OgType;
  /** When false, omit robots index. Default true for shareable pages. */
  index?: boolean;
}

const META_NAME = [
  "description",
  "robots",
  "twitter:card",
  "twitter:title",
  "twitter:description",
  "twitter:image",
] as const;

const META_PROPERTY = [
  "og:type",
  "og:site_name",
  "og:title",
  "og:description",
  "og:url",
  "og:image",
] as const;

function cssEscape(value: string): string {
  if (typeof CSS !== "undefined" && typeof CSS.escape === "function") {
    return CSS.escape(value);
  }
  return value.replace(/["\\]/g, "\\$&");
}

function ensureMeta(
  attr: "name" | "property",
  key: string,
): HTMLMetaElement | null {
  if (typeof document === "undefined") return null;
  const selector = `meta[${attr}="${cssEscape(key)}"]`;
  let el = document.head.querySelector(selector) as HTMLMetaElement | null;
  if (!el) {
    el = document.createElement("meta");
    el.setAttribute(attr, key);
    document.head.appendChild(el);
  }
  return el;
}

function ensureLink(rel: string): HTMLLinkElement | null {
  if (typeof document === "undefined") return null;
  let el = document.head.querySelector(
    `link[rel="${cssEscape(rel)}"]`,
  ) as HTMLLinkElement | null;
  if (!el) {
    el = document.createElement("link");
    el.rel = rel;
    document.head.appendChild(el);
  }
  return el;
}

function absoluteUrl(pathOrUrl: string): string {
  if (/^https?:\/\//i.test(pathOrUrl)) return pathOrUrl;
  if (typeof window === "undefined") return pathOrUrl;
  const path = pathOrUrl.startsWith("/") ? pathOrUrl : `/${pathOrUrl}`;
  return `${window.location.origin}${withBase(path)}`;
}

function documentTitle(pageTitle: string): string {
  const trimmed = pageTitle.trim();
  if (!trimmed || trimmed === APP_NAME) return APP_NAME;
  if (
    trimmed.endsWith(` · ${APP_NAME}`) ||
    trimmed.endsWith(` - ${APP_NAME}`)
  ) {
    return trimmed;
  }
  return `${trimmed} · ${APP_NAME}`;
}

/** Build default meta for a static route definition. */
export function metaFromRoute(route: RouteDefinition, path: string): PageMeta {
  return {
    title: route.title?.trim() || APP_NAME,
    description: route.description?.trim() || APP_DESCRIPTION,
    path,
    type: "website",
    image: DEFAULT_OG_IMAGE,
    index: route.index !== false,
  };
}

/** Apply page meta to document head (title, description, Open Graph, Twitter). */
export function applyPageMeta(meta: PageMeta): void {
  if (typeof document === "undefined") return;

  const title = documentTitle(meta.title || APP_NAME);
  const description = (meta.description?.trim() || APP_DESCRIPTION).slice(
    0,
    300,
  );
  const path =
    meta.path ??
    (typeof window !== "undefined"
      ? `${window.location.pathname}${window.location.search}`
      : "/");
  const url = absoluteUrl(path.startsWith("http") ? path : path || "/");
  const image = absoluteUrl(meta.image?.trim() || DEFAULT_OG_IMAGE);
  const type = meta.type ?? "website";
  const robots = meta.index === false ? "noindex, nofollow" : "index, follow";

  document.title = title;

  const byName: Record<(typeof META_NAME)[number], string> = {
    description,
    robots,
    "twitter:card": "summary_large_image",
    "twitter:title": title,
    "twitter:description": description,
    "twitter:image": image,
  };
  for (const key of META_NAME) {
    ensureMeta("name", key)?.setAttribute("content", byName[key]);
  }

  const byProp: Record<(typeof META_PROPERTY)[number], string> = {
    "og:type": type,
    "og:site_name": APP_NAME,
    "og:title": title,
    "og:description": description,
    "og:url": url,
    "og:image": image,
  };
  for (const key of META_PROPERTY) {
    ensureMeta("property", key)?.setAttribute("content", byProp[key]);
  }

  const canonical = ensureLink("canonical");
  if (canonical) canonical.href = url;
}

/** Convenience for pages that load entity names after mount. */
export function setPageMeta(
  partial: Partial<PageMeta> & { title: string },
): void {
  applyPageMeta({
    description: APP_DESCRIPTION,
    image: DEFAULT_OG_IMAGE,
    type: "website",
    ...partial,
  });
}
