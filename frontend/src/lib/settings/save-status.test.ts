// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from "vitest";
import { createSaveStatus } from "./save-status.svelte";

describe("SaveStatus", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("starts idle", () => {
    const status = createSaveStatus();
    expect(status.state).toBe("idle");
    expect(status.error).toBe("");
  });

  it("tracks the saving to saved lifecycle", () => {
    const status = createSaveStatus();
    status.begin();
    expect(status.state).toBe("saving");
    status.saved();
    expect(status.state).toBe("saved");
    expect(status.error).toBe("");
  });

  it("returns to idle after the saved flash", () => {
    vi.useFakeTimers();
    const status = createSaveStatus();
    status.saved();
    expect(status.state).toBe("saved");
    vi.advanceTimersByTime(1600);
    expect(status.state).toBe("idle");
  });

  it("keeps the saved flash when a new save starts before it clears", () => {
    vi.useFakeTimers();
    const status = createSaveStatus();
    status.saved();
    status.begin();
    expect(status.state).toBe("saving");
    vi.advanceTimersByTime(1600);
    // The stale timer must not drag an in-progress save back to idle.
    expect(status.state).toBe("saving");
  });

  it("records the failure message and clears it on the next save", () => {
    const status = createSaveStatus();
    status.failed("network unreachable");
    expect(status.state).toBe("error");
    expect(status.error).toBe("network unreachable");
    status.begin();
    expect(status.state).toBe("saving");
    expect(status.error).toBe("");
  });
});
