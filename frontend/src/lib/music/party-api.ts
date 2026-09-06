// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import { ApiPaths } from "$lib/core/http/api-paths";
import { requireOk } from "$lib/core/http/errors";
import { resolveMediaUrl } from "$lib/config/runtime";
import type { SyncPlaybackSnapshot } from "$lib/music/device-sync.svelte";

export interface PartyInviteLink {
  sessionId: string;
  token: string;
  url: string;
}

export interface PartyMember {
  deviceId: string;
  userId?: string;
  username?: string;
  name: string;
  isHost: boolean;
}

export interface PartyStatus {
  sessionId: string;
  hostId: string;
  hostUserId?: string;
  crossUser: boolean;
  members: PartyMember[];
}

export interface PartyJoinResult {
  sessionId: string;
  state?: SyncPlaybackSnapshot | null;
}

export async function createPartyInviteLink(): Promise<PartyInviteLink> {
  const response = await fetchWithRetry(ApiPaths.partyInviteLink, {
    method: "POST",
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to create invite link");
  return (await response.json()) as PartyInviteLink;
}

export async function invitePartyUser(username: string): Promise<
  PartyInviteLink & {
    username: string;
    userId: string;
  }
> {
  const response = await fetchWithRetry(ApiPaths.partyInvite, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify({ username }),
  });
  await requireOk(response, "Failed to invite user");
  return (await response.json()) as PartyInviteLink & {
    username: string;
    userId: string;
  };
}

export async function joinPartyByToken(
  token: string,
): Promise<PartyJoinResult> {
  const response = await fetchWithRetry(ApiPaths.partyJoin, {
    method: "POST",
    headers: apiHeaders("application/json"),
    body: JSON.stringify({ token }),
  });
  await requireOk(response, "Failed to join party");
  return (await response.json()) as PartyJoinResult;
}

export async function getPartyStatus(sessionId: string): Promise<PartyStatus> {
  const response = await fetchWithRetry(ApiPaths.partyStatus(sessionId), {
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to load party status");
  return (await response.json()) as PartyStatus;
}

export function partyStreamUrl(sessionId: string, trackId: string): string {
  return resolveMediaUrl(ApiPaths.partyStream(sessionId, trackId));
}

export function partyCoverUrl(sessionId: string, trackId: string): string {
  return resolveMediaUrl(ApiPaths.partyCover(sessionId, trackId));
}
