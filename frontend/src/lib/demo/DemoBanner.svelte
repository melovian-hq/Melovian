<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { auth } from "$lib/features/auth/store.svelte";
  import { isStaticDemo } from "$lib/config/runtime";

  const show = $derived(auth.demoMode || isStaticDemo());
</script>

{#if show}
  <div class="demo-banner" role="status">
    <MdiIcon name="info" size={18} />
    <span>
      Demo mode. You are browsing a sample library and changes are not saved.
    </span>
  </div>
{/if}

<style>
  .demo-banner {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    flex-wrap: wrap;
    padding: var(--jb-space-2) var(--jb-space-4);
    background: var(--jb-accent-muted);
    border-bottom: 1px solid var(--jb-border);
    color: var(--jb-text);
    font-size: 0.875rem;
  }

  @media (max-width: 768px) {
    .demo-banner {
      flex-wrap: nowrap;
      padding: var(--jb-space-1) var(--jb-space-3);
      font-size: 0.75rem;
      line-height: 1.4;
    }

    .demo-banner :global(svg) {
      width: 0.875rem;
      height: 0.875rem;
      flex-shrink: 0;
    }

    .demo-banner span {
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }
</style>
