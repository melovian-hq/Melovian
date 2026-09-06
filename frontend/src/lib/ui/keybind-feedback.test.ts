// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from "vitest";
import {
  formatSeekLabel,
  KeybindFeedbackStore,
  KEYBIND_FEEDBACK_HOLD_MS,
  KEYBIND_FEEDBACK_STACK_MS,
} from "./keybind-feedback.svelte";

describe("formatSeekLabel", () => {
  it("signs positive and negative seeks", () => {
    expect(formatSeekLabel(5)).toBe("+5s");
    expect(formatSeekLabel(-5)).toBe("-5s");
    expect(formatSeekLabel(10.4)).toBe("+10s");
  });
});

describe("KeybindFeedbackStore", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("stacks seeks in the same direction", () => {
    const store = new KeybindFeedbackStore();
    store.seek(5, 1_000);
    expect(store.current?.label).toBe("+5s");
    store.seek(5, 1_000 + KEYBIND_FEEDBACK_STACK_MS - 1);
    expect(store.current?.label).toBe("+10s");
    expect(store.current?.direction).toBe("forward");
  });

  it("starts a new total after the stack window", () => {
    const store = new KeybindFeedbackStore();
    store.seek(-5, 1_000);
    store.seek(-5, 1_000 + KEYBIND_FEEDBACK_STACK_MS + 1);
    expect(store.current?.label).toBe("-5s");
  });

  it("resets when seek direction flips", () => {
    const store = new KeybindFeedbackStore();
    store.seek(5, 1_000);
    store.seek(-5, 1_100);
    expect(store.current?.label).toBe("-5s");
    expect(store.current?.direction).toBe("back");
  });

  it("shows volume percent and play or skip labels", () => {
    const store = new KeybindFeedbackStore();
    store.volume(0.42);
    expect(store.current?.kind).toBe("volume");
    expect(store.current?.label).toBe("42%");
    expect(store.current?.volumePct).toBe(42);

    store.playback(true);
    expect(store.current?.kind).toBe("play");
    store.playback(false);
    expect(store.current?.kind).toBe("pause");
    store.skip("next");
    expect(store.current?.label).toBe("Next");
    store.skip("prev");
    expect(store.current?.label).toBe("Previous");
  });

  it("ignores non-finite volume levels", () => {
    const store = new KeybindFeedbackStore();
    store.volume(0.5);
    store.volume(Number.NaN);
    expect(store.current?.label).toBe("50%");
    store.volume(Number.POSITIVE_INFINITY);
    expect(store.current?.label).toBe("50%");
  });

  it("clears after the hold timeout", () => {
    vi.useFakeTimers();
    const store = new KeybindFeedbackStore();
    store.seek(5);
    expect(store.current).not.toBeNull();
    vi.advanceTimersByTime(KEYBIND_FEEDBACK_HOLD_MS);
    expect(store.current).toBeNull();
  });
});
