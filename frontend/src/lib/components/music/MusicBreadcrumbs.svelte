<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Link from "$lib/router/Link.svelte";

  export interface MusicBreadcrumb {
    label: string;
    href?: string;
  }

  interface Props {
    items?: MusicBreadcrumb[];
  }

  let { items = [] }: Props = $props();
</script>

<nav class="music-breadcrumbs" aria-label="Breadcrumb">
  <Link href="/music" class="music-breadcrumbs__link">Music</Link>
  {#each items as item, i (i)}
    <MdiIcon name="chevronRight" size={14} aria-hidden="true" />
    {#if item.href}
      <Link href={item.href} class="music-breadcrumbs__link">{item.label}</Link>
    {:else}
      <span class="music-breadcrumbs__current" aria-current="page"
        >{item.label}</span
      >
    {/if}
  {/each}
</nav>

<style>
  .music-breadcrumbs {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
    margin-bottom: var(--jb-space-5);
    font-size: 0.875rem;
    color: var(--jb-text-muted);
  }

  :global(a.music-breadcrumbs__link) {
    color: var(--jb-music-accent);
    text-decoration: none;
    font-weight: 600;
  }

  :global(a.music-breadcrumbs__link:hover) {
    text-decoration: underline;
  }

  .music-breadcrumbs__current {
    color: var(--jb-text);
    font-weight: 600;
  }
</style>
