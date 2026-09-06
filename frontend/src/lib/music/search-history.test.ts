// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it } from "vitest";
import {
  addSearchHistory,
  loadSearchHistory,
  saveSearchHistory,
} from "./search-history";

describe("search-history", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("stores recent searches with newest first", () => {
    let history = addSearchHistory("atl", []);
    history = addSearchHistory("phoebe bridgers", history);
    expect(history).toEqual(["phoebe bridgers", "atl"]);
  });

  it("deduplicates case-insensitively and caps at six items", () => {
    let history: string[] = [];
    for (const term of [
      "one",
      "two",
      "three",
      "four",
      "five",
      "six",
      "seven",
      "ONE",
    ]) {
      history = addSearchHistory(term, history);
    }
    expect(history).toHaveLength(6);
    expect(history[0]).toBe("ONE");
    expect(history).not.toContain("one");
  });

  it("persists to localStorage", () => {
    saveSearchHistory(["atl", "punisher"]);
    expect(loadSearchHistory()).toEqual(["atl", "punisher"]);
  });
});
