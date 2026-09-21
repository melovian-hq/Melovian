// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { Attachment } from "svelte/attachments";

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

// contextMenuPositionForTrigger positions a menu opened from a "more
// actions" button. Pointer clicks anchor at the cursor while keyboard
// activation reports clientX/Y of 0 and anchors to the button instead.
export function contextMenuPositionForTrigger(
  event: MouseEvent,
): ContextMenuPosition | null {
  event.preventDefault();
  event.stopPropagation();
  if (event.clientX !== 0 || event.clientY !== 0) {
    return { x: event.clientX, y: event.clientY };
  }
  const el = event.currentTarget;
  if (!(el instanceof HTMLElement)) return null;
  const rect = el.getBoundingClientRect();
  return { x: rect.left + rect.width / 2, y: rect.bottom + 4 };
}

const LONG_PRESS_MS = 500;
const LONG_PRESS_MOVE_TOLERANCE_PX = 10;

/**
 * Svelte action unifying right-click and touch long-press into one callback.
 * Use instead of oncontextmenu so every menu works on touch devices too.
 * Long-press fires after 500ms without finger movement, suppresses the
 * trailing click so rows do not also run their primary action, and skips
 * editable targets so text fields keep the native menu.
 */
export function contextMenu(
  node: HTMLElement,
  onMenu: (pos: ContextMenuPosition) => void,
) {
  let handler = onMenu;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let startX = 0;
  let startY = 0;
  let fired = false;

  function clearTimer() {
    if (timer === undefined) return;
    clearTimeout(timer);
    timer = undefined;
  }

  function onContextMenu(event: MouseEvent) {
    const pos = contextMenuPositionFromEvent(event);
    if (pos) handler(pos);
  }

  function onTouchStart(event: TouchEvent) {
    fired = false;
    if (event.touches.length !== 1 || isEditableContextTarget(event.target)) {
      return;
    }
    const touch = event.touches[0];
    startX = touch.clientX;
    startY = touch.clientY;
    clearTimer();
    timer = setTimeout(() => {
      timer = undefined;
      fired = true;
      handler({ x: startX, y: startY });
      if (typeof navigator !== "undefined" && navigator.vibrate) {
        navigator.vibrate(10);
      }
    }, LONG_PRESS_MS);
  }

  function onTouchMove(event: TouchEvent) {
    if (timer === undefined) return;
    const touch = event.touches[0];
    if (!touch) return;
    if (
      Math.abs(touch.clientX - startX) > LONG_PRESS_MOVE_TOLERANCE_PX ||
      Math.abs(touch.clientY - startY) > LONG_PRESS_MOVE_TOLERANCE_PX
    ) {
      clearTimer();
    }
  }

  // The click that follows a long-press must not trigger the row action.
  function onClickCapture(event: MouseEvent) {
    if (!fired) return;
    fired = false;
    event.preventDefault();
    event.stopPropagation();
  }

  // Suppress the iOS callout so long-press reaches our timer instead.
  const previousCallout = node.style.getPropertyValue("-webkit-touch-callout");
  node.style.setProperty("-webkit-touch-callout", "none");

  node.addEventListener("contextmenu", onContextMenu);
  node.addEventListener("touchstart", onTouchStart, { passive: true });
  node.addEventListener("touchmove", onTouchMove, { passive: true });
  node.addEventListener("touchend", clearTimer);
  node.addEventListener("touchcancel", clearTimer);
  node.addEventListener("click", onClickCapture, { capture: true });

  return {
    update(next: typeof onMenu) {
      handler = next;
    },
    destroy() {
      clearTimer();
      if (previousCallout) {
        node.style.setProperty("-webkit-touch-callout", previousCallout);
      } else {
        node.style.removeProperty("-webkit-touch-callout");
      }
      node.removeEventListener("contextmenu", onContextMenu);
      node.removeEventListener("touchstart", onTouchStart);
      node.removeEventListener("touchmove", onTouchMove);
      node.removeEventListener("touchend", clearTimer);
      node.removeEventListener("touchcancel", clearTimer);
      node.removeEventListener("click", onClickCapture, { capture: true });
    },
  };
}

/**
 * Attachment adapter for components that cannot take use:contextMenu, like
 * Link. Renders nothing when no handler is given so links without menus keep
 * the native browser context menu.
 */
export function contextMenuAttachment(
  onMenu: ((pos: ContextMenuPosition) => void) | undefined,
): Attachment<HTMLElement> {
  return (node) => {
    if (!onMenu) return;
    const action = contextMenu(node, onMenu);
    return () => action.destroy?.();
  };
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
