// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { PersistedState } from "runed";
import { StorageKeys } from "$lib/brand";

const MOBILE_MEDIA = "(max-width: 768px)";

// Created lazily because localStorage may not exist yet at module load.
// JSON.stringify(true) writes "true", matching the format this key always used.
let collapsedState: PersistedState<boolean> | undefined;

function collapsed(): PersistedState<boolean> {
  collapsedState ??= new PersistedState<boolean>(
    StorageKeys.sidebarCollapsed,
    false,
  );
  return collapsedState;
}

function readMobileViewport(): boolean {
  if (typeof window === "undefined") return false;
  return window.matchMedia(MOBILE_MEDIA).matches;
}

class LayoutStore {
  sidebarOpen = $state(false);
  tvMode = $state(false);
  isMobileViewport = $state(readMobileViewport());

  get sidebarCollapsed() {
    return collapsed().current;
  }

  set sidebarCollapsed(value: boolean) {
    collapsed().current = value;
  }

  setMobileViewport(mobile: boolean) {
    this.isMobileViewport = mobile;
  }

  toggleCollapsed() {
    this.sidebarCollapsed = !this.sidebarCollapsed;
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
