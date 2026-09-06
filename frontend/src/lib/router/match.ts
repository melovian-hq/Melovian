// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export interface RouteMatch {
  params: Record<string, string>;
  query: Record<string, string>;
}

export function matchPath(
  pattern: string,
  pathname: string,
): RouteMatch | null {
  const patternParts = pattern.split("/").filter(Boolean);
  const pathParts = pathname.split("/").filter(Boolean);

  if (patternParts.length !== pathParts.length) return null;

  const params: Record<string, string> = {};

  for (let i = 0; i < patternParts.length; i++) {
    const part = patternParts[i];
    const value = pathParts[i];

    if (part.startsWith(":")) {
      params[part.slice(1)] = decodeURIComponent(value);
    } else if (part !== value) {
      return null;
    }
  }

  return { params, query: {} };
}

export function parseQuery(search: string): Record<string, string> {
  const query: Record<string, string> = {};
  const params = new URLSearchParams(search);
  params.forEach((value, key) => {
    query[key] = value;
  });
  return query;
}

export function matchRoute(
  routes: { path: string }[],
  pathname: string,
  search = "",
): { path: string; match: RouteMatch } | null {
  const query = parseQuery(search);

  for (const route of routes) {
    const result = matchPath(route.path, pathname);
    if (result) {
      return { path: route.path, match: { ...result, query } };
    }
  }

  return null;
}
