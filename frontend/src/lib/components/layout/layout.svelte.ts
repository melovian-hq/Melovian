// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { StorageKeys } from "$lib/brand";

const SIDEBAR_COLLAPSED_KEY = StorageKeys.sidebarCollapsed;
const MOBILE_MEDIA = "(max-width: 768px)";

function loadCollapsed(): boolean {
  try {
    return localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === "true";
  } catch {
    return false;
  }
}

function readMobileViewport(): boolean {
  if (typeof window === "undefined") return false;
  return window.matchMedia(MOBILE_MEDIA).matches;
}

class LayoutStore {
  sidebarCollapsed = $state(loadCollapsed());
  sidebarOpen = $state(false);
  tvMode = $state(false);
  isMobileViewport = $state(readMobileViewport());

  setMobileViewport(mobile: boolean) {
    this.isMobileViewport = mobile;
  }

  toggleCollapsed() {
    this.sidebarCollapsed = !this.sidebarCollapsed;
    try {
      localStorage.setItem(
        SIDEBAR_COLLAPSED_KEY,
        String(this.sidebarCollapsed),
      );
    } catch {
      /* ignore */
    }
  }

  openSidebar() {
    this.sidebarOpen = true;
  }

  closeSidebar() {
    this.sidebarOpen = false;
  }

  toggleSidebar() {
    this.sidebarOpen = !this.sidebarOpen;
  }

  enterTvMode() {
    this.tvMode = true;
  }

  exitTvMode() {
    this.tvMode = false;
  }

  toggleTvMode() {
    this.tvMode = !this.tvMode;
  }
}

export const layout = new LayoutStore();
export { MOBILE_MEDIA };
