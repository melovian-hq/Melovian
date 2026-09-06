// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { getActiveInstanceId } from "$lib/features/instances/context";

const VOLUME_KEY = "mel-music-volume";
const PLAYBACK_KEY = "mel-music-playback";
const LEGACY_PLAYBACK_KEY = PLAYBACK_KEY;
const NATIVE_PLAYBACK_KEY = "mel-native-playback";
const NATIVE_BACKEND_KEY = "mel-native-backend";

export type NativeBackendPref = "auto" | "mpv" | "vlc";
const QUEUE_PANEL_POSITION_KEY = "mel-queue-panel-position";
const LYRICS_PANEL_POSITION_KEY = "mel-lyrics-panel-position";
const LYRICS_PANEL_SIZE_KEY = "mel-lyrics-panel-size";

export interface QueuePanelPosition {
  x: number;
  y: number;
}

export interface PanelSize {
  width: number;
  height: number;
}

import type { ContinuousMode } from "$lib/music/continuous-pool";

export interface SavedPlayback {
  trackIds: string[];
  queueIndex: number;
  positionMs: number;
  shuffle: boolean;
  autoplay: boolean;
  continuousMode?: ContinuousMode;
  randomRadio?: boolean;
}

export function loadVolume(): number {
  try {
    const raw = localStorage.getItem(VOLUME_KEY);
    if (!raw) return 0.85;
    const value = Number.parseFloat(raw);
    if (!Number.isFinite(value)) return 0.85;
    return Math.max(0, Math.min(1, value));
  } catch {
    return 0.85;
  }
}

export function saveVolume(volume: number): void {
  localStorage.setItem(VOLUME_KEY, String(volume));
}

function playbackStorageKey(): string {
  const scope = getActiveInstanceId() ?? "default";
  return `${PLAYBACK_KEY}:${scope}`;
}

function parseSavedPlayback(raw: string | null): SavedPlayback | null {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as SavedPlayback;
    if (!Array.isArray(parsed.trackIds) || parsed.trackIds.length === 0) {
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

export function loadSavedPlayback(): SavedPlayback | null {
  try {
    const scoped = parseSavedPlayback(
      localStorage.getItem(playbackStorageKey()),
    );
    if (scoped) return scoped;

    const legacy = parseSavedPlayback(
      localStorage.getItem(LEGACY_PLAYBACK_KEY),
    );
    if (!legacy) return null;

    localStorage.setItem(playbackStorageKey(), JSON.stringify(legacy));
    localStorage.removeItem(LEGACY_PLAYBACK_KEY);
    return legacy;
  } catch {
    return null;
  }
}

export function savePlayback(state: SavedPlayback): void {
  localStorage.setItem(playbackStorageKey(), JSON.stringify(state));
}

export function clearSavedPlayback(): void {
  localStorage.removeItem(playbackStorageKey());
  localStorage.removeItem(LEGACY_PLAYBACK_KEY);
}

export function loadNativePlaybackPref(): boolean {
  try {
    const raw = localStorage.getItem(NATIVE_PLAYBACK_KEY);
    if (raw === null) return false;
    return raw === "true";
  } catch {
    return false;
  }
}

export function saveNativePlaybackPref(enabled: boolean): void {
  localStorage.setItem(NATIVE_PLAYBACK_KEY, enabled ? "true" : "false");
}

export function loadNativeBackendPref(): NativeBackendPref {
  try {
    const raw = localStorage.getItem(NATIVE_BACKEND_KEY);
    if (raw === "mpv" || raw === "vlc" || raw === "auto") return raw;
    return "auto";
  } catch {
    return "auto";
  }
}

export function saveNativeBackendPref(backend: NativeBackendPref): void {
  localStorage.setItem(NATIVE_BACKEND_KEY, backend);
}

function clampPanelCoordinate(value: number, max: number): number {
  return Math.max(0, Math.min(max, value));
}

export function loadQueuePanelPosition(): QueuePanelPosition | null {
  try {
    const raw = localStorage.getItem(QUEUE_PANEL_POSITION_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<QueuePanelPosition>;
    if (
      typeof parsed.x !== "number" ||
      typeof parsed.y !== "number" ||
      !Number.isFinite(parsed.x) ||
      !Number.isFinite(parsed.y)
    ) {
      return null;
    }
    return { x: parsed.x, y: parsed.y };
  } catch {
    return null;
  }
}

export function saveQueuePanelPosition(position: QueuePanelPosition): void {
  if (typeof localStorage === "undefined") return;
  localStorage.setItem(QUEUE_PANEL_POSITION_KEY, JSON.stringify(position));
}

export function clampQueuePanelPosition(
  position: QueuePanelPosition,
  panelWidth: number,
  panelHeight: number,
  viewportWidth = window.innerWidth,
  viewportHeight = window.innerHeight,
): QueuePanelPosition {
  return {
    x: clampPanelCoordinate(
      position.x,
      Math.max(0, viewportWidth - panelWidth),
    ),
    y: clampPanelCoordinate(
      position.y,
      Math.max(0, viewportHeight - panelHeight),
    ),
  };
}

export function loadLyricsPanelPosition(): QueuePanelPosition | null {
  try {
    const raw = localStorage.getItem(LYRICS_PANEL_POSITION_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<QueuePanelPosition>;
    if (
      typeof parsed.x !== "number" ||
      typeof parsed.y !== "number" ||
      !Number.isFinite(parsed.x) ||
      !Number.isFinite(parsed.y)
    ) {
      return null;
    }
    return { x: parsed.x, y: parsed.y };
  } catch {
    return null;
  }
}

export function saveLyricsPanelPosition(position: QueuePanelPosition): void {
  if (typeof localStorage === "undefined") return;
  localStorage.setItem(LYRICS_PANEL_POSITION_KEY, JSON.stringify(position));
}

export function loadLyricsPanelSize(): PanelSize | null {
  try {
    const raw = localStorage.getItem(LYRICS_PANEL_SIZE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<PanelSize>;
    if (
      typeof parsed.width !== "number" ||
      typeof parsed.height !== "number" ||
      !Number.isFinite(parsed.width) ||
      !Number.isFinite(parsed.height)
    ) {
      return null;
    }
    return { width: parsed.width, height: parsed.height };
  } catch {
    return null;
  }
}

export function saveLyricsPanelSize(size: PanelSize): void {
  if (typeof localStorage === "undefined") return;
  localStorage.setItem(LYRICS_PANEL_SIZE_KEY, JSON.stringify(size));
}

export function clampLyricsPanelSize(
  size: PanelSize,
  viewportWidth = window.innerWidth,
  viewportHeight = window.innerHeight,
): PanelSize {
  const minWidth = 256;
  const minHeight = 192;
  const maxWidth = Math.max(minWidth, viewportWidth * 0.75);
  const maxHeight = Math.max(minHeight, viewportHeight * 0.85);
  return {
    width: Math.max(minWidth, Math.min(maxWidth, size.width)),
    height: Math.max(minHeight, Math.min(maxHeight, size.height)),
  };
}

export function clampLyricsPanelPosition(
  position: QueuePanelPosition,
  panelWidth: number,
  panelHeight: number,
  viewportWidth = window.innerWidth,
  viewportHeight = window.innerHeight,
): QueuePanelPosition {
  return clampQueuePanelPosition(
    position,
    panelWidth,
    panelHeight,
    viewportWidth,
    viewportHeight,
  );
}
