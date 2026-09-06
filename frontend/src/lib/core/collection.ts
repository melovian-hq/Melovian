// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export function dedupeBy<T>(items: T[], keyFn: (item: T) => string): T[] {
  const seen = new Set<string>();
  const unique: T[] = [];
  for (const item of items) {
    const key = keyFn(item);
    if (seen.has(key)) continue;
    seen.add(key);
    unique.push(item);
  }
  return unique;
}

export function stableItemKey(
  id: string | number | undefined | null,
  index: number,
  fallback = "item",
): string {
  if (id != null && String(id) !== "") return String(id);
  return `${fallback}-${index}`;
}

export function listItemKey(
  id: string | number | undefined | null,
  index: number,
  fallback = "item",
): string {
  const base =
    id != null && String(id) !== "" ? String(id) : `${fallback}-${index}`;
  return `${base}#${index}`;
}
