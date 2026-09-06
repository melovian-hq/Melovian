// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { getVideoSettings } from "./api";
import {
  defaultVideoSettings,
  mergeVideoSettings,
  type VideoSettings,
} from "./ids";

const CACHE_TTL_MS = 30_000;

class VideoFeatureStore {
  enabled = $state(false);
  loaded = $state(false);
  settings = $state<VideoSettings>(defaultVideoSettings());

  private inflight: Promise<VideoSettings> | null = null;
  private loadedAt = 0;

  /** Coalesced settings fetch with a short TTL to avoid duplicate GETs. */
  async refresh(force = false): Promise<VideoSettings> {
    if (!force && this.loaded && Date.now() - this.loadedAt < CACHE_TTL_MS) {
      return this.settings;
    }
    if (this.inflight) return this.inflight;

    this.inflight = this.load();
    try {
      return await this.inflight;
    } finally {
      this.inflight = null;
    }
  }

  applySettings(next: VideoSettings) {
    const merged = mergeVideoSettings(next);
    this.settings = merged;
    this.enabled = merged.enabled;
    this.loaded = true;
    this.loadedAt = Date.now();
  }

  setEnabled(value: boolean) {
    this.enabled = value;
    this.settings = { ...this.settings, enabled: value };
    this.loaded = true;
    this.loadedAt = Date.now();
  }

  private async load(): Promise<VideoSettings> {
    try {
      const settings = mergeVideoSettings(await getVideoSettings());
      this.applySettings(settings);
      return settings;
    } catch {
      this.enabled = false;
      this.settings = defaultVideoSettings();
      this.loaded = true;
      this.loadedAt = Date.now();
      return this.settings;
    }
  }
}

export const videoFeature = new VideoFeatureStore();
