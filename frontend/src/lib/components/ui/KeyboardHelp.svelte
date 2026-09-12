<script lang="ts">
  import { Dialog } from "bits-ui";
  import { keyboardHelp } from "$lib/ui/keyboard-help.svelte";

  const shortcuts = [
    { keys: "Ctrl/Cmd + K", action: "Open command palette" },
    { keys: "Q", action: "Toggle queue" },
    { keys: "Space", action: "Play / pause" },
    { keys: "Shift + ← / →", action: "Previous / next track" },
    { keys: "← / →", action: "Seek 5 seconds back / forward" },
    { keys: "↑ / ↓", action: "Volume up / down" },
    { keys: "T", action: "TV mode on now playing" },
    { keys: "?", action: "Show this help" },
    { keys: "Esc", action: "Close panels and help" },
    { keys: "Right-click track", action: "Track actions menu" },
  ];
</script>

<Dialog.Root bind:open={keyboardHelp.open}>
  <Dialog.Portal>
    <Dialog.Overlay class="keyboard-help__backdrop">
      {#snippet child({ props })}
        <div {...props}></div>
      {/snippet}
    </Dialog.Overlay>
    <Dialog.Content class="keyboard-help">
      {#snippet child({ props })}
        <div {...props}>
          <header class="keyboard-help__header">
            <Dialog.Title>
              {#snippet child({ props: titleProps })}
                <h2 {...titleProps}>Keyboard shortcuts</h2>
              {/snippet}
            </Dialog.Title>
            <Dialog.Close class="keyboard-help__close">
              {#snippet child({ props: closeProps })}
                <button {...closeProps} type="button">Close</button>
              {/snippet}
            </Dialog.Close>
          </header>
          <dl class="keyboard-help__list">
            {#each shortcuts as item (item.keys)}
              <div class="keyboard-help__row">
                <dt><kbd>{item.keys}</kbd></dt>
                <dd>{item.action}</dd>
              </div>
            {/each}
          </dl>
        </div>
      {/snippet}
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>

<style>
  .keyboard-help__backdrop {
    position: fixed;
    inset: 0;
    border: none;
    background: rgb(0 0 0 / 0.45);
    z-index: 120;
    cursor: default;
  }

  .keyboard-help {
    position: fixed;
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%);
    z-index: 121;
    width: min(24rem, calc(100vw - 2rem));
    padding: var(--jb-space-5);
    border-radius: var(--jb-radius-xl);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
    box-shadow: var(--jb-shadow-lg);
  }

  .keyboard-help__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-3);
    margin-bottom: var(--jb-space-4);
  }

  .keyboard-help__header h2 {
    margin: 0;
    font-size: 1.125rem;
    font-weight: 700;
  }

  .keyboard-help__close {
    border: none;
    background: transparent;
    color: var(--jb-accent);
    font-weight: 600;
    cursor: pointer;
  }

  .keyboard-help__list {
    margin: 0;
    display: grid;
    gap: var(--jb-space-3);
  }

  .keyboard-help__row {
    display: grid;
    grid-template-columns: 7rem 1fr;
    gap: var(--jb-space-3);
    align-items: center;
  }

  .keyboard-help__row dt {
    margin: 0;
  }

  .keyboard-help__row dd {
    margin: 0;
    color: var(--jb-text-muted);
  }

  kbd {
    display: inline-block;
    padding: 0.2rem 0.45rem;
    border-radius: var(--jb-radius-sm);
    border: 1px solid var(--jb-border);
    background: var(--jb-bg-muted);
    font-family: inherit;
    font-size: 0.8125rem;
    font-weight: 600;
  }
</style>
