<script lang="ts">
  import Button from "$lib/components/ui/Button.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import {
    listDirectories,
    type DirectoryEntry,
  } from "$lib/features/filesystem/api";

  interface Props {
    open?: boolean;
    initialPath?: string;
    onselect?: (path: string) => void;
    onclose?: () => void;
  }

  let { open = false, initialPath = "", onselect, onclose }: Props = $props();

  let currentPath = $state("");
  let parentPath = $state("");
  let entries = $state<DirectoryEntry[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let loadToken = 0;

  async function load(path: string) {
    const token = ++loadToken;
    loading = true;
    error = null;
    try {
      const listing = await listDirectories(path);
      if (token !== loadToken) return;
      currentPath = listing.path;
      parentPath = listing.parent;
      entries = listing.entries;
    } catch (err) {
      if (token !== loadToken) return;
      error = err instanceof Error ? err.message : "Could not list folders";
      entries = [];
    } finally {
      if (token === loadToken) loading = false;
    }
  }

  $effect(() => {
    if (!open) return;
    void load(initialPath.trim());
  });

  function onKeydown(event: KeyboardEvent) {
    if (!open) return;
    if (event.key === "Escape") {
      event.preventDefault();
      onclose?.();
    }
  }
</script>

<svelte:window onkeydown={onKeydown} />

{#if open}
  <button
    type="button"
    class="folder-browser__backdrop"
    aria-label="Close folder browser"
    onclick={() => onclose?.()}
  ></button>
  <div
    class="folder-browser"
    role="dialog"
    aria-modal="true"
    aria-labelledby="folder-browser-title"
  >
    <header class="folder-browser__header">
      <h2 id="folder-browser-title">Choose folder</h2>
      <p class="folder-browser__path" title={currentPath}>
        {currentPath || "Loading…"}
      </p>
    </header>

    <div class="folder-browser__toolbar">
      <Button
        type="button"
        variant="surface"
        size="sm"
        disabled={loading || !parentPath}
        onclick={() => void load(parentPath)}
      >
        <MdiIcon name="chevronUp" size={16} />
        Up
      </Button>
      <Button
        type="button"
        variant="surface"
        size="sm"
        disabled={loading}
        onclick={() => void load(currentPath || initialPath)}
      >
        <MdiIcon name="refresh" size={16} />
        Refresh
      </Button>
    </div>

    <div class="folder-browser__body" role="listbox" aria-label="Folders">
      {#if loading}
        <div class="folder-browser__state"><Spinner /></div>
      {:else if error}
        <p class="folder-browser__error">{error}</p>
      {:else if entries.length === 0}
        <p class="folder-browser__empty">No subfolders here.</p>
      {:else}
        {#each entries as entry (entry.path)}
          <button
            type="button"
            class="folder-browser__entry"
            role="option"
            aria-selected="false"
            onclick={() => void load(entry.path)}
          >
            <MdiIcon name="folderOpen" size={18} />
            <span>{entry.name}</span>
          </button>
        {/each}
      {/if}
    </div>

    <footer class="folder-browser__footer">
      <Button type="button" variant="surface" onclick={() => onclose?.()}>
        Cancel
      </Button>
      <Button
        type="button"
        disabled={loading || !currentPath}
        onclick={() => {
          if (!currentPath) return;
          onselect?.(currentPath);
          onclose?.();
        }}
      >
        Use this folder
      </Button>
    </footer>
  </div>
{/if}

<style>
  .folder-browser__backdrop {
    position: fixed;
    inset: 0;
    z-index: 80;
    border: none;
    background: rgb(0 0 0 / 0.45);
    cursor: pointer;
  }

  .folder-browser {
    position: fixed;
    z-index: 81;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    display: flex;
    flex-direction: column;
    width: min(32rem, calc(100vw - 2rem));
    max-height: min(36rem, calc(100vh - 2rem));
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-xl);
    background: var(--jb-bg-elevated);
    box-shadow: var(--jb-shadow-lg);
    overflow: hidden;
  }

  .folder-browser__header {
    padding: var(--jb-space-4) var(--jb-space-5) var(--jb-space-3);
    border-bottom: 1px solid var(--jb-border);
  }

  .folder-browser__header h2 {
    margin: 0;
    font-size: 1.0625rem;
  }

  .folder-browser__path {
    margin: var(--jb-space-2) 0 0;
    color: var(--jb-text-muted);
    font-size: 0.8125rem;
    font-family: var(--jb-font-mono, ui-monospace, monospace);
    word-break: break-all;
    line-height: 1.4;
  }

  .folder-browser__toolbar {
    display: flex;
    gap: var(--jb-space-2);
    padding: var(--jb-space-3) var(--jb-space-5);
    border-bottom: 1px solid var(--jb-border);
  }

  .folder-browser__body {
    flex: 1;
    min-height: 12rem;
    overflow-y: auto;
    padding: var(--jb-space-2);
  }

  .folder-browser__entry {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    width: 100%;
    padding: 0.625rem 0.75rem;
    border: none;
    border-radius: var(--jb-radius-md);
    background: transparent;
    color: var(--jb-text);
    text-align: left;
    cursor: pointer;
  }

  .folder-browser__entry:hover {
    background: var(--jb-surface-hover);
  }

  .folder-browser__state,
  .folder-browser__empty,
  .folder-browser__error {
    display: grid;
    place-items: center;
    min-height: 10rem;
    margin: 0;
    padding: var(--jb-space-4);
    color: var(--jb-text-muted);
    text-align: center;
  }

  .folder-browser__error {
    color: var(--jb-danger);
  }

  .folder-browser__footer {
    display: flex;
    justify-content: flex-end;
    gap: var(--jb-space-2);
    padding: var(--jb-space-3) var(--jb-space-5) var(--jb-space-4);
    border-top: 1px solid var(--jb-border);
  }
</style>
