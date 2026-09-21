// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { StorageKeys } from "$lib/brand";
import {
  accentTextFor,
  activeColorFor,
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
import {
  CUSTOM_PALETTE_ID,
  DEFAULT_CUSTOM_PALETTE,
  DEFAULT_PALETTE_ID,
  DEFAULT_RADIUS_ID,
  DEFAULT_UI_SIZE_ID,
  isPaletteIdForMode,
  RADIUS_OVERRIDES,
  UI_SIZE_FONT_SCALE,
  type CustomPaletteColors,
  type RadiusStyleId,
  type UiSizeId,
} from "./palettes";

export type ThemeMode = "light" | "dark" | "system";

const STORAGE_KEY = StorageKeys.theme;
const ACCENT_PRESET_KEY = StorageKeys.themeAccentPreset;
const ACCENT_HUE_KEY = StorageKeys.themeAccentHue;
const LEGACY_ACCENT_KEY = StorageKeys.themeAccent;
const CUSTOM_CSS_KEY = StorageKeys.customCss;
const CUSTOM_STYLE_ID = StorageKeys.customCss;
const PALETTE_DARK_KEY = StorageKeys.themePaletteDark;
const PALETTE_LIGHT_KEY = StorageKeys.themePaletteLight;
const CUSTOM_PALETTE_KEY = StorageKeys.themeCustomPalette;
const RADIUS_KEY = StorageKeys.themeRadius;
const UI_SIZE_KEY = StorageKeys.themeUiSize;

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

function loadStoredPalette(mode: ResolvedTheme): string {
  const key = mode === "dark" ? PALETTE_DARK_KEY : PALETTE_LIGHT_KEY;
  const stored = readStorage(key)?.trim();
  return stored && isPaletteIdForMode(stored, mode)
    ? stored
    : DEFAULT_PALETTE_ID;
}

/** Normalize one custom palette field. Only #rrggbb hex passes through. */
function sanitizeCustomColor(
  value: unknown,
  fallback: string,
  allowEmpty: boolean,
): string {
  if (typeof value !== "string") return fallback;
  const hex = value.trim();
  if (allowEmpty && hex === "") return "";
  return isHexColor(hex) ? hex.toLowerCase() : fallback;
}

function loadStoredCustomPalettes(): Record<
  ResolvedTheme,
  CustomPaletteColors
> {
  const fallback = structuredClone(DEFAULT_CUSTOM_PALETTE);
  const raw = readStorage(CUSTOM_PALETTE_KEY);
  if (!raw) return fallback;
  try {
    const parsed = JSON.parse(raw) as Partial<
      Record<ResolvedTheme, Partial<CustomPaletteColors>>
    >;
    for (const mode of ["dark", "light"] as const) {
      const stored = parsed?.[mode];
      if (!stored) continue;
      fallback[mode] = {
        bg: sanitizeCustomColor(stored.bg, fallback[mode].bg, false),
        surface: sanitizeCustomColor(
          stored.surface,
          fallback[mode].surface,
          false,
        ),
        accent: sanitizeCustomColor(stored.accent, "", true),
      };
    }
    return fallback;
  } catch {
    return fallback;
  }
}

const RADIUS_STYLE_IDS: readonly RadiusStyleId[] = [
  "default",
  "sharp",
  "square",
  "round",
];
const UI_SIZE_IDS: readonly UiSizeId[] = ["default", "compact", "large"];

function loadStoredRadius(): RadiusStyleId {
  const stored = readStorage(RADIUS_KEY)?.trim();
  return RADIUS_STYLE_IDS.find((id) => id === stored) ?? DEFAULT_RADIUS_ID;
}

function loadStoredUiSize(): UiSizeId {
  const stored = readStorage(UI_SIZE_KEY)?.trim();
  return UI_SIZE_IDS.find((id) => id === stored) ?? DEFAULT_UI_SIZE_ID;
}

function applyPalette(
  palette: string,
  custom: CustomPaletteColors | undefined,
  resolved: ResolvedTheme,
) {
  const root = document.documentElement;
  if (palette === CUSTOM_PALETTE_ID && custom) {
    root.dataset.palette = CUSTOM_PALETTE_ID;
    root.style.setProperty("--jb-custom-bg", custom.bg);
    root.style.setProperty("--jb-custom-surface", custom.surface);
    if (custom.accent) {
      root.style.setProperty("--jb-custom-accent", custom.accent);
      root.style.setProperty(
        "--jb-active",
        activeColorFor(custom.accent, resolved),
      );
    } else {
      root.style.removeProperty("--jb-custom-accent");
      root.style.removeProperty("--jb-active");
    }
    return;
  }
  root.style.removeProperty("--jb-active");
  root.style.removeProperty("--jb-custom-bg");
  root.style.removeProperty("--jb-custom-surface");
  root.style.removeProperty("--jb-custom-accent");
  if (palette === DEFAULT_PALETTE_ID) {
    delete root.dataset.palette;
    return;
  }
  root.dataset.palette = palette;
}

function applyRadiusStyle(style: RadiusStyleId) {
  const root = document.documentElement.style;
  const sizes = ["sm", "md", "lg", "xl", "full"] as const;
  if (style === DEFAULT_RADIUS_ID) {
    for (const size of sizes) root.removeProperty(`--jb-radius-${size}`);
    return;
  }
  const overrides = RADIUS_OVERRIDES[style];
  for (const size of sizes) {
    const value = overrides[size];
    if (value === "9999px") {
      root.removeProperty(`--jb-radius-${size}`);
    } else {
      root.setProperty(`--jb-radius-${size}`, value);
    }
  }
}

function applyUiSize(size: UiSizeId) {
  const root = document.documentElement.style;
  if (size === DEFAULT_UI_SIZE_ID) {
    root.removeProperty("font-size");
    return;
  }
  root.setProperty("font-size", UI_SIZE_FONT_SCALE[size]);
}

function applyAccent(hex: string, resolved: ResolvedTheme) {
  const root = document.documentElement.style;
  root.setProperty("--jb-accent", hex);
  root.setProperty("--jb-accent-text", accentTextFor(hex));
  root.setProperty("--jb-active", activeColorFor(hex, resolved));
}

function clearAccentOverrides() {
  const root = document.documentElement.style;
  root.removeProperty("--jb-accent");
  root.removeProperty("--jb-accent-text");
  root.removeProperty("--jb-active");
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
  darkPalette = $state<string>(loadStoredPalette("dark"));
  lightPalette = $state<string>(loadStoredPalette("light"));
  customPalettes = $state<Record<ResolvedTheme, CustomPaletteColors>>(
    loadStoredCustomPalettes(),
  );
  radiusStyle = $state<RadiusStyleId>(loadStoredRadius());
  uiSize = $state<UiSizeId>(loadStoredUiSize());
  /** Palette id active for the resolved theme. */
  palette = $derived(
    this.resolved === "dark" ? this.darkPalette : this.lightPalette,
  );
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
        // re-applied with their per-theme hex when the theme flips. Reading
        // palette re-applies after applyPalette clears --jb-active on the
        // custom to builtin transition.
        void this.palette;
        const hex = this.accentColor;
        if (preset === DEFAULT_ACCENT_PRESET_ID) {
          removeStorage(ACCENT_PRESET_KEY);
          clearAccentOverrides();
          return;
        }
        writeStorage(ACCENT_PRESET_KEY, preset);
        applyAccent(hex, this.resolved);
      });

      $effect(() => {
        writeStorage(ACCENT_HUE_KEY, String(this.customHue));
      });

      $effect(() => {
        const palette = this.palette;
        const mode = this.resolved;
        const custom = this.customPalettes[mode];
        for (const [key, value] of [
          [PALETTE_DARK_KEY, this.darkPalette],
          [PALETTE_LIGHT_KEY, this.lightPalette],
        ] as const) {
          if (value === DEFAULT_PALETTE_ID) removeStorage(key);
          else writeStorage(key, value);
        }
        writeStorage(CUSTOM_PALETTE_KEY, JSON.stringify(this.customPalettes));
        applyPalette(palette, custom, mode);
      });

      $effect(() => {
        const style = this.radiusStyle;
        if (style === DEFAULT_RADIUS_ID) removeStorage(RADIUS_KEY);
        else writeStorage(RADIUS_KEY, style);
        applyRadiusStyle(style);
      });

      $effect(() => {
        const size = this.uiSize;
        if (size === DEFAULT_UI_SIZE_ID) removeStorage(UI_SIZE_KEY);
        else writeStorage(UI_SIZE_KEY, size);
        applyUiSize(size);
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

  /** Every palette ships a dark and a light ramp, so one pick sets both. */
  setPalette(id: string) {
    if (!isPaletteIdForMode(id, this.resolved)) return;
    this.darkPalette = id;
    this.lightPalette = id;
  }

  /** Update one color of the custom palette for a resolved theme. */
  setCustomPaletteColor(
    mode: ResolvedTheme,
    field: keyof CustomPaletteColors,
    hex: string,
  ) {
    const trimmed = hex.trim();
    if (field === "accent" && trimmed === "") {
      this.customPalettes[mode] = {
        ...this.customPalettes[mode],
        accent: "",
      };
      return;
    }
    if (!isHexColor(trimmed)) return;
    this.customPalettes[mode] = {
      ...this.customPalettes[mode],
      [field]: trimmed.toLowerCase(),
    };
    this.darkPalette = CUSTOM_PALETTE_ID;
    this.lightPalette = CUSTOM_PALETTE_ID;
  }

  setRadiusStyle(id: RadiusStyleId) {
    if (RADIUS_STYLE_IDS.includes(id)) this.radiusStyle = id;
  }

  setUiSize(id: UiSizeId) {
    if (UI_SIZE_IDS.includes(id)) this.uiSize = id;
  }

  setCustomCss(css: string) {
    this.customCss = css;
  }

  resetCustomization() {
    this.accentPreset = DEFAULT_ACCENT_PRESET_ID;
    this.customHue = DEFAULT_ACCENT_HUE;
    this.customCss = "";
    this.darkPalette = DEFAULT_PALETTE_ID;
    this.lightPalette = DEFAULT_PALETTE_ID;
    this.customPalettes = structuredClone(DEFAULT_CUSTOM_PALETTE);
    this.radiusStyle = DEFAULT_RADIUS_ID;
    this.uiSize = DEFAULT_UI_SIZE_ID;
    clearAccentOverrides();
    applyCustomCss("");
    removeStorage(ACCENT_PRESET_KEY);
    removeStorage(ACCENT_HUE_KEY);
    removeStorage(LEGACY_ACCENT_KEY);
    removeStorage(CUSTOM_CSS_KEY);
    removeStorage(PALETTE_DARK_KEY);
    removeStorage(PALETTE_LIGHT_KEY);
    removeStorage(CUSTOM_PALETTE_KEY);
    removeStorage(RADIUS_KEY);
    removeStorage(UI_SIZE_KEY);
  }

  toggle() {
    this.mode = this.resolved === "dark" ? "light" : "dark";
  }
}

export const theme = new ThemeStore();

export { DEFAULT_ACCENT };
