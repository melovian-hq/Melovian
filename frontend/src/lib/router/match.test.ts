// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { matchPath, matchRoute, parseQuery } from "$lib/router/match";

describe("router match", () => {
  it("matches static routes", () => {
    expect(matchPath("/music", "/music")).not.toBeNull();
    expect(matchPath("/login", "/login")).not.toBeNull();
  });

  it("matches album route params", () => {
    const match = matchPath("/music/album/:albumId", "/music/album/abc");
    expect(match?.params.albumId).toBe("abc");
  });

  it("returns null for unknown routes", () => {
    expect(matchPath("/music", "/unknown")).toBeNull();
  });

  it("decodes encoded route params", () => {
    const match = matchPath(
      "/music/genre/:genre",
      "/music/genre/Classic%20Rock",
    );
    expect(match?.params.genre).toBe("Classic Rock");
  });

  it("parses query strings", () => {
    expect(parseQuery("?q=hello&page=2")).toEqual({ q: "hello", page: "2" });
  });

  it("matches routes with query attached", () => {
    const routes = [{ path: "/music/album/:albumId" }];
    const result = matchRoute(routes, "/music/album/abc", "?from=search");
    expect(result?.path).toBe("/music/album/:albumId");
    expect(result?.match.params.albumId).toBe("abc");
    expect(result?.match.query.from).toBe("search");
  });
});
