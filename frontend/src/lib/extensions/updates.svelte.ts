// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { PersistedState } from "runed";
import { StorageKeys } from "$lib/brand";
import { router } from "$lib/router/router.svelte";
import { toast } from "$lib/ui/toast.svelte";
import { fetchExtensionRegistry, installRemoteExtension } from "./api";

let autoUpdate: PersistedState<boolean> | undefined;

// Created lazily because localStorage may not exist at module load.
function autoUpdateState(): PersistedState<boolean> {
  autoUpdate ??= new PersistedState<boolean>(
    StorageKeys.extensionAutoUpdate,
    false,
  );
  return autoUpdate;
}

export function extensionAutoUpdate(): boolean {
  return autoUpdateState().current;
}

export function setExtensionAutoUpdate(value: boolean): void {
  autoUpdateState().current = value;
}

let checked = false;

/**
 * Launch-time registry check. Toasts when installed extensions have
 * updates, or auto-installs them when the user opted in. Auto-update is
 * deliberately narrow: low-risk entries only, never delisted ones, and a
 * failed install falls back to the manual prompt.
 */
export async function checkExtensionUpdates(): Promise<void> {
  if (checked) return;
  checked = true;
  let updates;
  try {
    const payload = await fetchExtensionRegistry();
    updates = (payload.items ?? []).filter((item) => item.updateAvailable);
  } catch {
    return; // Offline or unsigned registry stays silent at launch.
  }
  if (updates.length === 0) return;
  const openSettings = () => router.navigate("/settings/extensions", true);
  if (!extensionAutoUpdate()) {
    toast.info(
      `${updates.length} extension update${updates.length === 1 ? "" : "s"} available`,
      {
        duration: 8000,
        action: { label: "Review", onClick: openSettings },
      },
    );
    return;
  }
  const safe = updates.filter(
    (item) => item.risk === "low" && !item.delisted && !item.hasWasm,
  );
  if (safe.length === 0) {
    toast.info(
      `${updates.length} extension update${updates.length === 1 ? "" : "s"} need a manual review`,
      {
        duration: 8000,
        action: { label: "Review", onClick: openSettings },
      },
    );
    return;
  }
  let done = 0;
  for (const item of safe) {
    try {
      await installRemoteExtension(item.id);
      done += 1;
    } catch {
      // Leave it for the manual path; the update toast still offers review.
    }
  }
  if (done > 0) {
    toast.success(
      `Updated ${done} extension${done === 1 ? "" : "s"} automatically`,
    );
  }
  if (done < updates.length) {
    toast.info("Some updates were skipped for review", {
      action: { label: "Review", onClick: openSettings },
    });
  }
}
