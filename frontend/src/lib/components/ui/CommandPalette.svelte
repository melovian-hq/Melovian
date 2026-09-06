<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { commandPalette } from "$lib/ui/command-palette.svelte";
  import {
    searchPaletteWithMusic,
    type PaletteCommand,
  } from "$lib/music/command-palette";

  let query = $state("");
  let commands = $state<PaletteCommand[]>([]);
  let activeIndex = $state(0);
  let loading = $state(false);
  let inputEl = $state<HTMLInputElement | null>(null);
  let searchToken = 0;

  $effect(() => {
    if (!commandPalette.open) {
      query = "";
      commands = [];
      activeIndex = 0;
      return;
    }

    queueMicrotask(() => inputEl?.focus());
  });

  $effect(() => {
    if (!commandPalette.open) return;
    const currentQuery = query;
    const token = ++searchToken;
    loading = true;

    const timer = setTimeout(() => {
      void searchPaletteWithMusic(currentQuery).then((results) => {
        if (token !== searchToken) return;
        commands = results;
        activeIndex = 0;
        loading = false;
      });
    }, 120);

    return () => clearTimeout(timer);
  });

  function close() {
    commandPalette.close();
  }

  async function runCommand(command: PaletteCommand) {
    close();
    await command.run();
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      if (commands.length === 0) return;
      activeIndex = (activeIndex + 1) % commands.length;
      return;
    }

    if (event.key === "ArrowUp") {
      event.preventDefault();
      if (commands.length === 0) return;
      activeIndex = (activeIndex - 1 + commands.length) % commands.length;
      return;
    }

    if (event.key === "Enter") {
      event.preventDefault();
      const command = commands[activeIndex];
      if (command) void runCommand(command);
      return;
    }

    if (event.key === "Escape") {
      event.preventDefault();
      close();
    }
  }

  const grouped = $derived.by(() => {
    const map = new Map<string, PaletteCommand[]>();
    for (const command of commands) {
      const group = map.get(command.group) ?? [];
      group.push(command);
      map.set(command.group, group);
    }
    return [...map.entries()];
  });
</script>

{#if commandPalette.open}
  <button
    type="button"
    class="command-palette__backdrop"
    aria-label="Close command palette"
    onclick={close}
  ></button>
  <div
    class="command-palette"
    role="dialog"
    aria-label="Command palette"
    tabindex="-1"
    onkeydown={onKeydown}
  >
    <div class="command-palette__input-wrap">
      <MdiIcon name="search" size={18} />
      <input
        bind:this={inputEl}
        class="command-palette__input"
        type="search"
        placeholder="Search pages, actions, music..."
        bind:value={query}
        aria-controls="command-palette-results"
        aria-activedescendant={commands[activeIndex]
          ? `palette-${commands[activeIndex].id}`
          : undefined}
        autocomplete="off"
        spellcheck="false"
      />
      <kbd class="command-palette__hint">Esc</kbd>
    </div>

    <div
      id="command-palette-results"
      class="command-palette__results"
      role="listbox"
      aria-label="Commands"
    >
      {#if loading && commands.length === 0}
        <p class="command-palette__empty">Searching...</p>
      {:else if commands.length === 0}
        <p class="command-palette__empty">No matching commands.</p>
      {:else}
        {#each grouped as [group, items] (group)}
          <section class="command-palette__group">
            <h3 class="command-palette__group-title">{group}</h3>
            <ul class="command-palette__list">
              {#each items as command (command.id)}
                {@const index = commands.indexOf(command)}
                <li>
                  <button
                    id="palette-{command.id}"
                    type="button"
                    class="command-palette__item"
                    class:command-palette__item--active={index === activeIndex}
                    role="option"
                    aria-selected={index === activeIndex}
                    onclick={() => void runCommand(command)}
                    onmouseenter={() => {
                      activeIndex = index;
                    }}
                  >
                    {#if command.icon}
                      <MdiIcon name={command.icon} size={18} />
                    {/if}
                    <span class="command-palette__label">{command.label}</span>
                  </button>
                </li>
              {/each}
            </ul>
          </section>
        {/each}
      {/if}
    </div>
  </div>
{/if}

<style>
  .command-palette__backdrop {
    position: fixed;
    inset: 0;
    border: none;
    background: rgb(0 0 0 / 0.45);
    z-index: 125;
    cursor: default;
  }

  .command-palette {
    position: fixed;
    left: 50%;
    top: min(18vh, 8rem);
    transform: translateX(-50%);
    z-index: 126;
    width: min(36rem, calc(100vw - 2rem));
    border-radius: var(--jb-radius-xl);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
    box-shadow: var(--jb-shadow-lg);
    overflow: hidden;
  }

  .command-palette__input-wrap {
    display: grid;
    grid-template-columns: auto 1fr auto;
    align-items: center;
    gap: var(--jb-space-3);
    padding: var(--jb-space-4);
    border-bottom: 1px solid var(--jb-border);
    color: var(--jb-text-muted);
  }

  .command-palette__input {
    width: 100%;
    border: none;
    background: transparent;
    color: var(--jb-text);
    font: inherit;
    font-size: 1rem;
    outline: none;
  }

  .command-palette__hint {
    padding: 0.15rem 0.4rem;
    border-radius: var(--jb-radius-sm);
    border: 1px solid var(--jb-border);
    background: var(--jb-bg-muted);
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--jb-text-subtle);
  }

  .command-palette__results {
    max-height: min(24rem, 50vh);
    overflow: auto;
    padding: var(--jb-space-2);
  }

  .command-palette__empty {
    margin: 0;
    padding: var(--jb-space-4);
    text-align: center;
    color: var(--jb-text-muted);
    font-size: 0.875rem;
  }

  .command-palette__group {
    padding: var(--jb-space-2) 0;
  }

  .command-palette__group-title {
    margin: 0 0 var(--jb-space-2);
    padding: 0 var(--jb-space-2);
    font-size: 0.75rem;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--jb-text-subtle);
  }

  .command-palette__list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 0.125rem;
  }

  .command-palette__item {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    width: 100%;
    padding: 0.65rem 0.75rem;
    border: none;
    border-radius: var(--jb-radius-lg);
    background: transparent;
    color: var(--jb-text);
    font: inherit;
    text-align: left;
    cursor: pointer;
  }

  .command-palette__item:hover,
  .command-palette__item--active {
    background: var(--jb-bg-muted);
  }

  .command-palette__label {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-weight: 600;
  }
</style>
