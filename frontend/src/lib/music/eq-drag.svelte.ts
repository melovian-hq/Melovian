// SPDX-FileCopyrightText: 2026 Quad4 Software
// SPDX-License-Identifier: Apache-2.0

import { music } from "$lib/config/music.svelte";
import { xToFreq, yToGain } from "$lib/music/eq-curve";
import { CURVE_HEIGHT, CURVE_WIDTH } from "$lib/music/eq-layout";

export class EqChartDrag {
  draggingBand = $state<number | null>(null);
  curveEl = $state<SVGSVGElement | null>(null);
  chartEl = $state<HTMLDivElement | null>(null);

  constructor(private readonly selectBand: (index: number) => void) {}

  onBandPointerDown(index: number, event: PointerEvent) {
    if (!music.eq.enabled) return;
    event.preventDefault();
    event.stopPropagation();
    this.selectBand(index);
    this.draggingBand = index;
    this.chartEl?.setPointerCapture(event.pointerId);
  }

  onChartPointerMove(event: PointerEvent) {
    if (this.draggingBand === null || !music.eq.enabled || !this.curveEl)
      return;
    const rect = this.curveEl.getBoundingClientRect();
    const x = ((event.clientX - rect.left) / rect.width) * CURVE_WIDTH;
    const y = ((event.clientY - rect.top) / rect.height) * CURVE_HEIGHT;
    music.setEqBandParam(this.draggingBand, {
      frequency: Math.round(xToFreq(x, CURVE_WIDTH)),
      gain: yToGain(y, CURVE_HEIGHT),
    });
  }

  onChartPointerUp(event: PointerEvent) {
    if (this.draggingBand === null) return;
    this.chartEl?.releasePointerCapture(event.pointerId);
    this.draggingBand = null;
  }
}
