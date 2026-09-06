// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  detectTextLanguage,
  detectTrackLanguage,
  inferLanguageWeights,
  languageMatchScore,
  topLanguages,
} from "./mix-language";

describe("mix-language", () => {
  it("detects cyrillic text as russian", () => {
    expect(detectTextLanguage("Монеточка - Каждый раз")).toBe("ru");
    expect(detectTrackLanguage({ title: "Звезда", artist: "Би-2" })).toBe("ru");
  });

  it("detects latin text as english", () => {
    expect(detectTextLanguage("Radiohead - Creep")).toBe("en");
  });

  it("infers language weights from listening history", () => {
    const weights = inferLanguageWeights(
      [
        { title: "Песня", artist: "Artist", album: "Album", weight: 10 },
        { title: "Song", artist: "Band", album: "Record", weight: 2 },
      ],
      ["ru"],
    );
    expect(weights.get("ru")).toBeGreaterThan(weights.get("en") ?? 0);
  });

  it("scores matching tracks higher", () => {
    const weights = inferLanguageWeights(
      [{ title: "Песня", artist: "Исполнитель", album: "Альбом", weight: 5 }],
      ["ru"],
    );
    expect(
      languageMatchScore({ title: "Другая песня", artist: "МакSim" }, weights),
    ).toBeGreaterThan(
      languageMatchScore({ title: "Hello", artist: "World" }, weights),
    );
  });

  it("returns top languages by weight", () => {
    const weights = inferLanguageWeights(
      [
        { title: "Песня", artist: "Артист", album: "Альбом", weight: 20 },
        { title: "Song", artist: "Band", album: "Record", weight: 3 },
      ],
      [],
    );
    expect(topLanguages(weights, 1)).toEqual(["ru"]);
  });
});
