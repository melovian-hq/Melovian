// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { PersistedState } from "runed";
import { StorageKeys } from "$lib/brand";

let collapsedState: PersistedState<boolean> | undefined;

// Created lazily because localStorage may not exist yet at module load.
function collapsed(): PersistedState<boolean> {
  collapsedState ??= new PersistedState<boolean>(
    StorageKeys.settingsNavAdvancedCollapsed,
    true,
  );
  return collapsedState;
}

/** Whether the user collapsed the Advanced group. Defaults to collapsed. */
export function settingsNavAdvancedCollapsed(): boolean {
  return collapsed().current;
}

export function toggleSettingsNavAdvanced(): void {
  collapsed().current = !collapsed().current;
}

/**
 * The Advanced group stays open once expanded, and force-opens while the
 * active tab is advanced so a deep link or search hit is never hidden. The
 * stored preference is left untouched in that case.
 */
export function settingsNavAdvancedOpen(
  collapsed: boolean,
  activeIsAdvanced: boolean,
): boolean {
  return !collapsed || activeIsAdvanced;
}
