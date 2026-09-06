<script lang="ts">
  import CoverArt from "$lib/components/ui/CoverArt.svelte";
  import Link from "$lib/router/Link.svelte";
  import { music, type PersonalMix } from "$lib/config/music.svelte";
  import { loadMixDisplay, type MixDisplayStyle } from "$lib/music/mix-display";
  import { mixTrackCount } from "$lib/music/mix-storage";
  import { coverArtUrl } from "$lib/subsonic";
  import CoverArtPlayOverlay from "./CoverArtPlayOverlay.svelte";
  import MixContextMenu from "./MixContextMenu.svelte";

  interface Props {
    mix: PersonalMix;
    displayStyle?: MixDisplayStyle;
    hideable?: boolean;
  }

  let {
    mix,
    displayStyle = loadMixDisplay(),
    hideable = false,
  }: Props = $props();

  const image = $derived(coverArtUrl(music.config, mix.coverArtId, 400));
  const isDisc = $derived(displayStyle === "discs");

  let menu = $state<{ x: number; y: number } | null>(null);

  function onContextMenu(event: MouseEvent) {
    event.preventDefault();
    event.stopPropagation();
    menu = { x: event.clientX, y: event.clientY };
  }

  function playMix(event: MouseEvent) {
    event.preventDefault();
    event.stopPropagation();
    music.playMix(mix);
  }
</script>

<Link
  href="/music/mix/{mix.id}"
  class={isDisc ? "mix-card mix-card--disc" : "mix-card"}
  style="--mix-gradient: {mix.gradient}"
  oncontextmenu={onContextMenu}
>
  <div class="mix-card__art">
    <CoverArt src={image} seed={mix.id} paletteKey={mix.title} />
    <button
      type="button"
      class="mix-card__play"
      aria-label="Play mix"
      onclick={playMix}
    >
      <CoverArtPlayOverlay size={isDisc ? 22 : 18} />
    </button>
  </div>
  <div class="mix-card__copy">
    <h3 class="mix-card__title">{mix.title}</h3>
    <p class="mix-card__subtitle">{mix.subtitle}</p>
    <p class="mix-card__count">{mixTrackCount(mix)} tracks</p>
  </div>
</Link>

{#if menu}
  <MixContextMenu
    {mix}
    x={menu.x}
    y={menu.y}
    {hideable}
    onclose={() => (menu = null)}
  />
{/if}

<style>
  :global(a.mix-card) {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-3);
    min-width: 0;
    text-decoration: none;
    color: inherit;
    border-radius: var(--jb-radius-lg);
    transition: transform var(--jb-transition);
  }

  :global(a.mix-card:hover) {
    transform: translateY(-2px);
  }

  :global(a.mix-card:hover) .mix-card__art {
    box-shadow: var(--jb-shadow-md);
  }

  :global(a.mix-card:hover) .mix-card__title {
    color: var(--jb-music-accent);
  }

  .mix-card__art {
    position: relative;
    aspect-ratio: 1;
    border-radius: var(--jb-radius-lg);
    overflow: hidden;
    background: var(--mix-gradient, var(--jb-bg-muted));
    transition: box-shadow var(--jb-transition);
  }

  .mix-card__art :global(.cover-art) {
    width: 100%;
    height: 100%;
  }

  :global(a.mix-card--disc) .mix-card__art {
    border-radius: 50%;
  }

  .mix-card__play {
    position: absolute;
    inset: 0;
    border: none;
    padding: 0;
    background: transparent;
    cursor: pointer;
  }

  .mix-card__art:hover :global(.cover-play-overlay),
  .mix-card__art:focus-within :global(.cover-play-overlay),
  :global(a.mix-card:hover) .mix-card__art :global(.cover-play-overlay) {
    opacity: 1;
  }

  .mix-card__copy {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    min-width: 0;
  }

  :global(a.mix-card--disc) .mix-card__copy {
    text-align: center;
  }

  .mix-card__title {
    margin: 0;
    font-size: 0.9375rem;
    font-weight: 650;
    color: var(--jb-text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    transition: color var(--jb-transition);
  }

  .mix-card__subtitle,
  .mix-card__count {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  @media (prefers-reduced-motion: reduce) {
    :global(a.mix-card:hover) {
      transform: none;
    }
  }
</style>
