// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import * as v from "valibot";

// WS frames carry { type, payload }; the payload shape depends on the type.
export const wsEventSchema = v.looseObject({
  type: v.string(),
  payload: v.unknown(),
});
