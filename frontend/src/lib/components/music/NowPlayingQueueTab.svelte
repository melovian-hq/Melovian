<script lang="ts">
  import CoverArt from "$lib/components/ui/CoverArt.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import VirtualList from "$lib/components/ui/VirtualList.svelte";
  import CoverArtPlayOverlay from "$lib/components/music/CoverArtPlayOverlay.svelte";
  import { contextMenuPositionFromEvent } from "$lib/components/ui/context-menu";
  import { music } from "$lib/config/music.svelte";
  import {
    trackCoverPaletteKey,
    trackCoverSeed,
  } from "$lib/music/cover-art-fallback";
  import {
    coverArtUrl,
    formatDuration,
    isInternetRadioTrack,
  } from "$lib/subsonic";
  import type { SubsonicSong } from "$lib/subsonic";
  import { confirmDialog } from "$lib/ui/confirm.svelte";

  interface Props {
    ontrackmenu?: (menu: {
      x: number;
      y: number;
      track: SubsonicSong;
      queueIndex: number;
    }) => void;
  }

  let { ontrackmenu }: Props = $props();

  let dragIndex = $state(-1);
  let overIndex = $state(-1);

  function onDragStart(index: number) {
    dragIndex = index;
  }

  function onDragOver(event: DragEvent, index: number) {
    event.preventDefault();
    overIndex = index;
  }

  function onDrop(index: number) {
    if (dragIndex >= 0 && dragIndex !== index) {
      music.moveInQueue(dragIndex, index);
    }
    dragIndex = -1;
    overIndex = -1;
  }

  function onDragEnd() {
    dragIndex = -1;
    overIndex = -1;
  }

  async function clearQueue() {
    const ok = await confirmDialog.confirm({
      title: "Clear queue",
      message: `Remove all ${music.queue.length} tracks from the queue?`,
      confirmLabel: "Clear queue",
      danger: true,
    });
    if (!ok) return;
    music.clearQueue();
  }
</script>

{#if music.queue.length === 0}
  <EmptyState
    title="Queue is empty"
    message="Play an album, playlist, or track to build your queue."
    icon="listMusic"
    embedded
  />
{:else}
  <header class="now-playing-queue__header">
    <span>{music.queue.length} tracks</span>
    <button
      type="button"
      class="now-playing-queue__clear"
      onclick={() => void clearQueue()}
    >
      Clear
    </button>
  </header>
  <div class="now-playing-queue__list">
    <VirtualList
      items={music.queue}
      itemHeight={52}
      maxHeight="100%"
      scrollMode="internal"
    >
      {#snippet children({ item: queueTrack, index })}
        {@const queueImage = coverArtUrl(
          music.config,
          queueTrack.coverArt ?? queueTrack.albumId ?? queueTrack.id,
          64,
        )}
        {@const isCurrent = index === music.queueIndex}
        <div
          class="now-playing-queue__item"
          role="button"
          tabindex="0"
          class:now-playing-queue__item--current={isCurrent}
          class:now-playing-queue__item--over={overIndex === index &&
            dragIndex !== index}
          draggable="true"
          ondragstart={() => onDragStart(index)}
          ondragover={(e) => onDragOver(e, index)}
          ondrop={() => onDrop(index)}
          ondragend={onDragEnd}
          onclick={() => music.playQueueIndex(index)}
          onkeydown={(event) => {
            if (event.key === "Enter" || event.key === " ") {
              event.preventDefault();
              music.playQueueIndex(index);
            }
          }}
          oncontextmenu={(event) => {
            if (isInternetRadioTrack(queueTrack)) return;
            const pos = contextMenuPositionFromEvent(event);
            if (!pos) return;
            ontrackmenu?.({
              ...pos,
              track: queueTrack,
              queueIndex: index,
            });
          }}
        >
          <div class="now-playing-queue__art">
            <CoverArt
              src={queueImage}
              seed={trackCoverSeed(queueTrack)}
              paletteKey={trackCoverPaletteKey(queueTrack)}
            />
            <CoverArtPlayOverlay
              playing={isCurrent && music.playing}
              size={14}
            />
          </div>
          <div class="now-playing-queue__meta">
            <span class="now-playing-queue__track">{queueTrack.title}</span>
            <span class="now-playing-queue__artist"
              >{queueTrack.artist ?? "Unknown"}</span
            >
          </div>
          <span class="now-playing-queue__duration"
            >{formatDuration(queueTrack.duration)}</span
          >
          <button
            type="button"
            class="now-playing-queue__remove"
            aria-label="Remove from queue"
            onclick={(event) => {
              event.stopPropagation();
              music.removeFromQueue(index);
            }}
          >
            <MdiIcon name="x" size={14} />
          </button>
        </div>
      {/snippet}
    </VirtualList>
  </div>
{/if}

<style>
  .now-playing-queue__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--jb-space-3);
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .now-playing-queue__clear {
    border: none;
    background: transparent;
    color: var(--jb-text-muted);
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
    padding: 0.25rem 0.5rem;
    border-radius: var(--jb-radius-sm);
  }

  .now-playing-queue__clear:hover {
    color: var(--jb-text);
    background: var(--jb-accent-muted);
  }

  .now-playing-queue__list {
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }

  .now-playing-queue__item {
    display: grid;
    grid-template-columns: auto 1fr auto auto;
    align-items: center;
    gap: var(--jb-space-2);
    padding: var(--jb-space-2);
    border-radius: var(--jb-radius-md);
    min-height: 3.25rem;
    cursor: pointer;
  }

  .now-playing-queue__item--current {
    background: var(--jb-accent-muted);
  }

  .now-playing-queue__item--over {
    box-shadow: inset 0 2px 0 0 var(--jb-accent);
  }

  .now-playing-queue__remove {
    display: grid;
    place-content: center;
    width: 1.75rem;
    height: 1.75rem;
    border: none;
    border-radius: var(--jb-radius-full);
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
  }

  .now-playing-queue__art {
    position: relative;
    width: 2.25rem;
    height: 2.25rem;
    border-radius: var(--jb-radius-sm);
    overflow: hidden;
    flex-shrink: 0;
    pointer-events: none;
  }

  .now-playing-queue__item:hover
    .now-playing-queue__art
    :global(.cover-play-overlay) {
    opacity: 1;
  }

  .now-playing-queue__art :global(.cover-art) {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .now-playing-queue__meta {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }

  .now-playing-queue__track,
  .now-playing-queue__artist {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .now-playing-queue__track {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-text);
  }

  .now-playing-queue__artist {
    font-size: 0.75rem;
    color: var(--jb-text-muted);
  }

  .now-playing-queue__duration {
    font-size: 0.75rem;
    color: var(--jb-text-subtle);
    font-variant-numeric: tabular-nums;
  }
</style>
