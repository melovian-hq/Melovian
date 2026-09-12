<script lang="ts">
  import { Dialog } from "bits-ui";
  import Button from "$lib/components/ui/Button.svelte";
  import { APP_NAME } from "$lib/brand";
  import Toggle from "$lib/components/ui/Toggle.svelte";
  import { closePrompt } from "$lib/desktop/close-prompt.svelte";
  import {
    confirmCloseChoice,
    executeCloseAction,
  } from "$lib/desktop/window-close";
  import {
    loadDesktopIntegrationSettings,
    type ResolvedCloseAction,
  } from "$lib/desktop/desktop-integration-settings";

  let rememberChoice = $state(false);

  const settings = $derived(loadDesktopIntegrationSettings());
  const backgroundAction = $derived<ResolvedCloseAction>(
    settings.taskbarEnabled ? "background" : "minimize",
  );
  const backgroundLabel = $derived(
    settings.taskbarEnabled
      ? "Keep running in background"
      : "Minimize to taskbar",
  );
  const backgroundDescription = $derived(
    settings.taskbarEnabled
      ? `Hide the window and keep ${APP_NAME} in the system tray.`
      : `Minimize ${APP_NAME} to the taskbar and keep playback running.`,
  );

  function cancel() {
    rememberChoice = false;
    closePrompt.hide();
  }

  async function choose(action: ResolvedCloseAction) {
    if (rememberChoice) {
      await confirmCloseChoice({ action, remember: true });
      rememberChoice = false;
      return;
    }
    closePrompt.hide();
    await executeCloseAction(action);
  }

  function getOpen() {
    return closePrompt.open;
  }

  function setOpen(open: boolean) {
    if (!open) cancel();
  }
</script>

<Dialog.Root bind:open={getOpen, setOpen}>
  <Dialog.Portal>
    <Dialog.Overlay class="close-prompt__backdrop">
      {#snippet child({ props })}
        <div {...props}></div>
      {/snippet}
    </Dialog.Overlay>
    <Dialog.Content class="close-prompt">
      {#snippet child({ props })}
        <div {...props}>
          <header class="close-prompt__header">
            <Dialog.Title>
              {#snippet child({ props: titleProps })}
                <h2 {...titleProps}>Close {APP_NAME}?</h2>
              {/snippet}
            </Dialog.Title>
          </header>
          <Dialog.Description class="close-prompt__description">
            {#snippet child({ props: descriptionProps })}
              <p {...descriptionProps}>
                Choose whether to quit the application or keep it running.
              </p>
            {/snippet}
          </Dialog.Description>
          <div class="close-prompt__actions">
            <button
              type="button"
              class="close-prompt__option"
              onclick={() => void choose("quit")}
            >
              <span class="close-prompt__option-title">Quit application</span>
              <span class="close-prompt__option-description">
                Stop playback and exit {APP_NAME} completely.
              </span>
            </button>
            <button
              type="button"
              class="close-prompt__option"
              onclick={() => void choose(backgroundAction)}
            >
              <span class="close-prompt__option-title">{backgroundLabel}</span>
              <span class="close-prompt__option-description">
                {backgroundDescription}
              </span>
            </button>
          </div>
          <div class="close-prompt__remember">
            <div class="close-prompt__remember-copy">
              <p class="close-prompt__remember-label">Remember my choice</p>
              <p class="close-prompt__remember-description">
                Use the same action next time without asking.
              </p>
            </div>
            <Toggle
              checked={rememberChoice}
              ariaLabel="Remember my choice"
              onchange={(checked) => {
                rememberChoice = checked;
              }}
            />
          </div>
          <div class="close-prompt__footer">
            <Button variant="ghost" onclick={cancel}>Cancel</Button>
          </div>
        </div>
      {/snippet}
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>

<style>
  .close-prompt__backdrop {
    position: fixed;
    inset: 0;
    border: none;
    background: rgb(0 0 0 / 0.45);
    z-index: 130;
    cursor: default;
  }

  .close-prompt {
    position: fixed;
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%);
    z-index: 131;
    width: min(28rem, calc(100vw - 2rem));
    padding: var(--jb-space-5);
    border-radius: var(--jb-radius-xl);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
    box-shadow: var(--jb-shadow-lg);
  }

  .close-prompt__header h2 {
    margin: 0;
    font-size: 1.125rem;
    color: var(--jb-text);
  }

  .close-prompt__description {
    margin: var(--jb-space-3) 0 var(--jb-space-4);
    font-size: 0.875rem;
    line-height: 1.5;
    color: var(--jb-text-muted);
  }

  .close-prompt__actions {
    display: grid;
    gap: var(--jb-space-3);
  }

  .close-prompt__option {
    display: grid;
    gap: 0.25rem;
    width: 100%;
    padding: var(--jb-space-4);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-lg);
    background: var(--jb-bg-muted);
    color: var(--jb-text);
    text-align: left;
    cursor: pointer;
    transition:
      border-color var(--jb-transition),
      background var(--jb-transition);
  }

  .close-prompt__option:hover {
    border-color: var(--jb-accent);
    background: var(--jb-surface);
  }

  .close-prompt__option-title {
    font-size: 0.9375rem;
    font-weight: 600;
  }

  .close-prompt__option-description {
    font-size: 0.8125rem;
    line-height: 1.5;
    color: var(--jb-text-muted);
  }

  .close-prompt__remember {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-4);
    margin-top: var(--jb-space-4);
    padding-top: var(--jb-space-4);
    border-top: 1px solid var(--jb-border);
  }

  .close-prompt__remember-copy {
    min-width: 0;
  }

  .close-prompt__remember-label {
    margin: 0;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--jb-text);
  }

  .close-prompt__remember-description {
    margin: 0.25rem 0 0;
    font-size: 0.8125rem;
    line-height: 1.4;
    color: var(--jb-text-muted);
  }

  .close-prompt__footer {
    display: flex;
    justify-content: flex-end;
    margin-top: var(--jb-space-4);
  }
</style>
