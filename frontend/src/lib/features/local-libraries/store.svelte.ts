// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as instanceApi from "../instances/api";
import * as libraryApi from "./api";
import { music } from "$lib/config/music.svelte";
import { resetSubsonicDetailCaches } from "$lib/subsonic/detail-cache";
import {
  eventSocket,
  type LibraryUpdatedPayload,
  type ScanCompletePayload,
  type ScanErrorPayload,
  type ScanProgressPayload,
} from "$lib/core/events/ws.svelte";
import { toast } from "$lib/ui/toast.svelte";
import type {
  LocalLibrary,
  LocalLibraryConfig,
  LocalLibraryInput,
} from "./types";

const POLL_MS = 1500;

function delay(ms: number) {
  return new Promise<void>((resolve) => setTimeout(resolve, ms));
}

class LocalLibraryStore {
  items = $state.raw<LocalLibrary[]>([]);
  activeId = $state<string | null>(null);
  config = $state.raw<LocalLibraryConfig>({
    enabled: false,
    defaultPath: "",
    allowCustomPath: false,
  });
  loading = $state(true);
  switching = $state(false);
  error = $state<string | null>(null);
  watchingScans = $state(false);
  wsConnected = $state(false);

  private scanWatchIds = new Set<string>();
  private scanNames = new Map<string, string>();
  private scanResolvers = new Map<
    string,
    {
      resolve: (library: LocalLibrary) => void;
      reject: (error: Error) => void;
    }
  >();
  private wsUnsubscribers: Array<() => void> = [];

  active = $derived(
    this.items.find((item) => item.id === this.activeId) ?? null,
  );

  scanningItems = $derived(
    this.items.filter((item) => item.scanStatus === "scanning"),
  );

  ready = $derived(!this.loading);
  enabled = $derived(this.config.enabled);

  private initialized = false;

  bindEvents() {
    if (this.wsUnsubscribers.length > 0) return;
    this.wsUnsubscribers = [
      eventSocket.on("scan.progress", (event) => {
        const payload = event.payload as ScanProgressPayload;
        this.applyScanProgress(payload);
      }),
      eventSocket.on("scan.complete", (event) => {
        const payload = event.payload as ScanCompletePayload;
        void this.handleScanComplete(payload.libraryId);
      }),
      eventSocket.on("scan.error", (event) => {
        const payload = event.payload as ScanErrorPayload;
        void this.handleScanError(payload.libraryId, payload.error);
      }),
      eventSocket.on("library.updated", (event) => {
        const payload = event.payload as LibraryUpdatedPayload;
        void this.handleLibraryUpdated(payload.libraryId);
      }),
    ];
    const syncConnected = () => {
      this.wsConnected = eventSocket.connected;
    };
    syncConnected();
    const interval = window.setInterval(syncConnected, 1000);
    return () => {
      for (const unsub of this.wsUnsubscribers) unsub();
      this.wsUnsubscribers = [];
      clearInterval(interval);
    };
  }

  private applyScanProgress(payload: ScanProgressPayload) {
    if (!this.scanWatchIds.has(payload.libraryId)) return;
    this.items = this.items.map((item) =>
      item.id === payload.libraryId
        ? {
            ...item,
            scanStatus: "scanning",
            scanProgress: {
              processed: payload.processed,
              phase:
                payload.phase === "reconciling" ? "reconciling" : "scanning",
            },
          }
        : item,
    );
  }

  private async handleScanComplete(libraryId: string) {
    if (!this.scanWatchIds.has(libraryId)) return;
    await this.refresh();
    const library = this.items.find((item) => item.id === libraryId) ?? null;
    this.finishScan(libraryId, library, false);
  }

  private async handleScanError(libraryId: string, message: string) {
    if (!this.scanWatchIds.has(libraryId)) return;
    await this.refresh();
    const library =
      this.items.find((item) => item.id === libraryId) ??
      ({
        id: libraryId,
        scanError: message,
        scanStatus: "error",
      } as LocalLibrary);
    this.finishScan(libraryId, library, true);
  }

  private async handleLibraryUpdated(libraryId: string) {
    if (!this.config.enabled) return;
    await this.refresh();
    if (libraryId === this.activeId) {
      resetSubsonicDetailCaches();
    }
  }

  async init() {
    if (this.initialized) {
      await this.refresh();
      return;
    }
    this.initialized = true;
    this.loading = true;
    this.error = null;
    try {
      this.config = await libraryApi.fetchLocalLibraryConfig();
      if (!this.config.enabled) {
        this.items = [];
        this.activeId = null;
        return;
      }
      const [items, active] = await Promise.all([
        libraryApi.listLocalLibraries(),
        libraryApi.getActiveLocalLibrary(),
      ]);
      this.items = items;
      this.activeId = active?.id ?? null;
      for (const item of items) {
        if (item.scanStatus === "scanning") {
          this.trackScan(item.id, item.name);
        }
      }
    } catch (err) {
      this.error =
        err instanceof Error ? err.message : "Failed to load local libraries";
    } finally {
      this.loading = false;
    }
  }

  async refresh() {
    if (!this.config.enabled) return;
    const [items, active] = await Promise.all([
      libraryApi.listLocalLibraries(),
      libraryApi.getActiveLocalLibrary(),
    ]);
    this.items = items;
    this.activeId = active?.id ?? null;
  }

  private trackScan(id: string, name: string) {
    this.scanWatchIds.add(id);
    this.scanNames.set(id, name);
    void this.ensureScanWatcher();
  }

  private finishScan(
    id: string,
    library: LocalLibrary | null,
    failed: boolean,
  ) {
    this.scanWatchIds.delete(id);
    const name = this.scanNames.get(id) ?? library?.name ?? "Library";
    this.scanNames.delete(id);

    const pending = this.scanResolvers.get(id);
    if (pending) {
      this.scanResolvers.delete(id);
      if (failed || !library) {
        pending.reject(new Error(library?.scanError ?? "Scan failed"));
      } else {
        pending.resolve(library);
      }
    }

    if (failed) {
      toast.error(
        library?.scanError
          ? `Scan failed for ${name}: ${library.scanError}`
          : `Scan failed for ${name}`,
      );
      return;
    }

    if (library) {
      toast.success(
        `Finished scanning ${name} (${library.trackCount.toLocaleString()} tracks)`,
      );
      if (library.id === this.activeId) {
        resetSubsonicDetailCaches();
        void music.connect({ force: true });
      }
    }
  }

  waitForScan(id: string): Promise<LocalLibrary> {
    const current = this.items.find((item) => item.id === id);
    if (current && current.scanStatus !== "scanning") {
      return Promise.resolve(current);
    }
    return new Promise((resolve, reject) => {
      this.scanResolvers.set(id, { resolve, reject });
    });
  }

  private async ensureScanWatcher() {
    if (this.watchingScans) return;
    this.watchingScans = true;

    while (this.scanWatchIds.size > 0) {
      if (!eventSocket.connected) {
        await this.refresh();

        for (const id of [...this.scanWatchIds]) {
          const library = this.items.find((item) => item.id === id);
          if (!library) {
            this.finishScan(id, null, true);
            continue;
          }
          if (library.scanStatus === "scanning") continue;
          this.finishScan(id, library, library.scanStatus === "error");
        }
      }

      if (this.scanWatchIds.size > 0 && !eventSocket.connected) {
        await delay(POLL_MS);
      } else if (this.scanWatchIds.size > 0) {
        await delay(500);
      }
    }

    this.watchingScans = false;
  }

  async add(input: LocalLibraryInput) {
    const created = await libraryApi.createLocalLibrary(input);
    await this.refresh();
    this.trackScan(created.id, created.name);
    const activeInstance = await instanceApi
      .getActiveInstance()
      .catch(() => null);
    if (!this.activeId && !activeInstance?.id) {
      await this.activate(created.id);
    }
    return created;
  }

  async activate(id: string) {
    this.switching = true;
    try {
      await libraryApi.activateLocalLibrary(id);
      await this.refresh();
    } finally {
      this.switching = false;
    }
  }

  async scan(id: string) {
    const existing = this.items.find((item) => item.id === id);
    const name = existing?.name ?? "Library";
    const wait = this.waitForScan(id);
    await libraryApi.scanLocalLibrary(id);
    await this.refresh();
    this.trackScan(id, name);
    return wait;
  }

  async remove(id: string) {
    this.scanWatchIds.delete(id);
    this.scanNames.delete(id);
    this.scanResolvers.delete(id);
    await libraryApi.deleteLocalLibrary(id);
    await this.refresh();
  }
}

export const localLibraries = new LocalLibraryStore();
