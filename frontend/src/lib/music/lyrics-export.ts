// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { ParsedLyrics } from "./lyrics";
import { downloadTextFile, sanitizeFilename } from "$lib/utils/download";

function formatLrcTimestamp(ms: number): string {
  const total = Math.max(0, Math.round(ms));
  const minutes = Math.floor(total / 60_000);
  const seconds = Math.floor((total % 60_000) / 1000);
  const centiseconds = Math.floor((total % 1000) / 10);
  return `${minutes}:${seconds.toString().padStart(2, "0")}.${centiseconds.toString().padStart(2, "0")}`;
}

export function lyricsToPlainText(lyrics: ParsedLyrics): string {
  if (lyrics.lines.length > 0) {
    return lyrics.lines.map((line) => line.text).join("\n");
  }
  return lyrics.rawValue.trim();
}

export function lyricsToLrc(lyrics: ParsedLyrics): string {
  if (
    lyrics.synced &&
    lyrics.rawValue.trim() &&
    /^\[\d+:\d{2}/m.test(lyrics.rawValue)
  ) {
    return lyrics.rawValue.trim();
  }

  const lines: string[] = [];
  if (lyrics.offsetMs) {
    lines.push(`[offset:${lyrics.offsetMs}]`);
  }
  if (lyrics.title) lines.push(`[ti:${lyrics.title}]`);
  if (lyrics.artist) lines.push(`[ar:${lyrics.artist}]`);

  for (const line of lyrics.lines) {
    if (line.startMs !== undefined) {
      lines.push(`[${formatLrcTimestamp(line.startMs)}]${line.text}`);
    } else {
      lines.push(line.text);
    }
  }

  return lines.join("\n").trim();
}

export function lyricsExportFilename(
  lyrics: ParsedLyrics,
  fallbackTitle?: string,
): string {
  const title = lyrics.title?.trim() || fallbackTitle?.trim() || "lyrics";
  const artist = lyrics.artist?.trim();
  const base = artist ? `${artist} - ${title}` : title;
  const ext = lyrics.synced ? ".lrc" : ".txt";
  return `${sanitizeFilename(base, "lyrics")}${ext}`;
}

export function exportLyrics(
  lyrics: ParsedLyrics,
  fallbackTitle?: string,
): void {
  const filename = lyricsExportFilename(lyrics, fallbackTitle);
  if (lyrics.synced) {
    downloadTextFile(filename, lyricsToLrc(lyrics), "text/plain;charset=utf-8");
    return;
  }
  downloadTextFile(filename, lyricsToPlainText(lyrics));
}
