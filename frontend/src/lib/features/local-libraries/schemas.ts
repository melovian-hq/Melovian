// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

export const localLibrarySchema = v.looseObject({
  id: v.string(),
  type: v.literal("local"),
  name: v.string(),
  path: v.string(),
  trackCount: v.number(),
  missingCount: v.number(),
  duplicateCount: v.number(),
  scanStatus: v.string(),
  scanError: v.optional(v.string()),
  scanProgress: v.optional(
    v.looseObject({
      processed: v.number(),
      phase: v.picklist(["scanning", "reconciling"]),
    }),
  ),
  createdAt: v.string(),
  updatedAt: v.string(),
  lastScannedAt: v.optional(v.string()),
});

export const localLibrariesResponseSchema = v.looseObject({
  libraries: v.optional(v.nullable(v.array(localLibrarySchema))),
});

// The active-library endpoint returns an empty object when nothing is set.
export const activeLocalLibraryProbeSchema = v.looseObject({
  id: v.optional(v.string()),
});

export const localLibraryConfigResponseSchema = v.looseObject({
  localLibrary: v.optional(
    v.nullable(
      v.looseObject({
        enabled: v.optional(v.boolean()),
        defaultPath: v.optional(v.string()),
        allowCustomPath: v.optional(v.boolean()),
      }),
    ),
  ),
});
