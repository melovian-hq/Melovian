// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from "vitest";
import { toast } from "./toast.svelte";

describe("toast store", () => {
  beforeEach(() => {
    for (const item of [...toast.items]) {
      toast.dismiss(item.id);
    }
  });

  it("keeps at most four toasts", () => {
    toast.info("one");
    toast.info("two");
    toast.info("three");
    toast.info("four");
    toast.info("five");
    expect(toast.items).toHaveLength(4);
    expect(toast.items.map((t) => t.message)).toEqual([
      "two",
      "three",
      "four",
      "five",
    ]);
  });

  it("supports title and action", () => {
    const onClick = vi.fn();
    toast.success("Shared", {
      title: "Playlist",
      action: { label: "Open", onClick },
      duration: 0,
    });
    expect(toast.items[0]?.title).toBe("Playlist");
    expect(toast.items[0]?.action?.label).toBe("Open");
    expect(toast.items[0]?.actions).toHaveLength(1);
    toast.items[0]?.action?.onClick();
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("supports multiple actions", () => {
    const copy = vi.fn();
    const invite = vi.fn();
    toast.info("Share an invite", {
      title: "Listen together",
      actions: [
        { label: "Copy link", onClick: copy },
        { label: "Invite", onClick: invite },
      ],
      duration: 0,
    });
    expect(toast.items[0]?.actions?.map((a) => a.label)).toEqual([
      "Copy link",
      "Invite",
    ]);
    expect(toast.items[0]?.action?.label).toBe("Copy link");
  });

  it("sets the kind through the helper methods", () => {
    toast.success("a", { duration: 0 });
    toast.error("b", { duration: 0 });
    toast.warning("c", { duration: 0 });
    toast.info("d", { duration: 0 });
    expect(toast.items.map((t) => t.kind)).toEqual([
      "success",
      "error",
      "warning",
      "info",
    ]);
  });

  it("dedupes a visible toast with the same kind and message", () => {
    const first = toast.info("Saved", { duration: 0 });
    const second = toast.info("Saved", { duration: 0 });
    expect(second).toBe(first);
    expect(toast.items).toHaveLength(1);
  });

  it("does not dedupe across kinds", () => {
    toast.error("Saved", { duration: 0 });
    toast.info("Saved", { duration: 0 });
    expect(toast.items).toHaveLength(2);
  });

  it("restarts the timer on a duplicate", () => {
    vi.useFakeTimers();
    try {
      const first = toast.info("Saved");
      vi.advanceTimersByTime(3900);
      const second = toast.info("Saved");
      expect(second).toBe(first);
      expect(toast.items).toHaveLength(1);
      vi.advanceTimersByTime(3900);
      expect(toast.items).toHaveLength(1);
      vi.advanceTimersByTime(200);
      expect(toast.items).toHaveLength(0);
    } finally {
      vi.useRealTimers();
    }
  });

  it("keeps a paused duplicate paused while resetting its remaining time", () => {
    vi.useFakeTimers();
    try {
      const id = toast.info("Saved");
      vi.advanceTimersByTime(3900);
      toast.pause(id);
      expect(toast.info("Saved")).toBe(id);
      toast.resume(id);
      vi.advanceTimersByTime(3900);
      expect(toast.items).toHaveLength(1);
      vi.advanceTimersByTime(200);
      expect(toast.items).toHaveLength(0);
    } finally {
      vi.useRealTimers();
    }
  });
});
