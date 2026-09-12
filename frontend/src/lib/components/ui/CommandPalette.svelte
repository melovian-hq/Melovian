<script lang="ts">
  import { Command, Dialog } from "bits-ui";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { commandPalette } from "$lib/ui/command-palette.svelte";
  import {
    searchPaletteWithMusic,
    type PaletteCommand,
  } from "$lib/music/command-palette";

  let query = $state("");
  let commands = $state<PaletteCommand[]>([]);
  let selected = $state("");
  let loading = $state(false);
  let searchToken = 0;

  $effect(() => {
    if (commandPalette.open) return;
    query = "";
    commands = [];
    selected = "";
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

<Dialog.Root bind:open={commandPalette.open}>
  <Dialog.Portal>
    <Dialog.Overlay class="command-palette__backdrop">
      {#snippet child({ props })}
        <button
          {...props}
          type="button"
          aria-label="Close command palette"
          onclick={close}
        ></button>
      {/snippet}
    </Dialog.Overlay>
    <Dialog.Content class="command-palette" aria-label="Command palette">
      {#snippet child({ props })}
        <div {...props}>
          <Command.Root
            shouldFilter={false}
            loop
            vimBindings={false}
            bind:value={selected}
            label="Command palette"
          >
            <div class="command-palette__input-wrap">
              <MdiIcon name="search" size={18} />
              <Command.Input
                class="command-palette__input"
                type="search"
                placeholder="Search pages, actions, music..."
                bind:value={query}
                autofocus
                autocomplete="off"
                spellcheck="false"
              />
              <kbd class="command-palette__hint">Esc</kbd>
            </div>

            <Command.List
              id="command-palette-results"
              class="command-palette__results"
              aria-label="Commands"
            >
              <Command.Viewport>
                {#if loading && commands.length === 0}
                  <p class="command-palette__empty">Searching...</p>
                {:else}
                  <Command.Empty class="command-palette__empty">
                    {#snippet child({ props })}
                      <p {...props}>No matching commands.</p>
                    {/snippet}
                  </Command.Empty>
                {/if}
                {#each grouped as [group, items] (group)}
                  <Command.Group value={group} class="command-palette__group">
                    {#snippet child({ props: groupProps })}
                      <section {...groupProps}>
                        <Command.GroupHeading
                          class="command-palette__group-title"
                        >
                          {#snippet child({ props: headingProps })}
                            <h3 {...headingProps}>{group}</h3>
                          {/snippet}
                        </Command.GroupHeading>
                        <Command.GroupItems class="command-palette__list">
                          {#snippet child({ props: listProps })}
                            <ul {...listProps}>
                              {#each items as command (command.id)}
                                <li>
                                  <Command.Item
                                    value={command.id}
                                    onSelect={() => void runCommand(command)}
                                  >
                                    {#snippet child({ props: itemProps })}
                                      <button
                                        {...itemProps}
                                        type="button"
                                        class="command-palette__item"
                                        class:command-palette__item--active={selected ===
                                          command.id}
                                      >
                                        {#if command.icon}
                                          <MdiIcon
                                            name={command.icon}
                                            size={18}
                                          />
                                        {/if}
                                        <span class="command-palette__label"
                                          >{command.label}</span
                                        >
                                      </button>
                                    {/snippet}
                                  </Command.Item>
                                </li>
                              {/each}
                            </ul>
                          {/snippet}
                        </Command.GroupItems>
                      </section>
                    {/snippet}
                  </Command.Group>
                {/each}
              </Command.Viewport>
            </Command.List>
          </Command.Root>
        </div>
      {/snippet}
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>

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
