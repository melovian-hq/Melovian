<script lang="ts">
  import EnhancedCoverArt from "$lib/components/ui/EnhancedCoverArt.svelte";
  import AmbientCoverBackdrop from "$lib/components/ui/AmbientCoverBackdrop.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Link from "$lib/router/Link.svelte";
  import { music } from "$lib/config/music.svelte";
  import type { HomeShortcut } from "$lib/music/home-feed";
  import { genrePalette } from "$lib/music/genre-art";
  import { coverArtUrl } from "$lib/subsonic";

  interface Props {
    shortcuts: HomeShortcut[];
  }

  let { shortcuts }: Props = $props();

  function shortcutIcon(kind: HomeShortcut["kind"]): string {
    if (kind === "favorites") return "star";
    if (kind === "history") return "history";
    if (kind === "mix") return "listMusic";
    if (kind === "playlist") return "listMusic";
    if (kind === "artist") return "accountMusic";
    return "album";
  }

  function shortcutCover(item: HomeShortcut): string | null {
    if (!item.coverArtId) return null;
    return coverArtUrl(music.config, item.coverArtId, 200);
  }

  function shortcutEnhanceKind(
    kind: HomeShortcut["kind"],
  ): "artist" | "album" | "track" {
    if (kind === "artist") return "artist";
    return "album";
  }
</script>

{#if shortcuts.length > 0}
  <nav class="home-shortcuts" aria-label="Quick access">
    {#each shortcuts as item (item.id)}
      {@const palette = genrePalette(item.title)}
      {@const cover = shortcutCover(item)}
      <Link
        href={item.href}
        class="home-shortcut"
        style="--shortcut-bg: {palette[0]}; --shortcut-accent: {palette[1]}; --shortcut-mid: {palette[2]}"
      >
        {#if cover}
          <div class="home-shortcut__wash" aria-hidden="true">
            <AmbientCoverBackdrop
              src={cover}
              seed={item.seed}
              paletteKey={item.title}
              opacity={0.95}
              blur={28}
              saturate={1.7}
              scale={1.6}
            />
          </div>
        {/if}
        <span class="home-shortcut__scrim" aria-hidden="true"></span>
        <div class="home-shortcut__art">
          {#if cover}
            <EnhancedCoverArt
              kind={shortcutEnhanceKind(item.kind)}
              entity={{
                id: item.id,
                name: item.title,
                title: item.title,
                artist: item.subtitle,
                coverArt: item.coverArtId,
              }}
              src={cover}
              seed={item.seed}
              paletteKey={item.title}
            />
          {:else}
            <span class="home-shortcut__icon" aria-hidden="true">
              <MdiIcon name={shortcutIcon(item.kind)} size={22} />
            </span>
          {/if}
        </div>
        <span class="home-shortcut__copy">
          <span class="home-shortcut__title">{item.title}</span>
          {#if item.subtitle}
            <span class="home-shortcut__sub">{item.subtitle}</span>
          {/if}
        </span>
      </Link>
    {/each}
  </nav>
{/if}

<style>
  .home-shortcuts {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.625rem;
    width: 100%;
    min-width: 0;
  }

  :global(a.home-shortcut) {
    position: relative;
    isolation: isolate;
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    min-width: 0;
    min-height: 4.5rem;
    padding-right: var(--jb-space-3);
    overflow: hidden;
    border-radius: var(--jb-radius-lg);
    background: linear-gradient(
      105deg,
      var(--shortcut-mid) 0%,
      var(--shortcut-bg) 55%,
      color-mix(in srgb, var(--shortcut-accent) 55%, #0a0a0a) 100%
    );
    color: white;
    text-decoration: none;
    box-shadow:
      inset 0 1px 0 rgb(255 255 255 / 0.12),
      0 8px 22px rgb(0 0 0 / 0.22);
  }

  :global(a.home-shortcut:hover) {
    filter: brightness(1.08);
  }

  .home-shortcut__wash {
    position: absolute;
    inset: 0;
    z-index: 0;
    overflow: hidden;
    pointer-events: none;
  }

  .home-shortcut__scrim {
    position: absolute;
    inset: 0;
    z-index: 1;
    background: linear-gradient(
      90deg,
      rgb(0 0 0 / 0.28) 0%,
      rgb(0 0 0 / 0.42) 55%,
      rgb(0 0 0 / 0.55) 100%
    );
    pointer-events: none;
  }

  .home-shortcut__art {
    position: relative;
    z-index: 2;
    width: 4.5rem;
    height: 4.5rem;
    flex-shrink: 0;
    overflow: hidden;
    background: rgb(0 0 0 / 0.25);
    box-shadow: 4px 0 16px rgb(0 0 0 / 0.28);
  }

  .home-shortcut__art :global(.cover-art) {
    width: 100%;
    height: 100%;
  }

  .home-shortcut__icon {
    display: grid;
    place-content: center;
    width: 100%;
    height: 100%;
    color: white;
    background: color-mix(in srgb, var(--shortcut-accent) 70%, transparent);
  }

  .home-shortcut__copy {
    position: relative;
    z-index: 2;
    display: flex;
    flex-direction: column;
    min-width: 0;
    gap: 0.125rem;
  }

  .home-shortcut__title {
    font-size: 0.9375rem;
    font-weight: 700;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-shadow: 0 1px 8px rgb(0 0 0 / 0.45);
  }

  .home-shortcut__sub {
    font-size: 0.75rem;
    color: rgb(255 255 255 / 0.78);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  @media (min-width: 900px) {
    .home-shortcuts {
      grid-template-columns: repeat(4, minmax(0, 1fr));
    }
  }
</style>
