// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export interface ContextMenuItem {
  id: string;
  label: string;
  icon?: string;
  disabled?: boolean;
  danger?: boolean;
  keepOpen?: boolean;
  onclick: () => void;
}

export type ContextMenuEntry =
  ContextMenuItem | { id: string; separator: true };

export interface ContextMenuPosition {
  x: number;
  y: number;
}

export function isEditableContextTarget(target: EventTarget | null): boolean {
  if (!(target instanceof Element)) return false;
  const el = target.closest(
    "input, textarea, select, [contenteditable='true']",
  );
  if (!el) return false;
  if (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement) {
    return !el.readOnly && !el.disabled;
  }
  return true;
}

export function isContextMenuItem(
  entry: ContextMenuEntry,
): entry is ContextMenuItem {
  return !("separator" in entry);
}

export function contextMenuPositionFromEvent(
  event: MouseEvent,
): ContextMenuPosition | null {
  if (isEditableContextTarget(event.target)) return null;
  event.preventDefault();
  event.stopPropagation();
  return { x: event.clientX, y: event.clientY };
}

function parseCssLengthPx(value: string, fallback = 0): number {
  const trimmed = value.trim();
  if (!trimmed) return fallback;
  if (trimmed.endsWith("px")) {
    const parsed = Number.parseFloat(trimmed);
    return Number.isFinite(parsed) ? parsed : fallback;
  }
  if (trimmed.endsWith("rem")) {
    const parsed = Number.parseFloat(trimmed);
    if (!Number.isFinite(parsed)) return fallback;
    const rootFont =
      typeof document !== "undefined"
        ? Number.parseFloat(getComputedStyle(document.documentElement).fontSize)
        : 16;
    return parsed * (Number.isFinite(rootFont) ? rootFont : 16);
  }
  const parsed = Number.parseFloat(trimmed);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function safeAreaInsetBottomPx(): number {
  if (typeof document === "undefined") return 0;
  const value = getComputedStyle(document.documentElement)
    .getPropertyValue("env(safe-area-inset-bottom)")
    .trim();
  return parseCssLengthPx(value, 0);
}

export function getPlayerBarInsetPx(): number {
  if (typeof document === "undefined") return 0;
  const root = document.querySelector(".app-root");
  if (!root) return 0;

  const hasPlayer =
    root.classList.contains("app-root--player") ||
    root.classList.contains("app-root--mini-player");
  const hasMobileNav = root.classList.contains("app-root--mobile-nav");
  if (!hasPlayer && !hasMobileNav) return 0;

  const rootStyles = getComputedStyle(document.documentElement);
  const styles = getComputedStyle(root);
  let inset = 0;

  if (hasPlayer) {
    if (hasMobileNav) {
      const slim =
        rootStyles.getPropertyValue("--jb-slim-player-height").trim() ||
        "3.25rem";
      inset += parseCssLengthPx(slim, 52);
    } else {
      const barHeight = styles
        .getPropertyValue("--jb-player-bar-height")
        .trim();
      inset += parseCssLengthPx(barHeight, 88);
    }
  }

  if (hasMobileNav) {
    const navHeight =
      rootStyles.getPropertyValue("--jb-bottom-nav-height").trim() || "3.5rem";
    inset += parseCssLengthPx(navHeight, 56);
  }

  return inset + safeAreaInsetBottomPx();
}

export function clampContextMenuPosition(
  x: number,
  y: number,
  width: number,
  height: number,
  viewportWidth = typeof window !== "undefined" ? window.innerWidth : 1280,
  viewportHeight = typeof window !== "undefined" ? window.innerHeight : 800,
  pad = 8,
  bottomInset = getPlayerBarInsetPx(),
): ContextMenuPosition {
  const maxY = viewportHeight - height - pad - bottomInset;
  return {
    x: Math.max(pad, Math.min(x, viewportWidth - width - pad)),
    y: Math.max(pad, Math.min(y, maxY)),
  };
}
