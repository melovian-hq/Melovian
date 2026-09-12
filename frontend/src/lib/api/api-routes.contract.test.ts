// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { readFileSync, readdirSync, statSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { ApiPaths } from "$lib/core/http/api-paths";

type ApiRoute = {
  method: string;
  pattern: string;
  prefix?: boolean;
};

type ApiRouteManifest = {
  routes: ApiRoute[];
};

const repoRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "../../../../",
);
const manifestPath = path.join(
  repoRoot,
  "internal/api/testdata/api-routes.json",
);
const frontendSrc = path.join(repoRoot, "frontend/src");

const manifest = JSON.parse(
  readFileSync(manifestPath, "utf8"),
) as ApiRouteManifest;

const apiPathRE = /["'`](\/api\/[^"'`?]+)(?:\?[^"'`]*)?["'`]|`(\/api\/[^`?]+)/g;
const templateApiPathRE = /`(\/api\/[^`]+)`/g;

function normalizeFrontendPath(raw: string): string {
  let pathValue = raw;
  pathValue = pathValue.replace(/\$\{query[^}]*\}/gi, "");
  pathValue = pathValue.replace(/\$\{encodeURIComponent\([^)]+\)\}/g, "{id}");
  pathValue = pathValue.replace(/\$\{[a-zA-Z]+\([^)]*\)\}/g, "");
  pathValue = pathValue.replace(/\$\{[^}]+\}/g, "{id}");
  pathValue = pathValue.split("?")[0] ?? pathValue;
  pathValue = pathValue.replace(/\{id\}\{id\}/g, "{id}");
  return pathValue;
}

function isStableApiPath(normalized: string): boolean {
  return (
    normalized.startsWith("/api/") &&
    !normalized.includes("${") &&
    !normalized.includes("{id}{")
  );
}

function extractApiPaths(content: string): string[] {
  const paths: string[] = [];
  for (const match of content.matchAll(apiPathRE)) {
    const raw = match[1] ?? match[2];
    if (raw) {
      paths.push(raw);
    }
  }
  for (const match of content.matchAll(templateApiPathRE)) {
    const raw = match[1];
    if (raw) {
      paths.push(raw);
    }
  }
  return paths;
}

function normalizePattern(pattern: string): string {
  return pattern.replace(/\{[^}]+\}/g, "{id}");
}

function routeMatches(frontendPath: string, route: ApiRoute): boolean {
  const pattern = normalizePattern(route.pattern);
  const normalized = normalizePattern(frontendPath);

  if (route.prefix) {
    const base = route.pattern.replace(/\/$/, "");
    return normalized === base || normalized.startsWith(`${base}/`);
  }

  // Prefix probes like "/api/party/" match any concrete route under that tree.
  if (frontendPath.endsWith("/") && frontendPath.length > 1) {
    return (
      pattern === frontendPath.slice(0, -1) ||
      pattern.startsWith(frontendPath) ||
      pattern.startsWith(`${frontendPath.slice(0, -1)}/`)
    );
  }

  if (pattern === normalized) {
    return true;
  }

  const patternParts = pattern.split("/");
  const pathParts = normalized.split("/");
  if (patternParts.length !== pathParts.length) {
    return false;
  }
  return patternParts.every(
    (part, index) => part === "{id}" || part === pathParts[index],
  );
}

function collectSourceFiles(dir: string): string[] {
  const files: string[] = [];
  for (const entry of readdirSync(dir)) {
    const fullPath = path.join(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      files.push(...collectSourceFiles(fullPath));
      continue;
    }
    if (
      /\.(ts|svelte)$/.test(entry) &&
      !/\.(test|contract|property|leak)\./.test(entry)
    ) {
      files.push(fullPath);
    }
  }
  return files;
}

function resolveApiPathValue(value: unknown): string {
  if (typeof value === "function") {
    return (value as (...args: string[]) => string)("id", "id");
  }
  return String(value);
}

function collectFrontendApiPaths(): Map<string, Set<string>> {
  const usages = new Map<string, Set<string>>();
  for (const file of collectSourceFiles(frontendSrc)) {
    const content = readFileSync(file, "utf8");
    for (const raw of extractApiPaths(content)) {
      const normalized = normalizeFrontendPath(raw);
      if (!isStableApiPath(normalized)) {
        continue;
      }
      const rel = path.relative(repoRoot, file);
      const files = usages.get(normalized) ?? new Set<string>();
      files.add(rel);
      usages.set(normalized, files);
    }
  }
  return usages;
}

describe("frontend api route contract", () => {
  it("uses only routes declared in the backend manifest", () => {
    const frontendPaths = collectFrontendApiPaths();
    const unknown: string[] = [];

    for (const [frontendPath, files] of frontendPaths) {
      const matched = manifest.routes.some((route) =>
        routeMatches(frontendPath, route),
      );
      if (!matched) {
        unknown.push(`${frontendPath} (${[...files].join(", ")})`);
      }
    }

    expect(
      unknown,
      `frontend references API paths missing from internal/api/testdata/api-routes.json:\n${unknown.join("\n")}`,
    ).toEqual([]);
  });

  it("covers critical frontend API paths in the manifest", () => {
    const required = [
      "/api/config",
      "/api/auth/status",
      "/api/instances",
      "/api/music/status",
      "/api/music/history",
      "/api/downloads",
      "/api/local-music/search",
    ];
    const frontendPaths = new Set(collectFrontendApiPaths().keys());
    const missing = required.filter(
      (route) =>
        ![...frontendPaths].some((pathValue) =>
          routeMatches(pathValue, {
            method: "GET",
            pattern: route,
          }),
        ),
    );
    expect(missing).toEqual([]);
  });

  it("declares every ApiPaths entry in the backend manifest", () => {
    const missing: string[] = [];

    for (const [name, value] of Object.entries(ApiPaths)) {
      const resolved = resolveApiPathValue(value);
      const matched = manifest.routes.some((route) =>
        routeMatches(resolved, route),
      );
      if (!matched) {
        missing.push(`${name} -> ${resolved}`);
      }
    }

    expect(
      missing,
      `ApiPaths entries missing from internal/api/testdata/api-routes.json:\n${missing.join("\n")}`,
    ).toEqual([]);
  });
});
