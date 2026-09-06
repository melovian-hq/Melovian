// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

export type RepeatMode = "off" | "all" | "one";

export function shuffleIndices(length: number, excludeIndex = -1): number[] {
  const indices: number[] = [];
  for (let i = 0; i < length; i++) {
    if (i !== excludeIndex) indices.push(i);
  }
  for (let i = indices.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [indices[i], indices[j]] = [indices[j], indices[i]];
  }
  return indices;
}

export function shuffleNewIndices(
  from: number,
  to: number,
  excludeIndex: number,
): number[] {
  const indices: number[] = [];
  for (let i = from; i < to; i++) {
    if (i !== excludeIndex) indices.push(i);
  }
  for (let i = indices.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [indices[i], indices[j]] = [indices[j], indices[i]];
  }
  return indices;
}

export function shiftIndicesForInsert(
  indices: number[],
  insertAt: number,
  count: number,
): number[] {
  if (count <= 0) return [...indices];
  return indices.map((i) => (i >= insertAt ? i + count : i));
}

export function playNextShuffleUpcoming(
  upcoming: number[],
  insertAt: number,
  count: number,
): number[] {
  if (count <= 0) return [...upcoming];
  const shifted = shiftIndicesForInsert(upcoming, insertAt, count);
  const inserted: number[] = [];
  for (let i = 0; i < count; i++) inserted.push(insertAt + i);
  return [...inserted, ...shifted];
}

export function queueIndexAfterRemove(
  queueIndex: number,
  removedIndex: number,
  newLength: number,
): { queueIndex: number; restart: boolean } {
  if (newLength <= 0) return { queueIndex: -1, restart: false };
  if (removedIndex === queueIndex) {
    const next = queueIndex >= newLength ? newLength - 1 : queueIndex;
    return { queueIndex: next, restart: true };
  }
  if (removedIndex < queueIndex) {
    return { queueIndex: queueIndex - 1, restart: false };
  }
  if (queueIndex >= newLength) {
    return { queueIndex: newLength - 1, restart: false };
  }
  return { queueIndex, restart: false };
}

export function nextSequentialIndex(
  queueIndex: number,
  queueLength: number,
  repeat: RepeatMode,
): number | null {
  if (queueLength === 0 || queueIndex < 0) return null;
  if (queueIndex >= queueLength - 1) {
    return repeat === "all" ? 0 : null;
  }
  return queueIndex + 1;
}

export function shouldStopAtQueueEnd(
  atEnd: boolean,
  repeat: RepeatMode,
  autoplayExtended: boolean,
): boolean {
  if (!atEnd) return false;
  if (repeat === "all") return false;
  if (autoplayExtended) return false;
  return true;
}

export function canAdvanceQueue(
  queueIndex: number,
  queueLength: number,
  repeat: RepeatMode,
  shuffle: boolean,
): boolean {
  if (queueLength === 0) return false;
  if (shuffle && queueLength > 1) return true;
  return nextSequentialIndex(queueIndex, queueLength, repeat) !== null;
}

export type ShufflePreviousStep = {
  previousIndex: number;
  history: number[];
  prependUpcoming: number;
};

export function popShuffleHistory(
  history: number[],
  currentIndex: number,
): ShufflePreviousStep | null {
  if (history.length === 0 || currentIndex < 0) return null;
  const previousIndex = history[history.length - 1]!;
  return {
    previousIndex,
    history: history.slice(0, -1),
    prependUpcoming: currentIndex,
  };
}
