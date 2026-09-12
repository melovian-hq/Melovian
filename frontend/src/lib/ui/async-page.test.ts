// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { afterEach, describe, expect, it, vi } from "vitest";
import { flushSync, mount, unmount } from "svelte";
import AsyncPageHarness from "../../test-fixtures/AsyncPageHarness.svelte";
import PagedListHarness from "../../test-fixtures/PagedListHarness.svelte";
import type { AsyncPageOptions, PagedListOptions } from "./async-page.svelte";

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (err: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

function mountPage(options: AsyncPageOptions<string>) {
  const target = document.createElement("div");
  document.body.appendChild(target);
  const harness = mount(AsyncPageHarness, { target, props: { options } });
  flushSync();
  return {
    page: harness.getPage(),
    cleanup: () => {
      void unmount(harness);
      target.remove();
    },
  };
}

function mountList(options: PagedListOptions<string>) {
  const target = document.createElement("div");
  document.body.appendChild(target);
  const harness = mount(PagedListHarness, { target, props: { options } });
  flushSync();
  return {
    list: harness.getList(),
    cleanup: () => {
      void unmount(harness);
      target.remove();
    },
  };
}

describe("createAsyncPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("applies the load result and clears loading", async () => {
    const pending = deferred<string>();
    const applied: string[] = [];
    const { page, cleanup } = mountPage({
      load: () => pending.promise,
      apply: (value) => applied.push(value),
    });
    expect(page.loading).toBe(true);
    expect(page.error).toBeNull();

    pending.resolve("ok");
    await vi.waitFor(() => expect(page.loading).toBe(false));
    expect(applied).toEqual(["ok"]);
    expect(page.error).toBeNull();
    cleanup();
  });

  it("captures thrown errors into error and runs onError", async () => {
    const onError = vi.fn();
    const { page, cleanup } = mountPage({
      errorMessage: "fallback",
      onError,
      load: () => Promise.reject(new Error("boom")),
    });
    await vi.waitFor(() => expect(page.loading).toBe(false));
    expect(page.error).toBe("boom");
    expect(onError).toHaveBeenCalledOnce();
    cleanup();
  });

  it("uses errorMessage for non-Error throws", async () => {
    const { page, cleanup } = mountPage({
      errorMessage: "fallback",
      load: () => Promise.reject("nope"),
    });
    await vi.waitFor(() => expect(page.loading).toBe(false));
    expect(page.error).toBe("fallback");
    cleanup();
  });

  it("ends quietly on run.skip()", async () => {
    const apply = vi.fn();
    const { page, cleanup } = mountPage({
      apply,
      load: (run) => run.skip(),
    });
    await vi.waitFor(() => expect(page.loading).toBe(false));
    expect(page.error).toBeNull();
    expect(apply).not.toHaveBeenCalled();
    cleanup();
  });

  it("keeps loading pinned on run.wait()", async () => {
    const apply = vi.fn();
    const { page, cleanup } = mountPage({
      apply,
      load: (run) => run.wait(),
    });
    await Promise.resolve();
    await Promise.resolve();
    expect(page.loading).toBe(true);
    expect(apply).not.toHaveBeenCalled();
    cleanup();
  });

  it("reload() re-runs the load and cancels the stale run", async () => {
    const first = deferred<string>();
    const second = deferred<string>();
    const applied: string[] = [];
    const calls: string[] = [];
    const { page, cleanup } = mountPage({
      apply: (value) => applied.push(value),
      load: () => {
        calls.push("load");
        return calls.length === 1 ? first.promise : second.promise;
      },
    });
    const reloading = page.reload();
    first.resolve("stale");
    second.resolve("fresh");
    await reloading;
    await vi.waitFor(() => expect(page.loading).toBe(false));
    expect(applied).toEqual(["fresh"]);
    cleanup();
  });

  it("does not apply after unmount", async () => {
    const pending = deferred<string>();
    const apply = vi.fn();
    const { page, cleanup } = mountPage({
      apply,
      load: () => pending.promise,
    });
    cleanup();
    pending.resolve("late");
    await Promise.resolve();
    expect(apply).not.toHaveBeenCalled();
    expect(page.loading).toBe(true);
  });
});

describe("createPagedList", () => {
  it("loads the first page and appends via loadMore", async () => {
    const { list, cleanup } = mountList({
      load: async (offset) => ({
        items: [`item-${offset}`],
        hasMore: true,
      }),
    });
    await vi.waitFor(() => expect(list.loading).toBe(false));
    expect(list.items).toEqual(["item-0"]);
    expect(list.hasMore).toBe(true);

    await list.loadMore();
    expect(list.items).toEqual(["item-0", "item-1"]);
    expect(list.loadingMore).toBe(false);
    cleanup();
  });

  it("blocks loadMore while loading or without hasMore", async () => {
    const load = vi.fn(async () => ({ items: ["a"], hasMore: false }));
    const { list, cleanup } = mountList({ load });
    await vi.waitFor(() => expect(list.loading).toBe(false));
    expect(list.hasMore).toBe(false);
    await list.loadMore();
    expect(load).toHaveBeenCalledTimes(1);
    cleanup();
  });

  it("respects canLoadMore", async () => {
    const load = vi.fn(async () => ({ items: ["a"], hasMore: true }));
    const { list, cleanup } = mountList({ load, canLoadMore: () => false });
    await vi.waitFor(() => expect(list.loading).toBe(false));
    await list.loadMore();
    expect(load).toHaveBeenCalledTimes(1);
    cleanup();
  });

  it("closes the list when loadMore fails", async () => {
    const { list, cleanup } = mountList({
      load: async (offset) => {
        if (offset > 0) throw new Error("boom");
        return { items: ["a"], hasMore: true };
      },
    });
    await vi.waitFor(() => expect(list.loading).toBe(false));
    await list.loadMore();
    expect(list.hasMore).toBe(false);
    expect(list.error).toBeNull();
    cleanup();
  });

  it("captures initial load errors", async () => {
    const { list, cleanup } = mountList({
      errorMessage: "Failed to load",
      load: () => Promise.reject(new Error("boom")),
    });
    await vi.waitFor(() => expect(list.loading).toBe(false));
    expect(list.error).toBe("boom");
    expect(list.items).toEqual([]);
    cleanup();
  });
});
