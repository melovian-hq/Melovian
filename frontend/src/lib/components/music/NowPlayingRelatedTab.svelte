<script lang="ts">
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import TrackVirtualList from "$lib/components/music/TrackVirtualList.svelte";
  import { music } from "$lib/config/music.svelte";
  import type { SubsonicSong } from "$lib/subsonic";

  interface Props {
    relatedTracks: SubsonicSong[];
    relatedLoading: boolean;
  }

  let { relatedTracks, relatedLoading }: Props = $props();
</script>

{#if relatedLoading}
  <div class="now-playing-side__loading"><Spinner /></div>
{:else if relatedTracks.length === 0}
  <EmptyState
    title="No related tracks"
    message="Your server did not return similar songs for this track."
    icon="radio"
    embedded
  />
{:else}
  <div class="now-playing-related">
    <TrackVirtualList
      tracks={relatedTracks}
      onplay={(i) => music.playTracks(relatedTracks, i)}
      lazyThreshold={16}
      maxHeight="100%"
      scrollMode="internal"
    />
  </div>
{/if}

<style>
  .now-playing-side__loading {
    flex: 1;
    display: grid;
    place-content: center;
    min-height: 0;
  }

  .now-playing-related {
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }
</style>
