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
 * Pages mount AppShell with the sidebar, so a fly/slide moves the sidebar too.
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

/** Pin a leaving page so stacked intros and outros do not grow layout height. */
export function pinOutgoingPage(node: HTMLElement) {
  node.style.position = "absolute";
  node.style.top = "0";
  node.style.left = "0";
  node.style.right = "0";
  node.style.width = "100%";
}
