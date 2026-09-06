// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export function normalizeSearchQuery(query: string): string {
  return query.trim().toLowerCase().normalize("NFKD").replace(/\p{M}/gu, "");
}

export function matchesLocalSearch(
  query: string,
  ...fields: (string | undefined | null)[]
): boolean {
  const normalized = normalizeSearchQuery(query);
  if (!normalized) return true;
  return fields.some((field) => {
    if (!field) return false;
    return normalizeSearchQuery(field).includes(normalized);
  });
}

export function filterByLocalSearch<T>(
  items: readonly T[],
  query: string,
  getFields: (item: T) => (string | undefined | null)[],
): T[] {
  if (!normalizeSearchQuery(query)) return [...items];
  return items.filter((item) => matchesLocalSearch(query, ...getFields(item)));
}
