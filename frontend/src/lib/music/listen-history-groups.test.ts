// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import type { ListenEvent } from "$lib/subsonic/types";
import {
  groupListenEventsByDay,
  listenDayLabel,
  localDayKey,
} from "./listen-history-groups";

function eventAt(
  year: number,
  month: number,
  day: number,
  hour = 12,
): ListenEvent {
  return {
    id: year * 10000 + month * 100 + day + hour,
    trackId: `${year}-${month}-${day}`,
    trackTitle: "Track",
    artistName: "Artist",
    albumId: "alb",
    albumTitle: "Album",
    durationMs: 180_000,
    coverArtId: "cov",
    playedAt: new Date(year, month - 1, day, hour).toISOString(),
  };
}

describe("listen-history-groups", () => {
  const now = new Date(2026, 7, 17, 18, 0, 0);

  it("builds a local day key", () => {
    expect(localDayKey(new Date(2026, 7, 17, 9, 30))).toBe("2026-08-17");
    expect(localDayKey(new Date("not a date"))).toBe("unknown");
  });

  it("labels today yesterday weekdays and older dates", () => {
    expect(listenDayLabel("2026-08-17", now)).toBe("Today");
    expect(listenDayLabel("2026-08-16", now)).toBe("Yesterday");
    expect(listenDayLabel("2026-08-15", now)).toBe("Saturday");
    expect(listenDayLabel("2026-08-01", now)).toBe("August 1");
    expect(listenDayLabel("2025-07-03", now)).toBe("July 3, 2025");
    expect(listenDayLabel("unknown", now)).toBe("Unknown date");
  });

  it("groups events in first-seen day order", () => {
    const groups = groupListenEventsByDay(
      [
        eventAt(2026, 8, 17, 18),
        eventAt(2026, 8, 17, 10),
        eventAt(2026, 8, 16, 21),
        eventAt(2026, 8, 1, 8),
      ],
      now,
    );

    expect(groups.map((group) => group.label)).toEqual([
      "Today",
      "Yesterday",
      "August 1",
    ]);
    expect(groups[0].events).toHaveLength(2);
    expect(groups[1].events).toHaveLength(1);
    expect(groups[2].events).toHaveLength(1);
  });
});
