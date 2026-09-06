// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { createRequire } from "node:module";
import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const require = createRequire(import.meta.url);
const {
  PATHS,
  DEMO_REDIRECT_PATHS,
}: {
  PATHS: string[];
  DEMO_REDIRECT_PATHS: string[];
} = require("../../../lighthouse-urls.cjs");

const routesSource = readFileSync(
  path.join(path.dirname(fileURLToPath(import.meta.url)), "../../routes.ts"),
  "utf8",
);

const ROUTE_PATH_RE = /path:\s*"([^"]+)"/g;

function routePathsFromSource(source: string): string[] {
  const paths: string[] = [];
  for (const match of source.matchAll(ROUTE_PATH_RE)) {
    const value = match[1];
    if (value) paths.push(value);
  }
  return paths;
}

function isParamRoute(routePath: string): boolean {
  return routePath.includes(":");
}

function isDemoRedirect(routePath: string): boolean {
  if (DEMO_REDIRECT_PATHS.includes(routePath)) return true;
  if (routePath === "/settings/:tab") return true;
  return false;
}

describe("lighthouse route coverage", () => {
  it("audits every demo-reachable static route", () => {
    const staticPaths = routePathsFromSource(routesSource).filter(
      (routePath) => !isParamRoute(routePath) && !isDemoRedirect(routePath),
    );

    const missing = staticPaths.filter(
      (routePath) => !PATHS.includes(routePath),
    );
    expect(
      missing,
      `Add these static routes to frontend/lighthouse-urls.cjs:\n${missing.join("\n")}`,
    ).toEqual([]);
  });

  it("audits one URL per parameterized route kind", () => {
    const kinds = [
      { prefix: "/music/genre/", label: "genre" },
      { prefix: "/music/album/", label: "album" },
      { prefix: "/music/artist/", label: "artist" },
      { prefix: "/music/mix/", label: "mix" },
      { prefix: "/music/server-playlist/", label: "server-playlist" },
      { prefix: "/music/playlist/", label: "playlist" },
      { prefix: "/play/", label: "video player" },
      { prefix: "/share/", label: "share" },
      { prefix: "/listen/", label: "listen" },
    ];

    const missing = kinds.filter(
      (kind) => !PATHS.some((routePath) => routePath.startsWith(kind.prefix)),
    );
    expect(
      missing.map((k) => k.label),
      "Add a demo fixture URL for each parameterized route kind",
    ).toEqual([]);
  });

  it("does not audit demo redirect targets", () => {
    for (const routePath of DEMO_REDIRECT_PATHS) {
      expect(PATHS).not.toContain(routePath);
    }
  });
});
