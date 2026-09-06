<script lang="ts">
  import CoverArt from "$lib/components/ui/CoverArt.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { music } from "$lib/config/music.svelte";
  import { coverArtUrl } from "$lib/subsonic";

  interface Props {
    coverArtIds?: string[];
    seed: string;
    variant?: "compact" | "grid" | "card";
    emptyIcon?: string;
  }

  let {
    coverArtIds = [],
    seed,
    variant = "compact",
    emptyIcon = "listMusic",
  }: Props = $props();

  const covers = $derived(coverArtIds.filter(Boolean).slice(0, 3));
  const count = $derived(covers.length);
  const mosaic = $derived(variant !== "compact");

  const tileSize = $derived(mosaic ? 160 : 50);
  const imageSize = $derived(mosaic ? 360 : 120);
  const emptyIconSize = $derived(mosaic ? 40 : 22);
</script>

<div
  class="playlist-cover-stack playlist-cover-stack--{variant}"
  class:playlist-cover-stack--multi={count > 1}
  class:playlist-cover-stack--mosaic={mosaic}
  class:playlist-cover-stack--duo={mosaic && count === 2}
  style:--pcs-count={count}
  aria-hidden="true"
>
  {#if !mosaic}
    <div class="playlist-cover-stack__glow"></div>
  {/if}

  {#if covers.length === 0}
    <span class="playlist-cover-stack__empty">
      <MdiIcon name={emptyIcon} size={emptyIconSize} />
    </span>
  {:else}
    {#each covers as coverId, index (coverId)}
      <span
        class="playlist-cover-stack__tile"
        style:--pcs-index={index}
        style:--pcs-tile="{tileSize}px"
        style:grid-row={mosaic && count === 3 && index === 0
          ? "1 / -1"
          : undefined}
      >
        <CoverArt
          src={coverArtUrl(music.config, coverId, imageSize)}
          {seed}
          paletteKey={coverId}
          alt=""
          width={tileSize}
          height={tileSize}
          loading="lazy"
        />
      </span>
    {/each}
  {/if}
</div>

<style>
  .playlist-cover-stack {
    --pcs-tile: 50px;
    position: relative;
    flex: 0 0 auto;
    display: grid;
    place-items: end start;
  }

  .playlist-cover-stack--compact {
    width: calc(var(--pcs-tile) + 1.1rem);
    height: calc(var(--pcs-tile) + 0.55rem);
  }

  .playlist-cover-stack--mosaic {
    width: 100%;
    height: 100%;
    min-height: 0;
    place-items: stretch;
    padding: 0;
  }

  .playlist-cover-stack__glow {
    position: absolute;
    inset: 12% 8% 4%;
    border-radius: 50%;
    background: radial-gradient(
      circle at 40% 60%,
      color-mix(in srgb, var(--jb-music-accent) 28%, transparent),
      transparent 68%
    );
    opacity: 0.55;
    pointer-events: none;
    filter: blur(10px);
  }

  .playlist-cover-stack--mosaic:not(.playlist-cover-stack--multi)
    .playlist-cover-stack__tile {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    border-radius: 0;
    border: none;
    box-shadow: none;
    transform: none;
  }

  .playlist-cover-stack--mosaic :global(.cover-art),
  .playlist-cover-stack--mosaic :global(img) {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .playlist-cover-stack--mosaic.playlist-cover-stack--multi {
    display: grid;
    grid-template-columns: 1fr 1fr;
    grid-template-rows: 1fr 1fr;
    gap: 2px;
    padding: 0;
    place-items: stretch;
    overflow: hidden;
  }

  .playlist-cover-stack--duo {
    grid-template-rows: 1fr;
  }

  .playlist-cover-stack--mosaic.playlist-cover-stack--multi
    .playlist-cover-stack__tile {
    position: relative;
    left: auto;
    bottom: auto;
    width: 100%;
    height: 100%;
    min-width: 0;
    min-height: 0;
    border-radius: 0;
    border: none;
    box-shadow: none;
    transform: none;
  }

  .playlist-cover-stack__empty {
    position: relative;
    z-index: 1;
    display: grid;
    place-items: center;
    width: var(--pcs-tile);
    height: var(--pcs-tile);
    border-radius: var(--jb-radius-md);
    background: linear-gradient(
      145deg,
      color-mix(in srgb, var(--jb-music-accent) 22%, var(--jb-bg-muted)),
      var(--jb-bg-muted)
    );
    color: var(--jb-music-accent);
    border: 1px solid
      color-mix(in srgb, var(--jb-music-accent) 24%, transparent);
    box-shadow: var(--jb-shadow-sm);
  }

  .playlist-cover-stack--mosaic .playlist-cover-stack__empty {
    width: 100%;
    height: 100%;
    border-radius: 0;
  }

  .playlist-cover-stack__tile {
    position: absolute;
    left: 0;
    bottom: 0;
    width: var(--pcs-tile);
    height: var(--pcs-tile);
    border-radius: calc(var(--jb-radius-md) + 1px);
    overflow: hidden;
    border: 2px solid color-mix(in srgb, var(--jb-bg) 88%, transparent);
    box-shadow:
      0 4px 14px color-mix(in srgb, var(--jb-bg) 22%, transparent),
      0 1px 0 color-mix(in srgb, var(--jb-text) 8%, transparent) inset;
    transform: translate(
        calc(var(--pcs-index) * 0.82rem),
        calc(var(--pcs-index) * -0.38rem)
      )
      rotate(calc(-10deg + var(--pcs-index) * 7deg))
      scale(calc(1 - var(--pcs-index) * 0.03));
    transform-origin: 50% 100%;
    z-index: calc(12 - var(--pcs-index));
    transition:
      transform 280ms cubic-bezier(0.22, 1, 0.36, 1),
      box-shadow 280ms ease;
  }

  .playlist-cover-stack:not(.playlist-cover-stack--mosaic):hover
    .playlist-cover-stack__tile {
    box-shadow:
      0 8px 22px color-mix(in srgb, var(--jb-bg) 28%, transparent),
      0 1px 0 color-mix(in srgb, var(--jb-text) 10%, transparent) inset;
  }

  :global(a:hover)
    .playlist-cover-stack:not(
      .playlist-cover-stack--mosaic
    ).playlist-cover-stack--multi
    .playlist-cover-stack__tile,
  :global(a:focus-visible)
    .playlist-cover-stack:not(
      .playlist-cover-stack--mosaic
    ).playlist-cover-stack--multi
    .playlist-cover-stack__tile {
    transform: translate(
        calc(var(--pcs-index) * 1.05rem + (var(--pcs-index) - 1) * 0.15rem),
        calc(var(--pcs-index) * -0.55rem - 2px)
      )
      rotate(calc(-12deg + var(--pcs-index) * 8deg))
      scale(calc(1.01 - var(--pcs-index) * 0.02));
  }

  .playlist-cover-stack__tile :global(.cover-art) {
    width: 100%;
    height: 100%;
  }
</style>
