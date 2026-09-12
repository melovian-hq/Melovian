// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

export const directoryEntrySchema = v.looseObject({
  name: v.string(),
  path: v.string(),
});

export const directoryListingSchema = v.looseObject({
  path: v.optional(v.string()),
  parent: v.optional(v.string()),
  entries: v.optional(v.nullable(v.array(directoryEntrySchema))),
});
