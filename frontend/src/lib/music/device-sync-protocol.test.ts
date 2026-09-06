// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  applySessionUpdated,
  compensatedPositionMs,
  connectionLabel,
  deviceErrorCopy,
  emptyRoleState,
  linkStateFromSocket,
  roleFromSelf,
  sessionHostName,
  sessionJoinedCopy,
  sessionLeftCopy,
  sessionEndedCopy,
  sessionMemberCount,
  sessionMemberDisplayName,
  sessionMemberNames,
  sessionPresenceLabel,
  sessionRoleLabel,
  sessionSelfJoinedCopy,
} from "./device-sync-protocol";

describe("device-sync-protocol", () => {
  it("maps self device session into host or guest role", () => {
    expect(roleFromSelf(undefined)).toEqual(emptyRoleState());
    expect(roleFromSelf({ sessionId: "  " })).toEqual(emptyRoleState());
    expect(roleFromSelf({ sessionId: "lt-1", isHost: true })).toEqual({
      sessionId: "lt-1",
      isHost: true,
      following: false,
      role: "host",
    });
    expect(roleFromSelf({ sessionId: "lt-1", isHost: false })).toEqual({
      sessionId: "lt-1",
      isHost: false,
      following: true,
      role: "guest",
    });
  });

  it("applies session events only to this device", () => {
    const none = emptyRoleState();
    expect(
      applySessionUpdated(none, "self", {
        action: "joined",
        sessionId: "lt-x",
        deviceId: "other",
      }),
    ).toEqual(none);

    const hosted = applySessionUpdated(none, "self", {
      action: "created",
      sessionId: "lt-self",
      hostId: "self",
    });
    expect(hosted.role).toBe("host");
    expect(hosted.sessionId).toBe("lt-self");

    const guest = applySessionUpdated(none, "self", {
      action: "joined",
      sessionId: "lt-host",
      deviceId: "self",
    });
    expect(guest.role).toBe("guest");
    expect(guest.following).toBe(true);

    expect(
      applySessionUpdated(guest, "self", {
        action: "ended",
        sessionId: "lt-host",
      }),
    ).toEqual(none);

    expect(
      applySessionUpdated(guest, "self", {
        action: "ended",
        sessionId: "lt-other",
      }),
    ).toEqual(guest);

    expect(
      applySessionUpdated(guest, "self", {
        action: "left",
        deviceId: "self",
        sessionId: "lt-host",
      }),
    ).toEqual(none);
  });

  it("derives link state from socket flags", () => {
    expect(linkStateFromSocket({ connected: true })).toBe("connected");
    expect(linkStateFromSocket({ connected: false, connecting: true })).toBe(
      "connecting",
    );
    expect(linkStateFromSocket({ connected: false, failed: true })).toBe(
      "error",
    );
    expect(linkStateFromSocket({ connected: false })).toBe("disconnected");
  });

  it("labels connection and session roles without leaking internals", () => {
    expect(connectionLabel("connecting", null)).toBe(
      "Connecting to other devices…",
    );
    expect(connectionLabel("connected", "transfer")).toBe(
      "Transferring playback…",
    );
    expect(sessionRoleLabel("host")).toBe("Hosting listen together");
    expect(sessionRoleLabel("guest")).toBe("Following the host");
    expect(sessionRoleLabel("none")).toBe("");
    expect(sessionPresenceLabel("none", 0)).toBe("");
    expect(sessionPresenceLabel("host", 1)).toBe(
      "Listening together · waiting",
    );
    expect(sessionPresenceLabel("host", 2)).toBe(
      "Listening together with 1 other",
    );
    expect(sessionPresenceLabel("host", 3)).toBe(
      "Listening together with 2 others",
    );
    expect(sessionPresenceLabel("guest", 2, "Kitchen")).toBe(
      "Listening with Kitchen",
    );
    expect(sessionPresenceLabel("guest", 2)).toBe("Listening together");

    const copy = deviceErrorCopy("session_unavailable");
    expect(copy.toLowerCase()).not.toContain("token");
    expect(copy.toLowerCase()).not.toContain("http");
    expect(deviceErrorCopy("nope", "Could not transfer playback.")).toBe(
      "Could not transfer playback.",
    );
  });

  it("advances playing position by time since the server stamp", () => {
    expect(
      compensatedPositionMs(
        { positionMs: 10_000, paused: true, serverTime: 1_000 },
        4_000,
      ),
    ).toBe(10_000);
    expect(
      compensatedPositionMs(
        { positionMs: 10_000, paused: false, serverTime: 1_000 },
        4_000,
      ),
    ).toBe(13_000);
    expect(
      compensatedPositionMs(
        {
          positionMs: 10_000,
          durationMs: 12_000,
          paused: false,
          serverTime: 1_000,
        },
        8_000,
      ),
    ).toBe(11_800);
    expect(
      compensatedPositionMs({ positionMs: 5_000, paused: false }, 9_000),
    ).toBe(5_000);
  });

  it("names session members and counts them", () => {
    const devices = [
      {
        deviceId: "self",
        name: "Laptop",
        sessionId: "lt-1",
        isHost: true,
      },
      {
        deviceId: "phone",
        name: "Phone",
        sessionId: "lt-1",
        isHost: false,
      },
      {
        deviceId: "other",
        name: "TV",
        sessionId: "lt-other",
        isHost: true,
      },
    ];
    expect(sessionMemberCount(devices, "lt-1")).toBe(2);
    expect(sessionMemberCount(devices, null)).toBe(0);
    expect(sessionHostName(devices, "lt-1")).toBe("Laptop");
    expect(sessionMemberNames(devices, "lt-1", "self")).toBe(
      "Laptop (you, host) · Phone",
    );
  });

  it("builds session presence event copy", () => {
    expect(sessionMemberDisplayName({ username: "ada", name: "Phone" })).toBe(
      "ada",
    );
    expect(sessionMemberDisplayName({ name: "Kitchen" })).toBe("Kitchen");
    expect(sessionJoinedCopy("ada")).toBe("ada joined");
    expect(sessionLeftCopy("ada")).toBe("ada left");
    expect(sessionLeftCopy("ada", "disconnected")).toBe("ada disconnected");
    expect(sessionEndedCopy("host_left")).toBe("Host ended listen together.");
    expect(sessionEndedCopy("host_disconnected")).toBe(
      "Host disconnected. Listen together ended.",
    );
    expect(sessionSelfJoinedCopy(true, "ada")).toBe("Joined ada's party");
    expect(sessionSelfJoinedCopy(false, "Kitchen")).toBe("Joined Kitchen");
  });
});
