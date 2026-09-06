<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import SourceIcon from "$lib/components/ui/SourceIcon.svelte";
  import { instances } from "$lib/features/instances/store.svelte";
  import { localLibraries } from "$lib/features/local-libraries/store.svelte";
  import {
    activateLocalSource,
    activateSubsonicSource,
    activateUnifiedSource,
  } from "$lib/features/sources/activate";
  import { sources } from "$lib/features/sources/store.svelte";
  import { music } from "$lib/config/music.svelte";
  import { resetSubsonicDetailCaches } from "$lib/subsonic/detail-cache";
  import { router } from "$lib/router/router.svelte";
  import { toast } from "$lib/ui/toast.svelte";

  interface Props {
    compact?: boolean;
    class?: string;
  }

  let { compact = false, class: className = "" }: Props = $props();

  let open = $state(false);

  async function switchSubsonic(id: string) {
    if (
      id === instances.activeId ||
      instances.switching ||
      localLibraries.switching
    ) {
      return;
    }
    try {
      await activateSubsonicSource(id);
      resetSubsonicDetailCaches();
      await music.connect({ force: true });
      toast.success("Switched server");
      open = false;
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to switch server",
      );
    }
  }

  async function switchLocal(id: string) {
    if (
      id === localLibraries.activeId ||
      instances.switching ||
      localLibraries.switching
    ) {
      return;
    }
    try {
      await activateLocalSource(id);
      resetSubsonicDetailCaches();
      await music.connect({ force: true });
      toast.success("Switched to local library");
      open = false;
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to switch library",
      );
    }
  }

  async function switchUnified() {
    if (
      sources.hasUnifiedMode ||
      instances.switching ||
      localLibraries.switching
    ) {
      return;
    }
    try {
      await activateUnifiedSource();
      resetSubsonicDetailCaches();
      await music.connect({ force: true });
      toast.success("Switched to all sources");
      open = false;
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to switch to all sources",
      );
    }
  }

  function addInstance() {
    open = false;
    router.navigate("/setup");
  }

  function addLocalLibrary() {
    open = false;
    router.navigate("/setup?source=local");
  }

  async function refreshActiveLibrary(event: MouseEvent) {
    event.stopPropagation();
    if (music.libraryRefreshing) return;
    try {
      await music.refreshLibrary();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to refresh library",
      );
    }
  }
</script>

<div
  class="instance-switcher {className}"
  class:instance-switcher--compact={compact}
>
  <button
    type="button"
    class="instance-switcher__trigger"
    aria-haspopup="listbox"
    aria-expanded={open}
    disabled={instances.switching || localLibraries.switching}
    onclick={() => {
      open = !open;
    }}
  >
    <SourceIcon kind="auto" size={16} />
    <span class="instance-switcher__label">{sources.activeLabel}</span>
    <MdiIcon name="chevronDown" size={16} />
  </button>

  {#if open}
    <div class="instance-switcher__menu" role="listbox">
      {#if sources.canUseUnified}
        <p class="instance-switcher__section">All sources</p>
        <button
          type="button"
          class="instance-switcher__option"
          class:instance-switcher__option--active={sources.hasUnifiedMode}
          role="option"
          aria-selected={sources.hasUnifiedMode}
          onclick={() => void switchUnified()}
        >
          <SourceIcon kind="unified" size={16} />
          <span class="instance-switcher__option-copy">
            <span class="instance-switcher__option-name">All music</span>
            <span class="instance-switcher__option-meta"
              >Subsonic and local library</span
            >
          </span>
        </button>
      {/if}

      {#if instances.items.length > 0}
        <p class="instance-switcher__section">Subsonic servers</p>
        {#each instances.items as item (item.id)}
          <div
            class="instance-switcher__option-card"
            class:instance-switcher__option-card--active={item.id ===
              instances.activeId}
          >
            <button
              type="button"
              class="instance-switcher__option"
              class:instance-switcher__option--active={item.id ===
                instances.activeId}
              role="option"
              aria-selected={item.id === instances.activeId}
              onclick={() => void switchSubsonic(item.id)}
            >
              <SourceIcon
                kind="server"
                serverName={item.serverName}
                size={16}
              />
              <span class="instance-switcher__option-copy">
                <span class="instance-switcher__option-name">{item.name}</span>
                <span class="instance-switcher__option-meta"
                  >{item.serverName || item.serverUrl}</span
                >
              </span>
            </button>
            {#if item.id === instances.activeId}
              <button
                type="button"
                class="instance-switcher__refresh"
                title="Refresh library"
                aria-label="Refresh library"
                disabled={music.libraryRefreshing}
                onclick={(event) => void refreshActiveLibrary(event)}
              >
                <MdiIcon
                  name="refresh"
                  size={16}
                  class={music.libraryRefreshing
                    ? "instance-switcher__refresh-icon--spin"
                    : ""}
                />
              </button>
            {/if}
          </div>
        {/each}
      {/if}

      {#if localLibraries.enabled && localLibraries.items.length > 0}
        <p class="instance-switcher__section">Local libraries</p>
        {#each localLibraries.items as item (item.id)}
          <button
            type="button"
            class="instance-switcher__option"
            class:instance-switcher__option--active={item.id ===
              localLibraries.activeId}
            role="option"
            aria-selected={item.id === localLibraries.activeId}
            onclick={() => void switchLocal(item.id)}
          >
            <SourceIcon kind="local" size={16} />
            <span class="instance-switcher__option-copy">
              <span class="instance-switcher__option-name">{item.name}</span>
              <span class="instance-switcher__option-meta"
                >{item.trackCount} tracks · {item.path}</span
              >
            </span>
          </button>
        {/each}
      {/if}

      <button
        type="button"
        class="instance-switcher__add"
        onclick={addInstance}
      >
        <MdiIcon name="plus" size={16} />
        Add server
      </button>

      {#if localLibraries.enabled}
        <button
          type="button"
          class="instance-switcher__add"
          onclick={addLocalLibrary}
        >
          <MdiIcon name="folderPlus" size={16} />
          Add local library
        </button>
      {/if}
    </div>
  {/if}
</div>

<style>
  .instance-switcher {
    position: relative;
    width: fit-content;
    max-width: 100%;
  }

  .instance-switcher__trigger {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    width: 100%;
    min-width: 0;
    max-width: 13rem;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--jb-island-border);
    border-radius: var(--jb-radius-full);
    background: var(--jb-island-bg);
    color: var(--jb-text);
    cursor: pointer;
    backdrop-filter: blur(16px);
  }

  .instance-switcher--compact {
    width: 100%;
  }

  .instance-switcher--compact .instance-switcher__trigger {
    width: 100%;
    max-width: none;
  }

  .instance-switcher__label {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-align: left;
    font-size: 0.875rem;
  }

  .instance-switcher__menu {
    position: absolute;
    top: calc(100% + var(--jb-space-2));
    left: 0;
    z-index: 50;
    display: grid;
    gap: var(--jb-space-1);
    width: max-content;
    min-width: 100%;
    max-width: 16rem;
    padding: var(--jb-space-2);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-lg);
    background: var(--jb-bg-elevated);
    box-shadow: var(--jb-shadow-lg);
  }

  .instance-switcher__section {
    margin: var(--jb-space-1) var(--jb-space-3) 0;
    font-size: 0.6875rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--jb-text-subtle);
  }

  .instance-switcher__option-card {
    display: flex;
    align-items: stretch;
    border-radius: var(--jb-radius-md);
    overflow: hidden;
  }

  .instance-switcher__option-card--active {
    background: var(--jb-accent-muted);
  }

  .instance-switcher__option-card .instance-switcher__option {
    flex: 1;
    min-width: 0;
    background: transparent;
  }

  .instance-switcher__option-card--active .instance-switcher__option {
    color: var(--jb-accent);
  }

  .instance-switcher__option-card--active .instance-switcher__option-meta {
    color: color-mix(in srgb, var(--jb-accent) 72%, var(--jb-text-muted));
  }

  .instance-switcher__refresh {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    width: 2.25rem;
    padding: 0;
    border: none;
    border-radius: 0;
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
  }

  .instance-switcher__refresh:hover:not(:disabled) {
    background: color-mix(in srgb, var(--jb-accent) 12%, transparent);
    color: var(--jb-accent);
  }

  .instance-switcher__refresh:disabled {
    opacity: 0.6;
    cursor: default;
  }

  .instance-switcher__option-card--active .instance-switcher__refresh {
    color: var(--jb-accent);
  }

  :global(.instance-switcher__refresh-icon--spin) {
    animation: instance-refresh-spin 1.2s linear infinite;
  }

  @keyframes instance-refresh-spin {
    to {
      transform: rotate(360deg);
    }
  }

  .instance-switcher__option {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    padding: var(--jb-space-3);
    border: none;
    border-radius: var(--jb-radius-md);
    background: transparent;
    color: var(--jb-text);
    text-align: left;
    cursor: pointer;
  }

  .instance-switcher__option-copy {
    display: grid;
    gap: 0.125rem;
    min-width: 0;
    flex: 1;
  }

  .instance-switcher__option:hover {
    background: var(--jb-surface-hover);
  }

  .instance-switcher__option--active {
    color: var(--jb-accent);
  }

  .instance-switcher__option-name {
    font-weight: 600;
    font-size: 0.875rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .instance-switcher__option-meta {
    font-size: 0.75rem;
    color: var(--jb-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .instance-switcher__add {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--jb-space-2);
    width: 100%;
    margin-top: var(--jb-space-1);
    padding: var(--jb-space-3);
    border: 1px dashed var(--jb-border-strong);
    border-radius: var(--jb-radius-md);
    background: transparent;
    color: var(--jb-text-muted);
    font-size: 0.875rem;
    font-weight: 600;
    cursor: pointer;
  }

  .instance-switcher__add:hover {
    border-color: var(--jb-accent);
    color: var(--jb-accent);
    background: var(--jb-accent-muted);
  }
</style>
