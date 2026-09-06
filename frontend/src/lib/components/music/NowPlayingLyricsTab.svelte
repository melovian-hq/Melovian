<script lang="ts">
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import LyricsDisplay from "$lib/components/music/LyricsDisplay.svelte";
  import { music } from "$lib/config/music.svelte";
  import { filterLyricsLines } from "$lib/music/lyrics";
  import { extensionFeatures } from "$lib/extensions/features.svelte";

  let lyricsAutoScroll = $state(true);
  let lyricsSearchQuery = $state("");

  const displayLyrics = $derived(
    music.currentLyrics
      ? filterLyricsLines(music.currentLyrics, lyricsSearchQuery)
      : null,
  );

  const lyricsFollowScroll = $derived(
    lyricsAutoScroll && music.playing && Boolean(music.currentLyrics?.synced),
  );

  function exportCurrentLyrics() {
    if (!music.currentLyrics) return;
    void import("$lib/music/lyrics-export").then(({ exportLyrics }) =>
      exportLyrics(music.currentLyrics!, music.currentTrack?.title),
    );
  }
</script>

{#if music.lyricsLoading}
  <div class="now-playing-side__loading"><Spinner /></div>
{:else if !music.currentLyrics}
  <EmptyState
    title="No lyrics found"
    message="This track has no lyrics yet. Fetch from enabled providers or add lyrics on your server."
    icon="lyrics"
    embedded
  />
  <button
    type="button"
    class="now-playing-lyrics__fetch"
    disabled={music.lyricsFetching}
    onclick={() => void music.fetchCurrentLyrics()}
  >
    {music.lyricsFetching ? "Fetching lyrics…" : "Fetch lyrics"}
  </button>
  {#if extensionFeatures.lyricsWhisper}
    <button
      type="button"
      class="now-playing-lyrics__fetch"
      disabled={music.lyricsFetching}
      onclick={() => void music.generateWhisperLyrics()}
    >
      {music.lyricsFetching
        ? "Transcribing…"
        : "Generate synced lyrics with Whisper"}
    </button>
  {/if}
{:else}
  <header class="now-playing-lyrics-panel__header">
    <div class="now-playing-lyrics-panel__actions">
      {#if music.currentLyrics.synced}
        <span class="now-playing-lyrics__hint">Synced</span>
        <button
          type="button"
          class="now-playing-lyrics-panel__icon-btn"
          class:now-playing-lyrics-panel__icon-btn--active={lyricsAutoScroll}
          title={lyricsAutoScroll
            ? "Disable auto-scroll"
            : "Enable auto-scroll"}
          aria-label={lyricsAutoScroll
            ? "Disable auto-scroll"
            : "Enable auto-scroll"}
          aria-pressed={lyricsAutoScroll}
          onclick={() => (lyricsAutoScroll = !lyricsAutoScroll)}
        >
          <MdiIcon
            name={lyricsAutoScroll ? "scrollFollowActive" : "scrollFollow"}
            size={16}
          />
        </button>
      {/if}
      <button
        type="button"
        class="now-playing-lyrics-panel__icon-btn"
        title="Refetch lyrics"
        aria-label="Refetch lyrics"
        disabled={music.lyricsFetching}
        onclick={() => void music.fetchCurrentLyrics()}
      >
        <MdiIcon name="refresh" size={16} />
      </button>
      <button
        type="button"
        class="now-playing-lyrics-panel__icon-btn"
        title="Export lyrics"
        aria-label="Export lyrics"
        onclick={exportCurrentLyrics}
      >
        <MdiIcon name="export" size={16} />
      </button>
    </div>
  </header>
  <label class="now-playing-lyrics__search">
    <MdiIcon name="search" size={15} />
    <input
      type="search"
      placeholder="Search lyrics"
      bind:value={lyricsSearchQuery}
      autocomplete="off"
    />
  </label>
  <div class="now-playing-lyrics now-playing-lyrics--tab">
    <div class="now-playing-lyrics__scroll">
      {#if displayLyrics && displayLyrics.lines.length === 0}
        <p class="now-playing-lyrics__no-matches">No matching lines.</p>
      {:else if displayLyrics}
        <LyricsDisplay
          lyrics={displayLyrics}
          currentTimeMs={music.currentTime * 1000}
          playing={music.playing}
          autoScroll={lyricsFollowScroll}
          onSeek={(startMs) => music.seekToLyricLine(startMs)}
        />
      {/if}
    </div>
  </div>
{/if}

<style>
  .now-playing-side__loading {
    flex: 1;
    display: grid;
    place-content: center;
    min-height: 0;
  }

  .now-playing-lyrics-panel__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-3);
    margin-bottom: var(--jb-space-3);
    padding-bottom: var(--jb-space-3);
    border-bottom: 1px solid var(--jb-border);
  }

  .now-playing-lyrics-panel__actions {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
  }

  .now-playing-lyrics-panel__icon-btn {
    display: inline-grid;
    place-items: center;
    width: 1.75rem;
    height: 1.75rem;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-sm);
    background: var(--jb-surface);
    color: var(--jb-text-muted);
    cursor: pointer;
  }

  .now-playing-lyrics-panel__icon-btn:hover:not(:disabled) {
    color: var(--jb-text);
    background: var(--jb-surface-hover);
  }

  .now-playing-lyrics-panel__icon-btn:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }

  .now-playing-lyrics-panel__icon-btn--active {
    color: var(--jb-accent);
    border-color: color-mix(in srgb, var(--jb-accent) 45%, var(--jb-border));
    background: color-mix(in srgb, var(--jb-accent) 12%, var(--jb-surface));
  }

  .now-playing-lyrics {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .now-playing-lyrics--tab {
    flex: 1;
    min-height: 0;
  }

  .now-playing-lyrics__scroll {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding-right: var(--jb-space-1);
  }

  .now-playing-lyrics__search {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    flex-shrink: 0;
    margin-bottom: var(--jb-space-3);
    padding-bottom: var(--jb-space-2);
    border: none;
    border-bottom: 1px solid var(--jb-border);
    color: var(--jb-text-subtle);
  }

  .now-playing-lyrics__search input {
    flex: 1;
    min-width: 0;
    border: none;
    outline: none;
    background: transparent;
    color: var(--jb-text);
    font: inherit;
    font-size: 0.8125rem;
  }

  .now-playing-lyrics__search input::-webkit-search-cancel-button {
    display: none;
  }

  .now-playing-lyrics__no-matches {
    margin: 0;
    color: var(--jb-text-muted);
    text-align: center;
  }

  .now-playing-lyrics__hint {
    margin: 0 0 var(--jb-space-3);
    font-size: 0.75rem;
    color: var(--jb-text-subtle);
    text-transform: uppercase;
    letter-spacing: 0.06em;
    font-weight: 700;
  }

  .now-playing-lyrics__fetch {
    align-self: center;
    margin-top: var(--jb-space-3);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-full);
    background: var(--jb-surface);
    color: var(--jb-text);
    font-size: 0.8125rem;
    font-weight: 600;
    padding: 0.5rem 1rem;
    cursor: pointer;
  }

  .now-playing-lyrics__fetch:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
</style>
