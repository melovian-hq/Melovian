// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  eventSocket,
  type ScanCompletePayload,
  type ScanErrorPayload,
  type ScanProgressPayload,
} from "$lib/core/events/ws.svelte";

export type TaskStatus = "running" | "done" | "error";

export interface BackgroundTask {
  id: string;
  title: string;
  status: TaskStatus;
  detail: string;
  startedAt: number;
  updatedAt: number;
}

const MAX_RECENT = 20;

function scanTaskId(libraryId: string): string {
  return `scan:${libraryId}`;
}

class TasksStore {
  items = $state.raw<BackgroundTask[]>([]);

  private wsUnsubscribers: Array<() => void> = [];
  private mixTaskId: string | null = null;
  private lyricsTaskId: string | null = null;

  upsert(input: {
    id: string;
    title: string;
    status?: TaskStatus;
    detail?: string;
  }): BackgroundTask {
    const now = Date.now();
    const existing = this.items.find((item) => item.id === input.id);
    const next: BackgroundTask = {
      id: input.id,
      title: input.title,
      status: input.status ?? existing?.status ?? "running",
      detail: input.detail ?? existing?.detail ?? "",
      startedAt: existing?.startedAt ?? now,
      updatedAt: now,
    };
    this.items = [next, ...this.items.filter((item) => item.id !== input.id)];
    this.trim();
    return next;
  }

  complete(id: string, detail?: string): void {
    const existing = this.items.find((item) => item.id === id);
    if (!existing) return;
    this.upsert({
      id,
      title: existing.title,
      status: "done",
      detail: detail ?? existing.detail,
    });
  }

  fail(id: string, detail?: string): void {
    const existing = this.items.find((item) => item.id === id);
    if (!existing) return;
    this.upsert({
      id,
      title: existing.title,
      status: "error",
      detail: detail ?? existing.detail,
    });
  }

  list(): BackgroundTask[] {
    const running = this.items.filter((item) => item.status === "running");
    const finished = this.items
      .filter((item) => item.status !== "running")
      .slice(0, MAX_RECENT);
    return [...running, ...finished].slice(0, MAX_RECENT);
  }

  bindEvents(): (() => void) | undefined {
    if (this.wsUnsubscribers.length > 0) return;
    this.wsUnsubscribers = [
      eventSocket.on("scan.progress", (event) => {
        const payload = event.payload as ScanProgressPayload;
        if (!payload?.libraryId) return;
        const name = payload.name?.trim() || "Library";
        const phase =
          payload.phase === "reconciling" ? "Reconciling" : "Scanning";
        this.upsert({
          id: scanTaskId(payload.libraryId),
          title: `Scan ${name}`,
          status: "running",
          detail: `${phase} · ${payload.processed} files`,
        });
      }),
      eventSocket.on("scan.complete", (event) => {
        const payload = event.payload as ScanCompletePayload;
        if (!payload?.libraryId) return;
        const name = payload.name?.trim() || "Library";
        this.upsert({
          id: scanTaskId(payload.libraryId),
          title: `Scan ${name}`,
          status: "done",
          detail: `${payload.trackCount} tracks indexed`,
        });
      }),
      eventSocket.on("scan.error", (event) => {
        const payload = event.payload as ScanErrorPayload;
        if (!payload?.libraryId) return;
        const name = payload.name?.trim() || "Library";
        this.upsert({
          id: scanTaskId(payload.libraryId),
          title: `Scan ${name}`,
          status: "error",
          detail: payload.error || "Scan failed",
        });
      }),
    ];
    return () => {
      for (const unsub of this.wsUnsubscribers) unsub();
      this.wsUnsubscribers = [];
    };
  }

  beginMixRegeneration(label = "Refreshing mixes"): string {
    if (this.mixTaskId) {
      this.complete(this.mixTaskId, "Replaced by a newer request");
    }
    const id = `mix:${Date.now()}`;
    this.mixTaskId = id;
    this.upsert({
      id,
      title: label,
      status: "running",
      detail: "Building personalized playlists",
    });
    return id;
  }

  endMixRegeneration(ok: boolean, detail?: string): void {
    const id = this.mixTaskId;
    this.mixTaskId = null;
    if (!id) return;
    if (ok) {
      this.complete(id, detail ?? "Mixes updated");
      return;
    }
    this.fail(id, detail ?? "Mix refresh failed");
  }

  beginLyricsFetch(label = "Fetching lyrics"): string {
    if (this.lyricsTaskId) {
      this.complete(this.lyricsTaskId, "Replaced by a newer request");
    }
    const id = `lyrics:${Date.now()}`;
    this.lyricsTaskId = id;
    this.upsert({
      id,
      title: label,
      status: "running",
      detail: "Looking up providers",
    });
    return id;
  }

  /** Finish a specific lyrics task without touching a newer in-flight one. */
  finishLyricsFetch(taskId: string, ok: boolean, detail?: string): void {
    if (this.lyricsTaskId === taskId) {
      this.lyricsTaskId = null;
    }
    const existing = this.items.find((item) => item.id === taskId);
    if (!existing || existing.status !== "running") return;
    if (ok) {
      this.complete(taskId, detail ?? "Lyrics ready");
      return;
    }
    this.fail(taskId, detail ?? "Lyrics fetch failed");
  }

  endLyricsFetch(ok: boolean, detail?: string): void {
    const id = this.lyricsTaskId;
    if (!id) return;
    this.finishLyricsFetch(id, ok, detail);
  }

  private trim(): void {
    const running = this.items.filter((item) => item.status === "running");
    const finished = this.items
      .filter((item) => item.status !== "running")
      .slice(0, MAX_RECENT);
    this.items = [...running, ...finished];
  }
}

export const tasks = new TasksStore();
