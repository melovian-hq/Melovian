// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

// Scheme check for URLs that end up in href or window.open. Anything
// that is not http or https (javascript:, data:, vbscript:) returns ""
// so callers can gate rendering with a truthy check.
export function safeHttpUrl(url: string | undefined | null): string {
  const trimmed = (url ?? "").trim();
  if (!/^https?:\/\//i.test(trimmed)) return "";
  return trimmed;
}
