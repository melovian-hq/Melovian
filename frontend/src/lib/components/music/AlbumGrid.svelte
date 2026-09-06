<script lang="ts">
  import AlbumCard from "./AlbumCard.svelte";
  import VirtualList from "$lib/components/ui/VirtualList.svelte";
  import { chunk } from "$lib/core/dom/chunk";
  import type { SubsonicAlbum } from "$lib/subsonic";

  interface Props {
    albums: SubsonicAlbum[];
    size?: "sm" | "md" | "lg";
    lazyThreshold?: number;
    onNearEnd?: () => void;
  }

  let { albums, size = "md", lazyThreshold = 12, onNearEnd }: Props = $props();

  let containerEl = $state<HTMLElement | null>(null);
  let containerWidth = $state(
    typeof window !== "undefined" ? window.innerWidth : 360,
  );

  const minColumnWidth = $derived(
    size === "sm" ? 152 : size === "lg" ? 200 : 176,
  );
  const rowHeight = $derived(size === "sm" ? 200 : size === "lg" ? 280 : 240);
  const columns = $derived(
    Math.max(2, Math.floor(containerWidth / minColumnWidth)),
  );
  const rows = $derived(chunk(albums, columns));
  const useVirtual = $derived(albums.length >= lazyThreshold);

  function measure(node: HTMLElement) {
    containerEl = node;
    const width = node.clientWidth;
    if (width > 0) {
      containerWidth = width;
    }
    const ro = new ResizeObserver(() => {
      const nextWidth = node.clientWidth;
      if (nextWidth > 0) {
        containerWidth = nextWidth;
      }
    });
    ro.observe(node);
    return {
      destroy() {
        ro.disconnect();
        if (containerEl === node) containerEl = null;
      },
    };
  }
</script>

<div class="album-grid-wrap" use:measure>
  {#if useVirtual}
    <VirtualList
      items={rows}
      itemHeight={rowHeight}
      scrollMode="document"
      {onNearEnd}
    >
      {#snippet children({ item: row })}
        <div
          class="album-grid album-grid--virtual"
          style:grid-template-columns="repeat({columns}, minmax(0, 1fr))"
        >
          {#each row as album (album.id)}
            <AlbumCard {album} {size} />
          {/each}
        </div>
      {/snippet}
    </VirtualList>
  {:else}
    <div class="album-grid">
      {#each albums as album (album.id)}
        <AlbumCard {album} {size} />
      {/each}
    </div>
  {/if}
</div>

<style>
  .album-grid-wrap {
    width: 100%;
  }

  .album-grid {
    display: grid;
    grid-template-columns: repeat(
      auto-fill,
      minmax(min(100%, var(--jb-card-grid-min-sm)), 1fr)
    );
    gap: var(--jb-space-5);
  }

  .album-grid--virtual {
    margin-bottom: var(--jb-space-5);
  }

  @media (min-width: 768px) {
    .album-grid {
      grid-template-columns: repeat(
        auto-fill,
        minmax(var(--jb-card-grid-min), 1fr)
      );
    }
  }
</style>
