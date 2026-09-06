// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
import { APP_NAME, STORAGE_BACKUP_PREFIXES, BACKUP_FILE_BASENAME } from "$lib/brand";
import * as instanceApi from "$lib/features/instances/api";
import * as libraryApi from "$lib/features/local-libraries/api";
import * as musicApi from "$lib/music/api";
import type {
  InstanceInput,
  SubsonicInstance,
} from "$lib/features/instances/types";
import type {
  LocalLibrary,
  LocalLibraryInput,
} from "$lib/features/local-libraries/types";
import type { MusicPlaylist } from "$lib/subsonic/types";
import { downloadTextFile, sanitizeFilename } from "$lib/utils/download";

export const BACKUP_VERSION = 1;

export const BACKUP_SECTION_IDS = [
  "devicePreferences",
  "serverSettings",
  "playlists",
  "instances",
  "localLibraries",
] as const;

export type BackupSectionId = (typeof BACKUP_SECTION_IDS)[number];

export interface BackupSectionMeta {
  id: BackupSectionId;
  label: string;
  description: string;
}

export const BACKUP_SECTIONS: BackupSectionMeta[] = [
  {
    id: "devicePreferences",
    label: "Device preferences",
    description:
      "Theme, playback, queue, mixes, EQ, and other settings stored in this browser.",
  },
  {
    id: "serverSettings",
    label: "Server settings",
    description:
      `EQ, connection, cache, and lyrics settings stored on this ${APP_NAME} server.`,
  },
  {
    id: "playlists",
    label: "Local playlists",
    description: `Playlists created in ${APP_NAME} with their track lists.`,
  },
  {
    id: "instances",
    label: "Subsonic servers",
    description:
      "Saved server names and URLs. Passwords are not included for security.",
  },
  {
    id: "localLibraries",
    label: "Local libraries",
    description: "Folder paths for music libraries scanned on this device.",
  },
];

export interface MelovianBackup {
  version: number;
  exportedAt: string;
  sections: Partial<Record<BackupSectionId, unknown>>;
}

export interface BackupImportResult {
  imported: BackupSectionId[];
  skipped: string[];
  warnings: string[];
}

const LOCAL_STORAGE_PREFIXES = STORAGE_BACKUP_PREFIXES;

export function collectLocalStorageBackup(): Record<string, string> {
  if (typeof localStorage === "undefined") return {};
  const entries: Record<string, string> = {};
  for (let i = 0; i < localStorage.length; i += 1) {
    const key = localStorage.key(i);
    if (!key) continue;
    if (!LOCAL_STORAGE_PREFIXES.some((prefix) => key.startsWith(prefix))) {
      continue;
    }
    const value = localStorage.getItem(key);
    if (value !== null) entries[key] = value;
  }
  return entries;
}

export function restoreLocalStorageBackup(data: Record<string, string>): void {
  if (typeof localStorage === "undefined") return;
  for (const [key, value] of Object.entries(data)) {
    if (!LOCAL_STORAGE_PREFIXES.some((prefix) => key.startsWith(prefix))) {
      continue;
    }
    localStorage.setItem(key, value);
  }
}

async function fetchServerSettings(): Promise<Record<string, unknown>> {
  const [eq, connection, cache, lyrics, sentry] = await Promise.all([
    fetchWithRetry("/api/music/settings/eq", { headers: apiHeaders() })
      .then((r) => (r.ok ? r.json() : null))
      .catch(() => null),
    fetchWithRetry("/api/music/settings/connection", { headers: apiHeaders() })
      .then((r) => (r.ok ? r.json() : null))
      .catch(() => null),
    fetchWithRetry("/api/music/settings/cache", { headers: apiHeaders() })
      .then((r) => (r.ok ? r.json() : null))
      .catch(() => null),
    musicApi.getLyricsSettings().catch(() => null),
    fetchWithRetry("/api/settings/sentry", { headers: apiHeaders() })
      .then((r) => (r.ok ? r.json() : null))
      .catch(() => null),
  ]);
  return { eq, connection, cache, lyrics, sentry };
}

async function fetchPlaylistsBackup(): Promise<MusicPlaylist[]> {
  const playlists = await musicApi.listPlaylists();
  return Promise.all(
    playlists.map((playlist) => musicApi.getPlaylist(playlist.id)),
  );
}

async function fetchInstancesBackup(): Promise<{
  instances: SubsonicInstance[];
  activeInstanceId: string | null;
}> {
  const [instances, active] = await Promise.all([
    instanceApi.listInstances(),
    instanceApi.getActiveInstance(),
  ]);
  return {
    instances,
    activeInstanceId: active?.id ?? null,
  };
}

async function fetchLocalLibrariesBackup(): Promise<LocalLibrary[]> {
  return libraryApi.listLocalLibraries();
}

export async function createBackup(
  sections: BackupSectionId[],
): Promise<MelovianBackup> {
  const selected = new Set(sections);
  const payload: MelovianBackup = {
    version: BACKUP_VERSION,
    exportedAt: new Date().toISOString(),
    sections: {},
  };

  if (selected.has("devicePreferences")) {
    payload.sections.devicePreferences = collectLocalStorageBackup();
  }
  if (selected.has("serverSettings")) {
    payload.sections.serverSettings = await fetchServerSettings();
  }
  if (selected.has("playlists")) {
    payload.sections.playlists = await fetchPlaylistsBackup();
  }
  if (selected.has("instances")) {
    payload.sections.instances = await fetchInstancesBackup();
  }
  if (selected.has("localLibraries")) {
    payload.sections.localLibraries = await fetchLocalLibrariesBackup();
  }

  return payload;
}

export function exportBackupFile(backup: MelovianBackup): void {
  const stamp = backup.exportedAt.slice(0, 10);
  const filename = sanitizeFilename(
    `${BACKUP_FILE_BASENAME}-${stamp}`,
    BACKUP_FILE_BASENAME,
  );
  downloadTextFile(
    `${filename}.json`,
    JSON.stringify(backup, null, 2),
    "application/json;charset=utf-8",
  );
}

export function parseBackupFile(raw: string): MelovianBackup {
  const parsed = JSON.parse(raw) as MelovianBackup;
  if (
    !parsed ||
    typeof parsed !== "object" ||
    parsed.version !== BACKUP_VERSION
  ) {
    throw new Error("Unsupported or invalid backup file");
  }
  if (!parsed.sections || typeof parsed.sections !== "object") {
    throw new Error("Backup file is missing section data");
  }
  return parsed;
}

async function restoreServerSettings(
  data: Record<string, unknown>,
): Promise<void> {
  const tasks: Promise<unknown>[] = [];
  if (data.eq) {
    tasks.push(
      fetchWithRetry("/api/music/settings/eq", {
        method: "PUT",
        headers: apiHeaders("application/json"),
        body: JSON.stringify(data.eq),
      }),
    );
  }
  if (data.connection) {
    tasks.push(
      fetchWithRetry("/api/music/settings/connection", {
        method: "PUT",
        headers: apiHeaders("application/json"),
        body: JSON.stringify(data.connection),
      }),
    );
  }
  if (data.cache) {
    tasks.push(
      fetchWithRetry("/api/music/settings/cache", {
        method: "PUT",
        headers: apiHeaders("application/json"),
        body: JSON.stringify(data.cache),
      }),
    );
  }
  if (data.lyrics) {
    tasks.push(musicApi.saveLyricsSettingsRemote(data.lyrics as never));
  }
  if (data.sentry && typeof data.sentry === "object") {
    const sentryData = data.sentry as { stored?: Record<string, unknown> };
    if (sentryData.stored) {
      tasks.push(
        fetchWithRetry("/api/settings/sentry", {
          method: "PUT",
          headers: apiHeaders("application/json"),
          body: JSON.stringify(sentryData.stored),
        }),
      );
    }
  }
  const results = await Promise.allSettled(tasks);
  const failed = results.filter((result) => result.status === "rejected");
  if (failed.length > 0) {
    throw new Error("Some server settings could not be restored");
  }
}

async function restorePlaylists(playlists: MusicPlaylist[]): Promise<number> {
  let restored = 0;
  for (const playlist of playlists) {
    const created = await musicApi.createPlaylist(playlist.name);
    const tracks = playlist.tracks ?? [];
    if (tracks.length === 0) {
      restored += 1;
      continue;
    }
    const response = await fetchWithRetry(
      `/api/music/playlists/${created.id}/tracks`,
      {
        method: "PUT",
        headers: apiHeaders("application/json"),
        body: JSON.stringify({ tracks }),
      },
    );
    if (response.ok) restored += 1;
  }
  return restored;
}

interface BackupInstance extends SubsonicInstance {
  password?: string;
}

async function restoreInstances(data: {
  instances: BackupInstance[];
  activeInstanceId?: string | null;
}): Promise<{ restored: number; warnings: string[] }> {
  const warnings: string[] = [];
  let restored = 0;
  let activeId: string | null = null;

  for (const instance of data.instances) {
    if (!instance.password?.trim()) {
      warnings.push(
        `Skipped server "${instance.name}". No password in backup. Re-add it manually in Settings.`,
      );
      continue;
    }
    const input: InstanceInput = {
      name: instance.name,
      serverUrl: instance.serverUrl,
      username: instance.username,
      password: instance.password,
    };
    const created = await instanceApi.createInstance(input);
    restored += 1;
    if (data.activeInstanceId && instance.id === data.activeInstanceId) {
      activeId = created.id;
    }
  }

  if (activeId) {
    await instanceApi.activateInstance(activeId).catch(() => undefined);
  }

  return { restored, warnings };
}

async function restoreLocalLibraries(
  libraries: LocalLibrary[],
): Promise<number> {
  let restored = 0;
  for (const library of libraries) {
    const input: LocalLibraryInput = {
      name: library.name,
      path: library.path,
    };
    await libraryApi.createLocalLibrary(input);
    restored += 1;
  }
  return restored;
}

export async function importBackup(
  backup: MelovianBackup,
  sections: BackupSectionId[],
): Promise<BackupImportResult> {
  const selected = new Set(sections);
  const imported: BackupSectionId[] = [];
  const skipped: string[] = [];
  const warnings: string[] = [];

  if (selected.has("devicePreferences")) {
    const data = backup.sections.devicePreferences;
    if (data && typeof data === "object") {
      restoreLocalStorageBackup(data as Record<string, string>);
      imported.push("devicePreferences");
    } else {
      skipped.push("devicePreferences");
    }
  }

  if (selected.has("serverSettings")) {
    const data = backup.sections.serverSettings;
    if (data && typeof data === "object") {
      await restoreServerSettings(data as Record<string, unknown>);
      imported.push("serverSettings");
    } else {
      skipped.push("serverSettings");
    }
  }

  if (selected.has("playlists")) {
    const data = backup.sections.playlists;
    if (Array.isArray(data) && data.length > 0) {
      const count = await restorePlaylists(data as MusicPlaylist[]);
      if (count > 0) imported.push("playlists");
      else skipped.push("playlists");
    } else {
      skipped.push("playlists");
    }
  }

  if (selected.has("instances")) {
    const data = backup.sections.instances;
    if (
      data &&
      typeof data === "object" &&
      Array.isArray((data as { instances?: unknown }).instances)
    ) {
      const result = await restoreInstances(
        data as {
          instances: BackupInstance[];
          activeInstanceId?: string | null;
        },
      );
      warnings.push(...result.warnings);
      if (result.restored > 0) imported.push("instances");
      else skipped.push("instances");
    } else {
      skipped.push("instances");
    }
  }

  if (selected.has("localLibraries")) {
    const data = backup.sections.localLibraries;
    if (Array.isArray(data) && data.length > 0) {
      const count = await restoreLocalLibraries(data as LocalLibrary[]);
      if (count > 0) imported.push("localLibraries");
      else skipped.push("localLibraries");
    } else {
      skipped.push("localLibraries");
    }
  }

  return { imported, skipped, warnings };
}
