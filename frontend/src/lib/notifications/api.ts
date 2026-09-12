// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { ApiPaths } from "$lib/core/http/api-paths";
import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import { requireOk } from "$lib/core/http/errors";
import { parseJson } from "$lib/core/http/parse";
import {
  appNotificationSchema,
  notificationsReadAllResponseSchema,
  notificationsResponseSchema,
  unreadCountResponseSchema,
} from "./schemas";

export type NotificationKind =
  | "share_received"
  | "party_invite"
  | "party_joined"
  | "party_left"
  | "party_ended"
  | string;

export interface AppNotification {
  id: string;
  kind: NotificationKind;
  title: string;
  body: string;
  href: string;
  createdAt: string;
  read: boolean;
  readAt?: string;
  payload?: unknown;
}

export async function listNotifications(
  limit = 50,
  offset = 0,
): Promise<{ items: AppNotification[]; unread: number }> {
  const params = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });
  const response = await fetchWithRetry(`${ApiPaths.notifications}?${params}`, {
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to load notifications");
  const payload = await parseJson(
    notificationsResponseSchema,
    response,
    "notifications",
  );
  return {
    items: payload.items ?? [],
    unread: payload.unread ?? 0,
  };
}

export async function getUnreadCount(): Promise<number> {
  const response = await fetchWithRetry(ApiPaths.notificationsUnreadCount, {
    headers: apiHeaders(),
  });
  if (!response.ok) return 0;
  const payload = await parseJson(
    unreadCountResponseSchema,
    response,
    "unread count",
  );
  return payload.unread ?? 0;
}

export async function markNotificationRead(
  id: string,
): Promise<AppNotification> {
  const response = await fetchWithRetry(ApiPaths.notificationRead(id), {
    method: "POST",
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to mark notification read");
  return parseJson(appNotificationSchema, response, "notification");
}

export async function markAllNotificationsRead(): Promise<number> {
  const response = await fetchWithRetry(ApiPaths.notificationsReadAll, {
    method: "POST",
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to mark notifications read");
  const payload = await parseJson(
    notificationsReadAllResponseSchema,
    response,
    "mark all read",
  );
  return payload.updated ?? 0;
}

export async function deleteNotification(id: string): Promise<void> {
  const response = await fetchWithRetry(ApiPaths.notificationById(id), {
    method: "DELETE",
    headers: apiHeaders(),
  });
  await requireOk(response, "Failed to delete notification");
}
