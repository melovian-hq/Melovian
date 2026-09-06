// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { SyncDevice } from "$lib/music/device-sync.svelte";

export function resolveAmbientCoverArt(input: {
  localCoverArt?: string | null;
  activePeerCoverArt?: string | null;
}): string {
  const peer = input.activePeerCoverArt?.trim();
  if (peer) return peer;
  return input.localCoverArt?.trim() ?? "";
}

export function peerStatusLabel(
  peer: Pick<
    SyncDevice,
    "isActivePlayer" | "playback" | "sessionId" | "isHost"
  >,
): string {
  const parts: string[] = [];
  if (peer.isActivePlayer) {
    const title = peer.playback?.trackTitle?.trim();
    parts.push(title ? `Playing · ${title}` : "Playing");
  } else {
    parts.push("Idle");
  }
  if (peer.sessionId) {
    parts.push(peer.isHost ? "Hosting" : "Listening together");
  }
  return parts.join(" · ");
}

export type DeviceGlyph = "monitor" | "headphones" | "music" | "server";

export function deviceKindFromUserAgent(ua?: string | null): DeviceGlyph {
  const value = ua ?? "";
  if (/iPhone|iPad|Android/i.test(value)) return "headphones";
  if (/Macintosh|Mac OS|Windows|Linux|X11/i.test(value)) return "monitor";
  if (/Server/i.test(value)) return "server";
  return "music";
}
