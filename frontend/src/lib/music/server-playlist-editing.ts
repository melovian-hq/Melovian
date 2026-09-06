// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export function playlistOrderChanged(
  current: readonly string[],
  next: readonly string[],
): boolean {
  if (current.length !== next.length) return true;
  return current.some((id, index) => id !== next[index]);
}

export function buildPlaylistReorderParams(
  currentSongIds: readonly string[],
  nextSongIds: readonly string[],
): {
  songIndexToRemove: number[];
  songIdToAdd: string[];
} | null {
  if (!playlistOrderChanged(currentSongIds, nextSongIds)) {
    return null;
  }
  const currentSet = new Set(currentSongIds);
  const nextSet = new Set(nextSongIds);
  if (currentSet.size !== currentSongIds.length) {
    throw new Error("Duplicate song IDs in current playlist");
  }
  if (nextSet.size !== nextSongIds.length) {
    throw new Error("Duplicate song IDs in target playlist");
  }
  if (currentSet.size !== nextSet.size) {
    throw new Error("Target playlist must contain the same songs");
  }
  for (const id of currentSet) {
    if (!nextSet.has(id)) {
      throw new Error("Target playlist must contain the same songs");
    }
  }
  const songIndexToRemove = Array.from(
    { length: currentSongIds.length },
    (_, index) => currentSongIds.length - 1 - index,
  );
  return {
    songIndexToRemove,
    songIdToAdd: [...nextSongIds],
  };
}

export function movePlaylistSong(
  songIds: readonly string[],
  fromIndex: number,
  toIndex: number,
): string[] {
  if (
    fromIndex < 0 ||
    toIndex < 0 ||
    fromIndex >= songIds.length ||
    toIndex >= songIds.length ||
    fromIndex === toIndex
  ) {
    return [...songIds];
  }
  const next = [...songIds];
  const [item] = next.splice(fromIndex, 1);
  next.splice(toIndex, 0, item);
  return next;
}
