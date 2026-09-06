<script lang="ts">
  import Button from "$lib/components/ui/Button.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { localLibraries } from "$lib/features/local-libraries/store.svelte";
  import { activateLocalSource } from "$lib/features/sources/activate";
  import { toast } from "$lib/ui/toast.svelte";
  import { confirmDialog } from "$lib/ui/confirm.svelte";
  import type { LocalLibrary } from "$lib/features/local-libraries/types";

  interface Props {
    class?: string;
  }

  let { class: className = "" }: Props = $props();

  let scanningId = $state<string | null>(null);

  async function activate(item: LocalLibrary) {
    if (item.id === localLibraries.activeId) return;
    try {
      await activateLocalSource(item.id);
      toast.success(`Switched to ${item.name}`);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to switch library",
      );
    }
  }

  async function rescan(item: LocalLibrary) {
    scanningId = item.id;
    try {
      await localLibraries.scan(item.id);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Scan failed");
    } finally {
      scanningId = null;
    }
  }

  async function remove(item: LocalLibrary) {
    const ok = await confirmDialog.confirm({
      title: "Remove local library",
      message: `Remove ${item.name}? Files on disk are not deleted.`,
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok) return;
    try {
      await localLibraries.remove(item.id);
      toast.success("Local library removed");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to remove library",
      );
    }
  }
</script>

<div class="local-library-list {className}">
  {#if localLibraries.items.length > 1}
    <p class="local-library-list__summary">
      {localLibraries.items.length} libraries · one active at a time
    </p>
  {/if}
  {#each localLibraries.items as item (item.id)}
    <article
      class="local-library-list__item"
      class:local-library-list__item--active={item.id ===
        localLibraries.activeId}
    >
      <div class="local-library-list__copy">
        <h3 class="local-library-list__name">{item.name}</h3>
        <p class="local-library-list__meta">{item.path}</p>
        <p class="local-library-list__stats">
          {item.trackCount} tracks
          {#if item.missingCount > 0}
            · {item.missingCount} missing
          {/if}
          {#if item.duplicateCount > 0}
            · {item.duplicateCount} duplicates
          {/if}
          {#if item.scanStatus === "scanning"}
            · scanning
            {#if item.scanProgress}
              ({item.scanProgress.processed.toLocaleString()} files)
            {/if}
          {/if}
        </p>
      </div>
      <div class="local-library-list__actions">
        {#if item.id !== localLibraries.activeId}
          <Button size="sm" variant="ghost" onclick={() => void activate(item)}>
            Switch
          </Button>
        {:else}
          <span class="local-library-list__badge">Active</span>
        {/if}
        <Button
          size="sm"
          variant="ghost"
          disabled={scanningId === item.id}
          onclick={() => void rescan(item)}
        >
          {scanningId === item.id ? "Scanning..." : "Rescan"}
        </Button>
        <Button
          size="sm"
          variant="ghost"
          onclick={() => void remove(item)}
          aria-label="Remove local library"
        >
          <MdiIcon name="trash2" size={16} />
        </Button>
      </div>
    </article>
  {/each}
</div>

<style>
  .local-library-list {
    display: grid;
    gap: 0;
  }

  .local-library-list__summary {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--jb-text-subtle);
  }

  .local-library-list__item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-4);
    padding: var(--jb-space-4) 0;
    border-bottom: 1px solid var(--jb-border);
    background: transparent;
  }

  .local-library-list__item:last-child {
    border-bottom: none;
    padding-bottom: 0;
  }

  .local-library-list__item:first-child {
    padding-top: 0;
  }

  .local-library-list__item--active {
    background: transparent;
  }

  .local-library-list__item--active .local-library-list__name {
    color: var(--jb-accent);
  }

  .local-library-list__name {
    margin: 0;
    font-size: 1rem;
  }

  .local-library-list__meta,
  .local-library-list__stats {
    margin: 0.125rem 0 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
    word-break: break-all;
  }

  .local-library-list__actions {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    flex-shrink: 0;
  }

  .local-library-list__badge {
    font-size: 0.75rem;
    font-weight: 700;
    color: var(--jb-accent);
  }

  @media (max-width: 640px) {
    .local-library-list__item {
      flex-direction: column;
      align-items: stretch;
    }

    .local-library-list__actions {
      justify-content: flex-end;
    }
  }
</style>
