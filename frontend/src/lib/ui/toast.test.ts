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
});
