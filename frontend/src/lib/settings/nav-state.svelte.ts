// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { PersistedState } from "runed";
import { StorageKeys } from "$lib/brand";

export type SettingsNavMode = "simple" | "advanced";

let modeState: PersistedState<SettingsNavMode> | undefined;

// Created lazily because localStorage may not exist yet at module load.
function mode(): PersistedState<SettingsNavMode> {
  modeState ??= new PersistedState<SettingsNavMode>(
    StorageKeys.settingsNavMode,
    "simple",
  );
  return modeState;
}

/** The selected sidebar mode. Defaults to Simple. */
export function settingsNavMode(): SettingsNavMode {
  return mode().current;
}

export function setSettingsNavMode(next: SettingsNavMode): void {
  mode().current = next;
}
