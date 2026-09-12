<!-- SPDX-FileCopyrightText: 2026 Quad4 Software -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<script lang="ts">
  import { music } from "$lib/config/music.svelte";
  import {
    DEFAULT_BAND_FREQUENCIES,
    EQ_FREQ_MAX,
    EQ_FREQ_MIN,
    EQ_GAIN_MAX,
    EQ_GAIN_MIN,
    EQ_GAIN_STEP,
    EQ_Q_MAX,
    EQ_Q_MIN,
    EQ_Q_STEP,
    formatFrequency,
    formatGain,
    formatQ,
    freqToNorm,
    normToFreq,
    type EqBandParam,
  } from "$lib/music/eq";

  interface Props {
    bandIndex: number;
    band: EqBandParam;
  }

  let { bandIndex, band }: Props = $props();

  function resetBand() {
    music.setEqBandParam(bandIndex, {
      frequency: DEFAULT_BAND_FREQUENCIES[bandIndex],
      gain: 0,
      q: 1,
    });
  }
</script>

<div
  class="eq-panel__params"
  class:eq-panel__params--disabled={!music.eq.enabled}
>
  <div class="eq-panel__param">
    <label class="eq-panel__param-label" for="eq-freq-{bandIndex}">
      Frequency
    </label>
    <input
      id="eq-freq-{bandIndex}"
      type="range"
      min="0"
      max="1000"
      step="1"
      value={freqToNorm(band.frequency)}
      disabled={!music.eq.enabled}
      oninput={(e) =>
        music.setEqBandParam(bandIndex, {
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
      value={band.frequency}
      disabled={!music.eq.enabled}
      onchange={(e) =>
        music.setEqBandParam(bandIndex, {
          frequency: Number((e.currentTarget as HTMLInputElement).value),
        })}
    />
    <span class="eq-panel__param-unit">{formatFrequency(band.frequency)}</span>
  </div>

  <div class="eq-panel__param">
    <label class="eq-panel__param-label" for="eq-gain-{bandIndex}">
      Gain
    </label>
    <input
      id="eq-gain-{bandIndex}"
      type="range"
      min={EQ_GAIN_MIN}
      max={EQ_GAIN_MAX}
      step={EQ_GAIN_STEP}
      value={band.gain}
      disabled={!music.eq.enabled}
      oninput={(e) =>
        music.setEqBandParam(bandIndex, {
          gain: Number((e.currentTarget as HTMLInputElement).value),
        })}
    />
    <input
      type="number"
      class="eq-panel__param-input"
      min={EQ_GAIN_MIN}
      max={EQ_GAIN_MAX}
      step={EQ_GAIN_STEP}
      value={band.gain}
      disabled={!music.eq.enabled}
      onchange={(e) =>
        music.setEqBandParam(bandIndex, {
          gain: Number((e.currentTarget as HTMLInputElement).value),
        })}
    />
    <span class="eq-panel__param-unit">{formatGain(band.gain)} dB</span>
  </div>

  <div class="eq-panel__param">
    <label class="eq-panel__param-label" for="eq-q-{bandIndex}">Q</label>
    <input
      id="eq-q-{bandIndex}"
      type="range"
      min={EQ_Q_MIN}
      max={EQ_Q_MAX}
      step={EQ_Q_STEP}
      value={band.q}
      disabled={!music.eq.enabled}
      oninput={(e) =>
        music.setEqBandParam(bandIndex, {
          q: Number((e.currentTarget as HTMLInputElement).value),
        })}
    />
    <input
      type="number"
      class="eq-panel__param-input"
      min={EQ_Q_MIN}
      max={EQ_Q_MAX}
      step={EQ_Q_STEP}
      value={band.q}
      disabled={!music.eq.enabled}
      onchange={(e) =>
        music.setEqBandParam(bandIndex, {
          q: Number((e.currentTarget as HTMLInputElement).value),
        })}
    />
    <span class="eq-panel__param-unit">{formatQ(band.q)}</span>
  </div>

  <button
    type="button"
    class="eq-panel__reset-band"
    disabled={!music.eq.enabled}
    onclick={resetBand}
  >
    Reset band
  </button>
</div>

<style>
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

  @media (max-width: 768px) {
    .eq-panel__params {
      grid-template-columns: 1fr;
    }
  }
</style>
