<script lang="ts">
  import Button from "$lib/components/ui/Button.svelte";
  import { confirmDialog } from "$lib/ui/confirm.svelte";
  import { focusInitial, trapFocus } from "$lib/ui/focus-trap";

  const options = $derived(confirmDialog.options);

  let dialogEl = $state<HTMLDivElement | null>(null);

  function onKeydown(event: KeyboardEvent) {
    if (!confirmDialog.open) return;
    if (event.key === "Escape") {
      event.preventDefault();
      confirmDialog.cancel();
    }
  }

  $effect(() => {
    if (!confirmDialog.open || !options || !dialogEl) {
      return;
    }

    const previouslyFocused = document.activeElement as HTMLElement | null;
    const release = trapFocus(dialogEl);
    // Prefer Cancel so Enter does not immediately confirm a destructive action.
    focusInitial(dialogEl, false);

    return () => {
      release();
      previouslyFocused?.focus();
    };
  });
</script>

<svelte:window onkeydown={onKeydown} />

{#if confirmDialog.open && options}
  <button
    type="button"
    class="confirm__backdrop"
    aria-label="Close dialog"
    tabindex="-1"
    onclick={() => confirmDialog.cancel()}
  ></button>
  <div
    bind:this={dialogEl}
    class="confirm"
    role="alertdialog"
    aria-modal="true"
    aria-labelledby="confirm-title"
    aria-describedby="confirm-message"
  >
    <h2 id="confirm-title" class="confirm__title">{options.title}</h2>
    <p id="confirm-message" class="confirm__message">{options.message}</p>
    <div class="confirm__actions">
      <Button variant="surface" onclick={() => confirmDialog.cancel()}>
        {options.cancelLabel ?? "Cancel"}
      </Button>
      <Button
        variant="primary"
        class={options.danger ? "confirm__danger" : ""}
        onclick={() => confirmDialog.accept()}
      >
        {options.confirmLabel ?? "Confirm"}
      </Button>
    </div>
  </div>
{/if}

<style>
  .confirm__backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--jb-z-dialog);
    border: none;
    background: rgba(0, 0, 0, 0.45);
    cursor: default;
  }

  .confirm {
    position: fixed;
    top: 50%;
    left: 50%;
    z-index: calc(var(--jb-z-dialog) + 1);
    transform: translate(-50%, -50%);
    width: min(24rem, calc(100vw - 2rem));
    padding: var(--jb-space-5);
    border-radius: var(--jb-radius-lg);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
    box-shadow: var(--jb-shadow-lg);
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-4);
  }

  .confirm__title {
    margin: 0;
    font-size: 1.125rem;
  }

  .confirm__message {
    margin: 0;
    color: var(--jb-text-muted);
    line-height: 1.6;
  }

  .confirm__actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--jb-space-2);
  }

  :global(.confirm__danger) {
    background: var(--jb-danger, #c62828);
  }

  :global(.confirm__danger:hover) {
    background: var(--jb-danger-hover, #b71c1c);
  }
</style>
