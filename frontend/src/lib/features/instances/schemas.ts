// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

export const subsonicInstanceSchema = v.looseObject({
  id: v.string(),
  name: v.string(),
  serverUrl: v.string(),
  username: v.string(),
  serverName: v.string(),
  createdAt: v.string(),
  updatedAt: v.string(),
  lastUsedAt: v.optional(v.string()),
});

export const instancesResponseSchema = v.looseObject({
  instances: v.optional(v.nullable(v.array(subsonicInstanceSchema))),
});

// The active-instance endpoint returns an empty object when nothing is set.
export const activeInstanceProbeSchema = v.looseObject({
  id: v.optional(v.string()),
});

export const instanceTestResponseSchema = v.looseObject({
  serverName: v.optional(v.string()),
});

export const instancePingSchema = v.looseObject({
  online: v.boolean(),
  latencyMs: v.number(),
  serverName: v.optional(v.string()),
  version: v.optional(v.string()),
  songCount: v.optional(v.number()),
  error: v.optional(v.string()),
});
