<script lang="ts">
  import type { SettingsTab, SettingsTabId } from "$lib/settings/tabs";
  import { navTabsForMode } from "$lib/settings/tabs";
  import {
    settingsNavMode,
    setSettingsNavMode,
    type SettingsNavMode,
  } from "$lib/settings/nav-state.svelte";
  import { Tabs } from "bits-ui";

  interface Props {
    tabs: readonly SettingsTab[];
    active: SettingsTabId;
    onselect: (id: SettingsTabId) => void;
    searching?: boolean;
  }

  let { tabs, active, onselect, searching = false }: Props = $props();

  const mode = $derived(settingsNavMode());
  const items = $derived(navTabsForMode(tabs, mode, active, searching));
</script>

<nav class="settings-nav" aria-label="Settings sections">
  <Tabs.Root
    value={mode}
    onValueChange={(next) => setSettingsNavMode(next as SettingsNavMode)}
    style="display: contents"
  >
    <Tabs.List class="settings-nav__mode" aria-label="Settings level">
      <Tabs.Trigger value="simple" class="settings-nav__mode-btn">
        Simple
      </Tabs.Trigger>
      <Tabs.Trigger value="advanced" class="settings-nav__mode-btn">
        Advanced
      </Tabs.Trigger>
    </Tabs.List>
  </Tabs.Root>

  <div class="settings-nav__group">
    {#each items as tab (tab.id)}
      <button
        type="button"
        class="settings-nav__item"
        class:settings-nav__item--active={active === tab.id}
        aria-current={active === tab.id ? "page" : undefined}
        onclick={() => onselect(tab.id)}
      >
        <span class="settings-nav__label">{tab.label}</span>
        <span class="settings-nav__description">{tab.description}</span>
      </button>
    {/each}
  </div>
</nav>

<style>
  .settings-nav {
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
    gap: var(--jb-space-2);
    padding: var(--jb-space-2) 0;
    border-bottom: 1px solid var(--jb-border);
    background: transparent;
  }

  .settings-nav__group {
    display: flex;
    gap: var(--jb-space-1);
    overflow-x: auto;
    scrollbar-width: thin;
  }

  :global(.settings-nav__mode) {
    display: flex;
    flex-shrink: 0;
    gap: var(--jb-space-1);
    padding: 0.2rem;
    border-radius: var(--jb-radius-full);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
  }

  :global(.settings-nav__mode-btn) {
    flex: 1;
    border: none;
    background: transparent;
    color: var(--jb-text-muted);
    font-size: 0.8125rem;
    font-weight: 650;
    padding: 0.4rem 0.9rem;
    border-radius: var(--jb-radius-full);
    cursor: pointer;
    text-align: center;
    transition:
      background var(--jb-transition),
      color var(--jb-transition);
  }

  :global(.settings-nav__mode-btn:hover) {
    color: var(--jb-text);
  }

  :global(.settings-nav__mode-btn[data-state="active"]) {
    background: var(--jb-bg-muted);
    color: var(--jb-text);
    box-shadow: var(--jb-shadow-sm);
  }

  @media (prefers-reduced-motion: reduce) {
    :global(.settings-nav__mode-btn) {
      transition: none;
    }
  }

  .settings-nav__item {
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 0.125rem;
    padding: 0.625rem 0.75rem;
    border: none;
    border-radius: var(--jb-radius-lg);
    background: transparent;
    color: var(--jb-text-muted);
    text-align: left;
    cursor: pointer;
    transition:
      background var(--jb-transition),
      color var(--jb-transition);
  }

  .settings-nav__item:hover {
    background: var(--jb-bg-muted);
    color: var(--jb-text);
  }

  .settings-nav__item--active {
    background: var(--jb-accent);
    color: var(--jb-accent-text);
  }

  .settings-nav__item--active:hover {
    background: var(--jb-accent-hover);
    color: var(--jb-accent-text);
  }

  .settings-nav__label {
    font-size: 0.875rem;
    font-weight: 600;
  }

  .settings-nav__description {
    display: none;
    font-size: 0.75rem;
    line-height: 1.35;
    opacity: 0.85;
  }

  @media (max-width: 768px) {
    .settings-nav__group {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: var(--jb-space-2);
      overflow: visible;
    }

    .settings-nav__item {
      width: 100%;
      min-height: 3.125rem;
      justify-content: center;
    }
  }

  @media (min-width: 1024px) {
    .settings-nav {
      position: sticky;
      top: calc(var(--jb-window-chrome-offset, 0px) + var(--jb-space-4));
      align-self: flex-start;
      align-items: stretch;
      width: 16rem;
      max-height: calc(100vh - 6rem);
      overflow-y: auto;
      overflow-x: hidden;
      border-bottom: none;
      border-right: 1px solid var(--jb-border);
      border-radius: 0;
      padding: 0 var(--jb-space-4) 0 0;
      gap: var(--jb-space-4);
    }

    .settings-nav__group {
      flex-direction: column;
      align-items: stretch;
      overflow: visible;
      gap: var(--jb-space-1);
    }

    .settings-nav__item {
      width: 100%;
    }

    .settings-nav__description {
      display: block;
    }
  }

  @media (min-width: 1400px) {
    .settings-nav {
      width: 18rem;
    }
  }
</style>
