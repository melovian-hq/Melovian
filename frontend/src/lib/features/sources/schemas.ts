// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

export const sourceStatusSchema = v.looseObject({
  mode: v.picklist(["subsonic", "local", "unified"]),
  activeInstanceId: v.string(),
  activeLocalId: v.string(),
  multiLocalLibrary: v.boolean(),
  unifiedAvailable: v.boolean(),
});
