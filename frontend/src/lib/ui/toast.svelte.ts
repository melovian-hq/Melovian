// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { randomUUID } from "$lib/utils/uuid";

export type ToastKind = "success" | "error" | "info" | "warning";

export interface ToastAction {
  label: string;
  onClick: () => void;
}

export interface ToastShowOptions {
  kind?: ToastKind;
  duration?: number;
  title?: string;
  action?: ToastAction;
  actions?: ToastAction[];
}

export interface ToastItem {
  id: string;
  kind: ToastKind;
  message: string;
  duration: number;
  title?: string;
  action?: ToastAction;
  actions?: ToastAction[];
}

const MAX_TOASTS = 4;

const DEFAULT_DURATION: Record<ToastKind, number> = {
  info: 4000,
  success: 4000,
  warning: 5000,
  error: 6000,
};

class ToastStore {
  items = $state<ToastItem[]>([]);
  #timers = new Map<string, ReturnType<typeof setTimeout>>();
  #remaining = new Map<string, number>();
  #startedAt = new Map<string, number>();
  #paused = new Set<string>();

  show(
    message: string,
    kindOrOpts: ToastKind | ToastShowOptions = "info",
    duration?: number,
  ) {
    const opts: ToastShowOptions =
      typeof kindOrOpts === "string"
        ? { kind: kindOrOpts, duration }
        : kindOrOpts;
    const kind = opts.kind ?? "info";
    const resolvedDuration =
      opts.duration ?? duration ?? DEFAULT_DURATION[kind];
    const id = randomUUID();
    const actions =
      opts.actions && opts.actions.length > 0
        ? opts.actions
        : opts.action
          ? [opts.action]
          : undefined;
    const item: ToastItem = {
      id,
      kind,
      message,
      duration: resolvedDuration,
      title: opts.title,
      action: actions?.[0],
      actions,
    };
    const next = [...this.items, item];
    const dropped = next.length > MAX_TOASTS ? next.slice(0, -MAX_TOASTS) : [];
    for (const stale of dropped) {
      this.clearTimer(stale.id);
    }
    this.items = next.slice(-MAX_TOASTS);
    if (resolvedDuration > 0) {
      this.armTimer(id, resolvedDuration);
    }
    return id;
  }

  success(message: string, opts?: Omit<ToastShowOptions, "kind">) {
    return this.show(message, { ...opts, kind: "success" });
  }

  error(message: string, opts?: Omit<ToastShowOptions, "kind">) {
    return this.show(message, {
      ...opts,
      kind: "error",
      duration: opts?.duration ?? DEFAULT_DURATION.error,
    });
  }

  info(message: string, opts?: Omit<ToastShowOptions, "kind">) {
    return this.show(message, { ...opts, kind: "info" });
  }

  warning(message: string, opts?: Omit<ToastShowOptions, "kind">) {
    return this.show(message, {
      ...opts,
      kind: "warning",
      duration: opts?.duration ?? DEFAULT_DURATION.warning,
    });
  }

  dismiss(id: string) {
    this.clearTimer(id);
    this.items = this.items.filter((t) => t.id !== id);
  }

  pause(id: string) {
    if (this.#paused.has(id)) return;
    const started = this.#startedAt.get(id);
    const remaining = this.#remaining.get(id);
    if (started == null || remaining == null) return;
    const left = Math.max(0, remaining - (Date.now() - started));
    this.#remaining.set(id, left);
    this.#paused.add(id);
    this.clearTimer(id, false);
  }

  resume(id: string) {
    if (!this.#paused.has(id)) return;
    this.#paused.delete(id);
    const left = this.#remaining.get(id);
    if (left == null || left <= 0) {
      this.dismiss(id);
      return;
    }
    this.armTimer(id, left);
  }

  armTimer(id: string, ms: number) {
    this.clearTimer(id, false);
    this.#remaining.set(id, ms);
    this.#startedAt.set(id, Date.now());
    this.#timers.set(
      id,
      setTimeout(() => this.dismiss(id), ms),
    );
  }

  clearTimer(id: string, clearRemaining = true) {
    const timer = this.#timers.get(id);
    if (timer != null) clearTimeout(timer);
    this.#timers.delete(id);
    this.#startedAt.delete(id);
    this.#paused.delete(id);
    if (clearRemaining) this.#remaining.delete(id);
  }
}

export const toast = new ToastStore();
