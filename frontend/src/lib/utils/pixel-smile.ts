// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { APP_SLUG } from "$lib/brand";

export interface PixelSmile {
  gridSize: number;
  cells: string[];
}

export interface PixelSmileSpec {
  gridSize: number;
  background: string;
  cx: number;
  eyeY: number;
  eyeGap: number;
  bigEyes: boolean;
  wink: boolean;
  mouthY: number;
  mouthType: number;
  blush: boolean;
}

export interface GazeOffset {
  x: number;
  y: number;
}

function hashSeed(seed: string): number {
  let hash = 0;
  for (let i = 0; i < seed.length; i++) {
    hash = (hash << 5) - hash + seed.charCodeAt(i);
    hash |= 0;
  }
  return Math.abs(hash) || 1;
}

function createRng(seed: number) {
  return () => {
    seed = (seed * 1664525 + 1013904223) >>> 0;
    return seed / 0xffffffff;
  };
}

function setCell(
  cells: string[],
  size: number,
  x: number,
  y: number,
  color: string,
) {
  if (x < 0 || y < 0 || x >= size || y >= size) return;
  cells[y * size + x] = color;
}

const BACKGROUNDS = [
  "#fbbf24",
  "#f472b6",
  "#60a5fa",
  "#34d399",
  "#a78bfa",
  "#fb923c",
  "#f87171",
  "#22d3ee",
  "#e879f9",
  "#84cc16",
];

const FEATURE = "#18181b";
const EYE_WHITE = "#fafafa";

function drawOpenEye(
  cells: string[],
  size: number,
  x: number,
  y: number,
  gaze: GazeOffset,
  tall: boolean,
) {
  const width = 2;
  const height = tall ? 2 : 1;

  for (let row = 0; row < height; row++) {
    for (let col = 0; col < width; col++) {
      setCell(cells, size, x + col, y + row, EYE_WHITE);
    }
  }

  const pupilX = x + (gaze.x > 0 ? width - 1 : 0);
  const pupilY = y + (tall && gaze.y > 0 ? height - 1 : 0);
  setCell(cells, size, pupilX, pupilY, FEATURE);
}

function drawSmile(cells: string[], size: number, cx: number, mouthY: number) {
  setCell(cells, size, cx - 2, mouthY, FEATURE);
  setCell(cells, size, cx - 1, mouthY + 1, FEATURE);
  setCell(cells, size, cx, mouthY + 1, FEATURE);
  setCell(cells, size, cx + 1, mouthY + 1, FEATURE);
  setCell(cells, size, cx + 2, mouthY, FEATURE);
}

function drawGrin(cells: string[], size: number, cx: number, mouthY: number) {
  for (let x = cx - 2; x <= cx + 2; x++) {
    setCell(cells, size, x, mouthY, FEATURE);
    setCell(cells, size, x, mouthY + 1, FEATURE);
  }
}

function drawSurprised(
  cells: string[],
  size: number,
  cx: number,
  mouthY: number,
) {
  setCell(cells, size, cx, mouthY, FEATURE);
  setCell(cells, size, cx, mouthY + 1, FEATURE);
  setCell(cells, size, cx - 1, mouthY + 1, FEATURE);
  setCell(cells, size, cx + 1, mouthY + 1, FEATURE);
}

function drawNeutral(
  cells: string[],
  size: number,
  cx: number,
  mouthY: number,
) {
  setCell(cells, size, cx - 1, mouthY + 1, FEATURE);
  setCell(cells, size, cx, mouthY + 1, FEATURE);
  setCell(cells, size, cx + 1, mouthY + 1, FEATURE);
}

function drawEyes(
  cells: string[],
  size: number,
  cx: number,
  eyeY: number,
  eyeGap: number,
  bigEyes: boolean,
  wink: boolean,
  background: string,
  gaze: GazeOffset,
) {
  drawOpenEye(cells, size, cx - eyeGap, eyeY, gaze, bigEyes);

  if (!wink) {
    drawOpenEye(cells, size, cx + eyeGap, eyeY, gaze, bigEyes);
    return;
  }

  setCell(cells, size, cx + eyeGap, eyeY, background);
  setCell(cells, size, cx + eyeGap - 1, eyeY, FEATURE);
  setCell(cells, size, cx + eyeGap + 1, eyeY, FEATURE);
}

export function gazeFromPointer(
  rect: DOMRect,
  clientX: number,
  clientY: number,
): GazeOffset {
  const cx = rect.left + rect.width / 2;
  const cy = rect.top + rect.height / 2;
  const dx = clientX - cx;
  const dy = clientY - cy;
  const len = Math.hypot(dx, dy);

  if (len < 12) {
    return { x: 0, y: 0 };
  }

  const nx = dx / len;
  const ny = dy / len;

  return {
    x: nx > 0.35 ? 1 : nx < -0.35 ? -1 : 0,
    y: ny > 0.35 ? 1 : ny < -0.35 ? -1 : 0,
  };
}

export function generatePixelSmileSpec(seed = APP_SLUG): PixelSmileSpec {
  const rand = createRng(hashSeed(seed));
  const gridSize = 8;
  const background = BACKGROUNDS[Math.floor(rand() * BACKGROUNDS.length)];
  const cx = Math.floor(gridSize / 2);

  return {
    gridSize,
    background,
    cx,
    eyeY: 2 + Math.floor(rand() * 2),
    eyeGap: 1 + Math.floor(rand() * 2),
    bigEyes: rand() > 0.45,
    wink: rand() > 0.82,
    mouthY: gridSize - 3,
    mouthType: Math.floor(rand() * 4),
    blush: rand() > 0.55,
  };
}

export function renderPixelSmile(
  spec: PixelSmileSpec,
  gaze: GazeOffset = { x: 0, y: 0 },
): PixelSmile {
  const { gridSize, background, cx } = spec;
  const cells = Array.from({ length: gridSize * gridSize }, () => background);

  drawEyes(
    cells,
    gridSize,
    cx,
    spec.eyeY,
    spec.eyeGap,
    spec.bigEyes,
    spec.wink,
    background,
    gaze,
  );

  if (spec.mouthType === 0) drawSmile(cells, gridSize, cx, spec.mouthY);
  else if (spec.mouthType === 1) drawGrin(cells, gridSize, cx, spec.mouthY);
  else if (spec.mouthType === 2)
    drawSurprised(cells, gridSize, cx, spec.mouthY);
  else drawNeutral(cells, gridSize, cx, spec.mouthY);

  if (spec.blush) {
    const blush = "#fb7185";
    setCell(cells, gridSize, cx - spec.eyeGap - 1, spec.eyeY + 1, blush);
    setCell(cells, gridSize, cx + spec.eyeGap + 1, spec.eyeY + 1, blush);
  }

  return { gridSize, cells };
}

export function generatePixelSmile(seed = APP_SLUG): PixelSmile {
  return renderPixelSmile(generatePixelSmileSpec(seed));
}
