<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import CoverArt from "$lib/components/ui/CoverArt.svelte";
  import VirtualList from "$lib/components/ui/VirtualList.svelte";
  import { CONTINUOUS_MODE_LABELS, music } from "$lib/config/music.svelte";
  import {
    trackCoverPaletteKey,
    trackCoverSeed,
  } from "$lib/music/cover-art-fallback";
  import {
    clampQueuePanelPosition,
    loadQueuePanelPosition,
    saveQueuePanelPosition,
    type QueuePanelPosition,
  } from "$lib/music/prefs";
  import { queuePanelListHeight } from "$lib/components/ui/virtual-list-window";
  import { coverArtUrl, formatDuration } from "$lib/subsonic";
  import CoverArtPlayOverlay from "./CoverArtPlayOverlay.svelte";
  import TrackContextMenu from "./TrackContextMenu.svelte";
  import { contextMenuPositionFromEvent } from "$lib/components/ui/context-menu";
  import { confirmDialog } from "$lib/ui/confirm.svelte";

  let dragIndex = $state(-1);
  let overIndex = $state(-1);
  let panelEl = $state<HTMLDivElement | null>(null);
  let panelPosition = $state<QueuePanelPosition | null>(
    loadQueuePanelPosition(),
  );
  let draggingPanel = $state(false);
  let dragOrigin = { pointerX: 0, pointerY: 0, startX: 0, startY: 0 };
  let viewportHeight = $state(
    typeof window !== "undefined" ? window.innerHeight : 900,
  );
  let queueMenu = $state<{ x: number; y: number; index: number } | null>(null);

  const queueListHeight = $derived(
    queuePanelListHeight(music.queue.length, viewportHeight),
  );
  const continuousLabel = $derived(
    music.continuousMode === "off"
      ? null
      : CONTINUOUS_MODE_LABELS[music.continuousMode],
  );

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

  function ensurePanelPosition() {
    if (!panelEl || panelPosition) return;
    const rect = panelEl.getBoundingClientRect();
    panelPosition = { x: rect.left, y: rect.top };
  }

  function clampPanelPosition(
    position: QueuePanelPosition,
  ): QueuePanelPosition {
    if (!panelEl) return position;
    return clampQueuePanelPosition(
      position,
      panelEl.offsetWidth,
      panelEl.offsetHeight,
    );
  }

  function onHeaderPointerDown(event: PointerEvent) {
    if (!panelEl) return;
    if (event.button !== 0) return;
    const target = event.target as HTMLElement | null;
    if (target?.closest(".jb-no-drag")) return;

    ensurePanelPosition();
    if (!panelPosition) return;

    draggingPanel = true;
    dragOrigin = {
      pointerX: event.clientX,
      pointerY: event.clientY,
      startX: panelPosition.x,
      startY: panelPosition.y,
    };
    (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
    event.preventDefault();
  }

  function onHeaderPointerMove(event: PointerEvent) {
    if (!draggingPanel || !panelPosition) return;
    panelPosition = clampPanelPosition({
      x: dragOrigin.startX + (event.clientX - dragOrigin.pointerX),
      y: dragOrigin.startY + (event.clientY - dragOrigin.pointerY),
    });
  }

  function onHeaderPointerUp(event: PointerEvent) {
    if (!draggingPanel) return;
    draggingPanel = false;
    if (panelPosition) {
      saveQueuePanelPosition(panelPosition);
    }
    try {
      (event.currentTarget as HTMLElement).releasePointerCapture(
        event.pointerId,
      );
    } catch {
      // Capture may already be released.
    }
  }

  $effect(() => {
    if (typeof window === "undefined") return;
    const onResize = () => {
      viewportHeight = window.innerHeight;
      if (!panelPosition || !panelEl) return;
      panelPosition = clampPanelPosition(panelPosition);
    };
    viewportHeight = window.innerHeight;
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  });
</script>

{#if music.queueOpen && music.queue.length > 0 && !music.onPlayRoute}
  <div
    bind:this={panelEl}
    class="queue-panel"
    class:queue-panel--positioned={panelPosition !== null}
    class:queue-panel--dragging={draggingPanel}
    style:left={panelPosition ? `${panelPosition.x}px` : undefined}
    style:top={panelPosition ? `${panelPosition.y}px` : undefined}
  >
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <header
      class="queue-panel__header queue-panel__header--draggable"
      aria-label="Move queue panel"
      onpointerdown={onHeaderPointerDown}
      onpointermove={onHeaderPointerMove}
      onpointerup={onHeaderPointerUp}
      onpointercancel={onHeaderPointerUp}
    >
      <div class="queue-panel__title queue-panel__drag-handle">
        <MdiIcon name="listMusic" size={16} />
        <span>Queue</span>
        <span class="queue-panel__count">{music.queue.length}</span>
        {#if continuousLabel}
          <span class="queue-panel__mode">{continuousLabel}</span>
        {/if}
      </div>
      <div class="queue-panel__actions jb-no-drag">
        {#if continuousLabel}
          <button
            type="button"
            class="queue-panel__clear"
            onclick={() => music.clearContinuousMode()}
          >
            Stop refill
          </button>
        {/if}
        <button
          type="button"
          class="queue-panel__clear"
          onclick={() => void clearQueue()}
        >
          Clear
        </button>
        <button
          type="button"
          class="queue-panel__close"
          aria-label="Close queue"
          onclick={() => (music.queueOpen = false)}
        >
          <MdiIcon name="x" size={16} />
        </button>
      </div>
    </header>

    <div class="queue-panel__body" style:height="{queueListHeight}px">
      <VirtualList
        items={music.queue}
        itemHeight={52}
        maxHeight={`${queueListHeight}px`}
        scrollMode="internal"
        class="queue-panel__list"
        itemKey={(track, index) => track.id || index}
      >
        {#snippet children({ item: track, index })}
          {@const image = coverArtUrl(
            music.config,
            track.coverArt ?? track.albumId ?? track.id,
            64,
          )}
          {@const isCurrent = index === music.queueIndex}
          <div
            class="queue-panel__item"
            role="button"
            tabindex="0"
            class:queue-panel__item--current={isCurrent}
            class:queue-panel__item--over={overIndex === index &&
              dragIndex !== index}
            draggable="true"
            ondragstart={() => onDragStart(index)}
            ondragover={(e) => onDragOver(e, index)}
            ondrop={() => onDrop(index)}
            ondragend={onDragEnd}
            onclick={() => music.playQueueIndex(index)}
            oncontextmenu={(event) => {
              const pos = contextMenuPositionFromEvent(event);
              if (!pos) return;
              queueMenu = { ...pos, index };
            }}
            onkeydown={(event) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                music.playQueueIndex(index);
              }
            }}
          >
            <div class="queue-panel__art">
              <CoverArt
                src={image}
                seed={trackCoverSeed(track)}
                paletteKey={trackCoverPaletteKey(track)}
              />
              <CoverArtPlayOverlay
                playing={isCurrent && music.playing}
                size={14}
              />
            </div>
            <div class="queue-panel__meta">
              <span class="queue-panel__track">{track.title}</span>
              <span class="queue-panel__artist"
                >{track.artist ?? "Unknown"}</span
              >
            </div>
            <span class="queue-panel__duration"
              >{formatDuration(track.duration)}</span
            >
            <button
              type="button"
              class="queue-panel__remove"
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
  </div>
{/if}

{#if queueMenu && music.queue[queueMenu.index]}
  <TrackContextMenu
    track={music.queue[queueMenu.index]}
    x={queueMenu.x}
    y={queueMenu.y}
    onRemove={() => {
      const index = queueMenu?.index;
      if (index !== undefined) music.removeFromQueue(index);
    }}
    onclose={() => (queueMenu = null)}
  />
{/if}

<style>
  .queue-panel {
    position: fixed;
    z-index: 65;
    width: min(
      26rem,
      calc(100vw - var(--jb-sidebar-current-width, 0px) - 2rem)
    );
    display: flex;
    flex-direction: column;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-xl);
    background: color-mix(in srgb, var(--jb-bg-elevated) 96%, transparent);
    backdrop-filter: blur(20px);
    box-shadow: var(--jb-shadow-lg);
    overflow: hidden;
  }

  .queue-panel:not(.queue-panel--positioned) {
    bottom: calc(
      var(--jb-bottom-chrome-height, 6rem) + env(safe-area-inset-bottom, 0px) +
        var(--jb-space-2)
    );
    left: calc(var(--jb-sidebar-current-width, 0px) + var(--jb-space-4));
  }

  .queue-panel__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--jb-space-3) var(--jb-space-4);
    border-bottom: 1px solid var(--jb-border);
    flex-shrink: 0;
  }

  .queue-panel__header--draggable {
    cursor: grab;
    touch-action: none;
    user-select: none;
  }

  .queue-panel__drag-handle {
    min-width: 0;
  }

  .queue-panel--dragging .queue-panel__header--draggable {
    cursor: grabbing;
  }

  .queue-panel__body {
    flex: none;
    height: auto;
    min-height: 13.75rem;
    overflow: hidden;
  }

  .queue-panel :global(.queue-panel__list) {
    height: 100%;
    min-height: inherit;
    padding: var(--jb-space-2);
  }

  .queue-panel__title {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    font-size: 0.875rem;
    font-weight: 700;
    color: var(--jb-text);
  }

  .queue-panel__count {
    font-size: 0.75rem;
    color: var(--jb-text-subtle);
  }

  .queue-panel__mode {
    margin-left: var(--jb-space-1);
    padding: 0.1rem 0.45rem;
    border-radius: var(--jb-radius-sm);
    background: color-mix(in srgb, var(--jb-accent) 18%, transparent);
    color: var(--jb-accent);
    font-size: 0.6875rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .queue-panel__actions {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
  }

  .queue-panel__clear {
    border: none;
    background: transparent;
    color: var(--jb-text-muted);
    font-size: 0.75rem;
    font-weight: 600;
    cursor: pointer;
    padding: 0.25rem 0.5rem;
    border-radius: var(--jb-radius-sm);
  }

  .queue-panel__clear:hover {
    color: var(--jb-text);
    background: var(--jb-accent-muted);
  }

  .queue-panel__close {
    display: grid;
    place-content: center;
    width: 1.75rem;
    height: 1.75rem;
    border: none;
    border-radius: var(--jb-radius-sm);
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
  }

  .queue-panel__item--over {
    box-shadow: inset 0 2px 0 0 var(--jb-accent);
  }

  .queue-panel__item {
    display: grid;
    grid-template-columns: auto 1fr auto auto;
    align-items: center;
    gap: var(--jb-space-2);
    padding: var(--jb-space-2);
    border-radius: var(--jb-radius-md);
    min-height: 3.25rem;
    cursor: pointer;
  }

  .queue-panel__item--current {
    background: var(--jb-accent-muted);
  }

  .queue-panel__remove {
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

  .queue-panel__art {
    position: relative;
    width: 2.25rem;
    height: 2.25rem;
    border-radius: var(--jb-radius-sm);
    overflow: hidden;
    flex-shrink: 0;
    pointer-events: none;
  }

  .queue-panel__item:hover .queue-panel__art :global(.cover-play-overlay) {
    opacity: 1;
  }

  .queue-panel__art :global(.cover-art) {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .queue-panel__meta {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }

  .queue-panel__track,
  .queue-panel__artist {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .queue-panel__track {
    font-size: 0.8125rem;
    font-weight: 600;
  }

  .queue-panel__artist {
    font-size: 0.75rem;
    color: var(--jb-text-muted);
  }

  .queue-panel__duration {
    font-size: 0.75rem;
    color: var(--jb-text-subtle);
    font-variant-numeric: tabular-nums;
  }

  @media (max-width: 768px) {
    .queue-panel:not(.queue-panel--positioned) {
      left: var(--jb-space-3);
      right: var(--jb-space-3);
      width: auto;
    }
  }
</style>
