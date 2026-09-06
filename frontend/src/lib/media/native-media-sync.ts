// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

let syncFn: (() => void) | null = null;

export function setNativeMediaSync(fn: (() => void) | null): void {
  syncFn = fn;
}

export function requestNativeMediaSync(): void {
  syncFn?.();
}
