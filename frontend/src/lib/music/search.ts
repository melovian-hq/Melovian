// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

/**
 * String similarity helpers for "did you mean" suggestions and fuzzy ranking.
 * Uses the Sorensen-Dice coefficient over character bigrams, which handles
 * typos and word reordering better than raw edit distance for short queries.
 */

export function normalizeTerm(value: string): string {
  return value
    .toLowerCase()
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/[^a-z0-9\s]/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

function bigrams(value: string): Map<string, number> {
  const grams = new Map<string, number>();
  const clean = value.replace(/\s+/g, "");
  for (let i = 0; i < clean.length - 1; i++) {
    const gram = clean.slice(i, i + 2);
    grams.set(gram, (grams.get(gram) ?? 0) + 1);
  }
  return grams;
}

/**
 * diceCoefficient returns a similarity score in [0, 1] where 1 is an exact
 * match. Single-character strings fall back to equality.
 */
export function diceCoefficient(a: string, b: string): number {
  const x = normalizeTerm(a);
  const y = normalizeTerm(b);
  if (x === y) return 1;
  if (x.length < 2 || y.length < 2) return x === y ? 1 : 0;

  const aGrams = bigrams(x);
  const bGrams = bigrams(y);
  let intersection = 0;
  let aTotal = 0;
  for (const count of aGrams.values()) aTotal += count;
  let bTotal = 0;
  for (const count of bGrams.values()) bTotal += count;

  for (const [gram, countA] of aGrams) {
    const countB = bGrams.get(gram);
    if (countB) intersection += Math.min(countA, countB);
  }

  return (2 * intersection) / (aTotal + bTotal);
}

export interface ScoredMatch<T> {
  item: T;
  score: number;
}

/**
 * bestMatches ranks candidates by similarity to the query, keeping only those
 * above the threshold. Substring hits are boosted so partial queries surface
 * obvious matches even when the bigram score is low.
 */
export function bestMatches<T>(
  query: string,
  candidates: T[],
  label: (item: T) => string,
  options: { limit?: number; threshold?: number } = {},
): ScoredMatch<T>[] {
  const limit = options.limit ?? 5;
  const threshold = options.threshold ?? 0.3;
  const normalizedQuery = normalizeTerm(query);
  if (normalizedQuery === "") return [];

  const scored: ScoredMatch<T>[] = [];
  for (const item of candidates) {
    const name = label(item);
    const normalizedName = normalizeTerm(name);
    let score = diceCoefficient(normalizedQuery, normalizedName);
    if (normalizedName.includes(normalizedQuery)) {
      score = Math.max(score, 0.85);
    }
    if (score >= threshold) {
      scored.push({ item, score });
    }
  }

  scored.sort((a, b) => b.score - a.score);
  return scored.slice(0, limit);
}

/**
 * suggestion returns the closest candidate label when it differs meaningfully
 * from the query, suitable for a "did you mean" prompt.
 */
export function suggestion(query: string, candidates: string[]): string | null {
  const matches = bestMatches(query, candidates, (c) => c, {
    limit: 1,
    threshold: 0.4,
  });
  if (matches.length === 0) return null;
  const best = matches[0];
  if (normalizeTerm(best.item) === normalizeTerm(query)) return null;
  return best.item;
}
