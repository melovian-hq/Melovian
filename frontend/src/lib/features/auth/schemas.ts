// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

export const authUserSchema = v.looseObject({
  id: v.string(),
  username: v.string(),
});

export const authSessionSchema = v.looseObject({
  id: v.string(),
  expiresAt: v.string(),
  current: v.boolean(),
});

export const authSessionsResponseSchema = v.looseObject({
  sessions: v.optional(v.nullable(v.array(authSessionSchema))),
});

export const authStatusSchema = v.looseObject({
  enabled: v.boolean(),
  authenticated: v.boolean(),
  setupRequired: v.boolean(),
  demoMode: v.optional(v.boolean()),
  fakeCatalog: v.optional(v.boolean()),
  oidcEnabled: v.optional(v.boolean()),
  oidcLoginUrl: v.optional(v.string()),
  localLoginEnabled: v.optional(v.boolean()),
  user: v.optional(authUserSchema),
});

export const authUserResponseSchema = v.looseObject({
  user: authUserSchema,
});

export const subsonicKeyResponseSchema = v.looseObject({
  apiKey: v.optional(v.string()),
});
