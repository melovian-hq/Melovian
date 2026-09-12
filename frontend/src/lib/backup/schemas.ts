// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

/**
 * Server settings are snapshotted into backup files verbatim. Only object-ness
 * matters here; the detailed shape belongs to the owning feature schemas.
 */
export const settingsSnapshotSchema = v.looseObject({});

/** Backup files carry a version stamp plus per-section payloads. */
export const backupFileSchema = v.looseObject({
  version: v.number(),
  exportedAt: v.string(),
  sections: v.looseObject({}),
});
