// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { instances } from "$lib/features/instances/store.svelte";
import { localLibraries } from "$lib/features/local-libraries/store.svelte";
import * as sourceApi from "./api";

export async function activateSubsonicSource(id: string): Promise<void> {
  await instances.activate(id);
  await localLibraries.refresh();
  await sourceApi.setSourceViewMode("subsonic").catch(() => undefined);
}

export async function activateLocalSource(id: string): Promise<void> {
  await localLibraries.activate(id);
  await instances.refresh();
  await sourceApi.setSourceViewMode("local").catch(() => undefined);
}

export async function activateUnifiedSource(): Promise<void> {
  await sourceApi.setSourceViewMode("unified");
  await Promise.all([instances.refresh(), localLibraries.refresh()]);
}
