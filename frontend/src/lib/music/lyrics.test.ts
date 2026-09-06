// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  activeLineIndex,
  lyricsMatchText,
  lyricsSnippet,
  normalizeParsedLyrics,
  parseLyricsText,
  parseStructuredLyricsEntry,
  pickPreferredStructuredEntry,
} from "./lyrics";

describe("lyrics parsing", () => {
  it("parses LRC timestamps and offset metadata", () => {
    const parsed = parseLyricsText(
      "[offset:+500]\n[00:01.00]First line\n[00:03.50]Second line",
      { artist: "Artist", title: "Song" },
    );

    expect(parsed.synced).toBe(true);
    expect(parsed.offsetMs).toBe(500);
    expect(parsed.lines).toEqual([
      { startMs: 1000, text: "First line" },
      { startMs: 3500, text: "Second line" },
    ]);
  });

  it("parses plain text as unsynced lyrics", () => {
    const parsed = parseLyricsText("Line one\nLine two");
    expect(parsed.synced).toBe(false);
    expect(parsed.lines).toEqual([{ text: "Line one" }, { text: "Line two" }]);
  });

  it("parses structured lyrics entries", () => {
    const parsed = parseStructuredLyricsEntry(
      {
        synced: true,
        offset: 250,
        displayArtist: "Band",
        displayTitle: "Track",
        line: [
          { start: 0, value: "Intro" },
          { start: 4200, value: "Verse" },
        ],
      },
      { artist: "Fallback", title: "Fallback title" },
    );

    expect(parsed?.synced).toBe(true);
    expect(parsed?.offsetMs).toBe(250);
    expect(parsed?.artist).toBe("Band");
    expect(parsed?.lines[1]?.startMs).toBe(4200);
  });

  it("prefers synced main over synced translation", () => {
    const preferred = pickPreferredStructuredEntry([
      { kind: "translation", synced: true, line: [{ value: "Traduction" }] },
      { kind: "main", synced: true, line: [{ value: "Main line" }] },
    ]);
    expect(preferred?.kind).toBe("main");
  });

  it("prefers synced translation over unsynced main", () => {
    const preferred = pickPreferredStructuredEntry([
      { kind: "main", synced: false, line: [{ value: "Plain" }] },
      {
        kind: "translation",
        synced: true,
        line: [{ start: 1000, value: "Timed" }],
      },
    ]);
    expect(preferred?.kind).toBe("translation");
  });
});

describe("lyrics sync helpers", () => {
  const lines = [
    { startMs: 0, text: "One" },
    { startMs: 2000, text: "Two" },
    { startMs: 5000, text: "Three" },
  ];

  it("finds the active synced line for the current time", () => {
    expect(activeLineIndex(lines, 0, 0, true)).toBe(0);
    expect(activeLineIndex(lines, 2100, 0, true)).toBe(1);
    expect(activeLineIndex(lines, 4999, 0, true)).toBe(1);
    expect(activeLineIndex(lines, 5000, 0, true)).toBe(2);
  });

  it("applies lyric offsets when selecting the active line", () => {
    expect(activeLineIndex(lines, 1500, 500, true)).toBe(1);
  });

  it("does not highlight unsynced lyrics", () => {
    expect(activeLineIndex([{ text: "Plain" }], 1000, 0, false)).toBe(-1);
  });
});

describe("lyrics search helpers", () => {
  it("matches lyric text case-insensitively", () => {
    expect(lyricsMatchText("Hello World", "world")).toBe(true);
    expect(lyricsMatchText("Hello World", "missing")).toBe(false);
  });

  it("builds a readable snippet around the matched phrase", () => {
    const snippet = lyricsSnippet(
      "First line with a memorable phrase in the middle and more text after it",
      "memorable phrase",
      12,
    );
    expect(snippet).toContain("memorable phrase");
    expect(snippet.startsWith("…") || snippet.startsWith("First")).toBe(true);
  });
});

describe("normalizeParsedLyrics", () => {
  it("rejects malformed payloads without lines", () => {
    expect(
      normalizeParsedLyrics({ synced: true, offsetMs: 0, rawValue: "" }),
    ).toBeNull();
  });

  it("normalizes partial API payloads", () => {
    const parsed = normalizeParsedLyrics({
      synced: true,
      offsetMs: 0,
      rawValue: "hello",
      lines: [{ text: "hello", startMs: 1000 }],
    });
    expect(parsed?.lines).toHaveLength(1);
    expect(parsed?.lines[0]?.startMs).toBe(1000);
  });

  it("re-parses raw LRC text when structured lines are unsynced", () => {
    const parsed = normalizeParsedLyrics({
      synced: false,
      offsetMs: 0,
      rawValue: "[00:01.00]First line\n[00:03.00]Second line",
      lines: [{ text: "First line" }, { text: "Second line" }],
    });

    expect(parsed?.synced).toBe(true);
    expect(parsed?.lines[1]?.startMs).toBe(3000);
  });
});
