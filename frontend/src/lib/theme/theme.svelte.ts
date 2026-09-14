// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { StorageKeys } from "$lib/brand";
import {
  accentTextFor,
  CUSTOM_ACCENT_ID,
  DEFAULT_ACCENT_HUE,
  DEFAULT_ACCENT_PRESET_ID,
  hexToHue,
  hueToAccentHex,
  isAccentPresetId,
  isHexColor,
  presetAccentHex,
  type ResolvedTheme,
} from "./accent";
import {
  applyThemeDataset,
  DARK_THEME_COLORS,
  LIGHT_THEME_COLORS,
} from "./contrast";

export type ThemeMode = "light" | "dark" | "system";

const STORAGE_KEY = StorageKeys.theme;
const ACCENT_PRESET_KEY = StorageKeys.themeAccentPreset;
const ACCENT_HUE_KEY = StorageKeys.themeAccentHue;
const LEGACY_ACCENT_KEY = StorageKeys.themeAccent;
const CUSTOM_CSS_KEY = StorageKeys.customCss;
const CUSTOM_STYLE_ID = StorageKeys.customCss;

const DEFAULT_ACCENT = presetAccentHex(DEFAULT_ACCENT_PRESET_ID, "dark");

// localStorage can be missing or throw (test envs, locked-down webviews).
function readStorage(key: string): string | null {
  try {
    return typeof localStorage === "undefined"
      ? null
      : localStorage.getItem(key);
  } catch {
    return null;
  }
}

function writeStorage(key: string, value: string) {
  try {
    if (typeof localStorage !== "undefined") localStorage.setItem(key, value);
  } catch {
    /* storage unavailable */
  }
}

function removeStorage(key: string) {
  try {
    if (typeof localStorage !== "undefined") localStorage.removeItem(key);
  } catch {
    /* storage unavailable */
  }
}

/** Pure resolution: system follows the OS preference, others pass through. */
export function resolveThemeMode(
  mode: ThemeMode,
  prefersDark: boolean,
): ResolvedTheme {
  if (mode === "system") return prefersDark ? "dark" : "light";
  return mode;
}

function resolveTheme(mode: ThemeMode): ResolvedTheme {
  return resolveThemeMode(
    mode,
    typeof window !== "undefined" &&
      window.matchMedia("(prefers-color-scheme: dark)").matches,
  );
}

function loadStoredMode(): ThemeMode {
  if (typeof window !== "undefined") {
    const param = new URLSearchParams(window.location.search).get("theme");
    if (param === "light" || param === "dark" || param === "system") {
      return param;
    }
  }
  const stored = readStorage(STORAGE_KEY);
  if (stored === "light" || stored === "dark" || stored === "system") {
    return stored;
  }
  return "dark";
}

function loadStoredAccentPreset(): string {
  const stored = readStorage(ACCENT_PRESET_KEY)?.trim();
  if (stored && isAccentPresetId(stored)) return stored;
  const legacy = readStorage(LEGACY_ACCENT_KEY)?.trim();
  if (legacy && isHexColor(legacy)) return CUSTOM_ACCENT_ID;
  return DEFAULT_ACCENT_PRESET_ID;
}

function loadStoredCustomHue(): number {
  const raw = readStorage(ACCENT_HUE_KEY);
  const stored = raw === null ? Number.NaN : Number(raw);
  if (Number.isFinite(stored) && stored >= 0 && stored <= 360) {
    return Math.round(stored);
  }
  const legacy = readStorage(LEGACY_ACCENT_KEY)?.trim();
  if (legacy && isHexColor(legacy)) return hexToHue(legacy);
  return DEFAULT_ACCENT_HUE;
}

function loadStoredCustomCss(): string {
  return readStorage(CUSTOM_CSS_KEY) ?? "";
}

function applyAccent(hex: string) {
  const root = document.documentElement.style;
  root.setProperty("--jb-accent", hex);
  root.setProperty("--jb-accent-text", accentTextFor(hex));
}

function clearAccentOverrides() {
  const root = document.documentElement.style;
  root.removeProperty("--jb-accent");
  root.removeProperty("--jb-accent-text");
}

function applyCustomCss(css: string) {
  let el = document.getElementById(CUSTOM_STYLE_ID);
  if (!css.trim()) {
    el?.remove();
    return;
  }
  if (!el) {
    el = document.createElement("style");
    el.id = CUSTOM_STYLE_ID;
    document.head.appendChild(el);
  }
  el.textContent = css;
}

export class ThemeStore {
  mode = $state<ThemeMode>(loadStoredMode());
  resolved = $state<ResolvedTheme>(resolveTheme(loadStoredMode()));
  accentPreset = $state<string>(loadStoredAccentPreset());
  customHue = $state<number>(loadStoredCustomHue());
  customCss = $state(loadStoredCustomCss());
  accentColor = $derived(
    this.accentPreset === CUSTOM_ACCENT_ID
      ? hueToAccentHex(this.customHue, this.resolved)
      : presetAccentHex(this.accentPreset, this.resolved),
  );

  constructor() {
    // Consume the legacy free-form accent key once so it stops lingering.
    removeStorage(LEGACY_ACCENT_KEY);

    $effect.root(() => {
      const media = window.matchMedia("(prefers-color-scheme: dark)");

      const sync = () => {
        this.resolved = resolveTheme(this.mode);
        applyThemeDataset(this.resolved);
      };

      sync();

      const onMediaChange = () => {
        if (this.mode === "system") sync();
      };

      media.addEventListener("change", onMediaChange);

      $effect(() => {
        writeStorage(STORAGE_KEY, this.mode);
        sync();
      });

      $effect(() => {
        const resolved = this.resolved;
        const meta = document.querySelector('meta[name="theme-color"]');
        if (meta) {
          meta.setAttribute(
            "content",
            resolved === "dark" ? DARK_THEME_COLORS.bg : LIGHT_THEME_COLORS.bg,
          );
        }
      });

      $effect(() => {
        const preset = this.accentPreset;
        // Reading accentColor also tracks resolved, so preset accents are
        // re-applied with their per-theme hex when the theme flips.
        const hex = this.accentColor;
        if (preset === DEFAULT_ACCENT_PRESET_ID) {
          removeStorage(ACCENT_PRESET_KEY);
          clearAccentOverrides();
          return;
        }
        writeStorage(ACCENT_PRESET_KEY, preset);
        applyAccent(hex);
      });

      $effect(() => {
        writeStorage(ACCENT_HUE_KEY, String(this.customHue));
      });

      $effect(() => {
        const css = this.customCss;
        if (!css.trim()) {
          removeStorage(CUSTOM_CSS_KEY);
          applyCustomCss("");
          return;
        }
        writeStorage(CUSTOM_CSS_KEY, css);
        applyCustomCss(css);
      });

      return () => media.removeEventListener("change", onMediaChange);
    });
  }

  setMode(mode: ThemeMode) {
    this.mode = mode;
  }

  setAccentPreset(id: string) {
    if (isAccentPresetId(id)) this.accentPreset = id;
  }

  setCustomHue(hue: number) {
    this.customHue = Math.min(360, Math.max(0, Math.round(hue) || 0));
    this.accentPreset = CUSTOM_ACCENT_ID;
  }

  /** Back-compat: a raw hex maps onto the custom accent hue control. */
  setAccentColor(color: string) {
    const hex = color.trim();
    if (!isHexColor(hex)) return;
    this.customHue = hexToHue(hex);
    this.accentPreset = CUSTOM_ACCENT_ID;
  }

  setCustomCss(css: string) {
    this.customCss = css;
  }

  resetCustomization() {
    this.accentPreset = DEFAULT_ACCENT_PRESET_ID;
    this.customHue = DEFAULT_ACCENT_HUE;
    this.customCss = "";
    clearAccentOverrides();
    applyCustomCss("");
    removeStorage(ACCENT_PRESET_KEY);
    removeStorage(ACCENT_HUE_KEY);
    removeStorage(LEGACY_ACCENT_KEY);
    removeStorage(CUSTOM_CSS_KEY);
  }

  toggle() {
    this.mode = this.resolved === "dark" ? "light" : "dark";
  }
}

export const theme = new ThemeStore();

export { DEFAULT_ACCENT };
