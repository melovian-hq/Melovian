<script lang="ts">
  import MusicBreadcrumbs from "$lib/components/music/MusicBreadcrumbs.svelte";
  import LyricsDisplay from "$lib/components/music/LyricsDisplay.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Input from "$lib/components/ui/Input.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Section from "$lib/components/ui/Section.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import { music } from "$lib/config/music.svelte";
  import { isInternetRadioTrack } from "$lib/subsonic";
  import type { LyricsSearchHit } from "$lib/subsonic";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import Link from "$lib/router/Link.svelte";

  let searchQuery = $state("");
  let searchHits = $state<LyricsSearchHit[]>([]);
  let searchLoading = $state(false);
  let searchError = $state<string | null>(null);
  let lastSearchQuery = $state("");
  let searchDebounce: ReturnType<typeof setTimeout> | undefined;

  $effect(() => {
    if (!extensionFeatures.lyrics) return;
    if (!music.currentTrack || isInternetRadioTrack(music.currentTrack)) return;
    void music.loadCurrentLyrics();
  });

  async function runLyricsSearch(term: string) {
    const trimmed = term.trim();
    if (!trimmed) {
      lastSearchQuery = "";
      searchHits = [];
      searchError = null;
      searchLoading = false;
      return;
    }
    if (trimmed === lastSearchQuery) return;

    lastSearchQuery = trimmed;
    searchLoading = true;
    searchError = null;
    try {
      if (!music.connected) {
        const ok = await music.connect({ quiet: true });
        if (!ok) {
          searchError = "Connect to your music server to search lyrics.";
          searchHits = [];
          return;
        }
      }
      searchHits = await music.searchLyrics(trimmed);
    } catch {
      searchError = "Lyrics search failed. Try again in a moment.";
      searchHits = [];
    } finally {
      searchLoading = false;
    }
  }

  $effect(() => {
    const term = searchQuery.trim();
    clearTimeout(searchDebounce);
    const delay = term.length === 0 ? 0 : term.length < 3 ? 500 : 320;
    searchDebounce = setTimeout(() => void runLyricsSearch(searchQuery), delay);
    return () => clearTimeout(searchDebounce);
  });

  const showNowPlaying = $derived(
    music.currentTrack && !isInternetRadioTrack(music.currentTrack),
  );

  const showSearchSection = $derived(lastSearchQuery.length >= 3);
</script>

{#if !extensionFeatures.lyrics}
  <EmptyState
    title="Lyrics extension is off"
    message="Turn on Lyrics under Settings, Extensions to use lyrics search and synced display."
    icon="lyrics"
  >
    {#snippet actions()}
      <Link href="/settings/extensions">Open Extensions</Link>
    {/snippet}
  </EmptyState>
{:else}
  <div class="lyrics-page">
    <MusicBreadcrumbs items={[{ label: "Lyrics" }]} />

    <header class="lyrics-header">
      <h1>Lyrics</h1>
      <p class="lyrics-header__sub">
        Search by words you remember, or follow synced lyrics for the track that
        is playing.
      </p>
    </header>

    <div class="lyrics-search">
      <MdiIcon name="search" size={20} />
      <Input
        bind:value={searchQuery}
        placeholder="Search lyrics in your library"
      />
      {#if searchLoading}
        <Spinner />
      {/if}
    </div>

    {#if searchError}
      <p class="lyrics-page__message lyrics-page__message--error">
        {searchError}
      </p>
    {/if}

    {#if showSearchSection}
      <Section title="Lyrics matches">
        {#if searchLoading}
          <p class="lyrics-page__message" aria-live="polite">
            Searching for "{lastSearchQuery}"…
          </p>
        {:else if searchHits.length === 0}
          <EmptyState
            title="No lyric matches"
            message={`Nothing in the scanned library matched "${lastSearchQuery}". Try different words or a shorter phrase.`}
            icon="lyrics"
          />
        {:else}
          <div class="lyrics-search-results">
            {#each searchHits as hit (hit.songId)}
              <article class="lyrics-hit">
                <div class="lyrics-hit__meta">
                  <h2>{hit.title}</h2>
                  <p>
                    {hit.artist ?? "Unknown artist"}
                    {#if hit.album}
                      · {hit.album}
                    {/if}
                  </p>
                  <p class="lyrics-hit__snippet">{hit.snippet}</p>
                </div>
                <Button
                  variant="surface"
                  onclick={() =>
                    void music.playTrackById(hit.songId).catch(() => {})}
                >
                  Play
                </Button>
              </article>
            {/each}
          </div>
        {/if}
      </Section>
    {:else if searchQuery.trim().length > 0 && searchQuery.trim().length < 3}
      <p class="lyrics-page__message">Type at least 3 characters to search.</p>
    {/if}

    {#if showNowPlaying}
      <Section title="Now playing">
        <p class="lyrics-now-playing__track">
          {music.currentTrack!.title}
          {#if music.currentTrack!.artist}
            · {music.currentTrack!.artist}
          {/if}
        </p>

        {#if music.lyricsLoading}
          <div class="lyrics-page__loading"><Spinner /></div>
        {:else if !music.currentLyrics}
          <EmptyState
            title="No lyrics found"
            message="This track has no lyrics yet. Fetch from enabled providers or add lyrics on your server."
            icon="lyrics"
          />
          <button
            type="button"
            class="lyrics-fetch-btn"
            disabled={music.lyricsFetching}
            onclick={() => void music.fetchCurrentLyrics()}
          >
            {music.lyricsFetching ? "Fetching lyrics…" : "Fetch lyrics"}
          </button>
          {#if extensionFeatures.lyricsWhisper}
            <button
              type="button"
              class="lyrics-fetch-btn"
              disabled={music.lyricsFetching}
              onclick={() => void music.generateWhisperLyrics()}
            >
              {music.lyricsFetching
                ? "Transcribing…"
                : "Generate synced lyrics with Whisper"}
            </button>
          {/if}
        {:else}
          <article class="lyrics-body">
            {#if music.currentLyrics.artist || music.currentLyrics.title}
              <p class="lyrics-body__meta">
                {music.currentLyrics.artist ?? music.currentTrack!.artist}
                {#if music.currentLyrics.title}
                  · {music.currentLyrics.title}
                {/if}
                {#if music.currentLyrics.synced}
                  · Synced
                {/if}
              </p>
            {/if}
            <div class="lyrics-body__actions">
              <button
                type="button"
                class="lyrics-icon-btn"
                title="Export lyrics"
                aria-label="Export lyrics"
                onclick={() =>
                  import("$lib/music/lyrics-export").then(({ exportLyrics }) =>
                    exportLyrics(
                      music.currentLyrics!,
                      music.currentTrack?.title,
                    ),
                  )}
              >
                <MdiIcon name="export" size={18} />
              </button>
              <button
                type="button"
                class="lyrics-icon-btn"
                title="Refetch lyrics"
                aria-label="Refetch lyrics"
                disabled={music.lyricsFetching}
                onclick={() => void music.fetchCurrentLyrics()}
              >
                <MdiIcon name="refresh" size={18} />
              </button>
            </div>
            <LyricsDisplay
              lyrics={music.currentLyrics}
              currentTimeMs={music.currentTime * 1000}
              playing={music.playing}
              onSeek={(startMs) => music.seekToLyricLine(startMs)}
            />
          </article>
        {/if}
      </Section>
    {:else if !showSearchSection}
      <EmptyState
        title="Find songs by lyric"
        message="Search above with words you remember, or start playing a track to view synced lyrics."
        icon="lyrics"
      />
    {/if}
  </div>
{/if}

<style>
  .lyrics-page {
    max-width: var(--jb-content-narrow);
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-6);
  }

  .lyrics-header h1 {
    margin: 0 0 var(--jb-space-2);
    font-size: clamp(2rem, 4vw, 2.75rem);
    font-weight: 800;
  }

  .lyrics-header__sub {
    margin: 0;
    color: var(--jb-text-muted);
  }

  .lyrics-search {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    padding: 0 var(--jb-space-4);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-full);
    background: var(--jb-surface);
    color: var(--jb-text-muted);
  }

  .lyrics-search :global(.input) {
    border: none;
    background: transparent;
    padding-inline: 0;
  }

  .lyrics-search :global(.input:focus) {
    box-shadow: none;
  }

  .lyrics-page__message {
    margin: 0;
    color: var(--jb-text-muted);
    font-size: 0.875rem;
  }

  .lyrics-page__message--error {
    color: var(--jb-danger);
  }

  .lyrics-page__loading {
    display: grid;
    place-content: center;
    min-height: 8rem;
  }

  .lyrics-now-playing__track {
    margin: 0 0 var(--jb-space-4);
    color: var(--jb-text-muted);
  }

  .lyrics-body {
    padding: var(--jb-space-6);
    border-radius: var(--jb-radius-xl);
    background: var(--jb-surface);
    border: 1px solid var(--jb-border);
  }

  .lyrics-body__meta {
    margin: 0 0 var(--jb-space-4);
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--jb-text-muted);
  }

  .lyrics-body__actions {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    gap: var(--jb-space-2);
    margin-bottom: var(--jb-space-4);
  }

  .lyrics-icon-btn {
    display: inline-grid;
    place-items: center;
    width: 2.25rem;
    height: 2.25rem;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-full);
    background: var(--jb-surface);
    color: var(--jb-text);
    cursor: pointer;
  }

  .lyrics-icon-btn:hover {
    background: var(--jb-surface-hover);
  }

  .lyrics-fetch-btn {
    padding: 0.45rem 0.9rem;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-full);
    background: var(--jb-surface);
    color: var(--jb-text);
    font-weight: 600;
    cursor: pointer;
  }

  .lyrics-fetch-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .lyrics-search-results {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-3);
  }

  .lyrics-hit {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--jb-space-4);
    padding: var(--jb-space-4);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-lg);
    background: var(--jb-surface);
  }

  .lyrics-hit__meta h2 {
    margin: 0;
    font-size: 1rem;
    font-weight: 700;
  }

  .lyrics-hit__meta p {
    margin: 0.25rem 0 0;
    color: var(--jb-text-muted);
    font-size: 0.875rem;
  }

  .lyrics-hit__snippet {
    margin-top: var(--jb-space-3) !important;
    color: var(--jb-text) !important;
    line-height: 1.5;
  }

  @media (max-width: 768px) {
    .lyrics-page {
      max-width: none;
    }

    .lyrics-body {
      padding: var(--jb-space-4);
    }

    .lyrics-hit {
      flex-direction: column;
      align-items: stretch;
    }
  }
</style>
