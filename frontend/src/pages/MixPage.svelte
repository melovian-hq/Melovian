<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import AppShell from "$lib/components/layout/AppShell.svelte";
  import MusicBreadcrumbs from "$lib/components/music/MusicBreadcrumbs.svelte";
  import TrackVirtualList from "$lib/components/music/TrackVirtualList.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Link from "$lib/router/Link.svelte";
  import MixContextMenu from "$lib/components/music/MixContextMenu.svelte";
  import { contextMenuPositionFromEvent } from "$lib/components/ui/context-menu";
  import { music } from "$lib/config/music.svelte";
  import { mixTrackCount } from "$lib/music/mix-storage";
  import { coverArtUrl } from "$lib/subsonic";

  interface Props {
    mixId: string;
  }

  let { mixId }: Props = $props();

  let mixLoading = $state(false);
  let heroMenu = $state<{ x: number; y: number } | null>(null);

  const mix = $derived(music.getMix(mixId));
  const loading = $derived(
    !music.libraryReady ? music.loading : mixLoading && !mix,
  );

  $effect(() => {
    const id = mixId;
    return () => music.slimStoredMix(id);
  });

  $effect(() => {
    const id = mixId;
    if (!music.libraryReady) return;
    const current = music.getMix(id);
    const needsHydration =
      !current ||
      (current.tracks.length === 0 && (current.trackIds?.length ?? 0) > 0);
    if (!needsHydration) return;
    mixLoading = true;
    void music.ensureMix(id).finally(() => {
      mixLoading = false;
    });
  });

  const image = $derived(
    mix?.coverArtId ? coverArtUrl(music.config, mix.coverArtId, 400) : null,
  );
  const breadcrumbItems = $derived(
    mix ? [{ label: mix.title }] : [{ label: "Mix" }],
  );
</script>

<AppShell compactTop>
  <div class="mix-page">
    <MusicBreadcrumbs items={breadcrumbItems} />

    {#if loading}
      <div class="mix-page__loading"><Spinner /></div>
    {:else if !mix}
      <EmptyState
        title="Mix not found"
        message="This mix is no longer available."
        icon="listMusic"
      >
        {#snippet actions()}
          <Link href="/music">Back to music</Link>
        {/snippet}
      </EmptyState>
    {:else}
      <header
        class="mix-hero"
        style="--mix-gradient: {mix.gradient}"
        role="group"
        oncontextmenu={(event) => {
          heroMenu = contextMenuPositionFromEvent(event);
        }}
      >
        {#if image}
          <img class="mix-hero__bg" src={image} alt="" fetchpriority="high" />
        {/if}
        <div class="mix-hero__overlay"></div>
        <div class="mix-hero__content">
          <p class="mix-hero__type">Made for you</p>
          <h1 class="mix-hero__title">{mix.title}</h1>
          <p class="mix-hero__subtitle">{mix.subtitle}</p>
          <p class="mix-hero__meta">{mixTrackCount(mix)} tracks</p>
          <div class="mix-hero__actions">
            <Button size="lg" onclick={() => music.playMix(mix)}>
              <MdiIcon name="play" size={18} fill="currentColor" />
              Play mix
            </Button>
            <Button
              size="lg"
              variant="ghost"
              onclick={() => {
                music.shuffle = true;
                music.playMix(mix);
              }}
            >
              <MdiIcon name="shuffle" size={18} />
              Shuffle
            </Button>
          </div>
        </div>
      </header>

      <TrackVirtualList
        tracks={mix.tracks}
        onplay={(index) => music.playTracks(mix.tracks, index)}
        class="mix-tracks"
      />
    {/if}
  </div>
</AppShell>

{#if heroMenu && mix}
  <MixContextMenu
    {mix}
    x={heroMenu.x}
    y={heroMenu.y}
    onclose={() => (heroMenu = null)}
  />
{/if}

<style>
  .mix-page {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-6);
  }

  .mix-page__loading {
    display: grid;
    place-content: center;
    min-height: 12rem;
  }

  .mix-hero {
    position: relative;
    min-height: 14rem;
    border-radius: var(--jb-radius-xl);
    overflow: hidden;
    background: var(--mix-gradient);
    color: white;
  }

  .mix-hero__bg {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    object-fit: cover;
    opacity: 0.35;
    filter: blur(2px);
  }

  .mix-hero__overlay {
    position: absolute;
    inset: 0;
    background: linear-gradient(to top, rgb(0 0 0 / 0.8), rgb(0 0 0 / 0.25));
  }

  .mix-hero__content {
    position: relative;
    display: flex;
    flex-direction: column;
    justify-content: flex-end;
    min-height: 14rem;
    padding: var(--jb-space-6);
  }

  .mix-hero__type {
    margin: 0 0 var(--jb-space-2);
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    opacity: 0.85;
  }

  .mix-hero__title {
    margin: 0 0 var(--jb-space-2);
    font-size: clamp(1.75rem, 4vw, 2.5rem);
    font-weight: 800;
    line-height: 1.1;
  }

  .mix-hero__subtitle,
  .mix-hero__meta {
    margin: 0 0 var(--jb-space-2);
    opacity: 0.88;
  }

  .mix-hero__actions {
    display: flex;
    gap: var(--jb-space-3);
    margin-top: var(--jb-space-4);
  }

  :global(.mix-tracks.lazy-track-list),
  :global(.mix-tracks.virtual-list--document) {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: var(--jb-space-2) 0;
  }

  @media (max-width: 768px) {
    .mix-hero__content {
      padding: var(--jb-space-4);
      min-height: 12rem;
    }

    .mix-hero__actions {
      flex-wrap: nowrap;
      align-items: stretch;
    }

    .mix-hero__actions :global(.btn) {
      flex: 1 1 0;
      min-width: 0;
      justify-content: center;
      white-space: nowrap;
    }
  }
</style>
