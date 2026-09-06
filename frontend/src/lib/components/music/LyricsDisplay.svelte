<script lang="ts">
  import { activeLineIndex, type ParsedLyrics } from "$lib/music/lyrics";

  interface Props {
    lyrics: ParsedLyrics;
    currentTimeMs?: number;
    playing?: boolean;
    compact?: boolean;
    autoScroll?: boolean;
    onSeek?: (startMs: number) => void;
    class?: string;
  }

  let {
    lyrics,
    currentTimeMs = 0,
    playing = false,
    compact = false,
    autoScroll = true,
    onSeek,
    class: className = "",
  }: Props = $props();

  let lineRefs = $state<(HTMLElement | null)[]>([]);

  const lines = $derived(Array.isArray(lyrics.lines) ? lyrics.lines : []);

  const activeIndex = $derived(
    lines.length > 0
      ? activeLineIndex(
          lines,
          currentTimeMs,
          lyrics.offsetMs ?? 0,
          lyrics.synced,
        )
      : -1,
  );

  $effect(() => {
    if (!autoScroll || activeIndex < 0 || !playing) return;
    const node = lineRefs[activeIndex];
    node?.scrollIntoView?.({
      block: "center",
      behavior: "smooth",
      inline: "nearest",
    });
  });

  function handleLineClick(startMs?: number) {
    if (startMs === undefined || !onSeek) return;
    onSeek(startMs);
  }
</script>

<div
  class="lyrics-display {className}"
  class:lyrics-display--compact={compact}
  class:lyrics-display--synced={lyrics.synced}
>
  {#each lines as line, index (index)}
    {#if line.startMs !== undefined && onSeek}
      <button
        type="button"
        bind:this={lineRefs[index]}
        class="lyrics-display__line lyrics-display__line--seekable"
        class:lyrics-display__line--active={index === activeIndex}
        onclick={() => handleLineClick(line.startMs)}
      >
        {line.text}
      </button>
    {:else}
      <p
        bind:this={lineRefs[index]}
        class="lyrics-display__line"
        class:lyrics-display__line--active={index === activeIndex}
      >
        {line.text}
      </p>
    {/if}
  {/each}
</div>

<style>
  .lyrics-display {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    line-height: 1.7;
    font-size: 1rem;
    white-space: pre-wrap;
  }

  .lyrics-display--compact {
    gap: 0.35rem;
    font-size: 0.9375rem;
    line-height: 1.55;
  }

  .lyrics-display__line {
    margin: 0;
    color: var(--jb-text-muted);
    transition:
      color var(--jb-transition),
      font-weight var(--jb-transition);
  }

  .lyrics-display__line--seekable {
    border: none;
    background: transparent;
    padding: 0;
    text-align: left;
    width: 100%;
    cursor: pointer;
    font: inherit;
  }

  .lyrics-display__line--seekable:hover {
    color: var(--jb-text);
  }

  .lyrics-display--synced .lyrics-display__line--active {
    color: var(--jb-text);
    font-weight: 700;
    transform: scale(1.02);
    transform-origin: left center;
  }

  .lyrics-display--synced
    .lyrics-display__line:not(.lyrics-display__line--active) {
    opacity: 0.45;
  }
</style>
