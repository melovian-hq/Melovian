// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

export const storedSentrySettingsSchema = v.looseObject({
  enabled: v.boolean(),
  dsn: v.optional(v.string()),
  frontendDsn: v.optional(v.string()),
  environment: v.optional(v.string()),
  release: v.optional(v.string()),
  tracesSampleRate: v.optional(v.number()),
  clientReportingAllowed: v.boolean(),
});

export const sentryServerSettingsResponseSchema = v.looseObject({
  stored: storedSentrySettingsSchema,
  effective: v.looseObject({
    enabled: v.boolean(),
    dsnConfigured: v.boolean(),
    frontendDsnConfigured: v.boolean(),
    environment: v.optional(v.string()),
    release: v.optional(v.string()),
    tracesSampleRate: v.optional(v.number()),
    clientReportingAllowed: v.boolean(),
  }),
  envLocks: v.looseObject({
    dsn: v.optional(v.boolean()),
    frontendDsn: v.optional(v.boolean()),
    environment: v.optional(v.boolean()),
    release: v.optional(v.boolean()),
    tracesSampleRate: v.optional(v.boolean()),
  }),
});

export const sentryClientSettingsSchema = v.looseObject({
  enabled: v.boolean(),
});

// The test-event endpoint reports ok/eventId on success and may add an error
// or message on failure.
export const sentryTestEventSchema = v.looseObject({
  ok: v.optional(v.boolean()),
  eventId: v.optional(v.string()),
  message: v.optional(v.string()),
  error: v.optional(v.string()),
});
