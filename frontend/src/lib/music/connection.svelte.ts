// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import {
  appendConnectionEvent,
  closeLatestConnectionEvent,
  loadConnectionHistory,
  loadConnectionSettings,
  mergeConnectionSettings,
  saveConnectionHistory,
  saveConnectionSettings,
  type ConnectionEvent,
  type ConnectionSettings,
} from "./connection-settings";
import {
  computeReconnectDelayMs,
  phaseForAttempt,
  type ConnectionPhase,
} from "./connection-schedule";
import * as musicApi from "./api";

type ConnectFn = () => Promise<boolean>;
type PingFn = () => Promise<boolean>;
type HealFn = () => Promise<void>;

export interface ConnectionCallbacks {
  onReconnected?: () => void;
  onDisconnected?: () => void;
}

class ConnectionStore {
  settings = $state(loadConnectionSettings());
  history = $state.raw<ConnectionEvent[]>(loadConnectionHistory());

  browserOnline = $state(
    typeof navigator !== "undefined" ? navigator.onLine : true,
  );
  serverOnline = $state(false);
  /**
   * managed is true between init() and dispose(). It marks that this store
   * owns connect/ping/heal probes for the active source, so serverOnline is
   * authoritative and playback may hold the queue on failures. Sources that
   * never init the store (local-only, demo) stay unmanaged and unaffected.
   */
  managed = $state(false);
  phase = $state<ConnectionPhase>("offline");
  reconnectAttempt = $state(0);
  lastDisconnectedAt = $state<number | null>(null);
  lastConnectedAt = $state<number | null>(null);
  nextRetryAt = $state<number | null>(null);

  online = $derived(this.browserOnline && this.serverOnline);
  statusLabel = $derived(this.labelForPhase());

  private reconnectTimer: ReturnType<typeof setTimeout> | undefined;
  private healthTimer: ReturnType<typeof setInterval> | undefined;
  private healTimer: ReturnType<typeof setInterval> | undefined;
  private pendingConnect: ConnectFn | null = null;
  private pendingPing: PingFn | null = null;
  private pendingHeal: HealFn | null = null;
  private callbacks: ConnectionCallbacks = {};
  private listenersBound = false;
  private hadOutage = false;

  init(
    connect: ConnectFn,
    ping: PingFn,
    heal: HealFn,
    callbacks: ConnectionCallbacks = {},
  ) {
    this.pendingConnect = connect;
    this.pendingPing = ping;
    this.pendingHeal = heal;
    this.callbacks = callbacks;
    this.managed = true;
    if (typeof navigator !== "undefined") {
      this.browserOnline = navigator.onLine;
    }
    this.bindListeners();
    this.restartTimers();
    void this.tryConnect(true);
  }

  async loadRemoteSettings(): Promise<void> {
    try {
      const remote = await musicApi.getConnectionSettings();
      if (remote) {
        this.settings = mergeConnectionSettings(remote);
        saveConnectionSettings(this.settings);
      } else {
        this.settings = loadConnectionSettings();
      }
    } catch {
      this.settings = loadConnectionSettings();
    }
    this.restartTimers();
  }

  restartTimers() {
    this.startHealthLoop();
    this.startSelfHealLoop();
  }

  cancelScheduledReconnect() {
    this.stopReconnect();
    this.nextRetryAt = null;
  }

  async updateSettings(partial: Partial<ConnectionSettings>) {
    this.settings = { ...this.settings, ...partial };
    saveConnectionSettings(this.settings);
    try {
      await musicApi.saveConnectionSettingsRemote(this.settings);
    } catch {
      /* local cache remains available offline */
    }
    this.restartTimers();
  }

  async reloadSettings() {
    this.settings = loadConnectionSettings();
    try {
      const remote = await musicApi.getConnectionSettings();
      if (remote) {
        this.settings = mergeConnectionSettings(remote);
        saveConnectionSettings(this.settings);
      }
    } catch {
      /* keep local */
    }
    this.history = loadConnectionHistory();
  }

  onServerConnected() {
    const now = Date.now();
    this.serverOnline = true;
    this.phase = "online";
    this.reconnectAttempt = 0;
    this.nextRetryAt = null;
    this.lastConnectedAt = now;
    this.stopReconnect();

    if (this.settings.rememberHistory && this.lastDisconnectedAt !== null) {
      this.history = closeLatestConnectionEvent(this.history, now);
      saveConnectionHistory(this.history);
    }
    this.lastDisconnectedAt = null;

    if (this.hadOutage) {
      this.hadOutage = false;
      this.callbacks.onReconnected?.();
    }
  }

  onServerDisconnected(reason?: string, options?: { silent?: boolean }) {
    void reason;
    const wasOnline = this.serverOnline;
    const now = Date.now();
    this.serverOnline = false;

    if ((wasOnline || this.phase === "online") && !options?.silent) {
      this.hadOutage = true;
      this.callbacks.onDisconnected?.();
    }

    if (this.lastDisconnectedAt === null) {
      this.lastDisconnectedAt = now;
      if (this.settings.rememberHistory) {
        this.history = appendConnectionEvent(
          this.history,
          { disconnectedAt: now },
          this.settings.maxHistoryEntries,
        );
        saveConnectionHistory(this.history);
      }
    }

    if (!this.browserOnline) {
      this.phase = "offline";
      this.scheduleReconnect();
      return;
    }

    if (this.settings.autoReconnect) {
      this.scheduleReconnect();
    } else {
      this.phase = "offline";
    }
  }

  forceReconnect() {
    this.reconnectAttempt = 0;
    this.stopReconnect();
    void this.tryConnect(true);
  }

  /**
   * teardown detaches the store from the active source without recording an
   * outage. Deliberate sign-out and account deletion are not server
   * disconnects, so no history entry is written, no disconnect callback
   * fires, and no reconnect is scheduled. Runtime state returns to defaults
   * so a later init starts clean instead of inheriting a stale outage.
   */
  teardown() {
    this.stopReconnect();
    this.stopHealthLoop();
    this.stopSelfHealLoop();
    this.managed = false;
    this.pendingConnect = null;
    this.pendingPing = null;
    this.pendingHeal = null;
    this.callbacks = {};
    this.serverOnline = false;
    this.phase = "offline";
    this.reconnectAttempt = 0;
    this.lastDisconnectedAt = null;
    this.nextRetryAt = null;
    this.hadOutage = false;
    if (typeof window !== "undefined") {
      window.removeEventListener("online", this.handleBrowserOnline);
      window.removeEventListener("offline", this.handleBrowserOffline);
    }
    this.listenersBound = false;
  }

  dispose() {
    this.teardown();
  }

  private bindListeners() {
    if (this.listenersBound || typeof window === "undefined") return;
    window.addEventListener("online", this.handleBrowserOnline);
    window.addEventListener("offline", this.handleBrowserOffline);
    this.listenersBound = true;
  }

  private handleBrowserOnline = () => {
    this.browserOnline = true;
    if (
      this.settings.retryOnOnline &&
      !this.serverOnline &&
      this.settings.autoReconnect
    ) {
      this.reconnectAttempt = 0;
      this.stopReconnect();
      void this.tryConnect(true);
    }
  };

  private handleBrowserOffline = () => {
    this.browserOnline = false;
    this.serverOnline = false;
    this.phase = "offline";
    // Mirror the bookkeeping onServerDisconnected does so history and the
    // reconnect banner measure the outage from when the browser dropped,
    // not from whenever the next probe happens to run.
    const now = Date.now();
    if (this.lastDisconnectedAt === null) {
      this.lastDisconnectedAt = now;
      if (this.settings.rememberHistory) {
        this.history = appendConnectionEvent(
          this.history,
          { disconnectedAt: now },
          this.settings.maxHistoryEntries,
        );
        saveConnectionHistory(this.history);
      }
    }
    if (!this.hadOutage) {
      this.hadOutage = true;
      this.callbacks.onDisconnected?.();
    }
    this.scheduleReconnect();
  };

  private scheduleReconnect() {
    if (!this.settings.autoReconnect || !this.pendingConnect) return;
    if (this.reconnectTimer) return;

    const delay = computeReconnectDelayMs(
      this.settings,
      this.reconnectAttempt,
      this.history,
      this.browserOnline,
    );

    this.phase = phaseForAttempt(this.reconnectAttempt, false);
    this.nextRetryAt = Date.now() + delay;

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = undefined;
      void this.tryConnect(false);
    }, delay);
  }

  private async tryConnect(immediate: boolean) {
    if (!this.pendingConnect) return;

    if (!this.browserOnline) {
      this.phase = "offline";
      this.scheduleReconnect();
      return;
    }

    this.phase = phaseForAttempt(this.reconnectAttempt, false);
    this.nextRetryAt = null;

    const ok = await this.pendingConnect();
    if (ok) {
      this.onServerConnected();
      return;
    }

    this.reconnectAttempt += 1;
    this.scheduleReconnect();

    if (!immediate && this.reconnectAttempt === 1) {
      this.phase = "retrying";
    }
  }

  private startHealthLoop() {
    this.stopHealthLoop();
    if (typeof window === "undefined") return;

    this.healthTimer = setInterval(() => {
      if (!this.serverOnline || !this.pendingPing) return;
      void this.pendingPing().then((ok) => {
        if (ok && !this.serverOnline) {
          this.onServerConnected();
        }
      });
    }, this.settings.healthCheckIntervalMs);
  }

  private startSelfHealLoop() {
    this.stopSelfHealLoop();
    if (typeof window === "undefined") return;

    this.healTimer = setInterval(() => {
      if (!this.serverOnline || !this.pendingHeal) return;
      this.phase = "self-healing";
      void this.pendingHeal().finally(() => {
        if (this.serverOnline) this.phase = "online";
      });
    }, this.settings.selfHealIntervalMs);
  }

  private stopReconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = undefined;
    }
  }

  private stopHealthLoop() {
    if (this.healthTimer) {
      clearInterval(this.healthTimer);
      this.healthTimer = undefined;
    }
  }

  private stopSelfHealLoop() {
    if (this.healTimer) {
      clearInterval(this.healTimer);
      this.healTimer = undefined;
    }
  }

  private labelForPhase(): string {
    switch (this.phase) {
      case "online":
        return "Connected";
      case "offline":
        return this.browserOnline ? "Server unreachable" : "Offline";
      case "reconnecting":
        return "Reconnecting";
      case "retrying":
        return "Retrying";
      case "self-healing":
        return "Self-healing";
      default:
        return "Unknown";
    }
  }
}

export const connection = new ConnectionStore();
