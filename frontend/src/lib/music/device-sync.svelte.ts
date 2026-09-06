// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { eventSocket } from "$lib/core/events/ws.svelte";
import {
  getOrCreateDeviceId,
  loadDeviceName,
  saveDeviceName,
} from "$lib/music/device-id";
import { music } from "$lib/config/music.svelte";
import { toast } from "$lib/ui/toast.svelte";
import {
  applySessionUpdated,
  compensatedPositionMs,
  connectionLabel,
  DEVICE_OP_TIMEOUT_MS,
  DEVICE_PUBLISH_THROTTLE_MS,
  deviceErrorCopy,
  emptyRoleState,
  linkStateFromSocket,
  roleFromSelf,
  sessionEndedCopy,
  sessionHostName,
  sessionJoinedCopy,
  sessionLeftCopy,
  sessionMemberCount,
  sessionMemberDisplayName,
  sessionMemberNames,
  sessionPresenceLabel,
  sessionSelfJoinedCopy,
  TOGETHER_HARD_DRIFT_SEC,
  TOGETHER_HOST_THROTTLE_MS,
  TOGETHER_SOFT_DRIFT_SEC,
  type DeviceLinkState,
  type DevicePendingOp,
  type DeviceRoleState,
  type SessionRole,
} from "$lib/music/device-sync-protocol";
import {
  createPartyInviteLink,
  invitePartyUser,
  joinPartyByToken,
  partyCoverUrl,
  partyStreamUrl,
} from "$lib/music/party-api";
import { auth } from "$lib/features/auth/store.svelte";
import type { QueueTrack } from "$lib/subsonic/types";

export interface SyncPlaybackSnapshot {
  trackId?: string;
  trackTitle?: string;
  artistName?: string;
  coverArt?: string;
  positionMs: number;
  durationMs?: number;
  paused: boolean;
  queueIds?: string[];
  queueIndex?: number;
  serverTime?: number;
}

export interface SyncDevice {
  deviceId: string;
  name: string;
  username?: string;
  userAgent?: string;
  connectedAt: number;
  lastSeen: number;
  isActivePlayer: boolean;
  playback?: SyncPlaybackSnapshot | null;
  sessionId?: string;
  isHost?: boolean;
}

class DeviceSyncStore {
  devices = $state<SyncDevice[]>([]);
  panelOpen = $state(false);
  deviceId = $state(getOrCreateDeviceId());
  deviceName = $state(loadDeviceName());
  sessionId = $state<string | null>(null);
  isHost = $state(false);
  following = $state(false);
  crossUser = $state(false);
  lastError = $state<string | null>(null);
  pendingOp = $state<DevicePendingOp>(null);
  private started = false;
  private applyingRemote = false;
  private lastPublishMs = 0;
  private unsubscribers: Array<() => void> = [];
  private pendingTimer = 0;
  private pendingTargetId: string | null = null;

  get peers(): SyncDevice[] {
    return this.devices.filter((d) => d.deviceId !== this.deviceId);
  }

  get activePeer(): SyncDevice | null {
    return (
      this.devices.find(
        (d) => d.isActivePlayer && d.deviceId !== this.deviceId,
      ) ?? null
    );
  }

  get selfDevice(): SyncDevice | undefined {
    return this.devices.find((d) => d.deviceId === this.deviceId);
  }

  get isSelfActive(): boolean {
    return !!this.selfDevice?.isActivePlayer;
  }

  get role(): SessionRole {
    if (this.isHost) return "host";
    if (this.following) return "guest";
    return "none";
  }

  get linkState(): DeviceLinkState {
    return linkStateFromSocket({
      connected: eventSocket.connected,
      connecting: eventSocket.connecting,
      failed: eventSocket.failed,
    });
  }

  get statusLabel(): string {
    return connectionLabel(this.linkState, this.pendingOp);
  }

  get sessionMembers(): SyncDevice[] {
    if (!this.sessionId) return [];
    return this.devices.filter((d) => d.sessionId === this.sessionId);
  }

  get sessionMemberCount(): number {
    return sessionMemberCount(this.devices, this.sessionId);
  }

  get inListenTogether(): boolean {
    return !!this.sessionId;
  }

  get sessionLabel(): string {
    if (this.crossUser) {
      const host = sessionHostName(this.devices, this.sessionId);
      if (this.isHost) {
        return this.sessionMemberCount > 1
          ? `Party · ${this.sessionMemberCount}`
          : "Party · waiting";
      }
      return host ? `Party with ${host}` : "Party";
    }
    return sessionPresenceLabel(
      this.role,
      this.sessionMemberCount,
      sessionHostName(this.devices, this.sessionId),
    );
  }

  get sessionMemberLabel(): string {
    return sessionMemberNames(this.devices, this.sessionId, this.deviceId);
  }

  start() {
    if (this.started || typeof window === "undefined") return;
    this.started = true;
    this.deviceId = getOrCreateDeviceId();
    this.deviceName = loadDeviceName();

    this.unsubscribers.push(
      eventSocket.onOpen(() => {
        this.lastError = null;
        eventSocket.send("device.register", {
          deviceId: this.deviceId,
          name: this.deviceName,
          userAgent: navigator.userAgent,
        });
        this.publishPlayback(true);
      }),
    );

    this.unsubscribers.push(
      eventSocket.on("devices.updated", (event) => {
        const payload = event.payload as {
          devices?: SyncDevice[];
        };
        this.devices = payload.devices ?? [];
        const self = this.devices.find((d) => d.deviceId === this.deviceId);
        this.applyRole(roleFromSelf(self));
        this.resolvePendingFromDevices();
      }),
    );

    this.unsubscribers.push(
      eventSocket.on("playback.command", (event) => {
        void this.handleCommand(event.payload as Record<string, unknown>);
      }),
    );

    this.unsubscribers.push(
      eventSocket.on("playback.state", (event) => {
        void this.handlePlaybackState(event.payload as Record<string, unknown>);
      }),
    );

    this.unsubscribers.push(
      eventSocket.on("session.updated", (event) => {
        void this.handleSession(event.payload as Record<string, unknown>);
      }),
    );

    this.unsubscribers.push(
      eventSocket.on("device.error", (event) => {
        this.handleDeviceError(event.payload as Record<string, unknown>);
      }),
    );

    if (typeof window !== "undefined") {
      const timer = window.setInterval(() => {
        if (this.isHost && this.sessionId) {
          this.publishPlayback();
          return;
        }
        if (music.playing) this.publishPlayback();
      }, 1000);
      this.unsubscribers.push(() => clearInterval(timer));
    }

    eventSocket.connect();
  }

  retryConnect() {
    this.lastError = null;
    eventSocket.connect();
  }

  notifyPlaybackChanged() {
    this.publishPlayback(true);
  }

  togglePanel() {
    this.panelOpen = !this.panelOpen;
  }

  rename(name: string) {
    this.deviceName = name.trim() || this.deviceName;
    saveDeviceName(this.deviceName);
    if (!this.sendOrFail("device.rename", { name: this.deviceName })) return;
  }

  takeOver() {
    if (this.isSelfActive) {
      toast.info("Already playing on this device");
      return;
    }
    this.publishPlayback(true);
    if (
      !this.sendOrFail(
        "playback.takeOver",
        {},
        "play-here",
        "Could not play here",
      )
    ) {
      return;
    }
  }

  transferTo(deviceId: string) {
    if (!deviceId) {
      toast.error(deviceErrorCopy("missing_target"));
      return;
    }
    if (deviceId === this.deviceId) {
      this.takeOver();
      return;
    }
    const peer = this.devices.find((d) => d.deviceId === deviceId);
    if (peer?.isActivePlayer) {
      toast.info(`Already playing on ${peer.name}`);
      return;
    }
    this.publishPlayback(true);
    this.pendingTargetId = deviceId;
    if (
      !this.sendOrFail(
        "playback.handoffTo",
        { targetId: deviceId },
        "transfer",
        "Could not transfer playback",
      )
    ) {
      this.pendingTargetId = null;
    }
  }

  remoteCommand(action: string, extra: Record<string, unknown> = {}) {
    if (
      !this.sendOrFail(
        "playback.command",
        { action, ...extra },
        null,
        "Could not send the command",
      )
    ) {
      return;
    }
  }

  createListenTogether() {
    if (
      !this.sendOrFail(
        "session.create",
        {},
        "create-session",
        "Could not start listen together",
      )
    ) {
      return;
    }
  }

  joinListenTogether(sessionId: string) {
    if (!sessionId) {
      toast.error(deviceErrorCopy("session_missing"));
      return;
    }
    if (
      !this.sendOrFail(
        "session.join",
        { sessionId },
        "join-session",
        "Could not join listen together",
      )
    ) {
      return;
    }
  }

  inviteDevice(deviceId: string) {
    if (!deviceId) {
      toast.error(deviceErrorCopy("missing_target"));
      return;
    }
    if (!this.isHost || !this.sessionId) {
      toast.error(deviceErrorCopy("not_hosting"));
      return;
    }
    const peer = this.devices.find((d) => d.deviceId === deviceId);
    if (peer?.sessionId === this.sessionId) {
      toast.info(`${peer.name} is already listening together`);
      return;
    }
    if (
      !this.sendOrFail(
        "session.invite",
        { targetId: deviceId },
        null,
        "Could not invite that device",
      )
    ) {
      return;
    }
    toast.info(`Inviting ${peer?.name || "device"}…`, { duration: 2500 });
  }

  leaveListenTogether() {
    if (
      !this.sendOrFail("session.leave", {}, null, "Could not leave the session")
    ) {
      return;
    }
    this.applyRole(emptyRoleState());
    this.crossUser = false;
    toast.info("You left listen together");
  }

  async createInviteLink(): Promise<string | null> {
    try {
      const link = await createPartyInviteLink();
      this.crossUser = true;
      return link.url;
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Could not create invite",
      );
      return null;
    }
  }

  async copyInviteToClipboard(): Promise<boolean> {
    try {
      const url = await this.createInviteLink();
      if (!url) return false;
      const absolute = new URL(url, window.location.origin).toString();
      await navigator.clipboard.writeText(absolute);
      toast.success("Invite link copied");
      return true;
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not copy link");
      return false;
    }
  }

  async inviteByUsername(username: string): Promise<boolean> {
    try {
      await invitePartyUser(username);
      this.crossUser = true;
      toast.success(`Invited ${username}`, {
        action: {
          label: "Copy link",
          onClick: () => {
            void this.copyInviteToClipboard();
          },
        },
      });
      return true;
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not invite");
      return false;
    }
  }

  async joinByInviteToken(token: string): Promise<boolean> {
    this.beginPending("join-session");
    try {
      const result = await joinPartyByToken(token);
      this.crossUser = true;
      this.applyRole({
        sessionId: result.sessionId,
        isHost: false,
        following: true,
        role: "guest",
      });
      if (result.state) {
        const ok = await this.applySnapshot(result.state, false);
        if (!ok) {
          this.lastError = "Joined, but could not load the host track.";
          toast.error(this.lastError);
        }
      }
      this.clearPending();
      toast.success(sessionSelfJoinedCopy(true));
      return true;
    } catch (err) {
      this.clearPending();
      this.lastError =
        err instanceof Error ? err.message : "Could not join party";
      toast.error(this.lastError);
      return false;
    }
  }

  partyTrackFromSnapshot(
    trackId: string,
    state?: SyncPlaybackSnapshot,
  ): QueueTrack {
    const sessionId = this.sessionId ?? "";
    return {
      id: trackId,
      title: state?.trackTitle || "Track",
      artist: state?.artistName || "",
      coverArt: state?.coverArt
        ? partyCoverUrl(sessionId, state.coverArt)
        : partyCoverUrl(sessionId, trackId),
      duration: state?.durationMs
        ? Math.floor(state.durationMs / 1000)
        : undefined,
      streamUrl: partyStreamUrl(sessionId, trackId),
    };
  }

  dispatchTransport(
    action: "toggle" | "next" | "prev" | "seek",
    extra?: { positionSec?: number },
  ): boolean {
    if (!this.following) return false;
    if (action === "toggle") {
      this.remoteCommand(music.playing ? "pause" : "play");
      return true;
    }
    if (action === "seek") {
      const ms = Math.floor((extra?.positionSec ?? 0) * 1000);
      if (!Number.isFinite(ms)) return true;
      this.remoteCommand("seek", { positionMs: Math.max(0, ms) });
      return true;
    }
    this.remoteCommand(action);
    return true;
  }

  publishPlayback(force = false) {
    if (this.applyingRemote || this.following) return;
    const now = Date.now();
    const minGap =
      this.isHost && this.sessionId
        ? TOGETHER_HOST_THROTTLE_MS
        : DEVICE_PUBLISH_THROTTLE_MS;
    if (!force && now - this.lastPublishMs < minGap) return;
    this.lastPublishMs = now;

    const track = music.currentTrack;
    eventSocket.send("playback.state", {
      trackId: track?.id ?? "",
      trackTitle: track?.title ?? "",
      artistName: track?.artist ?? "",
      coverArt: track?.coverArt ?? track?.albumId ?? track?.id ?? "",
      positionMs: Math.floor(music.currentTime * 1000),
      durationMs: Math.floor((music.duration || 0) * 1000),
      paused: !music.playing,
      queueIds: music.queue.map((t) => t.id),
      queueIndex: music.queueIndex,
    } satisfies SyncPlaybackSnapshot);
  }

  private sendOrFail(
    type: string,
    payload: unknown,
    pending: DevicePendingOp = null,
    failCopy = "Could not reach other devices",
  ): boolean {
    if (!eventSocket.connected) {
      eventSocket.connect();
      this.lastError = deviceErrorCopy("socket_not_ready");
      toast.info("Connecting to other devices…");
      return false;
    }
    const sent = eventSocket.send(type, payload);
    if (!sent) {
      this.lastError = failCopy;
      toast.error(failCopy);
      return false;
    }
    this.lastError = null;
    if (pending) this.beginPending(pending);
    return true;
  }

  private beginPending(op: DevicePendingOp) {
    this.clearPendingTimer();
    this.pendingOp = op;
    if (typeof window === "undefined" || !op) return;
    this.pendingTimer = window.setTimeout(() => {
      if (this.pendingOp !== op) return;
      const message =
        op === "transfer"
          ? "Transfer timed out. The other device may be offline."
          : op === "play-here"
            ? "Could not move playback here in time."
            : op === "join-session"
              ? "Join timed out. The session may have ended."
              : "Listen together did not start in time.";
      this.pendingOp = null;
      this.pendingTargetId = null;
      this.lastError = message;
      toast.error(message);
    }, DEVICE_OP_TIMEOUT_MS);
  }

  private clearPending(successCopy?: string) {
    const hadPending = this.pendingOp !== null;
    this.clearPendingTimer();
    this.pendingOp = null;
    this.pendingTargetId = null;
    if (hadPending && successCopy) toast.success(successCopy);
  }

  private clearPendingTimer() {
    if (this.pendingTimer && typeof window !== "undefined") {
      window.clearTimeout(this.pendingTimer);
    }
    this.pendingTimer = 0;
  }

  private applyRole(next: DeviceRoleState) {
    this.sessionId = next.sessionId;
    this.isHost = next.isHost;
    this.following = next.following;
    if (!next.sessionId) this.crossUser = false;
  }

  private resolvePendingFromDevices() {
    if (this.pendingOp === "play-here" && this.isSelfActive) {
      this.clearPending("Playing on this device");
      return;
    }
    if (this.pendingOp === "transfer" && this.pendingTargetId) {
      const target = this.devices.find(
        (d) => d.deviceId === this.pendingTargetId,
      );
      if (target?.isActivePlayer) {
        this.clearPending(`Playback moved to ${target.name}`);
      }
    }
    if (this.pendingOp === "create-session" && this.isHost && this.sessionId) {
      this.clearPending();
      this.showHostStartedToast();
    }
    if (this.pendingOp === "join-session" && this.following && this.sessionId) {
      this.clearPending();
    }
  }

  private showHostStartedToast() {
    if (auth.enabled && auth.authenticated) {
      toast.info(
        "Invite another account on this server, or wait for your other devices to join.",
        {
          title: "Listening together",
          duration: 12000,
          actions: [
            {
              label: "Copy link",
              onClick: () => {
                void this.copyInviteToClipboard();
              },
            },
            {
              label: "Invite",
              onClick: () => {
                this.panelOpen = true;
              },
            },
          ],
        },
      );
      return;
    }
    toast.success(
      "Listening together started. Other devices can join from Devices.",
    );
  }

  private handleDeviceError(payload: Record<string, unknown>) {
    const code = String(payload.code ?? "");
    const message = deviceErrorCopy(code, String(payload.message ?? ""));
    this.lastError = message;
    this.clearPendingTimer();
    this.pendingOp = null;
    this.pendingTargetId = null;
    toast.error(message);
  }

  private async handleCommand(payload: Record<string, unknown>) {
    const targetId = String(payload.targetId ?? "");
    const action = String(payload.action ?? "");
    if (targetId && targetId !== this.deviceId) return;

    this.applyingRemote = true;
    try {
      switch (action) {
        case "pause_release":
          if (music.playing) music.togglePlay();
          toast.info("Playback moved to another device");
          break;
        case "take_over": {
          const state = payload.state as
            SyncPlaybackSnapshot | null | undefined;
          this.applyRole(emptyRoleState());
          if (!state || !state.trackId) {
            this.lastError = deviceErrorCopy("nothing_playing");
            toast.error(this.lastError);
            break;
          }
          const ok = await this.applySnapshot(state);
          if (ok) {
            toast.success("Playing here");
            this.clearPending();
          } else {
            this.lastError = deviceErrorCopy("apply_failed");
            toast.error(this.lastError);
          }
          break;
        }
        case "play":
          if (!music.playing) music.togglePlay();
          break;
        case "pause":
          if (music.playing) music.togglePlay();
          break;
        case "next":
          music.next();
          break;
        case "prev":
          music.previous();
          break;
        case "seek": {
          const ms = Number(payload.positionMs ?? 0);
          if (Number.isFinite(ms)) music.seek(ms / 1000);
          break;
        }
        default:
          break;
      }
    } catch (err) {
      this.lastError = deviceErrorCopy("apply_failed");
      toast.error(this.lastError);
      void err;
    } finally {
      this.applyingRemote = false;
      this.publishPlayback(true);
    }
  }

  private async handlePlaybackState(payload: Record<string, unknown>) {
    const together = !!payload.together;
    const deviceId = String(payload.deviceId ?? "");
    if (deviceId === this.deviceId) return;
    if (!together || !this.following || this.isHost) return;
    const state = payload.state as SyncPlaybackSnapshot | undefined;
    if (!state) return;
    const ok = await this.applySnapshot(state, true);
    if (!ok) {
      this.lastError = "Could not follow the host track.";
      toast.error(this.lastError);
    }
  }

  private async handleSession(payload: Record<string, unknown>) {
    const previous = {
      sessionId: this.sessionId,
      isHost: this.isHost,
      following: this.following,
      role: this.role,
    };
    const next = applySessionUpdated(previous, this.deviceId, payload);
    this.applyRole(next);
    if (payload.crossUser === true) this.crossUser = true;

    const action = String(payload.action ?? "");
    const actorId = String(payload.deviceId ?? "");
    const actorName = sessionMemberDisplayName({
      username: String(payload.username ?? ""),
      name: String(payload.name ?? ""),
    });

    if (action === "created" && next.role === "host") {
      const offerInvite = this.pendingOp === "create-session";
      this.clearPending();
      if (offerInvite) this.showHostStartedToast();
      this.publishPlayback(true);
      return;
    }

    if (action === "joined") {
      if (actorId === this.deviceId && next.role === "guest") {
        const state = payload.state as SyncPlaybackSnapshot | undefined;
        if (state) {
          const ok = await this.applySnapshot(state, false);
          if (!ok) {
            this.lastError = "Joined, but could not load the host track.";
            toast.error(this.lastError);
          }
        }
        const hadPending = this.pendingOp !== null;
        this.clearPending();
        if (hadPending) {
          toast.success(
            sessionSelfJoinedCopy(
              this.crossUser,
              sessionHostName(this.devices, this.sessionId),
            ),
          );
        }
        return;
      }
      if (
        previous.role !== "none" &&
        actorId &&
        actorId !== this.deviceId &&
        previous.sessionId ===
          (String(payload.sessionId ?? "") || previous.sessionId)
      ) {
        toast.info(sessionJoinedCopy(actorName), { duration: 4500 });
        if (this.isHost) this.publishPlayback(true);
      }
      return;
    }

    if (action === "left") {
      if (actorId === this.deviceId || previous.role === "none") return;
      if (
        previous.sessionId &&
        payload.sessionId &&
        previous.sessionId !== String(payload.sessionId)
      ) {
        return;
      }
      toast.info(sessionLeftCopy(actorName, String(payload.reason ?? "")), {
        duration: 4500,
      });
      return;
    }

    if (action === "ended" && previous.role !== "none") {
      this.crossUser = false;
      if (previous.isHost && actorId === this.deviceId) return;
      toast.info(sessionEndedCopy(String(payload.reason ?? "")));
    }
  }

  private async applySnapshot(
    state: SyncPlaybackSnapshot,
    soft = false,
  ): Promise<boolean> {
    this.applyingRemote = true;
    try {
      const targetSec = compensatedPositionMs(state, Date.now()) / 1000;
      const paused = !!state.paused;
      const queueIds = state.queueIds;
      const currentIds = music.queue.map((t) => t.id).join(",");
      const nextIds = queueIds?.join(",") ?? "";
      const queueChanged =
        !!queueIds && queueIds.length > 0 && currentIds !== nextIds;
      const indexChanged =
        typeof state.queueIndex === "number" &&
        state.queueIndex !== music.queueIndex;
      const trackChanged =
        !!state.trackId && music.currentTrack?.id !== state.trackId;
      const changingTrack = queueChanged || indexChanged || trackChanged;

      if (changingTrack) {
        music.armStartPosition(targetSec, paused);
        if (this.crossUser && this.following) {
          if (queueChanged && queueIds) {
            const tracks = queueIds.map((id) =>
              this.partyTrackFromSnapshot(
                id,
                id === state.trackId ? state : undefined,
              ),
            );
            music.playTracks(tracks, state.queueIndex ?? 0);
            if (music.queue.length === 0) return false;
          } else if (indexChanged) {
            music.playQueueIndex(state.queueIndex ?? 0);
          } else if (trackChanged && state.trackId) {
            music.playTracks(
              [this.partyTrackFromSnapshot(state.trackId, state)],
              0,
            );
          } else {
            return false;
          }
          return true;
        }
        if (queueChanged && queueIds) {
          await music.applyRemoteQueue(queueIds, state.queueIndex ?? 0);
          if (music.queue.length === 0) return false;
        } else if (indexChanged) {
          music.playQueueIndex(state.queueIndex ?? 0);
        } else if (trackChanged && state.trackId) {
          await music.playTrackById(state.trackId);
        } else {
          return false;
        }
        return true;
      }

      if (!state.trackId && (!queueIds || queueIds.length === 0)) {
        return false;
      }

      const drift = Math.abs(music.currentTime - targetSec);
      const threshold = soft
        ? TOGETHER_SOFT_DRIFT_SEC
        : TOGETHER_HARD_DRIFT_SEC;
      if (drift > threshold) {
        music.seek(targetSec);
      }

      if (paused && music.playing) music.togglePlay();
      if (!paused && !music.playing) music.togglePlay();
      return true;
    } catch (err) {
      void err;
      return false;
    } finally {
      queueMicrotask(() => {
        this.applyingRemote = false;
      });
    }
  }
}

export const deviceSync = new DeviceSyncStore();
