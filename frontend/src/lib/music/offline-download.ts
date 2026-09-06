// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { toast } from "$lib/ui/toast.svelte";

export interface OfflineDownloadResult {
  downloaded: number;
  failed: number;
}

function trackLabel(count: number): string {
  return count === 1 ? "track" : "tracks";
}

export function reportOfflineDownload(
  result: OfflineDownloadResult,
  entityLabel: string,
): void {
  if (result.downloaded === 0 && result.failed === 0) {
    toast.success(`${entityLabel} already available offline`);
    return;
  }
  if (result.failed > 0) {
    toast.warning(
      `Downloaded ${result.downloaded} ${trackLabel(result.downloaded)}, ${result.failed} failed`,
    );
    return;
  }
  toast.success(
    `Downloaded ${result.downloaded} ${trackLabel(result.downloaded)}`,
  );
}
