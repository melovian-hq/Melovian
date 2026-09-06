<script lang="ts">
  import Link from "$lib/router/Link.svelte";

  interface Props {
    title: string;
    href?: string;
    hrefLabel?: string;
    rows?: 2;
    children?: import("svelte").Snippet;
    action?: import("svelte").Snippet;
  }

  let {
    title,
    href,
    hrefLabel = "Show all",
    rows,
    children,
    action,
  }: Props = $props();
</script>

<section class="home-shelf">
  <header class="home-shelf__header">
    {#if href}
      <Link {href} class="home-shelf__heading-link">
        <h2 class="home-shelf__title">{title}</h2>
      </Link>
      <Link {href} class="home-shelf__all">{hrefLabel}</Link>
    {:else}
      <h2 class="home-shelf__title">{title}</h2>
      {#if action}
        <div class="home-shelf__action">
          {@render action()}
        </div>
      {/if}
    {/if}
  </header>
  <div class="home-shelf__items" class:home-shelf__items--two-rows={rows === 2}>
    {@render children?.()}
  </div>
</section>

<style>
  .home-shelf {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-3);
    min-width: 0;
    overflow-x: hidden;
  }

  .home-shelf__header {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--jb-space-4);
    min-width: 0;
  }

  .home-shelf__title {
    margin: 0;
    min-width: 0;
    font-size: 1.25rem;
    font-weight: 700;
    letter-spacing: -0.02em;
    color: var(--jb-text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  :global(a.home-shelf__heading-link) {
    min-width: 0;
    overflow: hidden;
    text-decoration: none;
    color: inherit;
  }

  :global(a.home-shelf__heading-link:hover) .home-shelf__title {
    text-decoration: underline;
  }

  :global(a.home-shelf__all) {
    flex-shrink: 0;
    font-size: 0.8125rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    text-decoration: none;
    color: var(--jb-text-muted);
  }

  :global(a.home-shelf__all:hover) {
    color: var(--jb-text);
  }

  .home-shelf__items {
    display: grid;
    grid-template-columns: repeat(
      auto-fill,
      minmax(min(100%, var(--jb-card-grid-min)), 1fr)
    );
    gap: var(--jb-space-4);
    overflow-x: hidden;
  }

  .home-shelf__items > :global(*) {
    min-width: 0;
    width: auto;
  }

  .home-shelf__items--two-rows {
    grid-template-columns: unset;
    grid-template-rows: repeat(2, auto);
    grid-auto-flow: column;
    grid-auto-columns: minmax(0, 1fr);
  }

  .home-shelf__items--two-rows > :global(.home-mix) {
    min-width: 0;
  }

  .home-shelf__items--two-rows :global(a.mix-card) {
    height: 100%;
  }

  @media (max-width: 640px) {
    .home-shelf__title {
      font-size: 1.0625rem;
    }

    .home-shelf__items {
      gap: var(--jb-space-3);
      grid-template-columns: repeat(
        auto-fill,
        minmax(min(100%, var(--jb-card-grid-min-sm)), 1fr)
      );
    }

    .home-shelf__items--two-rows {
      grid-template-columns: repeat(2, minmax(0, 1fr));
      grid-template-rows: none;
      grid-auto-flow: row;
      grid-auto-columns: auto;
    }
  }
</style>
