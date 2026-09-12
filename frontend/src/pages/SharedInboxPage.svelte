<script lang="ts">
  import { APP_NAME } from "$lib/brand";
  import MusicBreadcrumbs from "$lib/components/music/MusicBreadcrumbs.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import Link from "$lib/router/Link.svelte";
  import { auth } from "$lib/features/auth/store.svelte";
  import * as musicApi from "$lib/music/api";
  import type { MusicShare } from "$lib/music/api";

  let loading = $state(true);
  let error = $state<string | null>(null);
  let items = $state<MusicShare[]>([]);

  $effect(() => {
    if (!auth.enabled || !auth.authenticated) {
      loading = false;
      items = [];
      return;
    }
    let cancelled = false;
    loading = true;
    error = null;
    void (async () => {
      try {
        const next = await musicApi.listShareInbox();
        if (!cancelled) items = next;
      } catch (err) {
        if (!cancelled) {
          error = err instanceof Error ? err.message : "Failed to load inbox";
          items = [];
        }
      } finally {
        if (!cancelled) loading = false;
      }
    })();
    return () => {
      cancelled = true;
    };
  });

  function label(share: MusicShare): string {
    if (share.description?.trim()) return share.description.trim();
    return share.resourceType || "Shared item";
  }
</script>

<div class="shared-inbox">
  <MusicBreadcrumbs
    items={[{ label: "Home", href: "/music" }, { label: "Shared with you" }]}
  />

  <header class="shared-inbox__hero">
    <h1>Shared with you</h1>
    <p>Playlists and albums other {APP_NAME} users shared to your account.</p>
  </header>

  {#if !auth.enabled || !auth.authenticated}
    <EmptyState
      title="Sign in required"
      message={`Shared inbox needs a ${APP_NAME} account.`}
    >
      {#snippet actions()}
        <Link href="/account/login" class="shared-inbox__link">Sign in</Link>
      {/snippet}
    </EmptyState>
  {:else if loading}
    <div class="shared-inbox__loading">
      <Spinner />
    </div>
  {:else if error}
    <EmptyState title="Could not load inbox" message={error} />
  {:else if items.length === 0}
    <EmptyState
      title="Nothing shared yet"
      message="When someone shares a playlist with your username, it shows up here."
    />
  {:else}
    <ul class="shared-inbox__list">
      {#each items as share (share.id)}
        <li>
          <Link href="/share/{share.token}" class="shared-inbox__card">
            <span class="shared-inbox__icon" aria-hidden="true">
              <MdiIcon name="share" size={20} />
            </span>
            <span class="shared-inbox__copy">
              <span class="shared-inbox__title">{label(share)}</span>
              <span class="shared-inbox__meta">
                {share.resourceType}
                {#if share.tracks?.length}
                  · {share.tracks.length} tracks
                {/if}
              </span>
            </span>
            <MdiIcon name="chevronRight" size={18} />
          </Link>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .shared-inbox {
    max-width: var(--jb-content-narrow);
    display: grid;
    gap: var(--jb-space-5);
  }

  .shared-inbox__hero h1 {
    margin: 0 0 var(--jb-space-2);
    font-size: 1.75rem;
  }

  .shared-inbox__hero p {
    margin: 0;
    color: var(--jb-text-muted);
  }

  .shared-inbox__loading {
    display: grid;
    place-content: center;
    min-height: 12rem;
  }

  .shared-inbox__list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: var(--jb-space-2);
  }

  :global(.shared-inbox__card) {
    display: grid;
    grid-template-columns: auto 1fr auto;
    align-items: center;
    gap: var(--jb-space-3);
    padding: var(--jb-space-3) var(--jb-space-4);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-lg);
    background: var(--jb-surface);
    color: inherit;
    text-decoration: none;
  }

  :global(.shared-inbox__card:hover) {
    border-color: color-mix(in srgb, var(--jb-accent) 40%, var(--jb-border));
  }

  .shared-inbox__icon {
    display: grid;
    place-content: center;
    width: 2.5rem;
    height: 2.5rem;
    border-radius: var(--jb-radius-md);
    background: color-mix(in srgb, var(--jb-accent) 12%, transparent);
    color: var(--jb-accent);
  }

  .shared-inbox__copy {
    display: grid;
    gap: 0.15rem;
    min-width: 0;
  }

  .shared-inbox__title {
    font-weight: 600;
  }

  .shared-inbox__meta {
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
    text-transform: capitalize;
  }

  :global(.shared-inbox__link) {
    color: var(--jb-accent);
    font-weight: 600;
  }
</style>
