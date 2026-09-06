// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { StorageKeys } from "$lib/brand";

export type ThemeMode = "light" | "dark" | "system";

const STORAGE_KEY = StorageKeys.theme;
const ACCENT_KEY = StorageKeys.themeAccent;
const CUSTOM_CSS_KEY = StorageKeys.customCss;
const CUSTOM_STYLE_ID = StorageKeys.customCss;

const DEFAULT_ACCENT = "#dc2626";

function resolveTheme(mode: ThemeMode): "light" | "dark" {
  if (mode === "system") {
    return window.matchMedia("(prefers-color-scheme: dark)").matches
      ? "dark"
      : "light";
  }
  return mode;
}

function loadStoredMode(): ThemeMode {
  if (typeof window !== "undefined") {
    const param = new URLSearchParams(window.location.search).get("theme");
    if (param === "light" || param === "dark" || param === "system") {
      return param;
    }
  }
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === "light" || stored === "dark" || stored === "system") {
    return stored;
  }
  return "dark";
}

function loadStoredAccent(): string {
  const stored = localStorage.getItem(ACCENT_KEY)?.trim();
  if (!stored) return DEFAULT_ACCENT;
  return /^#[0-9a-fA-F]{6}$/.test(stored) ? stored : DEFAULT_ACCENT;
}

function loadStoredCustomCss(): string {
  return localStorage.getItem(CUSTOM_CSS_KEY) ?? "";
}

function applyTheme(resolved: "light" | "dark") {
  document.documentElement.dataset.theme = resolved;
  document.documentElement.style.colorScheme = resolved;
}

function applyAccent(color: string) {
  const root = document.documentElement.style;
  root.setProperty("--jb-accent", color);
  root.setProperty(
    "--jb-accent-hover",
    `color-mix(in srgb, ${color} 82%, black)`,
  );
  root.setProperty(
    "--jb-accent-muted",
    `color-mix(in srgb, ${color} 18%, transparent)`,
  );
  root.setProperty("--jb-music-accent", color);
  root.setProperty(
    "--jb-music-accent-hover",
    `color-mix(in srgb, ${color} 82%, black)`,
  );
  root.setProperty(
    "--jb-music-accent-muted",
    `color-mix(in srgb, ${color} 18%, transparent)`,
  );
  root.setProperty(
    "--jb-focus-ring",
    `0 0 0 3px color-mix(in srgb, ${color} 25%, transparent)`,
  );
}

function clearAccentOverrides() {
  const root = document.documentElement.style;
  for (const prop of [
    "--jb-accent",
    "--jb-accent-hover",
    "--jb-accent-muted",
    "--jb-music-accent",
    "--jb-music-accent-hover",
    "--jb-music-accent-muted",
    "--jb-focus-ring",
  ]) {
    root.removeProperty(prop);
  }
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

class ThemeStore {
  mode = $state<ThemeMode>(loadStoredMode());
  resolved = $state<"light" | "dark">(resolveTheme(loadStoredMode()));
  accentColor = $state(loadStoredAccent());
  customCss = $state(loadStoredCustomCss());

  constructor() {
    $effect.root(() => {
      const media = window.matchMedia("(prefers-color-scheme: dark)");

      const sync = () => {
        this.resolved = resolveTheme(this.mode);
        applyTheme(this.resolved);
      };

      sync();

      const onMediaChange = () => {
        if (this.mode === "system") sync();
      };

      media.addEventListener("change", onMediaChange);

      $effect(() => {
        localStorage.setItem(STORAGE_KEY, this.mode);
        sync();
      });

      $effect(() => {
        const resolved = this.resolved;
        const meta = document.querySelector('meta[name="theme-color"]');
        if (meta) {
          meta.setAttribute(
            "content",
            resolved === "dark" ? "#090909" : "#f4f4f5",
          );
        }
      });

      $effect(() => {
        const color = this.accentColor.trim();
        if (!color || color === DEFAULT_ACCENT) {
          localStorage.removeItem(ACCENT_KEY);
          clearAccentOverrides();
          return;
        }
        if (/^#[0-9a-fA-F]{6}$/.test(color)) {
          localStorage.setItem(ACCENT_KEY, color);
          applyAccent(color);
        }
      });

      $effect(() => {
        const css = this.customCss;
        localStorage.setItem(CUSTOM_CSS_KEY, css);
        applyCustomCss(css);
      });

      return () => media.removeEventListener("change", onMediaChange);
    });
  }

  setMode(mode: ThemeMode) {
    this.mode = mode;
  }

  setAccentColor(color: string) {
    this.accentColor = color;
  }

  setCustomCss(css: string) {
    this.customCss = css;
  }

  resetCustomization() {
    this.accentColor = DEFAULT_ACCENT;
    this.customCss = "";
    clearAccentOverrides();
    applyCustomCss("");
    localStorage.removeItem(ACCENT_KEY);
    localStorage.removeItem(CUSTOM_CSS_KEY);
  }

  toggle() {
    this.mode = this.resolved === "dark" ? "light" : "dark";
  }
}

export const theme = new ThemeStore();

export { DEFAULT_ACCENT };
