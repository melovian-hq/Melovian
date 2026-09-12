// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

// Update status fields are all optional: the shape is assembled from the
// desktop update service and the HTTP update handler.
export const updateStatusSchema = v.looseObject({
  currentVersion: v.optional(v.string()),
  latestVersion: v.optional(v.string()),
  releaseUrl: v.optional(v.string()),
  upToDate: v.optional(v.boolean()),
  checking: v.optional(v.boolean()),
  applying: v.optional(v.boolean()),
  stage: v.optional(v.string()),
  stageMessage: v.optional(v.string()),
  written: v.optional(v.number()),
  total: v.optional(v.number()),
  canApply: v.optional(v.boolean()),
  inContainer: v.optional(v.boolean()),
  autoUpdate: v.optional(v.boolean()),
  channel: v.optional(v.string()),
  signed: v.optional(v.boolean()),
  checkError: v.optional(v.string()),
  applyError: v.optional(v.string()),
  error: v.optional(v.string()),
  manual: v.optional(v.boolean()),
  appliedVersion: v.optional(v.string()),
  desktop: v.optional(
    v.looseObject({
      state: v.optional(v.string()),
      manual: v.optional(v.boolean()),
      releaseUrl: v.optional(v.string()),
      latestVersion: v.optional(v.string()),
    }),
  ),
});
