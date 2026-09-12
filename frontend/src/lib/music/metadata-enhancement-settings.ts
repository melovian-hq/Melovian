// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { StorageKeys } from "$lib/brand";

export interface MetadataEnhancementSettings {
  enabled: boolean;
  artists: boolean;
  albums: boolean;
  tracks: boolean;
  /** Prefer Navidrome/Subsonic artistImageUrl and coverArt over iTunes. */
  preferServerArtistArt: boolean;
}

const STORAGE_KEY = StorageKeys.metadataEnhancementSettings;

export function defaultMetadataEnhancementSettings(): MetadataEnhancementSettings {
  return {
    enabled: false,
    artists: true,
    albums: true,
    tracks: true,
    preferServerArtistArt: true,
  };
}

export function mergeMetadataEnhancementSettings(
  partial: Partial<MetadataEnhancementSettings> | null | undefined,
): MetadataEnhancementSettings {
  const defaults = defaultMetadataEnhancementSettings();
  if (!partial || typeof partial !== "object") return defaults;

  const artists =
    typeof partial.artists === "boolean" ? partial.artists : defaults.artists;
  const albums =
    typeof partial.albums === "boolean" ? partial.albums : defaults.albums;
  const tracks =
    typeof partial.tracks === "boolean" ? partial.tracks : defaults.tracks;
  const anyExplicitlyEnabled =
    (typeof partial.artists === "boolean" && partial.artists) ||
    (typeof partial.albums === "boolean" && partial.albums) ||
    (typeof partial.tracks === "boolean" && partial.tracks);

  return {
    enabled:
      typeof partial.enabled === "boolean"
        ? partial.enabled
        : anyExplicitlyEnabled
          ? true
          : defaults.enabled,
    artists,
    albums,
    tracks,
    preferServerArtistArt:
      typeof partial.preferServerArtistArt === "boolean"
        ? partial.preferServerArtistArt
        : defaults.preferServerArtistArt,
  };
}

export function loadMetadataEnhancementSettings(): MetadataEnhancementSettings {
  if (typeof localStorage === "undefined") {
    return defaultMetadataEnhancementSettings();
  }
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return defaultMetadataEnhancementSettings();
    return mergeMetadataEnhancementSettings(
      JSON.parse(raw) as Partial<MetadataEnhancementSettings>,
    );
  } catch {
    return defaultMetadataEnhancementSettings();
  }
}

export function saveMetadataEnhancementSettings(
  settings: MetadataEnhancementSettings,
): void {
  if (typeof localStorage === "undefined") return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
  } catch {
    /* storage full or unavailable */
  }
}
