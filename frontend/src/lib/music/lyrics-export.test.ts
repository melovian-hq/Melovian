// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import { lyricsToLrc, lyricsToPlainText } from "./lyrics-export";
import type { ParsedLyrics } from "./lyrics";

const syncedLyrics: ParsedLyrics = {
  artist: "Artist",
  title: "Song",
  synced: true,
  offsetMs: 0,
  lines: [
    { text: "First line", startMs: 1000 },
    { text: "Second line", startMs: 4500 },
  ],
  rawValue: "",
};

describe("lyrics-export", () => {
  it("serializes synced lyrics to LRC", () => {
    const lrc = lyricsToLrc(syncedLyrics);
    expect(lrc).toContain("[ti:Song]");
    expect(lrc).toContain("[ar:Artist]");
    expect(lrc).toContain("[0:01.00]First line");
    expect(lrc).toContain("[0:04.50]Second line");
  });

  it("serializes unsynced lyrics to plain text", () => {
    const plain = lyricsToPlainText({
      ...syncedLyrics,
      synced: false,
      lines: [{ text: "Line one" }, { text: "Line two" }],
    });
    expect(plain).toBe("Line one\nLine two");
  });
});
