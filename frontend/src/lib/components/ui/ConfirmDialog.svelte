<script lang="ts">
  import { AlertDialog } from "bits-ui";
  import Button from "$lib/components/ui/Button.svelte";
  import { confirmDialog } from "$lib/ui/confirm.svelte";

  const options = $derived(confirmDialog.options);

  let contentEl = $state<HTMLElement | null>(null);

  function getOpen() {
    return confirmDialog.open;
  }

  function setOpen(open: boolean) {
    // Dismissal through Escape or an outside click resolves as a cancel.
    if (!open) confirmDialog.cancel();
  }

  function onOpenAutoFocus(event: Event) {
    event.preventDefault();
    // Prefer Cancel so Enter does not immediately confirm a destructive action.
    requestAnimationFrame(() => {
      contentEl
        ?.querySelector<HTMLElement>(".confirm__actions button")
        ?.focus();
    });
  }
</script>

<AlertDialog.Root bind:open={getOpen, setOpen}>
  <AlertDialog.Portal>
    <AlertDialog.Overlay class="confirm__backdrop">
      {#snippet child({ props })}
        <div {...props}></div>
      {/snippet}
    </AlertDialog.Overlay>
    <AlertDialog.Content
      class="confirm"
      interactOutsideBehavior="close"
      {onOpenAutoFocus}
      bind:ref={contentEl}
    >
      {#snippet child({ props })}
        <div {...props}>
          {#if options}
            <AlertDialog.Title class="confirm__title">
              {#snippet child({ props: titleProps })}
                <h2 {...titleProps}>{options.title}</h2>
              {/snippet}
            </AlertDialog.Title>
            <AlertDialog.Description class="confirm__message">
              {#snippet child({ props: descriptionProps })}
                <p {...descriptionProps}>{options.message}</p>
              {/snippet}
            </AlertDialog.Description>
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
          {/if}
        </div>
      {/snippet}
    </AlertDialog.Content>
  </AlertDialog.Portal>
</AlertDialog.Root>

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
