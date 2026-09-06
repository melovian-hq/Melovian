// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export type LanguageBiasMode = "off" | "prefer" | "strict";
export type GenreSelectionMode = "personal" | "library" | "blend";

export const ALL_MIX_IDS = [
  "on-repeat",
  "replay",
  "for-you-50",
  "for-you-100",
  "daily-mix-1",
  "daily-mix-2",
  "daily-mix-3",
  "genre-mix-1",
  "genre-mix-2",
  "genre-mix-3",
  "discover",
  "deep-cuts",
  "release-radar",
  "throwback",
  "year-rewind",
  "popular",
  "your-artists",
  "overlooked",
  "favorites-mix",
  "decade-mix-1",
  "decade-mix-2",
  "decade-mix-3",
  "fresh-finds",
] as const;

export type MixId = (typeof ALL_MIX_IDS)[number];

export const MIX_TYPE_LABELS: Record<MixId, string> = {
  "on-repeat": "On Repeat",
  replay: "Replay Mix",
  "for-you-50": "For You Mix (50)",
  "for-you-100": "For You Mix (100)",
  "daily-mix-1": "Daily Mix 1",
  "daily-mix-2": "Daily Mix 2",
  "daily-mix-3": "Daily Mix 3",
  "genre-mix-1": "Genre Mix 1",
  "genre-mix-2": "Genre Mix 2",
  "genre-mix-3": "Genre Mix 3",
  discover: "Discover Weekly",
  "deep-cuts": "Deep Cuts",
  "release-radar": "Release Radar",
  throwback: "Throwback Mix",
  "year-rewind": "Year in Rewind",
  popular: "Trending Mix",
  "your-artists": "Your Artists Mix",
  overlooked: "Overlooked",
  "favorites-mix": "Favorites Mix",
  "decade-mix-1": "Decade Mix 1",
  "decade-mix-2": "Decade Mix 2",
  "decade-mix-3": "Decade Mix 3",
  "fresh-finds": "Fresh Finds",
};

export const LANGUAGE_BIAS_LABELS: Record<LanguageBiasMode, string> = {
  off: "Off",
  prefer: "Prefer my languages",
  strict: "Only my languages",
};

export const GENRE_SELECTION_LABELS: Record<GenreSelectionMode, string> = {
  personal: "From my listening",
  library: "From library size",
  blend: "Blend both",
};

const STORAGE_KEY = "mel-mix-settings";

export interface MixSettings {
  maxTracksPerMix: number;
  minTracksPerMix: number;
  discoverRecentDays: number;
  throwbackDays: number;
  deepCutMaxPlayCount: number;
  crossMixDedup: boolean;
  durationPacing: boolean;
  languageBias: LanguageBiasMode;
  preferredLanguages: string[];
  genreSelection: GenreSelectionMode;
  enabledMixIds: MixId[];
  mixSeedSuffix: string;
  personalRadioRecencyHours: number;
  personalRadioColdStartPlays: number;
  radioExploreBonus: number;
  flowAlbumLookback: number;
}

export function defaultMixSettings(): MixSettings {
  return {
    maxTracksPerMix: 50,
    minTracksPerMix: 30,
    discoverRecentDays: 21,
    throwbackDays: 30,
    deepCutMaxPlayCount: 2,
    crossMixDedup: true,
    durationPacing: true,
    languageBias: "prefer",
    preferredLanguages: [],
    genreSelection: "personal",
    enabledMixIds: [],
    mixSeedSuffix: "0",
    personalRadioRecencyHours: 2,
    personalRadioColdStartPlays: 5,
    radioExploreBonus: 0.6,
    flowAlbumLookback: 4,
  };
}

const VALID_BIAS = new Set<LanguageBiasMode>(["off", "prefer", "strict"]);
const VALID_GENRE = new Set<GenreSelectionMode>([
  "personal",
  "library",
  "blend",
]);
const VALID_MIX_IDS = new Set<string>(ALL_MIX_IDS);

function expandLegacyMixId(id: string): MixId[] {
  if (id === "genre-mix") {
    return ["genre-mix-1", "genre-mix-2", "genre-mix-3"];
  }
  if (id === "decade-mix") {
    return ["decade-mix-1", "decade-mix-2", "decade-mix-3"];
  }
  if (VALID_MIX_IDS.has(id)) return [id as MixId];
  return [];
}

function clampInt(
  value: unknown,
  min: number,
  max: number,
  fallback: number,
): number {
  if (typeof value !== "number" || !Number.isFinite(value)) return fallback;
  return Math.max(min, Math.min(max, Math.round(value)));
}

export function mergeMixSettings(
  partial: Partial<MixSettings> | null | undefined,
): MixSettings {
  const defaults = defaultMixSettings();
  if (!partial || typeof partial !== "object") return defaults;

  const languageBias = VALID_BIAS.has(partial.languageBias as LanguageBiasMode)
    ? (partial.languageBias as LanguageBiasMode)
    : defaults.languageBias;

  const genreSelection = VALID_GENRE.has(
    partial.genreSelection as GenreSelectionMode,
  )
    ? (partial.genreSelection as GenreSelectionMode)
    : defaults.genreSelection;

  const preferredLanguages = Array.isArray(partial.preferredLanguages)
    ? partial.preferredLanguages
        .map((lang) => String(lang).trim().toLowerCase())
        .filter(Boolean)
    : defaults.preferredLanguages;

  const enabledMixIds = Array.isArray(partial.enabledMixIds)
    ? [
        ...new Set(
          partial.enabledMixIds.flatMap((id) => expandLegacyMixId(String(id))),
        ),
      ]
    : defaults.enabledMixIds;

  const maxTracksPerMix = clampInt(
    partial.maxTracksPerMix,
    10,
    100,
    defaults.maxTracksPerMix,
  );
  const minTracksPerMix = Math.min(
    clampInt(partial.minTracksPerMix, 5, 50, defaults.minTracksPerMix),
    maxTracksPerMix,
  );

  return {
    maxTracksPerMix,
    minTracksPerMix,
    discoverRecentDays: clampInt(
      partial.discoverRecentDays,
      7,
      90,
      defaults.discoverRecentDays,
    ),
    throwbackDays: clampInt(
      partial.throwbackDays,
      14,
      365,
      defaults.throwbackDays,
    ),
    deepCutMaxPlayCount: clampInt(
      partial.deepCutMaxPlayCount,
      0,
      10,
      defaults.deepCutMaxPlayCount,
    ),
    crossMixDedup:
      typeof partial.crossMixDedup === "boolean"
        ? partial.crossMixDedup
        : defaults.crossMixDedup,
    durationPacing:
      typeof partial.durationPacing === "boolean"
        ? partial.durationPacing
        : defaults.durationPacing,
    languageBias,
    preferredLanguages,
    genreSelection,
    enabledMixIds,
    mixSeedSuffix:
      typeof partial.mixSeedSuffix === "string" && partial.mixSeedSuffix.trim()
        ? partial.mixSeedSuffix.trim()
        : defaults.mixSeedSuffix,
    personalRadioRecencyHours: clampInt(
      partial.personalRadioRecencyHours,
      0,
      24,
      defaults.personalRadioRecencyHours,
    ),
    personalRadioColdStartPlays: clampInt(
      partial.personalRadioColdStartPlays,
      1,
      50,
      defaults.personalRadioColdStartPlays,
    ),
    radioExploreBonus: clampNumber(
      partial.radioExploreBonus,
      0,
      2,
      defaults.radioExploreBonus,
    ),
    flowAlbumLookback: clampInt(
      partial.flowAlbumLookback,
      2,
      8,
      defaults.flowAlbumLookback,
    ),
  };
}

function clampNumber(
  value: unknown,
  min: number,
  max: number,
  fallback: number,
): number {
  if (typeof value !== "number" || !Number.isFinite(value)) return fallback;
  return Math.max(min, Math.min(max, value));
}

export function loadMixSettings(): MixSettings {
  if (typeof localStorage === "undefined") return defaultMixSettings();
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return defaultMixSettings();
    return mergeMixSettings(JSON.parse(raw) as Partial<MixSettings>);
  } catch {
    return defaultMixSettings();
  }
}

export function saveMixSettings(settings: MixSettings): void {
  if (typeof localStorage === "undefined") return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
  } catch {
    /* storage full or unavailable */
  }
}

export function isMixEnabled(id: string, settings: MixSettings): boolean {
  if (settings.enabledMixIds.length === 0) return true;
  return settings.enabledMixIds.includes(id as MixId);
}

export function parsePreferredLanguages(raw: string): string[] {
  return raw
    .split(/[,;\s]+/)
    .map((lang) => lang.trim().toLowerCase())
    .filter(Boolean);
}

export function formatPreferredLanguages(languages: readonly string[]): string {
  return languages.join(", ");
}
