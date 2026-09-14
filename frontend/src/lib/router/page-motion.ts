// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { cubicOut } from "svelte/easing";
import { prefersReducedMotion } from "svelte/motion";
import type { FadeParams, FlyParams } from "svelte/transition";

/** Matches --jb-duration-slow in tokens.css */
export const PAGE_DURATION_MS = 240;
/** Between normal and slow for tab panels */
export const TAB_DURATION_MS = 200;
/** Matches --jb-duration-normal in tokens.css */
export const OVERLAY_DURATION_MS = 160;
export const PAGE_SLIDE_PX = 9;
export const TAB_SLIDE_PX = 8;

const SETTINGS_SHELL_PATHS = new Set([
  "/settings",
  "/settings/:tab",
  "/instances",
]);

function motionScale(): number {
  return prefersReducedMotion.current ? 0 : 1;
}

/**
 * Page enter. Fade only.
 * The app shell persists above the outlet, so a fly/slide would move the
 * content while the sidebar stays put. Fade keeps the swap clean.
 */
export function pageInFly(): FadeParams {
  const m = motionScale();
  return {
    duration: PAGE_DURATION_MS * m,
    easing: cubicOut,
  };
}

/** Soft enter fade for nested panels if needed. */
export function pageInFade(): FadeParams {
  const m = motionScale();
  return {
    duration: PAGE_DURATION_MS * m,
    easing: cubicOut,
  };
}

/** Shorter leave so the incoming page leads the crossfade. */
export function pageOutFade(): FadeParams {
  const m = motionScale();
  return {
    duration: Math.round(PAGE_DURATION_MS * 0.65) * m,
    easing: cubicOut,
  };
}

/** Soft enter for settings tab panels. */
export function tabInFly(): FlyParams {
  const m = motionScale();
  return {
    duration: TAB_DURATION_MS * m,
    y: TAB_SLIDE_PX * m,
    easing: cubicOut,
  };
}

export function tabOutFade(): FadeParams {
  const m = motionScale();
  return {
    duration: Math.round(TAB_DURATION_MS * 0.7) * m,
    easing: cubicOut,
  };
}

/** Brief fade for source-switch and similar overlays. */
export function overlayFade(): FadeParams {
  const m = motionScale();
  return {
    duration: OVERLAY_DURATION_MS * m,
    easing: cubicOut,
  };
}

/**
 * Stable view key for the route outlet.
 * Settings shell routes share one key so tab changes only update props.
 * Param routes include params so detail to detail gets a soft transition.
 */
export function routeViewKey(
  path: string,
  params: Record<string, string>,
): string {
  if (SETTINGS_SHELL_PATHS.has(path)) {
    return "settings-shell";
  }
  const keys = Object.keys(params).sort();
  if (keys.length === 0) {
    return path;
  }
  return `${path}:${keys.map((k) => `${k}=${params[k]}`).join(",")}`;
}

export interface ViewBox {
  top: number;
  right: number;
  bottom: number;
  left: number;
  width: number;
  height: number;
}

export interface ClipBox {
  top: number;
  right: number;
  bottom: number;
  left: number;
}

export interface PinnedViewSnapshot {
  view: ViewBox;
  clip: ClipBox;
  /**
   * scrollTop of the shell scroller when the snapshot was taken. The view
   * rect is viewport-relative, so a later scroll moves the painted page by
   * the scroll delta and the pin must subtract it again.
   */
  scrollTop?: number;
}

function rectToBox(rect: ViewBox): ViewBox {
  return {
    top: rect.top,
    right: rect.right,
    bottom: rect.bottom,
    left: rect.left,
    width: rect.width,
    height: rect.height,
  };
}

/**
 * Visible region for content inside the shared shell scroller.
 * Overflow clips at the scroller padding box, so a pinned page must be
 * clipped the same way to keep lower content from leaking over chrome.
 */
export function scrollerClipBox(node: HTMLElement): ClipBox | null {
  const scroller = node.closest(".app-shell__content");
  if (!(scroller instanceof HTMLElement)) return null;
  const bounds = scroller.getBoundingClientRect();
  const left = bounds.left + scroller.clientLeft;
  const top = bounds.top + scroller.clientTop;
  return {
    top,
    right: left + scroller.clientWidth,
    bottom: top + scroller.clientHeight,
    left,
  };
}

/**
 * Snapshot the route view geometry while it is stable. The next commit can
 * flip the shell content padding instantly (compact to fill), which shifts
 * the outlet under a leaving page mid crossfade. Pinning uses this last
 * stable snapshot so the outgoing page stays exactly where it was painted.
 */
export function captureViewSnapshot(view: HTMLElement): PinnedViewSnapshot {
  const rect = rectToBox(view.getBoundingClientRect());
  const scroller = view.closest(".app-shell__content");
  const clip = scrollerClipBox(view) ?? {
    top: rect.top,
    right: rect.right,
    bottom: rect.bottom,
    left: rect.left,
  };
  return {
    view: rect,
    clip,
    scrollTop: scroller instanceof HTMLElement ? scroller.scrollTop : 0,
  };
}

/** Clip-path insets that keep rect inside clip, clamped to zero. */
export function clipInsets(rect: ClipBox, clip: ClipBox): ClipBox {
  return {
    top: Math.max(0, clip.top - rect.top),
    right: Math.max(0, rect.right - clip.right),
    bottom: Math.max(0, rect.bottom - clip.bottom),
    left: Math.max(0, clip.left - rect.left),
  };
}

/**
 * Pin a leaving page so it keeps its last painted position while the
 * incoming page commits. Fixed positioning plus a clip-path makes it immune
 * to shell padding flips and scroll resets that otherwise move a pinned
 * absolute page mid crossfade. Without a snapshot the current rect is
 * measured, which still freezes position and flow removal.
 */
export function pinOutgoingPage(
  node: HTMLElement,
  snapshot?: PinnedViewSnapshot | null,
) {
  const captured = snapshot?.view ?? rectToBox(node.getBoundingClientRect());
  // The snapshot rect is viewport-relative at capture time. If the shell
  // scroller moved since (scroll does not fire the ResizeObserver that
  // refreshes snapshots), shift the rect by the scroll delta so the pinned
  // page lands where it is actually painted now.
  let scrollShift = 0;
  if (snapshot) {
    const scroller = node.closest(".app-shell__content");
    const currentScrollTop =
      scroller instanceof HTMLElement ? scroller.scrollTop : 0;
    scrollShift = currentScrollTop - (snapshot.scrollTop ?? 0);
  }
  const rect: ViewBox = {
    ...captured,
    top: captured.top - scrollShift,
    bottom: captured.bottom - scrollShift,
  };
  const clip = snapshot?.clip ??
    scrollerClipBox(node) ?? {
      top: rect.top,
      right: rect.right,
      bottom: rect.bottom,
      left: rect.left,
    };
  const inset = clipInsets(rect, clip);
  node.style.position = "fixed";
  node.style.top = `${rect.top}px`;
  node.style.left = `${rect.left}px`;
  node.style.width = `${rect.width}px`;
  node.style.height = `${rect.height}px`;
  node.style.margin = "0";
  node.style.pointerEvents = "none";
  node.style.clipPath = `inset(${inset.top}px ${inset.right}px ${inset.bottom}px ${inset.left}px)`;
}

/** Reset the shared shell scroll container for the incoming page. */
export function resetPageScroll(node: HTMLElement) {
  const scroller = node.closest(".app-shell__content");
  if (scroller instanceof HTMLElement) {
    scroller.scrollTop = 0;
    scroller.scrollLeft = 0;
  }
}
