// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

const MINUTE_MS = 60_000;
const HOUR_MS = 3_600_000;
const DAY_MS = 86_400_000;
const MONTH_MS = 30 * DAY_MS;
const YEAR_MS = 365 * DAY_MS;

export function formatRelativePlayedAt(
  iso: string,
  nowMs = Date.now(),
): string {
  const playedMs = new Date(iso).getTime();
  if (!Number.isFinite(playedMs)) return "";
  const diff = Math.max(0, nowMs - playedMs);
  if (diff < MINUTE_MS) return "Just now";
  if (diff < HOUR_MS) {
    const minutes = Math.floor(diff / MINUTE_MS);
    return `${minutes} min ago`;
  }
  if (diff < DAY_MS) {
    const hours = Math.floor(diff / HOUR_MS);
    return hours === 1 ? "1 hour ago" : `${hours} hours ago`;
  }
  if (diff < MONTH_MS) {
    const days = Math.floor(diff / DAY_MS);
    return days === 1 ? "1 day ago" : `${days} days ago`;
  }
  if (diff < YEAR_MS) {
    const months = Math.floor(diff / MONTH_MS);
    return months === 1 ? "1 month ago" : `${months} months ago`;
  }
  const years = Math.floor(diff / YEAR_MS);
  return years === 1 ? "1 year ago" : `${years} years ago`;
}

export function formatPlayedClock(iso: string): string {
  const date = new Date(iso);
  if (!Number.isFinite(date.getTime())) return "";
  return date.toLocaleTimeString(undefined, {
    hour: "numeric",
    minute: "2-digit",
  });
}

export function formatExactPlayedAt(iso: string): string {
  const date = new Date(iso);
  if (!Number.isFinite(date.getTime())) return "";
  return date.toLocaleString(undefined, {
    weekday: "short",
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}
