// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

const connectionDefaultsSchema = v.looseObject({
  minDelayMs: v.optional(v.number()),
  maxDelayMs: v.optional(v.number()),
  backoffMultiplier: v.optional(v.number()),
  healthCheckIntervalMs: v.optional(v.number()),
  offlinePollMs: v.optional(v.number()),
  selfHealIntervalMs: v.optional(v.number()),
  maxHistoryEntries: v.optional(v.number()),
});

const sentryRuntimeSchema = v.looseObject({
  dsn: v.optional(v.string()),
  environment: v.optional(v.string()),
  release: v.optional(v.string()),
  tracesSampleRate: v.optional(v.number()),
  clientReporting: v.optional(v.boolean()),
});

/**
 * The /api/config payload. Only the keys the client reads are declared; the
 * rest pass through untouched. All fields are optional because trusted-only
 * keys (dataDir, extensions.dir, localLibrary.defaultPath) are omitted for
 * signed-out or remote callers.
 */
export const runtimeConfigSchema = v.looseObject({
  listenAddr: v.optional(v.string()),
  publicUrl: v.optional(v.string()),
  authEnabled: v.optional(v.boolean()),
  serverMode: v.optional(v.boolean()),
  demoMode: v.optional(v.boolean()),
  fakeCatalog: v.optional(v.boolean()),
  connectionDefaults: v.optional(v.nullable(connectionDefaultsSchema)),
  sentry: v.optional(v.nullable(sentryRuntimeSchema)),
  subsonicServer: v.optional(
    v.looseObject({ enabled: v.optional(v.boolean()) }),
  ),
  transcoding: v.optional(
    v.looseObject({ available: v.optional(v.boolean()) }),
  ),
  extensions: v.optional(v.looseObject({ dir: v.optional(v.string()) })),
  dlna: v.optional(
    v.looseObject({
      enabled: v.optional(v.boolean()),
      port: v.optional(v.number()),
    }),
  ),
  jukebox: v.optional(v.looseObject({ enabled: v.optional(v.boolean()) })),
  version: v.optional(v.string()),
  apiVersion: v.optional(v.number()),
  minClientVersion: v.optional(v.string()),
  minServerVersion: v.optional(v.string()),
  capabilities: v.optional(v.array(v.string())),
});

// A 426 Upgrade Required body carries an optional human-readable message.
export const upgradeRequiredSchema = v.looseObject({
  message: v.optional(v.string()),
});
