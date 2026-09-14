// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

export const extensionListItemSchema = v.looseObject({
  id: v.string(),
  name: v.string(),
  version: v.string(),
  description: v.optional(v.string()),
  author: v.optional(v.string()),
  enabled: v.boolean(),
  installed: v.boolean(),
  bundled: v.boolean(),
  hasScript: v.boolean(),
  scriptSafe: v.boolean(),
  hasWasm: v.boolean(),
  iconUrl: v.optional(v.string()),
  imageUrl: v.optional(v.string()),
  appTheme: v.optional(v.string()),
  installedAt: v.optional(v.string()),
});

const trackMatchSchema = v.looseObject({
  genreContains: v.optional(v.string()),
  artistContains: v.optional(v.string()),
  albumContains: v.optional(v.string()),
  titleContains: v.optional(v.string()),
  titleRegex: v.optional(v.string()),
  tagEquals: v.optional(v.string()),
  minRating: v.optional(v.number()),
  isLocal: v.optional(v.boolean()),
});

const trackDecorationSchema = v.looseObject({
  progressColor: v.optional(v.string()),
  progressGradient: v.optional(v.string()),
  progressThumbUrl: v.optional(v.string()),
  progressParticleUrl: v.optional(v.string()),
  icon: v.optional(v.string()),
  iconUrl: v.optional(v.string()),
  titlePrefix: v.optional(v.string()),
  coverOverlayIcon: v.optional(v.string()),
  playerTheme: v.optional(v.string()),
});

const trackRuleSchema = v.looseObject({
  match: trackMatchSchema,
  decoration: trackDecorationSchema,
});

const playerHookSchema = v.looseObject({
  when: v.string(),
  style: v.optional(v.record(v.string(), v.string())),
});

export const extensionManifestSchema = v.looseObject({
  id: v.string(),
  name: v.string(),
  version: v.string(),
  description: v.optional(v.string()),
  author: v.optional(v.string()),
  script: v.optional(v.string()),
  icon: v.optional(v.string()),
  image: v.optional(v.string()),
  appTheme: v.optional(v.string()),
  styles: v.optional(v.array(v.string())),
  trackRules: v.optional(v.array(trackRuleSchema)),
  playerHooks: v.optional(v.array(playerHookSchema)),
});

export const extensionsPayloadSchema = v.looseObject({
  items: v.optional(v.nullable(v.array(extensionListItemSchema))),
  manifests: v.optional(v.nullable(v.array(extensionManifestSchema))),
  dir: v.optional(v.string()),
});

export const registryItemSchema = v.looseObject({
  id: v.string(),
  name: v.string(),
  version: v.string(),
  description: v.optional(v.string()),
  author: v.optional(v.string()),
  iconUrl: v.optional(v.string()),
  imageUrl: v.optional(v.string()),
  packageUrl: v.optional(v.string()),
  sha256: v.optional(v.string()),
  bytes: v.optional(v.number()),
  hasScript: v.boolean(),
  hasWasm: v.boolean(),
  styles: v.optional(v.number()),
  appTheme: v.optional(v.boolean()),
  trackRules: v.optional(v.number()),
  playerHooks: v.optional(v.number()),
  auditStatus: v.optional(v.string()),
  auditWarnings: v.optional(v.array(v.string())),
  installed: v.boolean(),
  installedVersion: v.optional(v.string()),
  enabled: v.boolean(),
  updateAvailable: v.boolean(),
});

export const registryPayloadSchema = v.looseObject({
  url: v.optional(v.string()),
  generatedAt: v.optional(v.string()),
  items: v.optional(v.nullable(v.array(registryItemSchema))),
});
