// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import fc from "fast-check";
import {
  matchesLocalSearch,
  normalizeSearchQuery,
} from "../utils/local-search";
import { matchPath, matchRoute, parseQuery } from "../router/match";
import {
  contrastRatio,
  meetsWcagAaNormalText,
  parseHexColor,
  relativeLuminance,
} from "../theme/contrast";
import { formatRelativePlayedAt } from "../music/relative-time";

describe("local-search exploratory", () => {
  it("explores normalize never throws and is idempotent", () => {
    fc.assert(
      fc.property(fc.string({ maxLength: 60 }), (query) => {
        const once = normalizeSearchQuery(query);
        expect(normalizeSearchQuery(once)).toBe(once);
      }),
      { numRuns: 100 },
    );
  });

  it("explores empty query matches everything", () => {
    fc.assert(
      fc.property(
        fc.array(fc.string({ maxLength: 20 }), { maxLength: 5 }),
        (fields) => {
          expect(matchesLocalSearch("", ...fields)).toBe(true);
          expect(matchesLocalSearch("   ", ...fields)).toBe(true);
        },
      ),
    );
  });
});

describe("router oracle", () => {
  it("matchPath extracts params for valid patterns", () => {
    fc.assert(
      fc.property(
        fc.stringMatching(/^[a-z0-9-]{1,12}$/),
        fc.stringMatching(/^[a-z0-9-]{1,12}$/),
        (artistId, albumId) => {
          const match = matchPath(
            "/artist/:artistId/album/:albumId",
            `/artist/${artistId}/album/${albumId}`,
          );
          expect(match).toEqual({
            params: { artistId, albumId },
            query: {},
          });
        },
      ),
    );
  });

  it("matchRoute prefers first matching route and attaches query", () => {
    const routes = [{ path: "/a/:id" }, { path: "/b/:id" }];
    expect(matchRoute(routes, "/b/42", "?tab=tracks")).toEqual({
      path: "/b/:id",
      match: { params: { id: "42" }, query: { tab: "tracks" } },
    });
    expect(parseQuery("?x=1&y=two")).toEqual({ x: "1", y: "two" });
  });
});

describe("contrast oracle", () => {
  it("theme text colors meet WCAG AA against surfaces", () => {
    const darkText = parseHexColor("#18181b");
    const lightSurface = parseHexColor("#ffffff");
    const lightText = parseHexColor("#f5f5f5");
    const darkSurface = parseHexColor("#141414");
    expect(darkText && lightSurface).toBeTruthy();
    expect(lightText && darkSurface).toBeTruthy();
    expect(meetsWcagAaNormalText(darkText!, lightSurface!)).toBe(true);
    expect(meetsWcagAaNormalText(lightText!, darkSurface!)).toBe(true);
    expect(contrastRatio(darkText!, lightSurface!)).toBeGreaterThanOrEqual(4.5);
    expect(relativeLuminance(lightSurface!)).toBeGreaterThan(
      relativeLuminance(darkText!),
    );
  });
});

describe("relative-time exploratory", () => {
  it("explores invalid timestamps return empty string", () => {
    fc.assert(
      fc.property(fc.string({ maxLength: 20 }), (value) => {
        fc.pre(Number.isNaN(Date.parse(value)));
        expect(formatRelativePlayedAt(value, Date.now())).toBe("");
      }),
      { numRuns: 40 },
    );
  });

  it("explores future timestamps clamp to just now", () => {
    const now = Date.UTC(2026, 6, 18, 12, 0, 0);
    expect(
      formatRelativePlayedAt(new Date(now + 60_000).toISOString(), now),
    ).toBe("Just now");
  });
});
