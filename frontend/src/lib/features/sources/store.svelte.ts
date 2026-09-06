// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { instances } from "$lib/features/instances/store.svelte";
import { localLibraries } from "$lib/features/local-libraries/store.svelte";
import * as sourceApi from "./api";
import type { SourceViewMode } from "./api";

class SourceStore {
  mode = $state<SourceViewMode>("subsonic");
  unifiedAvailable = $state(false);
  multiLocalLibrary = $state(false);

  ready = $derived(instances.ready && localLibraries.ready);

  needsSetup = $derived(
    this.ready &&
      instances.activeId === null &&
      localLibraries.activeId === null &&
      this.mode !== "unified",
  );

  hasUnifiedMode = $derived(this.mode === "unified");

  activeLabel = $derived.by(() => {
    if (this.hasUnifiedMode) {
      return "All sources";
    }
    if (localLibraries.activeId) {
      return localLibraries.active?.name ?? "Local library";
    }
    if (instances.activeId) {
      return (
        instances.active?.name ??
        instances.active?.serverName ??
        "Subsonic server"
      );
    }
    return "Select source";
  });

  hasLocalActive = $derived(
    localLibraries.activeId !== null &&
      (this.hasUnifiedMode || this.mode === "local"),
  );
  hasSubsonicActive = $derived(
    instances.activeId !== null &&
      (this.hasUnifiedMode || this.mode === "subsonic"),
  );

  hasAnySource = $derived(
    instances.items.length > 0 ||
      (localLibraries.enabled && localLibraries.items.length > 0),
  );

  canUseUnified = $derived(
    this.unifiedAvailable ||
      (instances.items.length > 0 &&
        localLibraries.enabled &&
        localLibraries.items.length > 0),
  );

  async refreshStatus() {
    try {
      const status = await sourceApi.fetchSourceStatus();
      this.mode = status.mode;
      this.unifiedAvailable = status.unifiedAvailable;
      this.multiLocalLibrary = status.multiLocalLibrary;
    } catch {
      // keep current mode when status endpoint unavailable
    }
  }

  async setMode(mode: SourceViewMode) {
    const status = await sourceApi.setSourceViewMode(mode);
    this.mode = status.mode;
    this.unifiedAvailable = status.unifiedAvailable;
    this.multiLocalLibrary = status.multiLocalLibrary;
    await Promise.all([instances.refresh(), localLibraries.refresh()]);
  }

  async setMultiLocalLibrary(enabled: boolean) {
    const status = await sourceApi.setMultiLocalLibrary(enabled);
    this.multiLocalLibrary = status.multiLocalLibrary;
    await localLibraries.refresh();
  }
}

export const sources = new SourceStore();
