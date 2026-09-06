// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export type MixDisplayStyle = "cards" | "discs";

export const MIX_DISPLAY_KEY = "mel-mix-display";

export function loadMixDisplay(): MixDisplayStyle {
  try {
    const saved = localStorage.getItem(MIX_DISPLAY_KEY);
    if (saved === "cards" || saved === "discs") return saved;
  } catch {
    // ignore
  }
  return "cards";
}

export function saveMixDisplay(style: MixDisplayStyle): void {
  try {
    localStorage.setItem(MIX_DISPLAY_KEY, style);
    if (typeof window !== "undefined") {
      window.dispatchEvent(
        new CustomEvent("mel-mix-display", { detail: style }),
      );
    }
  } catch {
    // ignore
  }
}

export function subscribeMixDisplay(
  onChange: (style: MixDisplayStyle) => void,
): () => void {
  if (typeof window === "undefined") return () => {};
  const handler = (event: Event) => {
    const custom = event as CustomEvent<MixDisplayStyle>;
    if (custom.detail === "cards" || custom.detail === "discs") {
      onChange(custom.detail);
      return;
    }
    onChange(loadMixDisplay());
  };
  const storageHandler = (event: StorageEvent) => {
    if (event.key === MIX_DISPLAY_KEY) onChange(loadMixDisplay());
  };
  window.addEventListener("mel-mix-display", handler);
  window.addEventListener("storage", storageHandler);
  return () => {
    window.removeEventListener("mel-mix-display", handler);
    window.removeEventListener("storage", storageHandler);
  };
}
