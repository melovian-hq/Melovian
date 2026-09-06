// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export interface LyricsProviderSetting {
  id: string;
  name?: string;
  enabled: boolean;
  custom?: boolean;
  url?: string;
}

export interface BuiltinLyricsProvider {
  id: string;
  name: string;
  builtin: boolean;
  custom?: boolean;
}

export interface LyricsSettings {
  storageDir: string;
  autoFetch: boolean;
  providers: LyricsProviderSetting[];
  /** whisper.cpp-compatible server URL used by the lyrics-whisper extension. */
  whisperUrl?: string;
}

export interface LyricsSettingsResponse extends LyricsSettings {
  resolvedStorageDir: string;
  defaultStorageDir: string;
  builtinProviders: BuiltinLyricsProvider[];
  /** Server-side flag: the lyrics-whisper extension is installed and enabled. */
  whisperEnabled?: boolean;
  trackCount: number;
  usedBytes: number;
}

export function defaultLyricsSettings(): LyricsSettings {
  return {
    storageDir: "",
    autoFetch: false,
    providers: [
      { id: "subsonic", name: "Navidrome / Subsonic", enabled: true },
      { id: "lrclib", name: "LRCLIB", enabled: true },
      { id: "lyrics-ovh", name: "Lyrics.ovh", enabled: true },
    ],
    whisperUrl: "",
  };
}

export function mergeLyricsSettings(
  partial: Partial<LyricsSettings> | null | undefined,
): LyricsSettings {
  const defaults = defaultLyricsSettings();
  if (!partial || typeof partial !== "object") return defaults;

  const providers =
    partial.providers?.map((provider) => ({
      id: provider.id,
      name: provider.name,
      enabled: !!provider.enabled,
      custom: !!provider.custom,
      url: provider.url,
    })) ?? defaults.providers;

  return {
    storageDir:
      typeof partial.storageDir === "string"
        ? partial.storageDir
        : defaults.storageDir,
    autoFetch:
      typeof partial.autoFetch === "boolean"
        ? partial.autoFetch
        : defaults.autoFetch,
    providers,
    whisperUrl:
      typeof partial.whisperUrl === "string"
        ? partial.whisperUrl
        : defaults.whisperUrl,
  };
}

export { formatBytes as formatLyricsBytes } from "$lib/utils/format-bytes";
