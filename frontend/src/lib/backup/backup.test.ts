// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, beforeEach, vi } from "vitest";
import {
  BACKUP_VERSION,
  collectLocalStorageBackup,
  exportBackupFile,
  parseBackupFile,
  restoreLocalStorageBackup,
} from "./backup";

describe("backup", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it("collects melovian localStorage keys", () => {
    localStorage.setItem("mel-playback-settings", '{"crossfade":0}');
    localStorage.setItem("melovian-theme", "dark");
    localStorage.setItem("other-app", "ignore");

    expect(collectLocalStorageBackup()).toEqual({
      "mel-playback-settings": '{"crossfade":0}',
      "melovian-theme": "dark",
    });
  });

  it("restores localStorage keys with melovian prefixes", () => {
    restoreLocalStorageBackup({
      "mel-queue-settings": '{"maxQueueSize":500}',
      "melovian-theme-accent": "#336699",
      ignored: "value",
    });

    expect(localStorage.getItem("mel-queue-settings")).toBe(
      '{"maxQueueSize":500}',
    );
    expect(localStorage.getItem("melovian-theme-accent")).toBe("#336699");
    expect(localStorage.getItem("ignored")).toBeNull();
  });

  it("parses valid backup files", () => {
    const backup = {
      version: BACKUP_VERSION,
      exportedAt: "2026-07-05T00:00:00.000Z",
      sections: { devicePreferences: {} },
    };
    expect(parseBackupFile(JSON.stringify(backup))).toEqual(backup);
  });

  it("rejects unsupported backup versions", () => {
    expect(() =>
      parseBackupFile(
        JSON.stringify({
          version: 99,
          exportedAt: "2026-07-05T00:00:00.000Z",
          sections: {},
        }),
      ),
    ).toThrow(/unsupported/i);
  });

  it("downloads backup json", () => {
    const click = vi.fn();
    const remove = vi.fn();
    const anchor = {
      click,
      remove,
      href: "",
      download: "",
      rel: "",
    } as unknown as HTMLAnchorElement;
    vi.spyOn(document, "createElement").mockReturnValue(anchor);
    vi.spyOn(document.body, "append").mockImplementation(() => undefined);
    vi.stubGlobal("URL", {
      createObjectURL: vi.fn(() => "blob:test"),
      revokeObjectURL: vi.fn(),
    });

    exportBackupFile({
      version: BACKUP_VERSION,
      exportedAt: "2026-07-05T12:00:00.000Z",
      sections: {},
    });

    expect(click).toHaveBeenCalled();
    expect(anchor.download).toBe("melovian-backup-2026-07-05.json");
  });
});
