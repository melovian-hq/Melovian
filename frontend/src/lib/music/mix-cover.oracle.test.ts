// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { pickUniqueMixCover } from "./mix-generator";
import type { SubsonicSong } from "$lib/subsonic/types";

const song = (id: string, cover: string): SubsonicSong => ({
  id,
  title: id,
  coverArt: cover,
  albumId: `album-${cover}`,
});

describe("mix cover uniqueness oracle", () => {
  it("avoids reusing covers already claimed by earlier mixes", () => {
    const used = new Set<string>();
    const first = pickUniqueMixCover(
      [song("a", "cover-a"), song("b", "cover-b")],
      "cover-a",
      used,
      "seed-1",
    );
    const second = pickUniqueMixCover(
      [song("c", "cover-a"), song("d", "cover-b")],
      "cover-a",
      used,
      "seed-2",
    );

    expect(first).toBeTruthy();
    expect(second).toBeTruthy();
    expect(first).not.toBe(second);
    expect(used.size).toBe(2);
  });

  it("falls back when every candidate is already used", () => {
    const used = new Set(["only"]);
    const cover = pickUniqueMixCover(
      [{ id: "a", title: "a", coverArt: "only", albumId: "only" }],
      "only",
      used,
      "seed-fallback",
    );
    expect(cover).toBe("only");
  });
});
