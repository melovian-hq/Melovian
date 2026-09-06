// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export interface LyricLine {
  text: string;
  startMs?: number;
}

export interface ParsedLyrics {
  artist?: string;
  title?: string;
  synced: boolean;
  offsetMs: number;
  lines: LyricLine[];
  rawValue: string;
}

export interface LyricsSearchHit {
  songId: string;
  title: string;
  artist?: string;
  album?: string;
  duration?: number;
  coverArt?: string;
  lyrics: ParsedLyrics;
  snippet: string;
}

const LRC_LINE = /^\[(\d+):(\d{2})(?:[.:](\d{1,3}))?\]\s*(.*)$/;
const LRC_OFFSET = /^\[offset:\s*([+-]?\d+)\s*\]$/i;
const LRC_META =
  /^\[(?:ti|ar|al|by|au|length|re|ve|la|tool|key|bpm|sign|language|lang):/i;

function parseTimestamp(
  minutes: string,
  seconds: string,
  fraction?: string,
): number {
  const mins = Number(minutes);
  const secs = Number(seconds);
  let ms = mins * 60_000 + secs * 1000;
  if (fraction) {
    const digits = fraction.padEnd(3, "0").slice(0, 3);
    ms += Number(digits);
  }
  return ms;
}

export function parseLyricsText(
  value: string,
  meta?: { artist?: string; title?: string },
): ParsedLyrics {
  const rawValue = value.replace(/\r\n/g, "\n");
  const lines: LyricLine[] = [];
  let offsetMs = 0;
  let sawTimestamp = false;

  for (const rawLine of rawValue.split("\n")) {
    const line = rawLine.trim();
    if (!line) continue;

    const offsetMatch = line.match(LRC_OFFSET);
    if (offsetMatch) {
      offsetMs = Number(offsetMatch[1]);
      continue;
    }
    if (LRC_META.test(line)) continue;

    const lrcMatch = line.match(LRC_LINE);
    if (lrcMatch) {
      sawTimestamp = true;
      lines.push({
        startMs: parseTimestamp(lrcMatch[1], lrcMatch[2], lrcMatch[3]),
        text: lrcMatch[4].trim(),
      });
      continue;
    }

    lines.push({ text: line });
  }

  const filtered = lines.filter((entry) => entry.text.length > 0);
  return {
    artist: meta?.artist,
    title: meta?.title,
    synced:
      sawTimestamp && filtered.some((entry) => entry.startMs !== undefined),
    offsetMs,
    lines: filtered,
    rawValue,
  };
}

export function parseStructuredLyricsEntry(
  raw: Record<string, unknown>,
  meta?: { artist?: string; title?: string },
): ParsedLyrics | null {
  const lineEntries = unwrapStructuredLines(raw.line);
  if (lineEntries.length === 0) return null;

  const lines: LyricLine[] = lineEntries.map((entry) => ({
    text: String(entry.value ?? "").trim(),
    startMs: typeof entry.start === "number" ? entry.start : undefined,
  }));

  const filtered = lines.filter((entry) => entry.text.length > 0);
  if (filtered.length === 0) return null;

  const synced =
    raw.synced === true ||
    filtered.some((entry) => entry.startMs !== undefined);

  return {
    artist: (raw.displayArtist as string | undefined) ?? meta?.artist,
    title: (raw.displayTitle as string | undefined) ?? meta?.title,
    synced,
    offsetMs: typeof raw.offset === "number" ? raw.offset : 0,
    lines: filtered,
    rawValue: filtered.map((entry) => entry.text).join("\n"),
  };
}

function unwrapStructuredLines(value: unknown): Record<string, unknown>[] {
  if (!value) return [];
  if (Array.isArray(value)) {
    return value.filter(
      (entry): entry is Record<string, unknown> =>
        !!entry && typeof entry === "object",
    );
  }
  if (typeof value === "object") {
    return [value as Record<string, unknown>];
  }
  return [];
}

export function pickPreferredStructuredEntry(
  entries: Record<string, unknown>[],
): Record<string, unknown> | null {
  if (entries.length === 0) return null;

  let mainEntry: Record<string, unknown> | null = null;
  let syncedMain: Record<string, unknown> | null = null;
  let syncedAny: Record<string, unknown> | null = null;

  for (const entry of entries) {
    const kind = entry.kind;
    const isMain = kind === "main" || !kind;
    const synced = structuredEntryLooksSynced(entry);
    if (isMain && !mainEntry) mainEntry = entry;
    if (synced) {
      if (isMain && !syncedMain) syncedMain = entry;
      if (!syncedAny) syncedAny = entry;
    }
  }

  return syncedMain ?? syncedAny ?? mainEntry ?? entries[0];
}

function structuredEntryLooksSynced(entry: Record<string, unknown>): boolean {
  if (entry.synced === true) return true;
  const lines = unwrapStructuredLines(entry.line);
  return lines.some((line) => typeof line.start === "number");
}

export function activeLineIndex(
  lines: LyricLine[],
  timeMs: number,
  offsetMs = 0,
  synced = false,
): number {
  if (!synced || lines.length === 0) return -1;

  const adjusted = timeMs + offsetMs;
  let active = -1;
  for (let i = 0; i < lines.length; i++) {
    const start = lines[i].startMs;
    if (start === undefined) continue;
    if (start <= adjusted) active = i;
    else break;
  }
  return active;
}

export function lyricsMatchText(text: string, query: string): boolean {
  const needle = query.trim().toLowerCase();
  if (!needle) return false;
  return text.toLowerCase().includes(needle);
}

/**
 * Filters a parsed lyrics object down to the lines matching query.
 * Returns the original lyrics unchanged when query is blank.
 */
export function filterLyricsLines(
  lyrics: ParsedLyrics,
  query: string,
): ParsedLyrics {
  if (!query.trim()) return lyrics;
  return {
    ...lyrics,
    lines: lyrics.lines.filter((line) => lyricsMatchText(line.text, query)),
  };
}

export function lyricsSnippet(
  text: string,
  query: string,
  radius = 48,
): string {
  const source = text.replace(/\s+/g, " ").trim();
  const lower = source.toLowerCase();
  const needle = query.trim().toLowerCase();
  const index = lower.indexOf(needle);
  if (index < 0) return source.slice(0, radius * 2);

  const start = Math.max(0, index - radius);
  const end = Math.min(source.length, index + needle.length + radius);
  const prefix = start > 0 ? "…" : "";
  const suffix = end < source.length ? "…" : "";
  return `${prefix}${source.slice(start, end)}${suffix}`;
}

export async function mapWithConcurrency<T, R>(
  items: T[],
  limit: number,
  worker: (item: T) => Promise<R | null>,
): Promise<R[]> {
  const results: R[] = [];
  let index = 0;

  async function runWorker() {
    while (index < items.length) {
      const current = index;
      index += 1;
      const value = await worker(items[current]!);
      if (value !== null) results.push(value);
    }
  }

  const workers = Array.from({ length: Math.min(limit, items.length) }, () =>
    runWorker(),
  );
  await Promise.all(workers);
  return results;
}

export function normalizeParsedLyrics(
  value: Partial<ParsedLyrics> | null | undefined,
): ParsedLyrics | null {
  if (!value || typeof value !== "object") return null;

  const rawLines = Array.isArray(value.lines) ? value.lines : [];
  const lines: LyricLine[] = [];
  for (const entry of rawLines) {
    if (!entry || typeof entry.text !== "string") continue;
    const text = entry.text.trim();
    if (!text) continue;
    lines.push({
      text,
      startMs:
        typeof entry.startMs === "number" && Number.isFinite(entry.startMs)
          ? entry.startMs
          : undefined,
    });
  }

  if (lines.length === 0) {
    const rawValue = typeof value.rawValue === "string" ? value.rawValue : "";
    if (!rawValue.trim()) return null;
    return normalizeParsedLyrics(
      parseLyricsText(rawValue, {
        artist: value.artist,
        title: value.title,
      }),
    );
  }

  const rawValue =
    typeof value.rawValue === "string" && value.rawValue.trim()
      ? value.rawValue
      : lines.map((line) => line.text).join("\n");

  const hasSyncedLines = lines.some((line) => line.startMs !== undefined);
  if (!hasSyncedLines && rawValue.trim()) {
    const reparsed = parseLyricsText(rawValue, {
      artist: value.artist,
      title: value.title,
    });
    if (reparsed.synced) {
      return reparsed;
    }
  }

  return {
    artist: value.artist,
    title: value.title,
    synced:
      value.synced === true || lines.some((line) => line.startMs !== undefined),
    offsetMs:
      typeof value.offsetMs === "number" && Number.isFinite(value.offsetMs)
        ? value.offsetMs
        : 0,
    lines,
    rawValue,
  };
}
