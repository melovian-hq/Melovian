// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { music } from "$lib/config/music.svelte";
import { sources } from "$lib/features/sources/store.svelte";

/** True when browse pages should show setup / connection / missing-library UI. */
export function libraryUnavailable(): boolean {
  if (sources.needsSetup) return true;
  if (music.error) return true;
  if (!sources.hasAnySource) return true;
  if (!music.libraryReady && !music.loading) return true;
  return false;
}
