<script lang="ts">
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Link from "$lib/router/Link.svelte";
  import { router } from "$lib/router/router.svelte";

  const attemptedPath = $derived(router.pathname);
</script>

<div class="not-found-page">
  <EmptyState
    title="Page not found"
    titleAs="h1"
    message="No page exists at {attemptedPath}. Check the address or head back to your music."
    icon="search"
  >
    {#snippet actions()}
      <Link href="/music" class="not-found-page__primary">Back to music</Link>
      <Link href="/music/search" class="not-found-page__secondary"
        >Search your library</Link
      >
    {/snippet}
  </EmptyState>
</div>

<style>
  .not-found-page {
    flex: 1 1 auto;
    min-height: 0;
    display: grid;
    place-content: center;
    padding: var(--jb-space-6) var(--jb-space-4);
  }

  .not-found-page :global(.empty-state) {
    max-width: 32rem;
  }

  :global(a.not-found-page__primary),
  :global(a.not-found-page__secondary) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0.5rem 1rem;
    border-radius: var(--jb-radius-md);
    font-weight: 600;
    font-size: 0.9375rem;
    text-decoration: none;
    transition:
      background var(--jb-transition),
      border-color var(--jb-transition);
  }

  :global(a.not-found-page__primary) {
    background: var(--jb-accent);
    color: var(--jb-accent-text);
  }

  :global(a.not-found-page__primary:hover) {
    background: var(--jb-accent-hover);
  }

  :global(a.not-found-page__secondary) {
    background: var(--jb-surface);
    color: var(--jb-text);
    border: 1px solid var(--jb-border);
  }

  :global(a.not-found-page__secondary:hover) {
    background: var(--jb-surface-hover);
  }

  @media (prefers-reduced-motion: reduce) {
    :global(a.not-found-page__primary),
    :global(a.not-found-page__secondary) {
      transition: none;
    }
  }
</style>
