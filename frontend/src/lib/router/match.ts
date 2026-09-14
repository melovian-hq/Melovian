// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export interface RouteMatch {
  params: Record<string, string>;
  query: Record<string, string>;
}

// decodeURIComponent throws on malformed escapes like "%zz". A bad URL should
// fall back to the raw segment, not crash the match during render.
function decodeSegment(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

// A trailing ":name*" segment is a splat. It consumes every remaining path
// segment (including none) and joins them into one param. Only the final
// pattern segment may be a splat.
export function matchPath(
  pattern: string,
  pathname: string,
): RouteMatch | null {
  const patternParts = pattern.split("/").filter(Boolean);
  const pathParts = pathname.split("/").filter(Boolean);

  const last = patternParts[patternParts.length - 1] ?? "";
  const splatName =
    last.startsWith(":") && last.endsWith("*")
      ? last.slice(1, -1) || "splat"
      : null;

  if (splatName !== null) {
    if (pathParts.length < patternParts.length - 1) return null;
  } else if (patternParts.length !== pathParts.length) {
    return null;
  }

  const params: Record<string, string> = {};

  for (let i = 0; i < patternParts.length; i++) {
    const part = patternParts[i];
    const value = pathParts[i];

    if (splatName !== null && i === patternParts.length - 1) {
      params[splatName] = pathParts.slice(i).map(decodeSegment).join("/");
      break;
    }

    if (part.startsWith(":")) {
      if (value === undefined) return null;
      params[part.slice(1)] = decodeSegment(value);
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
