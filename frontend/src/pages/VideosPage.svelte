<script lang="ts">
  import AppShell from "$lib/components/layout/AppShell.svelte";
  import PageHeader from "$lib/components/ui/PageHeader.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import LocalSearchBox from "$lib/components/ui/LocalSearchBox.svelte";
  import Link from "$lib/router/Link.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { localLibraries } from "$lib/features/local-libraries/store.svelte";
  import { mediaTrackDownloadUrl } from "$lib/music/api";
  import { filterByLocalSearch } from "$lib/utils/local-search";
  import { formatDuration } from "$lib/subsonic";
  import { listLocalVideos } from "$lib/video/api";
  import { videoFeature } from "$lib/video/feature.svelte";
  import { videoPlayerPath, type LocalVideo } from "$lib/video/ids";

  let loading = $state(true);
  let error = $state<string | null>(null);
  let videos = $state<LocalVideo[]>([]);
  let searchQuery = $state("");
  let enabled = $state(false);

  const hasLibrary = $derived(Boolean(localLibraries.active?.id));
  const filtered = $derived(
    filterByLocalSearch(videos, searchQuery, (video) => [
      video.title,
      video.artist,
      video.album,
      video.relPath,
    ]),
  );

  $effect(() => {
    let cancelled = false;
    loading = true;
    error = null;
    void videoFeature
      .refresh()
      .then((settings) => {
        if (cancelled) return;
        enabled = settings.enabled;
        if (!settings.enabled) {
          videos = [];
          return;
        }
        if (!hasLibrary) {
          videos = [];
          return;
        }
        return listLocalVideos().then((items) => {
          if (!cancelled) videos = items;
        });
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          error = err instanceof Error ? err.message : String(err);
          videos = [];
        }
      })
      .finally(() => {
        if (!cancelled) loading = false;
      });
    return () => {
      cancelled = true;
    };
  });
</script>

<AppShell compactTop>
  <div class="videos-page">
    <PageHeader
      title="Videos"
      subtitle="Local music videos from your library folders."
    />

    {#if loading}
      <div class="videos-page__loading">
        <Spinner />
      </div>
    {:else if !enabled}
      <EmptyState
        title="Videos are off"
        message="Turn on videos in Settings → Video to browse local files and search music videos."
        icon="video"
      />
    {:else if !hasLibrary}
      <EmptyState
        title="No local library"
        message="Add a local music folder in Settings, Sources, then scan. Videos with .mp4, .m4v, or .webm extensions show up here."
        icon="folderMusic"
      />
    {:else if error}
      <EmptyState title="Could not load videos" message={error} icon="alert" />
    {:else}
      <LocalSearchBox bind:value={searchQuery} placeholder="Filter videos" />

      {#if filtered.length === 0}
        <EmptyState
          title={searchQuery.trim() ? "No matching videos" : "No videos yet"}
          message={searchQuery.trim()
            ? "Try a different search."
            : "Scan your library after adding .mp4, .m4v, or .webm files. Music mixes stay audio-only."}
          icon="video"
        />
      {:else}
        <ul class="videos-page__list">
          {#each filtered as video (video.id)}
            <li class="videos-page__row">
              <Link href={videoPlayerPath(video.id)} class="videos-page__play">
                <span class="videos-page__icon" aria-hidden="true">
                  <MdiIcon name="video" size={22} />
                </span>
                <span class="videos-page__meta">
                  <span class="videos-page__title">{video.title}</span>
                  <span class="videos-page__sub">
                    {[video.artist, video.format?.toUpperCase()]
                      .filter(Boolean)
                      .join(" · ")}
                  </span>
                </span>
                {#if video.duration}
                  <span class="videos-page__duration">
                    {formatDuration(video.duration)}
                  </span>
                {/if}
              </Link>
              <a
                class="videos-page__download"
                href={mediaTrackDownloadUrl(video.id, {
                  title: video.title,
                  artist: video.artist,
                })}
                download
                rel="noopener"
                aria-label={`Download ${video.title}`}
                onclick={(event) => event.stopPropagation()}
              >
                <MdiIcon name="download" size={18} />
              </a>
            </li>
          {/each}
        </ul>
      {/if}
    {/if}
  </div>
</AppShell>

<style>
  .videos-page {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-4);
    max-width: var(--jb-content-medium);
    margin: 0 auto;
    width: 100%;
  }

  .videos-page__loading {
    display: flex;
    justify-content: center;
    padding: var(--jb-space-8);
  }

  .videos-page__list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-1);
  }

  .videos-page__row {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    padding: var(--jb-space-3) var(--jb-space-4);
    border-radius: var(--jb-radius-md);
    background: color-mix(in srgb, var(--jb-surface) 70%, transparent);
  }

  .videos-page__row:hover {
    background: color-mix(in srgb, var(--jb-accent) 12%, var(--jb-surface));
  }

  .videos-page__list :global(.videos-page__play) {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    min-width: 0;
    flex: 1;
    color: inherit;
    text-decoration: none;
  }

  .videos-page__icon {
    display: flex;
    color: var(--jb-text-muted);
  }

  .videos-page__meta {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
    min-width: 0;
    flex: 1;
  }

  .videos-page__title {
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .videos-page__sub {
    font-size: 0.875rem;
    color: var(--jb-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .videos-page__duration {
    font-variant-numeric: tabular-nums;
    color: var(--jb-text-muted);
    font-size: 0.875rem;
  }

  .videos-page__download {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    width: 2rem;
    height: 2rem;
    border-radius: var(--jb-radius-md);
    color: var(--jb-text-muted);
    text-decoration: none;
  }

  .videos-page__download:hover {
    background: color-mix(in srgb, var(--jb-accent) 16%, transparent);
    color: var(--jb-text);
  }
</style>
