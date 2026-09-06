// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { isDemoMode } from "$lib/config/runtime";

export function canMutateInDemo(): boolean {
  return !isDemoMode();
}
