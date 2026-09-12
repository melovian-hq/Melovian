<script lang="ts">
  import { Dialog, Select } from "bits-ui";
  import Button from "$lib/components/ui/Button.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import SmartPlaylistGroupEditor from "./SmartPlaylistGroupEditor.svelte";
  import { SORT_ITEMS } from "$lib/music/smart-playlist/editor";
  import { SmartPlaylistEditor } from "$lib/music/smart-playlist/editor.svelte";
  import type { SmartPlaylistDraft } from "$lib/music/smart-playlist/types";

  interface Props {
    open?: boolean;
    onclose?: () => void;
    oncreate?: (draft: SmartPlaylistDraft) => Promise<void>;
  }

  let { open = false, onclose, oncreate }: Props = $props();

  const editor = new SmartPlaylistEditor({
    onclose: () => onclose?.(),
    oncreate: async (draft) => {
      await oncreate?.(draft);
    },
  });
</script>

<Dialog.Root
  {open}
  onOpenChange={(next) => {
    if (!next) editor.close();
  }}
>
  <Dialog.Portal>
    <Dialog.Overlay class="smart-creator__backdrop" />
    <Dialog.Content class="smart-creator">
      <header class="smart-creator__header">
        <div>
          <Dialog.Title id="smart-creator-title">
            {#snippet child({ props })}
              <h2 {...props}>Smart playlist</h2>
            {/snippet}
          </Dialog.Title>
          <Dialog.Description>
            {#snippet child({ props })}
              <p {...props}>
                Build dynamic playlists with AND/OR rules from your library.
              </p>
            {/snippet}
          </Dialog.Description>
        </div>
        <Dialog.Close
          type="button"
          class="smart-creator__close"
          aria-label="Close"
        >
          <MdiIcon name="x" size={18} />
        </Dialog.Close>
      </header>

      <div class="smart-creator__body">
        <div class="smart-creator__grid">
          <label class="smart-field">
            <span>Name</span>
            <input
              bind:value={editor.draft.name}
              placeholder="Evening jazz"
              autocomplete="off"
            />
            {#if editor.errorByPath.name}
              <span class="smart-field__error">{editor.errorByPath.name}</span>
            {/if}
          </label>

          <label class="smart-field">
            <span>Sort</span>
            <Select.Root
              type="single"
              items={SORT_ITEMS}
              bind:value={editor.draft.sort}
            >
              <Select.Trigger
                class="smart-field__select smart-select-trigger"
                aria-label="Sort"
              >
                <Select.Value />
                <MdiIcon name="chevronDown" size={14} />
              </Select.Trigger>
              <Select.Portal>
                <Select.Content
                  class="smart-creator__select-content"
                  sideOffset={4}
                >
                  <Select.Viewport>
                    {#each SORT_ITEMS as option (option.value)}
                      <Select.Item
                        class="smart-creator__select-item"
                        value={option.value}
                        label={option.label}
                      >
                        {#snippet children({ selected })}
                          {option.label}
                          {#if selected}
                            <MdiIcon name="check" size={14} />
                          {/if}
                        {/snippet}
                      </Select.Item>
                    {/each}
                  </Select.Viewport>
                </Select.Content>
              </Select.Portal>
            </Select.Root>
          </label>

          <label class="smart-field">
            <span>Track limit</span>
            <input
              type="number"
              min="1"
              value={editor.draft.limit ?? ""}
              oninput={(event) => {
                const raw = (event.currentTarget as HTMLInputElement).value;
                editor.draft.limit = raw === "" ? null : Number(raw);
                if (editor.draft.limit !== null)
                  editor.draft.limitPercent = null;
              }}
            />
            {#if editor.errorByPath.limit}
              <span class="smart-field__error">{editor.errorByPath.limit}</span>
            {/if}
          </label>

          <label class="smart-field smart-field--checkbox">
            <input type="checkbox" bind:checked={editor.draft.public} />
            <span>Share publicly on server</span>
          </label>
        </div>

        <label class="smart-field">
          <span>Comment</span>
          <input
            bind:value={editor.draft.comment}
            placeholder="Optional description"
            autocomplete="off"
          />
        </label>

        <div class="smart-creator__rules">
          <h3>Rules</h3>
          <SmartPlaylistGroupEditor
            draft={editor.draft}
            group={editor.draft.root}
            parentId={null}
            depth={0}
            errorByPath={editor.errorByPath}
            onchange={(next) => (editor.draft = next)}
          />
        </div>

        {#if editor.submitError}
          <p class="smart-creator__submit-error">{editor.submitError}</p>
        {/if}
      </div>

      <footer class="smart-creator__footer">
        <Button
          variant="surface"
          onclick={() => editor.close()}
          disabled={editor.submitting}
        >
          Cancel
        </Button>
        <Button
          onclick={() => void editor.submit()}
          disabled={editor.submitting}
        >
          {#if editor.submitting}
            <Spinner />
            Creating...
          {:else}
            <MdiIcon name="plus" size={16} />
            Create smart playlist
          {/if}
        </Button>
      </footer>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>

<style>
  :global(.smart-creator__backdrop) {
    position: fixed;
    inset: 0;
    border: none;
    background: rgb(0 0 0 / 0.5);
    z-index: 80;
    cursor: pointer;
  }

  :global(.smart-creator) {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    z-index: 81;
    width: min(44rem, calc(100vw - 1.5rem));
    max-height: min(44rem, 92vh);
    display: flex;
    flex-direction: column;
    border-radius: var(--jb-radius-xl);
    background: var(--jb-bg-elevated);
    border: 1px solid var(--jb-border);
    box-shadow: var(--jb-shadow-lg);
    overflow: hidden;
  }

  .smart-creator__header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--jb-space-4);
    padding: var(--jb-space-5);
    border-bottom: 1px solid var(--jb-border);
  }

  .smart-creator__header h2 {
    margin: 0 0 0.25rem;
    font-size: 1.25rem;
  }

  .smart-creator__header p {
    margin: 0;
    color: var(--jb-text-muted);
    font-size: 0.875rem;
  }

  :global(.smart-creator__close) {
    border: none;
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
    padding: 0.35rem;
    border-radius: var(--jb-radius-md);
  }

  :global(.smart-creator__close:hover) {
    color: var(--jb-text);
    background: var(--jb-surface-hover);
  }

  .smart-creator__body {
    padding: var(--jb-space-5);
    overflow: auto;
    display: grid;
    gap: var(--jb-space-4);
  }

  .smart-creator__grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--jb-space-3);
  }

  .smart-field {
    display: grid;
    gap: 0.35rem;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .smart-field input,
  :global(.smart-field__select) {
    width: 100%;
    padding: 0.55rem 0.7rem;
    border-radius: var(--jb-radius-md);
    border: 1px solid var(--jb-border);
    background: var(--jb-bg);
    color: var(--jb-text);
    font: inherit;
  }

  .smart-field--checkbox {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    align-self: end;
    color: var(--jb-text);
  }

  .smart-field--checkbox input {
    width: auto;
  }

  .smart-field__error,
  .smart-creator__submit-error {
    margin: 0;
    color: var(--jb-danger);
    font-size: 0.75rem;
  }

  .smart-creator__rules h3 {
    margin: 0 0 var(--jb-space-3);
    font-size: 0.9375rem;
  }

  :global(.smart-select-trigger) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.35rem;
    text-align: left;
    cursor: pointer;
  }

  :global(.smart-select-trigger svg) {
    flex-shrink: 0;
    color: var(--jb-text-subtle);
  }

  :global(.smart-creator__select-content) {
    z-index: 100;
    min-width: var(--bits-select-anchor-width);
    padding: var(--jb-space-1);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-bg-elevated);
    box-shadow: var(--jb-shadow-lg);
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }

  :global(.smart-creator__select-item) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-2);
    width: 100%;
    padding: 0.45rem 0.55rem;
    border-radius: var(--jb-radius-sm);
    color: var(--jb-text);
    font-size: 0.8125rem;
    cursor: pointer;
    user-select: none;
  }

  :global(.smart-creator__select-item[data-highlighted]) {
    background: var(--jb-surface-hover);
  }

  :global(.smart-creator__select-item[data-disabled]) {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .smart-creator__footer {
    display: flex;
    justify-content: flex-end;
    gap: var(--jb-space-2);
    padding: var(--jb-space-4) var(--jb-space-5);
    border-top: 1px solid var(--jb-border);
    background: color-mix(in srgb, var(--jb-bg-muted) 35%, transparent);
  }

  @media (max-width: 720px) {
    .smart-creator__grid {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 480px) {
    :global(.smart-creator) {
      width: calc(100vw - 0.75rem);
      max-height: 94dvh;
      border-radius: var(--jb-radius-lg);
    }

    .smart-creator__header,
    .smart-creator__body,
    .smart-creator__footer {
      padding-inline: var(--jb-space-4);
    }

    .smart-creator__footer {
      flex-wrap: wrap;
    }

    .smart-creator__footer :global(.button) {
      flex: 1;
      min-width: 8rem;
    }
  }
</style>
