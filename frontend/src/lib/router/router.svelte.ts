// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { Component } from "svelte";
import { matchPath, parseQuery, type RouteMatch } from "./match";

export type { RouteMatch } from "./match";
export { matchPath, parseQuery, matchRoute } from "./match";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type RouteComponent = Component<any>;
export type RouteLoader = () => Promise<{ default: RouteComponent }>;

export interface RouteDefinition {
  path: string;
  component?: RouteComponent;
  load?: RouteLoader;
  title?: string;
}

function appBase(): string {
  const raw =
    (typeof import.meta !== "undefined" && import.meta.env?.BASE_URL) || "/";
  if (!raw || raw === "/") return "";
  return raw.endsWith("/") ? raw.slice(0, -1) : raw;
}

function stripBase(pathname: string): string {
  const base = appBase();
  if (base && pathname.startsWith(base)) {
    const next = pathname.slice(base.length);
    const normalized = next.startsWith("/") ? next : `/${next}`;
    return normalized || "/";
  }
  return pathname || "/";
}

function withBase(path: string): string {
  const base = appBase();
  const normalized = path.startsWith("/") ? path : `/${path}`;
  return base ? `${base}${normalized}` : normalized;
}

export { withBase };

class RouterStore {
  pathname = $state(
    typeof window !== "undefined" ? stripBase(window.location.pathname) : "/",
  );
  search = $state(typeof window !== "undefined" ? window.location.search : "");

  constructor() {
    if (typeof window === "undefined") return;

    const onPopState = () => {
      this.pathname = stripBase(window.location.pathname);
      this.search = window.location.search;
    };

    window.addEventListener("popstate", onPopState);
  }

  navigate(path: string, replace = false) {
    const url = withBase(path.startsWith("/") ? path : `/${path}`);
    if (replace) {
      window.history.replaceState({}, "", url);
    } else {
      window.history.pushState({}, "", url);
    }
    this.pathname = stripBase(window.location.pathname);
    this.search = window.location.search;
  }

  match(
    routes: RouteDefinition[],
  ): { route: RouteDefinition; match: RouteMatch } | null {
    const query = parseQuery(this.search);

    for (const route of routes) {
      const result = matchPath(route.path, this.pathname);
      if (result) {
        return { route, match: { ...result, query } };
      }
    }

    return null;
  }
}

export const router = new RouterStore();

export function link(node: HTMLAnchorElement, href: string) {
  const onClick = (event: MouseEvent) => {
    if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey)
      return;
    event.preventDefault();
    router.navigate(href);
  };

  node.addEventListener("click", onClick);
  return {
    update(next: string) {
      href = next;
    },
    destroy() {
      node.removeEventListener("click", onClick);
    },
  };
}
