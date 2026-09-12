<!-- SPDX-FileCopyrightText: 2026 Quad4 Software -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<script lang="ts">
  import { music } from "$lib/config/music.svelte";
  import {
    combinedGainDbAt,
    computeCombinedCurve,
    curveAreaPath,
    curvePath,
    freqToX,
    gainToY,
  } from "$lib/music/eq-curve";
  import { formatFrequency, formatGain } from "$lib/music/eq";
  import { EqChartDrag } from "$lib/music/eq-drag.svelte";
  import {
    CURVE_HEIGHT,
    CURVE_WIDTH,
    FREQ_GRID,
    GAIN_GRID,
    GAIN_LABELS,
  } from "$lib/music/eq-layout";

  interface Props {
    selectedBand: number;
    onselect: (index: number) => void;
  }

  let { selectedBand, onselect }: Props = $props();

  const drag = new EqChartDrag((index) => onselect(index));

  const curvePoints = $derived(computeCombinedCurve(music.eq.bands));
  const curveLine = $derived(curvePath(curvePoints, CURVE_WIDTH, CURVE_HEIGHT));
  const curveFill = $derived(
    curveAreaPath(curvePoints, CURVE_WIDTH, CURVE_HEIGHT),
  );
  const zeroLineY = $derived(gainToY(0, CURVE_HEIGHT));
</script>

<div class="eq-panel__curve-wrap">
  <div class="eq-panel__chart-row">
    <div class="eq-panel__gain-axis" aria-hidden="true">
      {#each GAIN_LABELS as label (label)}
        <span>{label > 0 ? `+${label}` : label}</span>
      {/each}
    </div>

    <div
      class="eq-panel__chart"
      class:eq-panel__chart--disabled={!music.eq.enabled}
      role="group"
      aria-label="Equalizer band controls"
      bind:this={drag.chartEl}
      onpointermove={(event) => drag.onChartPointerMove(event)}
      onpointerup={(event) => drag.onChartPointerUp(event)}
      onpointercancel={(event) => drag.onChartPointerUp(event)}
    >
      <svg
        bind:this={drag.curveEl}
        class="eq-panel__curve"
        viewBox="0 0 {CURVE_WIDTH} {CURVE_HEIGHT}"
        preserveAspectRatio="none"
        role="img"
        aria-label="Equalizer frequency response"
      >
        <defs>
          <linearGradient id="eq-curve-fill" x1="0" y1="0" x2="0" y2="1">
            <stop
              offset="0%"
              stop-color="var(--jb-music-accent)"
              stop-opacity="0.28"
            />
            <stop
              offset="100%"
              stop-color="var(--jb-music-accent)"
              stop-opacity="0.04"
            />
          </linearGradient>
        </defs>

        {#each FREQ_GRID as freq (freq)}
          {@const x = freqToX(freq, CURVE_WIDTH)}
          <line
            x1={x}
            y1="0"
            x2={x}
            y2={CURVE_HEIGHT}
            class="eq-panel__grid-line eq-panel__grid-line--freq"
          />
        {/each}

        {#each GAIN_GRID as gridGain (gridGain)}
          {@const y = gainToY(gridGain, CURVE_HEIGHT)}
          <line
            x1="0"
            y1={y}
            x2={CURVE_WIDTH}
            y2={y}
            class="eq-panel__grid-line"
            class:eq-panel__grid-line--zero={gridGain === 0}
          />
        {/each}

        <path d={curveFill} class="eq-panel__curve-fill" />
        <line
          x1="0"
          y1={zeroLineY}
          x2={CURVE_WIDTH}
          y2={zeroLineY}
          class="eq-panel__zero-line"
        />
        <path d={curveLine} class="eq-panel__curve-line" />
      </svg>

      <div class="eq-panel__handles">
        {#each music.eq.bands as band, i (i)}
          {@const x =
            (freqToX(band.frequency, CURVE_WIDTH) / CURVE_WIDTH) * 100}
          {@const y =
            (gainToY(
              combinedGainDbAt(music.eq.bands, band.frequency),
              CURVE_HEIGHT,
            ) /
              CURVE_HEIGHT) *
            100}
          {@const isSelected = selectedBand === i}
          <button
            type="button"
            class="eq-panel__handle"
            class:eq-panel__handle--selected={isSelected}
            class:eq-panel__handle--disabled={!music.eq.enabled}
            style:left="{x}%"
            style:top="{y}%"
            aria-label="Band {i + 1}, {formatFrequency(
              band.frequency,
            )}, {formatGain(band.gain)} dB"
            aria-pressed={isSelected}
            disabled={!music.eq.enabled}
            onpointerdown={(e) => drag.onBandPointerDown(i, e)}
            onclick={() => onselect(i)}
          >
            {#if isSelected}
              <span class="eq-panel__handle-ring" aria-hidden="true"></span>
            {/if}
            <span class="eq-panel__handle-dot" aria-hidden="true"></span>
          </button>
        {/each}
      </div>
    </div>
  </div>

  <div class="eq-panel__curve-labels">
    <div class="eq-panel__freq-labels">
      <span>20 Hz</span>
      <span>100</span>
      <span>1 kHz</span>
      <span>10 kHz</span>
      <span>20 kHz</span>
    </div>
  </div>
</div>

<style>
  .eq-panel__curve-wrap {
    margin-bottom: var(--jb-space-3);
  }

  .eq-panel__chart-row {
    display: grid;
    grid-template-columns: 2rem 1fr;
    gap: var(--jb-space-2);
    align-items: stretch;
  }

  .eq-panel__gain-axis {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    padding-block: 0.125rem;
    font-size: 0.5625rem;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    color: var(--jb-text-subtle);
    text-align: right;
    line-height: 1;
  }

  .eq-panel__chart {
    position: relative;
    touch-action: none;
    width: 100%;
    aspect-ratio: 640 / 140;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-bg-subtle);
    overflow: hidden;
  }

  .eq-panel__chart--disabled {
    opacity: 0.5;
  }

  .eq-panel__curve {
    display: block;
    width: 100%;
    height: 100%;
  }

  .eq-panel__handles {
    position: absolute;
    inset: 0;
    pointer-events: none;
  }

  .eq-panel__grid-line {
    stroke: var(--jb-border);
    stroke-width: 1;
    vector-effect: non-scaling-stroke;
  }

  .eq-panel__grid-line--freq {
    stroke: color-mix(in srgb, var(--jb-border) 70%, transparent);
    stroke-dasharray: 3 5;
  }

  .eq-panel__grid-line--zero {
    stroke: transparent;
  }

  .eq-panel__zero-line {
    stroke: var(--jb-border-strong);
    stroke-width: 1;
    stroke-dasharray: 5 4;
    vector-effect: non-scaling-stroke;
  }

  .eq-panel__curve-fill {
    fill: url(#eq-curve-fill);
    pointer-events: none;
  }

  .eq-panel__curve-line {
    fill: none;
    stroke: var(--jb-music-accent);
    stroke-width: 2;
    vector-effect: non-scaling-stroke;
    pointer-events: none;
  }

  .eq-panel__handle {
    position: absolute;
    display: grid;
    place-items: center;
    width: 1.125rem;
    height: 1.125rem;
    padding: 0;
    border: none;
    background: transparent;
    transform: translate(-50%, -50%);
    pointer-events: auto;
    cursor: grab;
  }

  .eq-panel__handle-dot {
    display: block;
    width: 0.5625rem;
    height: 0.5625rem;
    border-radius: 50%;
    background: var(--jb-bg-elevated);
    border: 2px solid var(--jb-music-accent);
    box-sizing: border-box;
  }

  .eq-panel__handle-ring {
    position: absolute;
    width: 1.125rem;
    height: 1.125rem;
    border-radius: 50%;
    border: 2px solid
      color-mix(in srgb, var(--jb-music-accent) 35%, transparent);
    box-sizing: border-box;
    pointer-events: none;
  }

  .eq-panel__handle--selected .eq-panel__handle-dot {
    width: 0.6875rem;
    height: 0.6875rem;
    background: var(--jb-music-accent);
    border-color: var(--jb-bg-elevated);
  }

  .eq-panel__handle--disabled {
    cursor: not-allowed;
    opacity: 0.55;
    pointer-events: none;
  }

  .eq-panel__handle:focus-visible {
    outline: none;
  }

  .eq-panel__handle:focus-visible .eq-panel__handle-dot {
    box-shadow: 0 0 0 2px
      color-mix(in srgb, var(--jb-music-accent) 45%, transparent);
  }

  .eq-panel__curve-labels {
    display: grid;
    grid-template-columns: 2rem 1fr;
    gap: var(--jb-space-2);
    margin-top: var(--jb-space-1);
  }

  .eq-panel__freq-labels {
    grid-column: 2;
    display: flex;
    justify-content: space-between;
    font-size: 0.625rem;
    font-weight: 600;
    color: var(--jb-text-subtle);
    font-variant-numeric: tabular-nums;
  }
</style>
