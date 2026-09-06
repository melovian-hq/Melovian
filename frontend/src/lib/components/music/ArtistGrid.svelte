<script lang="ts">
  import ArtistCard from "./ArtistCard.svelte";
  import VirtualList from "$lib/components/ui/VirtualList.svelte";
  import { chunk } from "$lib/core/dom/chunk";
  import type { SubsonicArtist } from "$lib/subsonic";

  interface Props {
    artists: SubsonicArtist[];
    size?: "sm" | "md";
    lazyThreshold?: number;
  }

  let { artists, size = "md", lazyThreshold = 24 }: Props = $props();

  let containerWidth = $state(800);

  const minColumnWidth = $derived(size === "sm" ? 128 : 152);
  const rowHeight = $derived(size === "sm" ? 196 : 220);
  const rowGap = 20;
  const virtualRowHeight = $derived(rowHeight + rowGap);
  const columns = $derived(
    Math.max(2, Math.floor(containerWidth / minColumnWidth)),
  );
  const rows = $derived(chunk(artists, columns));
  const useVirtual = $derived(artists.length >= lazyThreshold);

  function measure(node: HTMLElement) {
    containerWidth = node.clientWidth;
    let resizeTimer: ReturnType<typeof setTimeout> | undefined;

    const ro = new ResizeObserver(() => {
      clearTimeout(resizeTimer);
      resizeTimer = setTimeout(() => {
        containerWidth = node.clientWidth;
      }, 100);
    });
    ro.observe(node);

    return {
      destroy() {
        clearTimeout(resizeTimer);
        ro.disconnect();
      },
    };
  }
</script>

<div class="artist-grid-wrap" use:measure>
  {#if useVirtual}
    <VirtualList
      items={rows}
      itemHeight={virtualRowHeight}
      overscan={2}
      scrollMode="document"
    >
      {#snippet children({ item: row })}
        <div
          class="artist-grid artist-grid--virtual"
          style:grid-template-columns="repeat({columns}, minmax(0, 1fr))"
        >
          {#each row as artist (artist.id)}
            <ArtistCard {artist} {size} />
          {/each}
        </div>
      {/snippet}
    </VirtualList>
  {:else}
    <div class="artist-grid">
      {#each artists as artist (artist.id)}
        <ArtistCard {artist} {size} />
      {/each}
    </div>
  {/if}
</div>

<style>
  .artist-grid-wrap {
    width: 100%;
  }

  .artist-grid {
    display: grid;
    grid-template-columns: repeat(
      auto-fill,
      minmax(min(100%, var(--jb-artist-grid-min)), 1fr)
    );
    gap: var(--jb-space-5);
  }

  .artist-grid--virtual {
    margin-bottom: 1.25rem;
  }

  @media (min-width: 768px) {
    .artist-grid {
      grid-template-columns: repeat(
        auto-fill,
        minmax(var(--jb-artist-grid-min-md), 1fr)
      );
    }
  }
</style>
