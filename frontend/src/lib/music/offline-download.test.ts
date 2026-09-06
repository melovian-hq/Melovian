// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it } from "vitest";
import { toast } from "$lib/ui/toast.svelte";
import { reportOfflineDownload } from "./offline-download";

describe("reportOfflineDownload", () => {
  beforeEach(() => {
    toast.items = [];
  });

  it("reports an already-cached library", () => {
    reportOfflineDownload({ downloaded: 0, failed: 0 }, "Album");
    expect(toast.items[0]?.kind).toBe("success");
    expect(toast.items[0]?.message).toBe("Album already available offline");
  });

  it("warns on partial failure", () => {
    reportOfflineDownload({ downloaded: 5, failed: 2 }, "Playlist");
    expect(toast.items[0]?.kind).toBe("warning");
    expect(toast.items[0]?.message).toBe("Downloaded 5 tracks, 2 failed");
  });

  it("reports a full download", () => {
    reportOfflineDownload({ downloaded: 1, failed: 0 }, "Album");
    expect(toast.items[0]?.kind).toBe("success");
    expect(toast.items[0]?.message).toBe("Downloaded 1 track");
  });
});
