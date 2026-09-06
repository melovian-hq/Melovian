// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/** Bundled feature extension ids. On by default until the user disables them. */
export const EXT_LYRICS = "lyrics";
export const EXT_METADATA = "metadata";
export const EXT_LYRICS_WHISPER = "lyrics-whisper";

const BUNDLED_DEFAULT_ON = new Set([EXT_LYRICS, EXT_METADATA]);

const DATA_ATTR = "extensionTheme";

export type ExtensionFeatureItem = {
  id: string;
  enabled: boolean;
  appTheme?: string;
};

function syncDocumentTheme(theme: string) {
  if (typeof document === "undefined") return;
  const root = document.documentElement;
  if (theme) {
    root.dataset[DATA_ATTR] = theme;
  } else {
    delete root.dataset[DATA_ATTR];
  }
}

class ExtensionFeaturesStore {
  enabledIds = $state<Set<string>>(new Set(BUNDLED_DEFAULT_ON));
  /** Themes offered by enabled extensions (manifest appTheme). */
  availableThemes = $state<Set<string>>(new Set());
  /** Active chrome theme from the current track decoration, or "". */
  appTheme = $state("");
  loaded = $state(false);

  isEnabled(id: string): boolean {
    return this.enabledIds.has(id);
  }

  get lyrics(): boolean {
    return this.isEnabled(EXT_LYRICS);
  }

  get metadata(): boolean {
    return this.isEnabled(EXT_METADATA);
  }

  get lyricsWhisper(): boolean {
    return this.isEnabled(EXT_LYRICS_WHISPER);
  }

  applyFromItems(items: ExtensionFeatureItem[]) {
    const next = new Set<string>();
    const themes = new Set<string>();
    for (const item of items) {
      if (!item.enabled) continue;
      next.add(item.id);
      const candidate = String(item.appTheme ?? "").trim();
      if (candidate) themes.add(candidate);
    }
    this.enabledIds = next;
    this.availableThemes = themes;
    this.loaded = true;
    if (this.appTheme && !this.isThemeAvailable(this.appTheme)) {
      this.appTheme = "";
      syncDocumentTheme("");
    }
  }

  /** Refresh themes offered by enabled manifests (does not activate chrome). */
  applyAppThemeFromManifests(
    manifests: Array<{ id?: string; appTheme?: string }>,
  ) {
    const themes = new Set<string>();
    for (const manifest of manifests) {
      const candidate = String(manifest.appTheme ?? "").trim();
      if (candidate) themes.add(candidate);
    }
    this.availableThemes = themes;
    if (this.appTheme && !this.isThemeAvailable(this.appTheme)) {
      this.appTheme = "";
      syncDocumentTheme("");
    }
  }

  /**
   * Activate chrome when the current track's playerTheme matches a theme
   * offered by an enabled extension.
   */
  syncFromTrackDecoration(decoration: { playerTheme?: string } | null) {
    const candidate = String(decoration?.playerTheme ?? "").trim();
    const theme = this.isThemeAvailable(candidate) ? candidate : "";
    if (theme === this.appTheme) {
      syncDocumentTheme(theme);
      return;
    }
    this.appTheme = theme;
    syncDocumentTheme(theme);
  }

  private isThemeAvailable(theme: string): boolean {
    if (!theme) return false;
    return this.availableThemes.has(theme);
  }

  /** Test helper. */
  resetForTests(
    enabled: string[] = [...BUNDLED_DEFAULT_ON],
    theme = "",
    available: string[] = theme ? [theme] : [],
  ) {
    this.enabledIds = new Set(enabled);
    this.availableThemes = new Set(available);
    this.appTheme = theme;
    this.loaded = true;
    syncDocumentTheme(theme);
  }
}

export const extensionFeatures = new ExtensionFeaturesStore();
