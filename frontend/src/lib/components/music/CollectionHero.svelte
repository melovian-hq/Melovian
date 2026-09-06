<script lang="ts">
  import CoverArt from "$lib/components/ui/CoverArt.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";

  interface Props {
    typeLabel: string;
    title: string;
    meta: string;
    tone?: "favorites" | "history";
    icon?: string;
    coverSrc?: string | null;
    coverSeed?: string;
    playLabel?: string;
    playDisabled?: boolean;
    onplay?: () => void;
    onshuffle?: () => void;
    onplaynext?: () => void;
    onqueue?: () => void;
    onselect?: () => void;
  }

  let {
    typeLabel,
    title,
    meta,
    tone = "favorites",
    icon = "star",
    coverSrc = null,
    coverSeed = title,
    playLabel = "Play",
    playDisabled = false,
    onplay,
    onshuffle,
    onplaynext,
    onqueue,
    onselect,
  }: Props = $props();
</script>

<header class="collection-hero collection-hero--{tone}">
  <div class="collection-hero__backdrop">
    {#if coverSrc}
      <CoverArt
        src={coverSrc}
        seed={coverSeed}
        paletteKey={coverSeed}
        fetchpriority="high"
        loading="eager"
      />
    {/if}
    <div class="collection-hero__gradient"></div>
  </div>

  <div class="collection-hero__layout">
    <div class="collection-hero__cover" aria-hidden="true">
      <MdiIcon name={icon} size={72} fill="currentColor" />
    </div>

    <div class="collection-hero__info">
      <p class="collection-hero__type">{typeLabel}</p>
      <h1>{title}</h1>
      <p class="collection-hero__meta">{meta}</p>

      <div class="collection-hero__actions">
        <button
          type="button"
          class="collection-hero__play"
          onclick={() => onplay?.()}
          disabled={playDisabled}
          aria-label={playLabel}
        >
          <MdiIcon name="play" size={28} fill="currentColor" />
        </button>
        {#if onshuffle}
          <button
            type="button"
            class="collection-hero__secondary"
            onclick={() => onshuffle?.()}
            disabled={playDisabled}
          >
            <MdiIcon name="shuffle" size={16} />
            Shuffle
          </button>
        {/if}
        {#if onplaynext}
          <button
            type="button"
            class="collection-hero__secondary"
            onclick={() => onplaynext?.()}
            disabled={playDisabled}
          >
            <MdiIcon name="playNext" size={16} />
            Play next
          </button>
        {/if}
        {#if onqueue}
          <button
            type="button"
            class="collection-hero__secondary"
            onclick={() => onqueue?.()}
            disabled={playDisabled}
          >
            <MdiIcon name="queueAdd" size={16} />
            Add to queue
          </button>
        {/if}
        {#if onselect}
          <button
            type="button"
            class="collection-hero__secondary"
            onclick={() => onselect?.()}
            disabled={playDisabled}
          >
            <MdiIcon name="squareCheck" size={16} />
            Select
          </button>
        {/if}
      </div>
    </div>
  </div>
</header>

<style>
  .collection-hero {
    position: relative;
    overflow: hidden;
    min-height: 18rem;
    margin-inline: calc(-1 * var(--jb-page-pad-x, var(--jb-space-6)));
    --collection-hero-from: #4c1d95;
    --collection-hero-mid: #7c3aed;
    --collection-hero-to: #1e1b4b;
  }

  .collection-hero::after {
    content: "";
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    height: 5rem;
    background: linear-gradient(to bottom, transparent, var(--jb-bg));
    pointer-events: none;
  }

  .collection-hero--history {
    --collection-hero-from: #0c4a6e;
    --collection-hero-mid: #0284c7;
    --collection-hero-to: #082f49;
  }

  .collection-hero__backdrop {
    position: absolute;
    inset: 0;
    background: linear-gradient(
      135deg,
      var(--collection-hero-from) 0%,
      var(--collection-hero-mid) 46%,
      var(--collection-hero-to) 100%
    );
  }

  .collection-hero__backdrop :global(.cover-art) {
    width: 100%;
    height: 100%;
    object-fit: cover;
    filter: blur(48px) saturate(1.35) brightness(0.55);
    transform: scale(1.15);
    opacity: 0.7;
  }

  .collection-hero__gradient {
    position: absolute;
    inset: 0;
    background: linear-gradient(
      to bottom,
      rgb(0 0 0 / 0.15) 0%,
      rgb(0 0 0 / 0.45) 48%,
      rgb(0 0 0 / 0.88) 100%
    );
  }

  .collection-hero__layout {
    position: relative;
    display: flex;
    flex-wrap: wrap;
    align-items: flex-end;
    gap: var(--jb-space-8);
    padding: clamp(1.5rem, 4vw, 2.5rem) var(--jb-page-pad-x, var(--jb-space-6));
    min-height: 18rem;
    z-index: 1;
  }

  .collection-hero__cover {
    flex-shrink: 0;
    display: grid;
    place-content: center;
    width: clamp(10rem, 22vw, 14rem);
    height: clamp(10rem, 22vw, 14rem);
    border-radius: var(--jb-radius-lg);
    color: white;
    background: linear-gradient(
      145deg,
      var(--collection-hero-mid) 0%,
      var(--collection-hero-from) 55%,
      var(--collection-hero-to) 100%
    );
    box-shadow: 0 16px 40px rgb(0 0 0 / 0.4);
  }

  .collection-hero__info {
    flex: 1;
    min-width: min(100%, 16rem);
    color: white;
    padding-bottom: var(--jb-space-2);
  }

  .collection-hero__type {
    margin: 0 0 var(--jb-space-2);
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    opacity: 0.75;
  }

  .collection-hero h1 {
    margin: 0 0 var(--jb-space-3);
    font-size: clamp(2.25rem, 6vw, 4.5rem);
    font-weight: 900;
    line-height: 1.02;
    letter-spacing: -0.04em;
  }

  .collection-hero__meta {
    margin: 0 0 var(--jb-space-5);
    font-size: 0.9375rem;
    opacity: 0.9;
    line-height: 1.5;
  }

  .collection-hero__actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-3);
    align-items: center;
  }

  .collection-hero__play {
    display: grid;
    place-content: center;
    width: 3.5rem;
    height: 3.5rem;
    border: none;
    border-radius: var(--jb-radius-full);
    background: var(--jb-accent);
    color: var(--jb-accent-text);
    cursor: pointer;
    box-shadow: 0 8px 24px rgb(0 0 0 / 0.35);
    transition: transform var(--jb-transition);
  }

  .collection-hero__play:hover:not(:disabled) {
    transform: scale(1.06);
  }

  .collection-hero__play:disabled,
  .collection-hero__secondary:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .collection-hero__secondary {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    padding: 0.5rem 1rem;
    border: 1px solid rgb(255 255 255 / 0.25);
    border-radius: var(--jb-radius-full);
    background: rgb(255 255 255 / 0.08);
    color: white;
    font-weight: 600;
    font-size: 0.875rem;
    cursor: pointer;
    backdrop-filter: blur(8px);
  }

  .collection-hero__secondary:hover:not(:disabled) {
    background: rgb(255 255 255 / 0.15);
  }

  @media (max-width: 640px) {
    .collection-hero {
      min-height: auto;
    }

    .collection-hero__layout {
      flex-direction: column;
      align-items: center;
      text-align: center;
      min-height: auto;
      padding: var(--jb-space-4) var(--jb-page-pad-x, var(--jb-space-4));
    }

    .collection-hero h1 {
      font-size: clamp(2rem, 12vw, 3rem);
    }

    .collection-hero__actions {
      justify-content: center;
      align-items: center;
    }

    .collection-hero__secondary {
      padding: 0.375rem 0.75rem;
      font-size: 0.8125rem;
      white-space: nowrap;
    }
  }
</style>
