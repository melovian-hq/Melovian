// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  bestMatches,
  diceCoefficient,
  normalizeTerm,
  suggestion,
} from "./search";

describe("normalizeTerm", () => {
  it("lowercases, strips punctuation, and collapses whitespace", () => {
    expect(normalizeTerm("  The Beatles!! ")).toBe("the beatles");
  });

  it("removes diacritics", () => {
    expect(normalizeTerm("Bjórk")).toBe("bjork");
  });
});

describe("diceCoefficient", () => {
  it("returns 1 for identical strings", () => {
    expect(diceCoefficient("radiohead", "radiohead")).toBe(1);
  });

  it("scores typos highly", () => {
    expect(diceCoefficient("radiohed", "radiohead")).toBeGreaterThan(0.7);
  });

  it("scores unrelated strings low", () => {
    expect(diceCoefficient("metallica", "beethoven")).toBeLessThan(0.3);
  });
});

describe("bestMatches", () => {
  const artists = ["Radiohead", "Metallica", "Daft Punk", "Pink Floyd"];

  it("ranks the closest candidate first", () => {
    const matches = bestMatches("radiohed", artists, (a) => a);
    expect(matches[0]?.item).toBe("Radiohead");
  });

  it("boosts substring matches", () => {
    const matches = bestMatches("daft", artists, (a) => a);
    expect(matches[0]?.item).toBe("Daft Punk");
    expect(matches[0]?.score).toBeGreaterThanOrEqual(0.85);
  });

  it("respects the limit", () => {
    const matches = bestMatches("a", artists, (a) => a, {
      limit: 2,
      threshold: 0,
    });
    expect(matches.length).toBeLessThanOrEqual(2);
  });
});

describe("suggestion", () => {
  it("suggests a close candidate for a typo", () => {
    expect(suggestion("metalica", ["Metallica", "Megadeth"])).toBe("Metallica");
  });

  it("returns null when the query already matches", () => {
    expect(suggestion("Metallica", ["Metallica"])).toBeNull();
  });

  it("returns null when nothing is close", () => {
    expect(suggestion("zzzzzz", ["Metallica", "Radiohead"])).toBeNull();
  });
});
