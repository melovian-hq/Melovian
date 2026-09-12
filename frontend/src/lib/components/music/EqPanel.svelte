<script lang="ts">
  import { music } from "$lib/config/music.svelte";
  import { EQ_PRESETS } from "$lib/music/eq";
  import EqCurveChart from "./EqCurveChart.svelte";
  import EqBandParams from "./EqBandParams.svelte";
  import EqGainFaders from "./EqGainFaders.svelte";

  let selectedBand = $state(0);
  const selected = $derived(music.eq.bands[selectedBand]);

  function selectBand(index: number) {
    selectedBand = index;
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

  <EqCurveChart {selectedBand} onselect={selectBand} />

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

  <EqBandParams bandIndex={selectedBand} band={selected} />

  <EqGainFaders {selectedBand} onselect={selectBand} />
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
</style>
