// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { formatDuration } from "$lib/subsonic";

export function playlistDurationSeconds(
  input:
    | {
        durationMs?: number;
        duration?: number;
        tracks?: { durationMs?: number }[];
      }
    | null
    | undefined,
): number {
  if (!input) return 0;
  if (typeof input.durationMs === "number" && input.durationMs > 0) {
    return Math.floor(input.durationMs / 1000);
  }
  if (typeof input.duration === "number" && input.duration > 0) {
    return input.duration;
  }
  if (input.tracks && input.tracks.length > 0) {
    const totalMs = input.tracks.reduce(
      (sum, track) => sum + (track.durationMs ?? 0),
      0,
    );
    if (totalMs > 0) return Math.floor(totalMs / 1000);
  }
  return 0;
}

export function formatHumanDuration(seconds: number): string | null {
  if (seconds <= 0) return null;

  const totalMinutes = Math.max(1, Math.ceil(seconds / 60));
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;

  if (hours > 0) {
    if (minutes > 0) {
      return `${hours} hr ${minutes} min`;
    }
    return `${hours} hr`;
  }

  return `${minutes} min`;
}

export function formatPlaylistDuration(
  input: Parameters<typeof playlistDurationSeconds>[0],
): string | null {
  const seconds = playlistDurationSeconds(input);
  return formatHumanDuration(seconds);
}

export function formatPlaylistDurationClock(
  input: Parameters<typeof playlistDurationSeconds>[0],
): string | null {
  const seconds = playlistDurationSeconds(input);
  if (seconds <= 0) return null;
  return formatDuration(seconds);
}
