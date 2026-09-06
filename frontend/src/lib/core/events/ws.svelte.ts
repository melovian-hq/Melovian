// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { resolveMediaUrl } from "$lib/config/runtime";
import { getOrCreateDeviceId } from "$lib/music/device-id";

export type WSEventType =
  | "scan.progress"
  | "scan.complete"
  | "scan.error"
  | "library.updated"
  | "devices.updated"
  | "playback.state"
  | "playback.command"
  | "session.updated"
  | "device.error";

export interface WSEvent<T = unknown> {
  type: WSEventType | string;
  payload: T;
}

export interface ScanProgressPayload {
  libraryId: string;
  name?: string;
  processed: number;
  phase: string;
}

export interface ScanCompletePayload {
  libraryId: string;
  name?: string;
  trackCount: number;
}

export interface ScanErrorPayload {
  libraryId: string;
  name?: string;
  error: string;
}

export interface LibraryUpdatedPayload {
  libraryId: string;
  name?: string;
  trackCount: number;
  missingCount: number;
  duplicateCount: number;
}

type EventHandler = (event: WSEvent) => void;

const INITIAL_BACKOFF_MS = 1000;
const MAX_BACKOFF_MS = 30000;

class EventSocketStore {
  connected = $state(false);
  connecting = $state(false);
  failed = $state(false);
  private handlers = new Map<string, Set<EventHandler>>();
  private socket: WebSocket | null = null;
  private disposed = false;
  private backoffMs = INITIAL_BACKOFF_MS;
  private reconnectTimer = 0;
  private openHandlers = new Set<() => void>();
  private generation = 0;

  connect() {
    if (typeof window === "undefined" || this.disposed) return;
    this.clearReconnect();
    if (
      this.socket &&
      (this.socket.readyState === WebSocket.OPEN ||
        this.socket.readyState === WebSocket.CONNECTING)
    ) {
      return;
    }

    const deviceId = getOrCreateDeviceId();
    const url = resolveMediaUrl(
      `/api/ws?deviceId=${encodeURIComponent(deviceId)}`,
    ).replace(/^http/, "ws");
    const generation = ++this.generation;
    this.connecting = true;
    this.failed = false;
    const socket = new WebSocket(url);
    this.socket = socket;

    socket.addEventListener("open", () => {
      if (generation !== this.generation) return;
      this.connected = true;
      this.connecting = false;
      this.failed = false;
      this.backoffMs = INITIAL_BACKOFF_MS;
      for (const handler of this.openHandlers) handler();
    });

    socket.addEventListener("message", (event) => {
      if (generation !== this.generation) return;
      try {
        const parsed = JSON.parse(String(event.data)) as WSEvent;
        if (!parsed?.type) return;
        this.dispatch(parsed);
      } catch (err) {
        void err;
      }
    });

    socket.addEventListener("close", () => {
      if (generation !== this.generation) return;
      this.connected = false;
      this.connecting = false;
      this.socket = null;
      this.scheduleReconnect();
    });

    socket.addEventListener("error", () => {
      if (generation !== this.generation) return;
      this.failed = true;
      this.connecting = false;
      socket.close();
    });
  }

  /** Soft close. connect() can reopen afterward. */
  disconnect() {
    this.clearReconnect();
    this.generation++;
    const socket = this.socket;
    this.socket = null;
    this.connected = false;
    this.connecting = false;
    socket?.close();
  }

  /** Permanent teardown until a new store is created. */
  dispose() {
    this.disposed = true;
    this.disconnect();
  }

  send(type: string, payload: unknown = {}) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) return false;
    this.socket.send(JSON.stringify({ type, payload }));
    return true;
  }

  onOpen(handler: () => void): () => void {
    this.openHandlers.add(handler);
    if (this.connected) handler();
    return () => this.openHandlers.delete(handler);
  }

  on(type: string, handler: EventHandler): () => void {
    let set = this.handlers.get(type);
    if (!set) {
      set = new Set();
      this.handlers.set(type, set);
    }
    set.add(handler);
    return () => set?.delete(handler);
  }

  private dispatch(event: WSEvent) {
    const typeHandlers = this.handlers.get(event.type);
    typeHandlers?.forEach((handler) => handler(event));
    const allHandlers = this.handlers.get("*");
    allHandlers?.forEach((handler) => handler(event));
  }

  private scheduleReconnect() {
    if (this.disposed || typeof window === "undefined") return;
    this.clearReconnect();
    this.reconnectTimer = window.setTimeout(() => {
      this.backoffMs = Math.min(this.backoffMs * 2, MAX_BACKOFF_MS);
      this.connect();
    }, this.backoffMs);
  }

  private clearReconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = 0;
    }
  }
}

export const eventSocket = new EventSocketStore();
