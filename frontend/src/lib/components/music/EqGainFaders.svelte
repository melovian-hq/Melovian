<!-- SPDX-FileCopyrightText: 2026 Quad4 Software -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<script lang="ts">
  import { Slider } from "bits-ui";
  import { music } from "$lib/config/music.svelte";
  import {
    EQ_GAIN_MAX,
    EQ_GAIN_MIN,
    EQ_GAIN_STEP,
    formatFrequency,
    formatGain,
  } from "$lib/music/eq";

  interface Props {
    selectedBand: number;
    onselect: (index: number) => void;
  }

  let { selectedBand, onselect }: Props = $props();

  function onBandGainInput(index: number, gain: number) {
    music.setEqBandParam(index, { gain });
  }
</script>

<div class="eq-panel__faders">
  {#each music.eq.bands as band, i (i)}
    <div
      class="eq-panel__fader"
      class:eq-panel__fader--selected={selectedBand === i}
      role="group"
      onfocusin={() => {
        if (music.eq.enabled) onselect(i);
      }}
      onpointerdown={() => {
        if (music.eq.enabled) onselect(i);
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

<style>
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
    .eq-panel__faders {
      grid-template-columns: repeat(5, 1fr);
    }
  }
</style>
