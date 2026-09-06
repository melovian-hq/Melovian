// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  getRouteComponentProps,
  requiredRouteParamKeys,
  routesMissingParamMappings,
  routesWithIncompleteProps,
} from "./route-props";

const PARAM_ROUTES = [
  "/library/:libraryId",
  "/item/:itemId",
  "/person/:personId",
  "/play/:itemId",
  "/music/album/:albumId",
  "/music/artist/:artistId",
  "/music/mix/:mixId",
  "/music/playlist/:playlistId",
  "/music/genre/:genre",
  "/share/:token",
] as const;

describe("route component props", () => {
  it("maps mixId for mix pages", () => {
    const props = getRouteComponentProps("/music/mix/:mixId", {
      mixId: "discover",
    });
    expect(props.mixId).toBe("discover");
  });

  it("maps album, artist, and playlist ids", () => {
    expect(
      getRouteComponentProps("/music/album/:albumId", { albumId: "al-1" })
        .albumId,
    ).toBe("al-1");
    expect(
      getRouteComponentProps("/music/artist/:artistId", { artistId: "ar-1" })
        .artistId,
    ).toBe("ar-1");
    expect(
      getRouteComponentProps("/music/playlist/:playlistId", {
        playlistId: "pl-1",
      }).playlistId,
    ).toBe("pl-1");
  });

  it("maps share token for public share pages", () => {
    expect(
      getRouteComponentProps("/share/:token", { token: "abc123" }).token,
    ).toBe("abc123");
  });

  it("maps item query props", () => {
    const props = getRouteComponentProps(
      "/item/:itemId",
      { itemId: "movie-1" },
      { season: "3" },
    );
    expect(props.itemId).toBe("movie-1");
    expect(props.seasonQuery).toBe("3");
  });

  it("flags missing required params", () => {
    expect(
      routesWithIncompleteProps("/music/mix/:mixId", { mixId: "" }),
    ).toContain("mixId");
    expect(
      routesWithIncompleteProps("/music/mix/:mixId", { mixId: "replay" }),
    ).toEqual([]);
  });
});

describe("route registry coverage", () => {
  it("every parameterized app route has a prop mapping", () => {
    expect(routesMissingParamMappings([...PARAM_ROUTES])).toEqual([]);
  });

  it("documents required keys for each parameterized route", () => {
    for (const path of PARAM_ROUTES) {
      expect(requiredRouteParamKeys(path).length).toBeGreaterThan(0);
    }
  });
});
