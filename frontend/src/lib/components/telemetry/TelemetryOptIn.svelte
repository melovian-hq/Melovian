<script lang="ts">
  import { Dialog } from "bits-ui";
  import Button from "$lib/components/ui/Button.svelte";
  import { APP_NAME } from "$lib/brand";
  import { telemetryConsent } from "$lib/core/telemetry-consent.svelte";

  const saving = $derived(telemetryConsent.saving);

  // A sanitized sample of what a crash report looks like. Values match the
  // scrubbing applied in telemetry-scrub.ts and internal/observability.
  const exampleReport = `{
  "event": "error",
  "message": "Cannot read properties of undefined",
  "release": "${APP_NAME.toLowerCase()}@1.4.2",
  "request": {
    "url": "https://telemetry.local/rest/stream?id=song-9&u=[redacted]&t=[redacted]"
  },
  "tags": { "source": "player" }
}`;

  function getOpen() {
    return telemetryConsent.promptOpen;
  }

  function setOpen(next: boolean) {
    // Closing without an answer stays opted out
    if (!next && telemetryConsent.promptOpen) {
      void telemetryConsent.answer("declined");
    }
  }
</script>

<Dialog.Root bind:open={getOpen, setOpen}>
  <Dialog.Portal>
    <Dialog.Overlay class="telemetry__backdrop">
      {#snippet child({ props })}
        <div {...props}></div>
      {/snippet}
    </Dialog.Overlay>
    <Dialog.Content class="telemetry">
      {#snippet child({ props })}
        <div {...props}>
          <Dialog.Title class="telemetry__title">
            {#snippet child({ props: titleProps })}
              <h2 {...titleProps}>Help improve {APP_NAME}</h2>
            {/snippet}
          </Dialog.Title>
          <Dialog.Description class="telemetry__body">
            {#snippet child({ props: descriptionProps })}
              <div {...descriptionProps}>
                <p>
                  {APP_NAME} can send anonymous crash and error reports to the developers
                  so bugs get fixed faster. Reports are off unless you turn them on
                  here.
                </p>
                <p>
                  Reports contain the app version, the error itself, and a short
                  trail of what the app was doing. Server addresses, usernames,
                  passwords, tokens, and track details are removed before
                  anything leaves this device. Nothing is collected about what
                  you listen to.
                </p>
                <p class="telemetry__example-label">Example report</p>
                <pre class="telemetry__example">{exampleReport}</pre>
                <p>
                  You can change this any time under Settings, Error tracking.
                </p>
              </div>
            {/snippet}
          </Dialog.Description>
          <div class="telemetry__actions">
            <Button
              variant="surface"
              disabled={saving}
              onclick={() => void telemetryConsent.answer("declined")}
            >
              No thanks
            </Button>
            <Button
              variant="primary"
              disabled={saving}
              onclick={() => void telemetryConsent.answer("accepted")}
            >
              Enable error reports
            </Button>
          </div>
        </div>
      {/snippet}
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>

<style>
  .telemetry__backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--jb-z-dialog);
    border: none;
    background: var(--jb-scrim);
  }

  .telemetry {
    position: fixed;
    top: 50%;
    left: 50%;
    z-index: calc(var(--jb-z-dialog) + 1);
    transform: translate(-50%, -50%);
    width: min(32rem, calc(100vw - 2rem));
    max-height: calc(100vh - 4rem);
    overflow-y: auto;
    padding: var(--jb-space-5);
    border-radius: var(--jb-radius-lg);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
    box-shadow: var(--jb-shadow-lg);
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-4);
  }

  .telemetry__title {
    margin: 0;
    font-size: 1.125rem;
  }

  .telemetry__body {
    color: var(--jb-text-muted);
    line-height: 1.6;
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-3);
  }

  .telemetry__body p {
    margin: 0;
  }

  .telemetry__example-label {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-text);
  }

  .telemetry__example {
    margin: 0;
    padding: var(--jb-space-3);
    border-radius: var(--jb-radius-md);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface-sunken, var(--jb-bg));
    font-size: 0.75rem;
    line-height: 1.5;
    overflow-x: auto;
    white-space: pre;
  }

  .telemetry__actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--jb-space-2);
  }
</style>
