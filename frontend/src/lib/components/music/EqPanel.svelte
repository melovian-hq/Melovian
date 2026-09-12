<script lang="ts">
  import { Slider } from "bits-ui";
  import { music } from "$lib/config/music.svelte";
  import {
    combinedGainDbAt,
    computeCombinedCurve,
    curveAreaPath,
    curvePath,
    freqToX,
    gainToY,
    xToFreq,
    yToGain,
  } from "$lib/music/eq-curve";
  import {
    DEFAULT_BAND_FREQUENCIES,
    EQ_FREQ_MAX,
    EQ_FREQ_MIN,
    EQ_GAIN_MAX,
    EQ_GAIN_MIN,
    EQ_GAIN_STEP,
    EQ_PRESETS,
    EQ_Q_MAX,
    EQ_Q_MIN,
    EQ_Q_STEP,
    formatFrequency,
    formatGain,
    formatQ,
    freqToNorm,
    normToFreq,
  } from "$lib/music/eq";

  const CURVE_WIDTH = 640;
  const CURVE_HEIGHT = 140;
  const FREQ_GRID = [100, 1000, 10000] as const;
  const GAIN_GRID = [-24, -12, 0, 12, 24] as const;
  const GAIN_LABELS = [24, 0, -24] as const;

  let selectedBand = $state(0);
  let draggingBand = $state<number | null>(null);
  let curveEl = $state<SVGSVGElement | null>(null);
  let chartEl = $state<HTMLDivElement | null>(null);

  const curvePoints = $derived(computeCombinedCurve(music.eq.bands));
  const curveLine = $derived(curvePath(curvePoints, CURVE_WIDTH, CURVE_HEIGHT));
  const curveFill = $derived(
    curveAreaPath(curvePoints, CURVE_WIDTH, CURVE_HEIGHT),
  );
  const zeroLineY = $derived(gainToY(0, CURVE_HEIGHT));
  const selected = $derived(music.eq.bands[selectedBand]);

  function selectBand(index: number) {
    selectedBand = index;
  }

  function onBandGainInput(index: number, gain: number) {
    music.setEqBandParam(index, { gain });
  }

  function onBandPointerDown(index: number, event: PointerEvent) {
    if (!music.eq.enabled) return;
    event.preventDefault();
    event.stopPropagation();
    selectBand(index);
    draggingBand = index;
    chartEl?.setPointerCapture(event.pointerId);
  }

  function onChartPointerMove(event: PointerEvent) {
    if (draggingBand === null || !music.eq.enabled || !curveEl) return;
    const rect = curveEl.getBoundingClientRect();
    const x = ((event.clientX - rect.left) / rect.width) * CURVE_WIDTH;
    const y = ((event.clientY - rect.top) / rect.height) * CURVE_HEIGHT;
    music.setEqBandParam(draggingBand, {
      frequency: Math.round(xToFreq(x, CURVE_WIDTH)),
      gain: yToGain(y, CURVE_HEIGHT),
    });
  }

  function onChartPointerUp(event: PointerEvent) {
    if (draggingBand === null) return;
    chartEl?.releasePointerCapture(event.pointerId);
    draggingBand = null;
  }

  function resetSelectedBand() {
    music.setEqBandParam(selectedBand, {
      frequency: DEFAULT_BAND_FREQUENCIES[selectedBand],
      gain: 0,
      q: 1,
    });
  }
</script>

<div class="eq-panel">
  <div class="eq-panel__header">
    <span>Parametric EQ</span>
    <button
      type="button"
      class="eq-toggle"
      class:eq-toggle--on={music.eq.enabled}
      onclick={() => music.toggleEq()}
      aria-label={music.eq.enabled ? "Disable equalizer" : "Enable equalizer"}
      aria-pressed={music.eq.enabled}
    >
      <span class="eq-toggle__track">
        <span class="eq-toggle__thumb"></span>
      </span>
      <span class="eq-toggle__label">{music.eq.enabled ? "On" : "Off"}</span>
    </button>
  </div>

  <div class="eq-panel__presets">
    {#each EQ_PRESETS as preset (preset.id)}
      <button
        type="button"
        class="eq-panel__preset"
        class:eq-panel__preset--active={music.eq.presetId === preset.id}
        onclick={() => music.setEqPreset(preset.id)}
      >
        {preset.name}
      </button>
    {/each}
    {#if music.eq.presetId === "custom"}
      <span class="eq-panel__preset eq-panel__preset--active">Custom</span>
    {/if}
  </div>

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
        bind:this={chartEl}
        onpointermove={onChartPointerMove}
        onpointerup={onChartPointerUp}
        onpointercancel={onChartPointerUp}
      >
        <svg
          bind:this={curveEl}
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
              onpointerdown={(e) => onBandPointerDown(i, e)}
              onclick={() => selectBand(i)}
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

  <div class="eq-panel__band-tabs">
    {#each music.eq.bands as _, i (i)}
      <button
        type="button"
        class="eq-panel__band-tab"
        class:eq-panel__band-tab--active={selectedBand === i}
        onclick={() => selectBand(i)}
      >
        {i + 1}
      </button>
    {/each}
  </div>

  <div
    class="eq-panel__params"
    class:eq-panel__params--disabled={!music.eq.enabled}
  >
    <div class="eq-panel__param">
      <label class="eq-panel__param-label" for="eq-freq-{selectedBand}">
        Frequency
      </label>
      <input
        id="eq-freq-{selectedBand}"
        type="range"
        min="0"
        max="1000"
        step="1"
        value={freqToNorm(selected.frequency)}
        disabled={!music.eq.enabled}
        oninput={(e) =>
          music.setEqBandParam(selectedBand, {
            frequency: normToFreq(
              Number((e.currentTarget as HTMLInputElement).value),
            ),
          })}
      />
      <input
        type="number"
        class="eq-panel__param-input"
        min={EQ_FREQ_MIN}
        max={EQ_FREQ_MAX}
        step="1"
        value={selected.frequency}
        disabled={!music.eq.enabled}
        onchange={(e) =>
          music.setEqBandParam(selectedBand, {
            frequency: Number((e.currentTarget as HTMLInputElement).value),
          })}
      />
      <span class="eq-panel__param-unit"
        >{formatFrequency(selected.frequency)}</span
      >
    </div>

    <div class="eq-panel__param">
      <label class="eq-panel__param-label" for="eq-gain-{selectedBand}">
        Gain
      </label>
      <input
        id="eq-gain-{selectedBand}"
        type="range"
        min={EQ_GAIN_MIN}
        max={EQ_GAIN_MAX}
        step={EQ_GAIN_STEP}
        value={selected.gain}
        disabled={!music.eq.enabled}
        oninput={(e) =>
          music.setEqBandParam(selectedBand, {
            gain: Number((e.currentTarget as HTMLInputElement).value),
          })}
      />
      <input
        type="number"
        class="eq-panel__param-input"
        min={EQ_GAIN_MIN}
        max={EQ_GAIN_MAX}
        step={EQ_GAIN_STEP}
        value={selected.gain}
        disabled={!music.eq.enabled}
        onchange={(e) =>
          music.setEqBandParam(selectedBand, {
            gain: Number((e.currentTarget as HTMLInputElement).value),
          })}
      />
      <span class="eq-panel__param-unit">{formatGain(selected.gain)} dB</span>
    </div>

    <div class="eq-panel__param">
      <label class="eq-panel__param-label" for="eq-q-{selectedBand}">Q</label>
      <input
        id="eq-q-{selectedBand}"
        type="range"
        min={EQ_Q_MIN}
        max={EQ_Q_MAX}
        step={EQ_Q_STEP}
        value={selected.q}
        disabled={!music.eq.enabled}
        oninput={(e) =>
          music.setEqBandParam(selectedBand, {
            q: Number((e.currentTarget as HTMLInputElement).value),
          })}
      />
      <input
        type="number"
        class="eq-panel__param-input"
        min={EQ_Q_MIN}
        max={EQ_Q_MAX}
        step={EQ_Q_STEP}
        value={selected.q}
        disabled={!music.eq.enabled}
        onchange={(e) =>
          music.setEqBandParam(selectedBand, {
            q: Number((e.currentTarget as HTMLInputElement).value),
          })}
      />
      <span class="eq-panel__param-unit">{formatQ(selected.q)}</span>
    </div>

    <button
      type="button"
      class="eq-panel__reset-band"
      disabled={!music.eq.enabled}
      onclick={resetSelectedBand}
    >
      Reset band
    </button>
  </div>

  <div class="eq-panel__faders">
    {#each music.eq.bands as band, i (i)}
      <div
        class="eq-panel__fader"
        class:eq-panel__fader--selected={selectedBand === i}
        role="group"
        onfocusin={() => {
          if (music.eq.enabled) selectBand(i);
        }}
        onpointerdown={() => {
          if (music.eq.enabled) selectBand(i);
        }}
      >
        <Slider.Root
          type="single"
          orientation="vertical"
          min={EQ_GAIN_MIN}
          max={EQ_GAIN_MAX}
          step={EQ_GAIN_STEP}
          value={band.gain}
          disabled={!music.eq.enabled}
          onValueChange={(value) => onBandGainInput(i, value)}
          class="eq-panel__fader-slider"
        >
          <span class="eq-panel__fader-track">
            <Slider.Range class="eq-panel__fader-range" />
          </span>
          <Slider.Thumb
            index={0}
            class="eq-panel__fader-thumb"
            aria-label="Band {i + 1} gain"
          />
        </Slider.Root>
        <span class="eq-panel__fader-label"
          >{formatFrequency(band.frequency)}</span
        >
        <span class="eq-panel__fader-value">{formatGain(band.gain)}</span>
      </div>
    {/each}
  </div>
</div>

<style>
  .eq-panel {
    padding: var(--jb-space-4) var(--jb-space-6) var(--jb-space-5);
    border-top: 1px solid var(--jb-border);
    background: color-mix(in srgb, var(--jb-bg) 60%, transparent);
  }

  .eq-panel__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--jb-space-3);
    font-size: 0.8125rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--jb-text-muted);
  }

  .eq-toggle {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    border: none;
    background: transparent;
    color: var(--jb-text);
    cursor: pointer;
    padding: 0;
    font: inherit;
  }

  .eq-toggle__track {
    position: relative;
    width: 2.5rem;
    height: 1.375rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-bg-muted);
    border: 1px solid var(--jb-border);
    transition:
      background var(--jb-transition),
      border-color var(--jb-transition);
  }

  .eq-toggle--on .eq-toggle__track {
    background: var(--jb-music-accent);
    border-color: var(--jb-music-accent);
  }

  .eq-toggle__thumb {
    position: absolute;
    top: 2px;
    left: 2px;
    width: 1rem;
    height: 1rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-bg-elevated);
    box-shadow: var(--jb-shadow-sm);
    transition: transform var(--jb-transition);
  }

  .eq-toggle--on .eq-toggle__thumb {
    transform: translateX(1.125rem);
  }

  .eq-toggle__label {
    font-size: 0.8125rem;
    font-weight: 600;
    min-width: 1.75rem;
    text-transform: none;
    letter-spacing: normal;
    color: var(--jb-text);
  }

  .eq-panel__presets {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
    margin-bottom: var(--jb-space-3);
  }

  .eq-panel__preset {
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
    color: var(--jb-text-muted);
    border-radius: var(--jb-radius-full);
    padding: 0.35rem 0.75rem;
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
    transition:
      background var(--jb-transition),
      border-color var(--jb-transition),
      color var(--jb-transition);
  }

  .eq-panel__preset:hover {
    background: var(--jb-surface-hover);
    color: var(--jb-text);
  }

  .eq-panel__preset--active {
    border-color: var(--jb-music-accent);
    color: var(--jb-music-accent);
    background: var(--jb-music-accent-muted);
  }

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

  .eq-panel__band-tabs {
    display: flex;
    gap: var(--jb-space-1);
    margin-bottom: var(--jb-space-3);
  }

  .eq-panel__band-tab {
    flex: 1;
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
    color: var(--jb-text-muted);
    border-radius: var(--jb-radius-sm);
    padding: 0.25rem 0;
    font-size: 0.6875rem;
    font-weight: 700;
    cursor: pointer;
    transition:
      background var(--jb-transition),
      border-color var(--jb-transition),
      color var(--jb-transition);
  }

  .eq-panel__band-tab:hover {
    background: var(--jb-surface-hover);
    color: var(--jb-text);
  }

  .eq-panel__band-tab--active {
    border-color: var(--jb-music-accent);
    color: var(--jb-music-accent);
    background: var(--jb-music-accent-muted);
  }

  .eq-panel__params {
    display: grid;
    grid-template-columns: 1fr 1fr 1fr auto;
    gap: var(--jb-space-3);
    align-items: end;
    margin-bottom: var(--jb-space-4);
    padding: var(--jb-space-3);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-surface);
  }

  .eq-panel__params--disabled {
    opacity: 0.55;
  }

  .eq-panel__param {
    display: grid;
    gap: var(--jb-space-1);
    min-width: 0;
  }

  .eq-panel__param-label {
    font-size: 0.6875rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--jb-text-subtle);
  }

  .eq-panel__param input[type="range"] {
    width: 100%;
    accent-color: var(--jb-music-accent);
  }

  .eq-panel__param-input {
    width: 100%;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-sm);
    background: var(--jb-bg-subtle);
    color: var(--jb-text);
    font-size: 0.75rem;
    font-variant-numeric: tabular-nums;
    padding: 0.2rem 0.35rem;
  }

  .eq-panel__param-input:disabled {
    color: var(--jb-text-muted);
  }

  .eq-panel__param-unit {
    font-size: 0.6875rem;
    color: var(--jb-text-muted);
    font-variant-numeric: tabular-nums;
  }

  .eq-panel__reset-band {
    border: 1px solid var(--jb-border);
    background: var(--jb-bg-subtle);
    color: var(--jb-text-muted);
    border-radius: var(--jb-radius-sm);
    padding: 0.45rem 0.65rem;
    font-size: 0.6875rem;
    font-weight: 600;
    cursor: pointer;
    white-space: nowrap;
    transition:
      background var(--jb-transition),
      color var(--jb-transition),
      border-color var(--jb-transition);
  }

  .eq-panel__reset-band:hover:not(:disabled) {
    background: var(--jb-surface-hover);
    color: var(--jb-text);
    border-color: var(--jb-border-strong);
  }

  .eq-panel__reset-band:disabled {
    cursor: not-allowed;
  }

  .eq-panel__faders {
    display: grid;
    grid-template-columns: repeat(10, 1fr);
    gap: var(--jb-space-2);
  }

  .eq-panel__fader {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--jb-space-1);
    font-size: 0.6875rem;
    color: var(--jb-text-subtle);
  }

  .eq-panel__fader--selected {
    color: var(--jb-music-accent);
  }

  :global(.eq-panel__fader-slider) {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    height: 5rem;
    width: 1.25rem;
  }

  .eq-panel__fader-track {
    position: relative;
    height: 100%;
    width: 0.25rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-bg-muted);
    overflow: hidden;
  }

  :global(.eq-panel__fader-range) {
    width: 100%;
    background: var(--jb-music-accent);
  }

  :global(.eq-panel__fader-thumb) {
    width: 0.875rem;
    height: 0.875rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-music-accent);
    box-shadow: var(--jb-shadow-sm);
    cursor: grab;
  }

  :global(.eq-panel__fader-thumb:active) {
    cursor: grabbing;
  }

  :global(.eq-panel__fader-thumb:focus-visible) {
    outline: 2px solid var(--jb-music-accent);
    outline-offset: 2px;
  }

  :global(.eq-panel__fader-slider[data-disabled]) {
    opacity: 0.55;
  }

  :global(.eq-panel__fader-slider[data-disabled] .eq-panel__fader-thumb) {
    cursor: not-allowed;
  }

  .eq-panel__fader-label,
  .eq-panel__fader-value {
    font-variant-numeric: tabular-nums;
    font-size: 0.625rem;
    text-align: center;
    line-height: 1.2;
  }

  @media (max-width: 768px) {
    .eq-panel__params {
      grid-template-columns: 1fr;
    }

    .eq-panel__faders {
      grid-template-columns: repeat(5, 1fr);
    }
  }
</style>
