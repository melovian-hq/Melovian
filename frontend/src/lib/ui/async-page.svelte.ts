// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { toast } from "$lib/ui/toast.svelte";

const SKIP = Symbol("async-page.skip");
type Skip = typeof SKIP;

/**
 * Controls one in-flight load run. `cancelled` flips when a newer run starts
 * or the owning component tears down, so writes after an `await` should check
 * it first. Return `run.skip()` to end a run quietly, or `run.wait(loading)`
 * to park it with `loading` pinned while an external gate (library connect,
 * auth) resolves. The gate's reactive reads are tracked, so the run retries
 * when they change.
 */
export interface AsyncPageRun {
  readonly cancelled: boolean;
  skip(): Skip;
  wait(loading?: boolean): Skip;
}

class Run implements AsyncPageRun {
  cancelled = false;
  hold: boolean | undefined;

  skip(): Skip {
    return SKIP;
  }

  wait(loading = true): Skip {
    this.hold = loading;
    return SKIP;
  }
}

function errorText(err: unknown, fallback?: string): string {
  if (err instanceof Error) return err.message;
  return fallback ?? String(err);
}

export interface AsyncPageOptions<T> {
  /**
   * Runs inside a tracked effect. Reactive values read synchronously (route
   * props, store state) retrigger the run when they change. Throw to land in
   * `error`, return a value for `apply`, or return `run.skip()`/`run.wait()`
   * to end the run without applying.
   */
  load: (run: AsyncPageRun) => T | Skip | Promise<T | Skip>;
  /** Called with the load result when the run is still current. */
  apply?: (value: T, run: AsyncPageRun) => void;
  /** Fallback message when the load throws something that is not an Error. */
  errorMessage?: string;
  /** Called after `error` is set, still guarded by cancellation. */
  onError?: () => void;
  /** Also surface the failure as a toast. */
  toastOnError?: boolean;
  /** Effect teardown, for releasing page-owned resources on re-run/unmount. */
  cleanup?: () => void;
}

class AsyncPage<T> {
  loading = $state(true);
  error = $state<string | null>(null);
  #run: Run | null = null;

  constructor(private options: AsyncPageOptions<T>) {
    $effect(() => {
      void this.start();
      return () => {
        if (this.#run) this.#run.cancelled = true;
        this.options.cleanup?.();
      };
    });
  }

  /** Re-run `load` outside the effect, e.g. after a retry action. */
  reload = (): Promise<void> => this.start();

  private async start(): Promise<void> {
    if (this.#run) this.#run.cancelled = true;
    const run = new Run();
    this.#run = run;
    this.loading = true;
    this.error = null;
    try {
      const value = await this.options.load(run);
      if (run.cancelled || value === SKIP) return;
      this.options.apply?.(value as T, run);
    } catch (err) {
      if (run.cancelled) return;
      this.error = errorText(err, this.options.errorMessage);
      if (this.options.toastOnError && this.error) toast.error(this.error);
      this.options.onError?.();
    } finally {
      if (!run.cancelled) this.loading = run.hold ?? false;
    }
  }
}

export function createAsyncPage<T>(options: AsyncPageOptions<T>): AsyncPage<T> {
  return new AsyncPage(options);
}

export interface PagedListResult<T> {
  items: T[];
  hasMore: boolean;
}

export interface PagedListOptions<T> {
  /**
   * Fetches one page at `offset` (0 for the initial/refresh run). The run
   * re-triggers on reactive reads like `createAsyncPage`. `loadMore` appends
   * the next page; a failed `loadMore` just closes the list (`hasMore` off).
   */
  load: (
    offset: number,
    run: AsyncPageRun,
  ) => PagedListResult<T> | Skip | Promise<PagedListResult<T> | Skip>;
  /** Extra gate for `loadMore`, e.g. an active local search. */
  canLoadMore?: () => boolean;
  errorMessage?: string;
  onError?: () => void;
  toastOnError?: boolean;
}

class PagedList<T> {
  items = $state<T[]>([]);
  loading = $state(true);
  loadingMore = $state(false);
  hasMore = $state(false);
  error = $state<string | null>(null);
  #run: Run | null = null;

  constructor(private options: PagedListOptions<T>) {
    $effect(() => {
      void this.start(0);
      return () => {
        if (this.#run) this.#run.cancelled = true;
      };
    });
  }

  loadMore = async (): Promise<void> => {
    if (this.loading || this.loadingMore || !this.hasMore) return;
    if (this.options.canLoadMore && !this.options.canLoadMore()) return;
    await this.start(this.items.length);
  };

  private async start(offset: number): Promise<void> {
    if (this.#run) this.#run.cancelled = true;
    const run = new Run();
    this.#run = run;
    const initial = offset === 0;
    if (initial) {
      this.loading = true;
      this.loadingMore = false;
      this.error = null;
      this.hasMore = false;
    } else {
      this.loadingMore = true;
    }
    try {
      const result = await this.options.load(offset, run);
      if (run.cancelled || result === SKIP) return;
      this.items = initial ? result.items : [...this.items, ...result.items];
      this.hasMore = result.hasMore;
    } catch (err) {
      if (run.cancelled) return;
      if (initial) {
        this.error = errorText(err, this.options.errorMessage);
        if (this.options.toastOnError && this.error) toast.error(this.error);
        this.options.onError?.();
      } else {
        this.hasMore = false;
      }
    } finally {
      if (!run.cancelled) {
        if (initial) this.loading = run.hold ?? false;
        else this.loadingMore = false;
      }
    }
  }
}

export function createPagedList<T>(options: PagedListOptions<T>): PagedList<T> {
  return new PagedList(options);
}
