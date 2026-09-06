<script lang="ts">
  import type { SettingsTab, SettingsTabId } from "$lib/settings/tabs";

  interface Props {
    tabs: readonly SettingsTab[];
    active: SettingsTabId;
    onselect: (id: SettingsTabId) => void;
  }

  let { tabs, active, onselect }: Props = $props();

  const recommended = $derived(tabs.filter((tab) => tab.tier !== "advanced"));
  const advanced = $derived(tabs.filter((tab) => tab.tier === "advanced"));
</script>

<nav class="settings-nav" aria-label="Settings sections">
  <div class="settings-nav__group">
    <p class="settings-nav__group-label">Settings</p>
    {#each recommended as tab (tab.id)}
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

  {#if advanced.length > 0}
    <div class="settings-nav__group">
      <p class="settings-nav__group-label">Advanced</p>
      {#each advanced as tab (tab.id)}
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
  {/if}
</nav>

<style>
  .settings-nav {
    display: flex;
    flex-shrink: 0;
    gap: var(--jb-space-1);
    overflow-x: auto;
    padding: var(--jb-space-2) 0;
    border-bottom: 1px solid var(--jb-border);
    background: transparent;
    scrollbar-width: thin;
  }

  .settings-nav__group {
    display: flex;
    gap: var(--jb-space-1);
  }

  .settings-nav__group-label {
    display: none;
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
    .settings-nav {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: var(--jb-space-2);
      overflow: visible;
      padding: var(--jb-space-2) 0;
    }

    .settings-nav__group {
      display: contents;
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
      flex-direction: column;
      align-items: stretch;
      width: 16rem;
      max-height: calc(100vh - 6rem);
      overflow-y: auto;
      overflow-x: hidden;
      border-bottom: none;
      border-right: 1px solid var(--jb-border);
      border-radius: 0;
      padding: 0 var(--jb-space-4) 0 0;
      background: transparent;
      scrollbar-width: thin;
      gap: var(--jb-space-4);
    }

    .settings-nav__group {
      flex-direction: column;
      align-items: stretch;
      gap: var(--jb-space-1);
    }

    .settings-nav__group-label {
      display: block;
      margin: 0 0 var(--jb-space-1);
      padding: 0 var(--jb-space-3);
      font-size: 0.6875rem;
      font-weight: 700;
      letter-spacing: 0.06em;
      text-transform: uppercase;
      color: var(--jb-text-subtle);
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
