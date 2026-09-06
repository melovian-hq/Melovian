// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import type { ListenEvent } from "$lib/subsonic/types";

const DAY_MS = 86_400_000;

export interface ListenDayGroup {
  key: string;
  label: string;
  events: ListenEvent[];
}

export function localDayKey(date: Date): string {
  if (!Number.isFinite(date.getTime())) return "unknown";
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function startOfLocalDay(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

function parseLocalDayKey(key: string): Date | null {
  if (key === "unknown") return null;
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(key);
  if (!match) return null;
  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  const parsed = new Date(year, month - 1, day);
  if (!Number.isFinite(parsed.getTime())) return null;
  return parsed;
}

export function listenDayLabel(key: string, now = new Date()): string {
  if (key === "unknown") return "Unknown date";
  const date = parseLocalDayKey(key);
  if (!date) return "Unknown date";

  const today = startOfLocalDay(now);
  const yesterday = new Date(today.getTime() - DAY_MS);
  const target = startOfLocalDay(date);

  if (target.getTime() === today.getTime()) return "Today";
  if (target.getTime() === yesterday.getTime()) return "Yesterday";

  const diffDays = Math.round((today.getTime() - target.getTime()) / DAY_MS);
  if (diffDays >= 2 && diffDays < 7) {
    return date.toLocaleDateString("en-US", { weekday: "long" });
  }

  if (date.getFullYear() === now.getFullYear()) {
    return date.toLocaleDateString("en-US", {
      month: "long",
      day: "numeric",
    });
  }

  return date.toLocaleDateString("en-US", {
    month: "long",
    day: "numeric",
    year: "numeric",
  });
}

export function groupListenEventsByDay(
  events: readonly ListenEvent[],
  now = new Date(),
): ListenDayGroup[] {
  const groups: ListenDayGroup[] = [];
  const indexByKey = new Map<string, number>();

  for (const event of events) {
    const key = localDayKey(new Date(event.playedAt));
    let index = indexByKey.get(key);
    if (index === undefined) {
      index = groups.length;
      indexByKey.set(key, index);
      groups.push({
        key,
        label: listenDayLabel(key, now),
        events: [],
      });
    }
    groups[index].events.push(event);
  }

  return groups;
}
