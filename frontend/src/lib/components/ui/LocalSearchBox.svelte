<script lang="ts">
  import Input from "$lib/components/ui/Input.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";

  interface Props {
    value?: string;
    placeholder?: string;
    disabled?: boolean;
    resultCount?: number;
    totalCount?: number;
    class?: string;
  }

  let {
    value = $bindable(""),
    placeholder = "Search",
    disabled = false,
    resultCount,
    totalCount,
    class: className = "",
  }: Props = $props();

  const showCount = $derived(
    value.trim().length > 0 &&
      resultCount !== undefined &&
      totalCount !== undefined,
  );
</script>

<div class="local-search-box {className}">
  <MdiIcon name="search" size={20} />
  <Input bind:value {placeholder} {disabled} autocomplete="off" />
  {#if showCount}
    <span class="local-search-box__count" aria-live="polite">
      {resultCount} of {totalCount}
    </span>
  {/if}
</div>

<style>
  .local-search-box {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    padding: 0 var(--jb-space-4);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-full);
    background: var(--jb-surface);
    color: var(--jb-text-muted);
  }

  .local-search-box :global(.input) {
    border: none;
    background: transparent;
    padding-inline: 0;
  }

  .local-search-box :global(.input:focus) {
    box-shadow: none;
  }

  .local-search-box__count {
    flex-shrink: 0;
    font-size: 0.75rem;
    font-variant-numeric: tabular-nums;
    color: var(--jb-text-subtle);
    white-space: nowrap;
  }

  @media (max-width: 640px) {
    .local-search-box {
      gap: var(--jb-space-2);
      padding: 0 var(--jb-space-3);
    }

    .local-search-box__count {
      display: none;
    }
  }
</style>
