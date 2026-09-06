// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

const CYRILLIC = /[\u0400-\u04FF]/;
const LATIN = /[A-Za-z\u00C0-\u024F]/;
const CJK = /[\u4E00-\u9FFF\u3040-\u30FF\uAC00-\uD7AF]/;

export function detectTextLanguage(text: string): string {
  const trimmed = text.trim();
  if (!trimmed) return "unknown";

  let cyrillic = 0;
  let latin = 0;
  let cjk = 0;

  for (const char of trimmed) {
    if (CYRILLIC.test(char)) cyrillic++;
    else if (CJK.test(char)) cjk++;
    else if (LATIN.test(char)) latin++;
  }

  const total = cyrillic + latin + cjk;
  if (total === 0) return "unknown";
  if (cyrillic >= latin && cyrillic >= cjk && cyrillic > 0) return "ru";
  if (cjk >= latin && cjk >= cyrillic && cjk > 0) return "cjk";
  if (latin > 0) return "en";
  return "unknown";
}

export function detectTrackLanguage(track: {
  title?: string;
  artist?: string;
  album?: string;
}): string {
  const combined = [track.title, track.artist, track.album]
    .filter(Boolean)
    .join(" ");
  return detectTextLanguage(combined);
}

export function inferLanguageWeights(
  entries: readonly {
    title: string;
    artist: string;
    album: string;
    weight: number;
  }[],
  preferredLanguages: readonly string[],
): Map<string, number> {
  const weights = new Map<string, number>();

  for (const entry of entries) {
    const lang = detectTrackLanguage({
      title: entry.title,
      artist: entry.artist,
      album: entry.album,
    });
    weights.set(lang, (weights.get(lang) ?? 0) + entry.weight);
  }

  for (const lang of preferredLanguages) {
    const key = lang.trim().toLowerCase();
    if (!key) continue;
    weights.set(key, (weights.get(key) ?? 0) + 50);
  }

  return weights;
}

export function languageMatchScore(
  track: { title?: string; artist?: string; album?: string },
  weights: ReadonlyMap<string, number>,
): number {
  const lang = detectTrackLanguage(track);
  const direct = weights.get(lang) ?? 0;
  if (direct > 0) return direct;

  if (
    lang === "mixed" &&
    CYRILLIC.test([track.title, track.artist, track.album].join(" "))
  ) {
    return weights.get("ru") ?? 0;
  }

  if (lang === "unknown") {
    let best = 0;
    for (const weight of weights.values()) {
      if (weight > best) best = weight;
    }
    return best * 0.15;
  }

  return 0;
}

export function topLanguages(
  weights: ReadonlyMap<string, number>,
  limit = 2,
): string[] {
  return [...weights.entries()]
    .filter(([lang]) => lang !== "unknown" && lang !== "mixed")
    .sort((a, b) => b[1] - a[1])
    .slice(0, limit)
    .map(([lang]) => lang);
}
