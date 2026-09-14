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

  it("matches a catch-all splat at any depth", () => {
    const match = matchPath("/:splat*", "/definitely/not/a/page");
    expect(match?.params.splat).toBe("definitely/not/a/page");
  });

  it("matches a splat with zero trailing segments", () => {
    const match = matchPath("/:splat*", "/");
    expect(match).not.toBeNull();
    expect(match?.params.splat).toBe("");
  });

  it("matches a splat after a static prefix", () => {
    expect(matchPath("/music/:rest*", "/music/a/b")?.params.rest).toBe("a/b");
    expect(matchPath("/music/:rest*", "/music")?.params.rest).toBe("");
    expect(matchPath("/music/:rest*", "/other/deep")).toBeNull();
  });

  it("decodes splat segments and tolerates malformed escapes", () => {
    expect(matchPath("/:splat*", "/Classic%20Rock/jazz")?.params.splat).toBe(
      "Classic Rock/jazz",
    );
    expect(matchPath("/:splat*", "/bad%zz")?.params.splat).toBe("bad%zz");
    expect(matchPath("/music/:id", "/music/bad%zz")?.params.id).toBe("bad%zz");
  });

  it("matchRoute falls through to a trailing catch-all", () => {
    const routes = [{ path: "/music" }, { path: "/:splat*" }];
    expect(matchRoute(routes, "/music")?.path).toBe("/music");
    expect(matchRoute(routes, "/nope/deep")?.path).toBe("/:splat*");
  });
});
