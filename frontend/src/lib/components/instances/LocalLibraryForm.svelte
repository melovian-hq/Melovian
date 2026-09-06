<script lang="ts">
  import Button from "$lib/components/ui/Button.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import Input from "$lib/components/ui/Input.svelte";
  import FolderBrowserDialog from "$lib/components/ui/FolderBrowserDialog.svelte";
  import {
    folderDialogAvailable,
    folderNameFromPath,
    pickFolder,
  } from "$lib/desktop/folder-dialog";
  import { localLibraries } from "$lib/features/local-libraries/store.svelte";
  import type { LocalLibraryInput } from "$lib/features/local-libraries/types";

  interface Props {
    submitLabel?: string;
    saving?: boolean;
    initial?: Partial<LocalLibraryInput>;
    onsubmit?: (input: LocalLibraryInput) => void | Promise<void>;
  }

  let {
    submitLabel = "Add local library",
    saving = false,
    initial = {},
    onsubmit,
  }: Props = $props();

  let name = $state("");
  let path = $state("");
  let picking = $state(false);
  let browserOpen = $state(false);

  const canBrowseNative = $derived(
    localLibraries.config.allowCustomPath && folderDialogAvailable(),
  );
  const canBrowseServer = $derived(
    localLibraries.config.allowCustomPath && !folderDialogAvailable(),
  );
  const canBrowse = $derived(canBrowseNative || canBrowseServer);

  $effect(() => {
    name = initial.name ?? "";
    path = initial.path ?? "";
  });

  function payload(): LocalLibraryInput {
    const input: LocalLibraryInput = { name: name.trim() };
    if (localLibraries.config.allowCustomPath) {
      input.path = path.trim();
    }
    return input;
  }

  async function browseFolder() {
    if (!canBrowse || picking) return;
    if (canBrowseServer) {
      browserOpen = true;
      return;
    }
    picking = true;
    try {
      const selected = await pickFolder({
        title: "Select music folder",
        directory:
          path.trim() || localLibraries.config.defaultPath || undefined,
      });
      if (!selected) return;
      applySelectedPath(selected);
    } finally {
      picking = false;
    }
  }

  function applySelectedPath(selected: string) {
    path = selected;
    if (!name.trim()) {
      name = folderNameFromPath(selected);
    }
  }
</script>

<form
  class="local-library-form"
  onsubmit={(event) => {
    event.preventDefault();
    void onsubmit?.(payload());
  }}
>
  <Field label="Display name">
    <Input bind:value={name} placeholder="My music folder" autocomplete="off" />
  </Field>

  {#if localLibraries.config.allowCustomPath}
    <Field
      label="Folder path"
      hint={canBrowse
        ? "Browse for a folder, or type an absolute path."
        : "Absolute path to a directory of audio files on this machine."}
    >
      <div class="local-library-form__path-row">
        <Input
          bind:value={path}
          placeholder="/home/user/Music"
          autocomplete="off"
        />
        {#if canBrowse}
          <Button
            type="button"
            variant="surface"
            disabled={picking || saving}
            onclick={() => void browseFolder()}
          >
            {picking ? "Opening..." : "Browse"}
          </Button>
        {/if}
      </div>
    </Field>
  {:else if localLibraries.config.defaultPath}
    <p class="local-library-form__hint">
      Files are loaded from the server default path:
      <code>{localLibraries.config.defaultPath}</code>
    </p>
  {/if}

  <div class="local-library-form__actions">
    <Button type="submit" disabled={saving || picking}>
      {saving ? "Saving..." : submitLabel}
    </Button>
  </div>
</form>

{#if browserOpen}
  <FolderBrowserDialog
    open={browserOpen}
    initialPath={path.trim() || localLibraries.config.defaultPath || ""}
    onselect={applySelectedPath}
    onclose={() => (browserOpen = false)}
  />
{/if}

<style>
  .local-library-form {
    display: grid;
    gap: var(--jb-space-4);
  }

  .local-library-form__path-row {
    display: flex;
    align-items: stretch;
    gap: var(--jb-space-2);
  }

  .local-library-form__path-row :global(.input) {
    flex: 1;
    min-width: 0;
  }

  .local-library-form__actions {
    display: flex;
    gap: var(--jb-space-3);
    flex-wrap: wrap;
  }

  .local-library-form__hint {
    margin: 0;
    color: var(--jb-text-muted);
    line-height: 1.6;
  }

  .local-library-form__hint code {
    display: inline-block;
    margin-top: var(--jb-space-2);
    padding: 0.125rem 0.375rem;
    border-radius: var(--jb-radius-sm);
    background: var(--jb-bg-muted);
    word-break: break-all;
  }

  @media (max-width: 640px) {
    .local-library-form__path-row {
      flex-direction: column;
    }
  }
</style>
