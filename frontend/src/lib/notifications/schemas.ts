// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

export const appNotificationSchema = v.looseObject({
  id: v.string(),
  kind: v.string(),
  title: v.string(),
  body: v.string(),
  href: v.string(),
  createdAt: v.string(),
  read: v.boolean(),
  readAt: v.optional(v.string()),
  payload: v.optional(v.unknown()),
});

export const notificationsResponseSchema = v.looseObject({
  items: v.optional(v.nullable(v.array(appNotificationSchema))),
  unread: v.optional(v.nullable(v.number())),
});

export const unreadCountResponseSchema = v.looseObject({
  unread: v.optional(v.nullable(v.number())),
});

export const notificationsReadAllResponseSchema = v.looseObject({
  updated: v.optional(v.nullable(v.number())),
});
