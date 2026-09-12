<script lang="ts">
  import { Dialog } from "bits-ui";
  import Button from "$lib/components/ui/Button.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import { homeHidden } from "$lib/music/home-hidden.svelte";

  interface Props {
    open?: boolean;
    onclose?: () => void;
  }

  let { open = false, onclose }: Props = $props();

  const hiddenCount = $derived(
    homeHidden.albums.size +
      homeHidden.mixes.size +
      homeHidden.playlists.size +
      homeHidden.artists.size,
  );

  function close() {
    onclose?.();
  }

  function resetHidden() {
    homeHidden.clearAll();
  }

  function getOpen() {
    return open;
  }

  function setOpen(next: boolean) {
    if (!next) close();
  }
</script>

<Dialog.Root bind:open={getOpen, setOpen}>
  <Dialog.Portal>
    <Dialog.Overlay class="home-customize__backdrop">
      {#snippet child({ props })}
        <div {...props}></div>
      {/snippet}
    </Dialog.Overlay>
    <Dialog.Content class="home-customize">
      {#snippet child({ props })}
        <div {...props}>
          <header class="home-customize__header">
            <Dialog.Title>
              {#snippet child({ props: titleProps })}
                <h2 {...titleProps}>Customize home</h2>
              {/snippet}
            </Dialog.Title>
            <Dialog.Close class="home-customize__close">
              {#snippet child({ props: closeProps })}
                <button {...closeProps} type="button" aria-label="Close">
                  <MdiIcon name="x" size={18} />
                </button>
              {/snippet}
            </Dialog.Close>
          </header>

          <div class="home-customize__body">
            <Dialog.Description>
              {#snippet child({ props: descriptionProps })}
                <p {...descriptionProps}>
                  Right-click albums, mixes, playlists, and artists on home to
                  hide them from your feed.
                </p>
              {/snippet}
            </Dialog.Description>

            <dl class="home-customize__stats">
              <div>
                <dt>Hidden albums</dt>
                <dd>{homeHidden.albums.size}</dd>
              </div>
              <div>
                <dt>Hidden mixes</dt>
                <dd>{homeHidden.mixes.size}</dd>
              </div>
              <div>
                <dt>Hidden playlists</dt>
                <dd>{homeHidden.playlists.size}</dd>
              </div>
              <div>
                <dt>Hidden artists</dt>
                <dd>{homeHidden.artists.size}</dd>
              </div>
            </dl>

            {#if hiddenCount === 0}
              <p class="home-customize__empty">Nothing hidden yet.</p>
            {/if}
          </div>

          <footer class="home-customize__footer">
            <Button
              variant="surface"
              disabled={hiddenCount === 0}
              onclick={resetHidden}
            >
              Restore hidden items
            </Button>
            <Button onclick={close}>Done</Button>
          </footer>
        </div>
      {/snippet}
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>

<style>
  .home-customize__backdrop {
    position: fixed;
    inset: 0;
    z-index: 110;
    border: none;
    background: rgba(0, 0, 0, 0.45);
  }

  .home-customize {
    position: fixed;
    top: 50%;
    left: 50%;
    z-index: 111;
    transform: translate(-50%, -50%);
    width: min(28rem, calc(100vw - 2rem));
    border-radius: var(--jb-radius-lg);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
    box-shadow: var(--jb-shadow-lg);
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-4);
    padding: var(--jb-space-5);
  }

  .home-customize__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-3);
  }

  .home-customize__header h2 {
    margin: 0;
    font-size: 1.125rem;
  }

  .home-customize__close {
    display: grid;
    place-content: center;
    width: 2rem;
    height: 2rem;
    border: none;
    border-radius: var(--jb-radius-md);
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
  }

  .home-customize__close:hover {
    background: var(--jb-surface-hover);
    color: var(--jb-text);
  }

  .home-customize__body {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-4);
    color: var(--jb-text-muted);
    line-height: 1.6;
    font-size: 0.9375rem;
  }

  .home-customize__body p {
    margin: 0;
  }

  .home-customize__stats {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--jb-space-3);
    margin: 0;
  }

  .home-customize__stats div {
    padding: var(--jb-space-3);
    border-radius: var(--jb-radius-md);
    background: var(--jb-bg-muted);
  }

  .home-customize__stats dt {
    font-size: 0.75rem;
    color: var(--jb-text-muted);
  }

  .home-customize__stats dd {
    margin: 0.25rem 0 0;
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--jb-text);
  }

  .home-customize__empty {
    font-size: 0.875rem;
  }

  .home-customize__footer {
    display: flex;
    justify-content: flex-end;
    gap: var(--jb-space-2);
  }
</style>
