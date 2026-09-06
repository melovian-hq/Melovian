<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { theme, type ThemeMode } from "$lib/theme/theme.svelte";

  interface Props {
    class?: string;
    embedded?: boolean;
  }

  let { class: className = "", embedded = false }: Props = $props();

  const modes: { value: ThemeMode; label: string; icon: string }[] = [
    { value: "light", label: "Light", icon: "sun" },
    { value: "dark", label: "Dark", icon: "moon" },
    { value: "system", label: "System", icon: "monitor" },
  ];
</script>

<div
  class="theme-toggle {className}"
  class:theme-toggle--embedded={embedded}
  role="group"
  aria-label="Theme"
>
  {#each modes as mode (mode.value)}
    <button
      type="button"
      class="theme-toggle__btn"
      class:theme-toggle__btn--active={theme.mode === mode.value}
      aria-pressed={theme.mode === mode.value}
      title={mode.label}
      onclick={() => theme.setMode(mode.value)}
    >
      <MdiIcon name={mode.icon} size={16} />
      <span class="sr-only">{mode.label}</span>
    </button>
  {/each}
</div>

<style>
  .theme-toggle {
    display: inline-flex;
    padding: var(--jb-space-1);
    border-radius: var(--jb-radius-full);
    background: var(--jb-island-bg);
    border: 1px solid var(--jb-island-border);
    backdrop-filter: blur(16px);
    box-shadow: var(--jb-shadow-md);
    gap: var(--jb-space-1);
  }

  .theme-toggle__btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 2rem;
    height: 2rem;
    border: none;
    border-radius: var(--jb-radius-full);
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
    transition:
      background var(--jb-transition),
      color var(--jb-transition);
  }

  .theme-toggle__btn:hover {
    background: var(--jb-surface-hover);
    color: var(--jb-text);
  }

  .theme-toggle__btn--active {
    background: var(--jb-accent);
    color: var(--jb-accent-text);
  }

  .theme-toggle--embedded {
    background: var(--jb-bg-muted);
    border-color: var(--jb-border);
    box-shadow: none;
    backdrop-filter: none;
  }

  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }
</style>
