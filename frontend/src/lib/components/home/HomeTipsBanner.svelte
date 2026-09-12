<script lang="ts">
  import { PersistedState } from "runed";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { layout } from "$lib/components/layout/layout.svelte";
  import { keyboardHelp } from "$lib/ui/keyboard-help.svelte";
  import { commandPalette } from "$lib/ui/command-palette.svelte";
  import { StorageKeys } from "$lib/brand";

  const STORAGE_KEY = StorageKeys.homeTipsDismissed;

  // Keep the "1" storage format so existing dismissals still apply
  const dismissed = new PersistedState<boolean>(STORAGE_KEY, false, {
    serializer: {
      serialize: (value) => (value ? "1" : "0"),
      deserialize: (value) => value === "1",
    },
  });

  function dismiss() {
    dismissed.current = true;
  }

  function openPalette() {
    keyboardHelp.close();
    commandPalette.openPalette();
  }
</script>

{#if !dismissed.current && !layout.isMobileViewport}
  <div class="home-tips" role="note">
    <MdiIcon name="lightbulb" size={18} />
    <p>
      Press <kbd>Ctrl/Cmd+K</kbd> for quick navigation or <kbd>?</kbd> for keyboard
      shortcuts.
    </p>
    <button type="button" class="home-tips__action" onclick={openPalette}>
      Open palette
    </button>
    <button
      type="button"
      class="home-tips__dismiss"
      aria-label="Dismiss tips"
      onclick={dismiss}
    >
      <MdiIcon name="x" size={16} />
    </button>
  </div>
{/if}

<style>
  .home-tips {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    flex-wrap: wrap;
    padding: var(--jb-space-3) var(--jb-space-4);
    border-radius: var(--jb-radius-md);
    border: 1px solid var(--jb-border);
    background: var(--jb-bg-muted);
    color: var(--jb-text-muted);
    font-size: 0.875rem;
  }

  .home-tips p {
    margin: 0;
    flex: 1;
    min-width: 12rem;
  }

  .home-tips kbd {
    padding: 0.125rem 0.375rem;
    border-radius: var(--jb-radius-sm);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
    font-family: inherit;
    font-size: 0.8125rem;
    color: var(--jb-text);
  }

  .home-tips__action {
    padding: 0.375rem 0.75rem;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-surface);
    color: var(--jb-text);
    font-size: 0.8125rem;
    font-weight: 600;
    cursor: pointer;
  }

  .home-tips__action:hover {
    background: var(--jb-surface-hover);
  }

  .home-tips__dismiss {
    display: grid;
    place-content: center;
    width: 1.75rem;
    height: 1.75rem;
    border: none;
    border-radius: var(--jb-radius-md);
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
  }

  .home-tips__dismiss:hover {
    background: var(--jb-surface-hover);
    color: var(--jb-text);
  }
</style>
