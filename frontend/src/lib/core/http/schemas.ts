// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

// httputil.WriteError emits { error: message, code: code } on the Go side.
export const apiErrorBodySchema = v.looseObject({
  error: v.optional(v.string()),
  code: v.optional(v.string()),
});
