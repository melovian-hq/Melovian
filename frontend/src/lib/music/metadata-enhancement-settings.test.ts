// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it } from "vitest";
import {
  defaultMetadataEnhancementSettings,
  loadMetadataEnhancementSettings,
  mergeMetadataEnhancementSettings,
  saveMetadataEnhancementSettings,
} from "./metadata-enhancement-settings";

const STORAGE_KEY = "mel-metadata-enhancement-settings";

describe("metadata-enhancement-settings", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it("returns disabled defaults", () => {
    expect(defaultMetadataEnhancementSettings()).toEqual({
      enabled: false,
      artists: true,
      albums: true,
      tracks: true,
      preferServerArtistArt: true,
    });
  });

  it("merges partial settings without dropping unspecified toggles", () => {
    expect(
      mergeMetadataEnhancementSettings({
        enabled: false,
        albums: false,
      }),
    ).toEqual({
      enabled: false,
      artists: true,
      albums: false,
      tracks: true,
      preferServerArtistArt: true,
    });
  });

  it("keeps enhancement defaults when enabled is omitted", () => {
    expect(mergeMetadataEnhancementSettings({ albums: false })).toEqual({
      enabled: false,
      artists: true,
      albums: false,
      tracks: true,
      preferServerArtistArt: true,
    });
  });

  it("persists settings to localStorage", () => {
    const settings = {
      enabled: true,
      artists: false,
      albums: true,
      tracks: false,
      preferServerArtistArt: false,
    };
    saveMetadataEnhancementSettings(settings);
    expect(JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "{}")).toEqual(
      settings,
    );
    expect(loadMetadataEnhancementSettings()).toEqual(settings);
  });

  it("defaults preferServerArtistArt on for older stored settings", () => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        enabled: true,
        artists: true,
        albums: true,
        tracks: true,
      }),
    );
    expect(loadMetadataEnhancementSettings().preferServerArtistArt).toBe(true);
  });
});
