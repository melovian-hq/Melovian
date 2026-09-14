// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from "vitest";
import {
  failuresSuggestOutage,
  OFFLINE_SUSPECT_SKIP_THRESHOLD,
  playbackHoldActive,
} from "./offline-gate";

describe("playbackHoldActive", () => {
  it("does not hold when the connection is unmanaged", () => {
    expect(
      playbackHoldActive({
        managed: false,
        browserOnline: false,
        serverOnline: false,
      }),
    ).toBe(false);
  });

  it("holds when the server is unreachable", () => {
    expect(
      playbackHoldActive({
        managed: true,
        browserOnline: true,
        serverOnline: false,
      }),
    ).toBe(true);
  });

  it("holds when the browser reports offline", () => {
    expect(
      playbackHoldActive({
        managed: true,
        browserOnline: false,
        serverOnline: true,
      }),
    ).toBe(true);
  });

  it("does not hold while the managed connection is healthy", () => {
    expect(
      playbackHoldActive({
        managed: true,
        browserOnline: true,
        serverOnline: true,
      }),
    ).toBe(false);
  });
});

describe("failuresSuggestOutage", () => {
  it("stays below the threshold for isolated failures", () => {
    expect(failuresSuggestOutage(0)).toBe(false);
    expect(failuresSuggestOutage(OFFLINE_SUSPECT_SKIP_THRESHOLD - 1)).toBe(
      false,
    );
  });

  it("fires at the threshold", () => {
    expect(failuresSuggestOutage(OFFLINE_SUSPECT_SKIP_THRESHOLD)).toBe(true);
    expect(failuresSuggestOutage(OFFLINE_SUSPECT_SKIP_THRESHOLD + 5)).toBe(
      true,
    );
  });
});
