<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { music } from "$lib/config/music.svelte";
  import { trackSelection } from "$lib/music/selection.svelte";
  import PlaylistPicker from "./PlaylistPicker.svelte";

  interface Props {
    allTracks?: import("$lib/subsonic").SubsonicSong[];
  }

  let { allTracks = [] }: Props = $props();

  let pickerOpen = $state(false);

  function playSelected(shuffle = false) {
    const tracks = trackSelection.tracks;
    if (tracks.length === 0) return;
    if (shuffle) music.shuffle = true;
    music.playTracks(tracks, 0);
    trackSelection.clear();
  }

  function selectAll() {
    if (allTracks.length > 0) trackSelection.selectAll(allTracks);
  }

  function addSelectedToQueue() {
    const tracks = trackSelection.tracks;
    if (tracks.length === 0) return;
    music.addTracksToQueue(tracks);
    trackSelection.clear();
  }

  function playSelectedNext() {
    const tracks = trackSelection.tracks;
    if (tracks.length === 0) return;
    music.playTracksNext(tracks);
    trackSelection.clear();
  }
</script>

{#if trackSelection.active && trackSelection.count > 0}
  <div class="selection-bar">
    <span class="selection-bar__count">{trackSelection.count} selected</span>
    <div class="selection-bar__actions">
      {#if allTracks.length > 0 && trackSelection.count < allTracks.length}
        <button type="button" onclick={selectAll}>Select all</button>
      {/if}
      <button type="button" onclick={() => playSelected(false)}>
        <MdiIcon name="play" size={16} /> Play
      </button>
      <button type="button" onclick={() => playSelected(true)}>
        <MdiIcon name="shuffle" size={16} /> Shuffle
      </button>
      <button type="button" onclick={playSelectedNext}>
        <MdiIcon name="playNext" size={16} /> Play next
      </button>
      <button type="button" onclick={addSelectedToQueue}>
        <MdiIcon name="queueAdd" size={16} /> Add to queue
      </button>
      <button type="button" onclick={() => (pickerOpen = true)}>
        <MdiIcon name="listMusic" size={16} /> Add to playlist
      </button>
      <button
        type="button"
        class="selection-bar__clear"
        onclick={() => trackSelection.clear()}
      >
        <MdiIcon name="x" size={16} />
      </button>
    </div>
  </div>

  <PlaylistPicker
    open={pickerOpen}
    tracks={trackSelection.tracks}
    onclose={() => {
      pickerOpen = false;
      trackSelection.clear();
    }}
  />
{/if}

<style>
  .selection-bar {
    position: sticky;
    top: 0;
    z-index: 10;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-4);
    padding: var(--jb-space-3) var(--jb-space-4);
    margin-bottom: var(--jb-space-3);
    border-radius: var(--jb-radius-lg);
    background: linear-gradient(
      135deg,
      var(--jb-music-accent-muted),
      var(--jb-surface)
    );
    border: 1px solid
      color-mix(in srgb, var(--jb-music-accent) 35%, var(--jb-border));
    box-shadow: var(--jb-shadow-sm);
  }

  .selection-bar__count {
    font-weight: 700;
    font-size: 0.875rem;
    color: var(--jb-music-accent);
  }

  .selection-bar__actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--jb-space-2);
  }

  .selection-bar__actions button {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-1);
    padding: 0.4rem 0.75rem;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-full);
    background: var(--jb-surface);
    color: var(--jb-text);
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
  }

  .selection-bar__actions button:hover {
    border-color: var(--jb-music-accent);
    color: var(--jb-music-accent);
  }

  .selection-bar__clear {
    border: none !important;
    background: transparent !important;
    color: var(--jb-text-muted) !important;
  }
</style>
