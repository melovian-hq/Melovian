<script lang="ts">
  import TrackRow from "$lib/components/music/TrackRow.svelte";
  import VirtualList from "$lib/components/ui/VirtualList.svelte";
  import { listItemKey } from "$lib/core/collection";
  import type { SubsonicSong } from "$lib/subsonic";

  interface Props {
    tracks: SubsonicSong[];
    onplay: (index: number) => void;
    onplayversion?: (track: SubsonicSong) => void;
    versionsFor?: (index: number) => SubsonicSong[] | undefined;
    selectable?: boolean;
    showIndex?: boolean;
    itemHeight?: number;
    maxHeight?: string;
    lazyThreshold?: number;
    scrollMode?: "internal" | "document";
    onNearEnd?: () => void;
    class?: string;
  }

  let {
    tracks,
    onplay,
    onplayversion,
    versionsFor,
    selectable = false,
    showIndex = true,
    itemHeight = 62,
    maxHeight = "none",
    lazyThreshold = 24,
    scrollMode = "document",
    onNearEnd,
    class: className = "",
  }: Props = $props();

  const useVirtual = $derived(tracks.length >= lazyThreshold);
  const effectiveScrollMode = $derived(
    maxHeight !== "none" ? "internal" : scrollMode,
  );
</script>

{#if useVirtual}
  <VirtualList
    items={tracks}
    {itemHeight}
    {maxHeight}
    scrollMode={effectiveScrollMode}
    {onNearEnd}
    itemKey={(track, index) =>
      listItemKey(track.id, index, track.title ?? track.album)}
    class={className}
  >
    {#snippet children({ item, index })}
      <TrackRow
        track={item}
        {index}
        {showIndex}
        {selectable}
        versions={versionsFor?.(index)}
        onplay={() => onplay(index)}
        {onplayversion}
      />
    {/snippet}
  </VirtualList>
{:else}
  <div class="lazy-track-list {className}">
    {#each tracks as track, index (listItemKey(track.id, index, track.title ?? track.album))}
      <TrackRow
        {track}
        {index}
        {showIndex}
        {selectable}
        versions={versionsFor?.(index)}
        onplay={() => onplay(index)}
        {onplayversion}
      />
    {/each}
  </div>
{/if}

<style>
  .lazy-track-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
    content-visibility: auto;
  }
</style>
