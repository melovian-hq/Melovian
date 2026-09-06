// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export type DeviceLinkState =
  "disconnected" | "connecting" | "connected" | "error";

export type SessionRole = "none" | "host" | "guest";

export type DevicePendingOp =
  "transfer" | "play-here" | "create-session" | "join-session" | null;

export interface DeviceRoleState {
  sessionId: string | null;
  isHost: boolean;
  following: boolean;
  role: SessionRole;
}

export function emptyRoleState(): DeviceRoleState {
  return {
    sessionId: null,
    isHost: false,
    following: false,
    role: "none",
  };
}

export function roleFromSelf(
  self:
    | {
        sessionId?: string | null;
        isHost?: boolean;
      }
    | null
    | undefined,
): DeviceRoleState {
  const sessionId = self?.sessionId?.trim() ? self.sessionId : null;
  if (!sessionId) return emptyRoleState();
  if (self?.isHost) {
    return { sessionId, isHost: true, following: false, role: "host" };
  }
  return { sessionId, isHost: false, following: true, role: "guest" };
}

export function applySessionUpdated(
  current: DeviceRoleState,
  selfDeviceId: string,
  payload: Record<string, unknown>,
): DeviceRoleState {
  const action = String(payload.action ?? "");
  const sessionId = String(payload.sessionId ?? "") || null;
  const hostId = String(payload.hostId ?? "");
  const deviceId = String(payload.deviceId ?? "");

  if (action === "ended") {
    if (current.sessionId && sessionId && current.sessionId !== sessionId) {
      return current;
    }
    return emptyRoleState();
  }
  if (action === "created" && hostId === selfDeviceId && sessionId) {
    return { sessionId, isHost: true, following: false, role: "host" };
  }
  if (action === "joined" && deviceId === selfDeviceId && sessionId) {
    return { sessionId, isHost: false, following: true, role: "guest" };
  }
  if (action === "left" && deviceId === selfDeviceId) {
    return emptyRoleState();
  }
  return current;
}

export function linkStateFromSocket(input: {
  connected: boolean;
  connecting?: boolean;
  failed?: boolean;
}): DeviceLinkState {
  if (input.connected) return "connected";
  if (input.failed) return "error";
  if (input.connecting) return "connecting";
  return "disconnected";
}

export function connectionLabel(
  state: DeviceLinkState,
  pending: DevicePendingOp,
): string {
  if (pending === "transfer") return "Transferring playback…";
  if (pending === "play-here") return "Moving playback here…";
  if (pending === "create-session") return "Starting listen together…";
  if (pending === "join-session") return "Joining listen together…";
  switch (state) {
    case "connecting":
      return "Connecting to other devices…";
    case "connected":
      return "Connected";
    case "error":
      return "Device connection failed";
    default:
      return "Not connected";
  }
}

export function sessionRoleLabel(role: SessionRole): string {
  switch (role) {
    case "host":
      return "Hosting listen together";
    case "guest":
      return "Following the host";
    default:
      return "";
  }
}

export const TOGETHER_SOFT_DRIFT_SEC = 0.6;
export const TOGETHER_HARD_DRIFT_SEC = 0.12;
export const TOGETHER_HOST_THROTTLE_MS = 350;
export const DEVICE_PUBLISH_THROTTLE_MS = 1500;

export function compensatedPositionMs(
  state: {
    positionMs?: number;
    durationMs?: number;
    paused?: boolean;
    serverTime?: number;
  },
  nowMs: number,
): number {
  const base = Math.max(0, Number(state.positionMs) || 0);
  if (state.paused || !state.serverTime) return base;
  const elapsed = nowMs - Number(state.serverTime);
  if (!Number.isFinite(elapsed) || elapsed <= 0) return base;
  const next = base + elapsed;
  const duration = Number(state.durationMs) || 0;
  if (duration > 0) return Math.min(next, Math.max(0, duration - 200));
  return next;
}

export function sessionMemberCount(
  devices: Array<{ sessionId?: string | null }>,
  sessionId: string | null,
): number {
  if (!sessionId) return 0;
  return devices.filter((d) => d.sessionId === sessionId).length;
}

export function sessionHostName(
  devices: Array<{
    sessionId?: string | null;
    isHost?: boolean;
    name?: string;
  }>,
  sessionId: string | null,
): string {
  if (!sessionId) return "";
  const host = devices.find((d) => d.sessionId === sessionId && d.isHost);
  return host?.name?.trim() ?? "";
}

export function sessionPresenceLabel(
  role: SessionRole,
  memberCount: number,
  hostName?: string,
): string {
  if (role === "none") return "";
  const others = Math.max(0, memberCount - 1);
  if (role === "host") {
    if (others === 0) return "Listening together · waiting";
    if (others === 1) return "Listening together with 1 other";
    return `Listening together with ${others} others`;
  }
  const name = hostName?.trim();
  if (name) return `Listening with ${name}`;
  return "Listening together";
}

export function sessionMemberNames(
  devices: Array<{
    sessionId?: string | null;
    isHost?: boolean;
    name?: string;
    deviceId: string;
  }>,
  sessionId: string | null,
  selfDeviceId: string,
): string {
  if (!sessionId) return "";
  return devices
    .filter((d) => d.sessionId === sessionId)
    .map((d) => {
      const name = d.name?.trim() || "Device";
      const self = d.deviceId === selfDeviceId ? "you" : "";
      const host = d.isHost ? "host" : "";
      const tag = [self, host].filter(Boolean).join(", ");
      return tag ? `${name} (${tag})` : name;
    })
    .join(" · ");
}

export function sessionMemberDisplayName(input: {
  username?: string | null;
  name?: string | null;
}): string {
  const username = input.username?.trim() ?? "";
  if (username) return username;
  const name = input.name?.trim() ?? "";
  if (name) return name;
  return "Someone";
}

export function sessionJoinedCopy(name: string): string {
  return `${sessionMemberDisplayName({ name })} joined`;
}

export function sessionLeftCopy(name: string, reason?: string): string {
  const label = sessionMemberDisplayName({ name });
  if (reason === "disconnected") return `${label} disconnected`;
  return `${label} left`;
}

export function sessionEndedCopy(reason?: string): string {
  if (reason === "host_disconnected") {
    return "Host disconnected. Listen together ended.";
  }
  if (reason === "host_left") {
    return "Host ended listen together.";
  }
  return "Listen together ended.";
}

export function sessionSelfJoinedCopy(
  crossUser: boolean,
  hostName?: string,
): string {
  if (crossUser) {
    const host = hostName?.trim();
    return host ? `Joined ${host}'s party` : "Joined the party";
  }
  const host = hostName?.trim();
  return host ? `Joined ${host}` : "Joined listen together";
}

export function deviceErrorCopy(code: string, fallback?: string): string {
  switch (code) {
    case "not_registered":
      return "This device is not connected yet. Wait a moment and try again.";
    case "target_offline":
      return "That device is not connected.";
    case "nothing_playing":
      return "Nothing is playing to transfer.";
    case "missing_target":
      return "Pick a device.";
    case "session_missing":
      return "That listen together session is missing.";
    case "session_unavailable":
      return "Could not join. The session ended or is on another account.";
    case "not_hosting":
      return "Start listen together before inviting a device.";
    case "already_member":
      return "That device is already listening together.";
    case "target_hosting":
      return "That device is hosting its own session. Ask them to leave first.";
    case "socket_not_ready":
      return "Not connected to other devices yet.";
    case "apply_failed":
      return "Could not load the transferred track.";
    default:
      return fallback?.trim() || "The device action failed.";
  }
}

export const DEVICE_OP_TIMEOUT_MS = 8000;
