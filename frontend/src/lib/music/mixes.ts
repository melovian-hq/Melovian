// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export interface MixLike {
  id: string;
}

export function findMix<T extends MixLike>(
  mixes: T[],
  id: string,
): T | undefined {
  if (!id) return undefined;
  const direct = mixes.find((mix) => mix.id === id);
  if (direct) return direct;
  // Legacy single-slot ids from before multi genre/decade mixes.
  if (id === "genre-mix") {
    return mixes.find((mix) => mix.id.startsWith("genre-mix-"));
  }
  if (id === "decade-mix") {
    return mixes.find((mix) => mix.id.startsWith("decade-mix-"));
  }
  return undefined;
}
