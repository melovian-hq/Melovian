<script lang="ts">
  import { Dialog, Select } from "bits-ui";
  import Button from "$lib/components/ui/Button.svelte";
  import Input from "$lib/components/ui/Input.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import { auth } from "$lib/features/auth/store.svelte";
  import { getActiveInstanceId } from "$lib/features/instances/context";
  import * as musicApi from "$lib/music/api";
  import type { MusicShare, ShareAccessMode } from "$lib/music/api";
  import { toast } from "$lib/ui/toast.svelte";

  interface Props {
    open?: boolean;
    playlistId: string;
    playlistName: string;
    resourceType?: "playlist";
    kind: "local" | "server";
    onclose?: () => void;
  }

  let {
    open = false,
    playlistId,
    playlistName,
    resourceType = "playlist",
    kind,
    onclose,
  }: Props = $props();

  const EXPIRES_OPTIONS: {
    value: "none" | "1d" | "7d" | "30d";
    label: string;
  }[] = [
    { value: "none", label: "None" },
    { value: "1d", label: "1 day" },
    { value: "7d", label: "7 days" },
    { value: "30d", label: "30 days" },
  ];

  let mode = $state<ShareAccessMode>("public");
  let password = $state("");
  let usernameDraft = $state("");
  let usernames = $state<string[]>([]);
  let expiresChoice = $state<"none" | "1d" | "7d" | "30d">("none");
  let creating = $state(false);
  let loadingShares = $state(false);
  let existingShares = $state<MusicShare[]>([]);
  let createdShare = $state<MusicShare | null>(null);
  let loadToken = 0;

  const expiresInSec = $derived(
    expiresChoice === "1d"
      ? 86400
      : expiresChoice === "7d"
        ? 604800
        : expiresChoice === "30d"
          ? 2592000
          : undefined,
  );

  const restrictedDisabled = $derived(!auth.enabled);

  function resetForm() {
    mode = "public";
    password = "";
    usernameDraft = "";
    usernames = [];
    expiresChoice = "none";
    createdShare = null;
  }

  async function loadExistingShares() {
    const token = ++loadToken;
    loadingShares = true;
    try {
      const items = await musicApi.listShares();
      if (token !== loadToken) return;
      existingShares = items.filter(
        (share) =>
          share.resourceType === resourceType &&
          share.resourceId === playlistId,
      );
    } catch (err) {
      if (token !== loadToken) return;
      existingShares = [];
      toast.error(err instanceof Error ? err.message : "Failed to load shares");
    } finally {
      if (token === loadToken) loadingShares = false;
    }
  }

  $effect(() => {
    if (!open) return;
    void playlistId;
    void resourceType;
    resetForm();
    void loadExistingShares();
  });

  function addUsername() {
    const name = usernameDraft.trim();
    if (!name) return;
    if (usernames.some((u) => u.toLowerCase() === name.toLowerCase())) {
      usernameDraft = "";
      return;
    }
    usernames = [...usernames, name];
    usernameDraft = "";
  }

  function removeUsername(name: string) {
    usernames = usernames.filter((u) => u !== name);
  }

  async function createShare() {
    if (mode === "password" && !password.trim()) {
      toast.error("Enter a password");
      return;
    }
    if (mode === "restricted" && usernames.length === 0) {
      toast.error("Add at least one username");
      return;
    }
    if (mode === "restricted" && restrictedDisabled) {
      toast.error("Sign-in accounts required");
      return;
    }

    creating = true;
    try {
      const instanceId =
        kind === "server" ? (getActiveInstanceId() ?? undefined) : undefined;
      const share = await musicApi.createShare({
        resourceType,
        resourceId: playlistId,
        description: playlistName,
        accessMode: mode,
        password: mode === "password" ? password : undefined,
        usernames: mode === "restricted" ? usernames : undefined,
        expiresInSec,
        instanceId,
      });
      createdShare = share;
      toast.success("Share created");
      await loadExistingShares();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to create share",
      );
    } finally {
      creating = false;
    }
  }

  async function copyUrl(url: string) {
    try {
      await navigator.clipboard.writeText(url);
      toast.success("Link copied");
    } catch {
      toast.error("Could not copy link");
    }
  }

  async function revokeShare(id: string) {
    try {
      await musicApi.deleteShare(id);
      if (createdShare?.id === id) createdShare = null;
      toast.success("Share revoked");
      await loadExistingShares();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to revoke share",
      );
    }
  }
</script>

<Dialog.Root
  {open}
  onOpenChange={(next) => {
    if (!next) onclose?.();
  }}
>
  <Dialog.Portal>
    <Dialog.Overlay class="share-dialog__backdrop" />
    <Dialog.Content class="share-dialog">
      <header class="share-dialog__header">
        <div>
          <Dialog.Title id="share-dialog-title">
            {#snippet child({ props })}
              <h2 {...props}>Share playlist</h2>
            {/snippet}
          </Dialog.Title>
          <Dialog.Description>
            {#snippet child({ props })}
              <p {...props} class="share-dialog__subtitle" title={playlistName}>
                {playlistName}
              </p>
            {/snippet}
          </Dialog.Description>
        </div>
        <Dialog.Close
          type="button"
          class="share-dialog__close"
          aria-label="Close"
        >
          <MdiIcon name="x" size={18} />
        </Dialog.Close>
      </header>

      <div class="share-dialog__body">
        <fieldset class="share-dialog__modes">
          <legend>Access</legend>
          <label class="share-dialog__mode">
            <input type="radio" bind:group={mode} value="public" />
            <span>
              <MdiIcon name="link" size={16} />
              Public
            </span>
          </label>
          <label class="share-dialog__mode">
            <input type="radio" bind:group={mode} value="password" />
            <span>
              <MdiIcon name="lock" size={16} />
              Password
            </span>
          </label>
          <label
            class="share-dialog__mode"
            class:share-dialog__mode--disabled={restrictedDisabled}
            title={restrictedDisabled ? "Sign-in accounts required" : undefined}
          >
            <input
              type="radio"
              bind:group={mode}
              value="restricted"
              disabled={restrictedDisabled}
            />
            <span>
              <MdiIcon name="accountMultiple" size={16} />
              Restricted
            </span>
          </label>
          {#if restrictedDisabled}
            <p class="share-dialog__hint">Sign-in accounts required</p>
          {/if}
        </fieldset>

        {#if mode === "password"}
          <label class="share-dialog__field">
            <span>Password</span>
            <Input
              type="password"
              bind:value={password}
              autocomplete="new-password"
              placeholder="Share password"
            />
          </label>
        {/if}

        {#if mode === "restricted" && !restrictedDisabled}
          <div class="share-dialog__field">
            <span>Allowed usernames</span>
            <form
              class="share-dialog__username-row"
              onsubmit={(event) => {
                event.preventDefault();
                addUsername();
              }}
            >
              <Input
                bind:value={usernameDraft}
                placeholder="username"
                autocomplete="off"
              />
              <Button type="submit" variant="surface" size="sm">Add</Button>
            </form>
            {#if usernames.length > 0}
              <ul class="share-dialog__chips">
                {#each usernames as name (name)}
                  <li>
                    <span>{name}</span>
                    <button
                      type="button"
                      aria-label="Remove {name}"
                      onclick={() => removeUsername(name)}
                    >
                      <MdiIcon name="x" size={14} />
                    </button>
                  </li>
                {/each}
              </ul>
            {/if}
          </div>
        {/if}

        <label class="share-dialog__field">
          <span>Expires</span>
          <Select.Root
            type="single"
            items={EXPIRES_OPTIONS}
            value={expiresChoice}
            onValueChange={(value) =>
              (expiresChoice =
                value as (typeof EXPIRES_OPTIONS)[number]["value"])}
          >
            <Select.Trigger class="share-dialog__select" aria-label="Expires">
              <Select.Value />
              <MdiIcon name="chevronDown" size={16} />
            </Select.Trigger>
            <Select.Portal>
              <Select.Content
                class="share-dialog__select-content"
                sideOffset={4}
              >
                <Select.Viewport>
                  {#each EXPIRES_OPTIONS as option (option.value)}
                    <Select.Item
                      class="share-dialog__select-item"
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

        <div class="share-dialog__actions">
          <Button
            type="button"
            disabled={creating || (mode === "restricted" && restrictedDisabled)}
            onclick={() => void createShare()}
          >
            <MdiIcon name="share" size={16} />
            {creating ? "Creating…" : "Create share"}
          </Button>
        </div>

        {#if createdShare}
          <div class="share-dialog__created">
            <p class="share-dialog__created-label">Share link</p>
            <div class="share-dialog__url-row">
              <code class="share-dialog__url">{createdShare.url}</code>
              <Button
                type="button"
                variant="surface"
                size="sm"
                onclick={() => {
                  if (createdShare) void copyUrl(createdShare.url);
                }}
              >
                Copy
              </Button>
            </div>
          </div>
        {/if}

        <section class="share-dialog__existing">
          <h3>Existing shares</h3>
          {#if loadingShares}
            <div class="share-dialog__state"><Spinner /></div>
          {:else if existingShares.length === 0}
            <p class="share-dialog__empty">No shares for this playlist yet.</p>
          {:else}
            <ul class="share-dialog__list">
              {#each existingShares as share (share.id)}
                <li>
                  <div class="share-dialog__list-main">
                    <span class="share-dialog__list-mode"
                      >{share.accessMode}</span
                    >
                    <code class="share-dialog__url" title={share.url}
                      >{share.url}</code
                    >
                  </div>
                  <div class="share-dialog__list-actions">
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onclick={() => void copyUrl(share.url)}
                    >
                      Copy
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onclick={() => void revokeShare(share.id)}
                    >
                      <MdiIcon name="trash2" size={14} />
                      Revoke
                    </Button>
                  </div>
                </li>
              {/each}
            </ul>
          {/if}
        </section>
      </div>

      <footer class="share-dialog__footer">
        <Button type="button" variant="surface" onclick={() => onclose?.()}>
          Close
        </Button>
      </footer>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>

<style>
  :global(.share-dialog__backdrop) {
    position: fixed;
    inset: 0;
    z-index: 80;
    border: none;
    background: rgb(0 0 0 / 0.45);
    cursor: pointer;
  }

  :global(.share-dialog) {
    position: fixed;
    z-index: 81;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    display: flex;
    flex-direction: column;
    width: min(32rem, calc(100vw - 2rem));
    max-height: min(40rem, calc(100vh - 2rem));
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-xl);
    background: var(--jb-bg-elevated);
    box-shadow: var(--jb-shadow-lg);
    overflow: hidden;
  }

  .share-dialog__header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--jb-space-3);
    padding: var(--jb-space-4) var(--jb-space-5) var(--jb-space-3);
    border-bottom: 1px solid var(--jb-border);
  }

  .share-dialog__header h2 {
    margin: 0;
    font-size: 1.0625rem;
  }

  .share-dialog__subtitle {
    margin: var(--jb-space-2) 0 0;
    color: var(--jb-text-muted);
    font-size: 0.8125rem;
    word-break: break-word;
  }

  :global(.share-dialog__close) {
    display: grid;
    place-items: center;
    width: 2rem;
    height: 2rem;
    border: none;
    border-radius: var(--jb-radius-md);
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
  }

  :global(.share-dialog__close:hover) {
    background: var(--jb-surface-hover);
    color: var(--jb-text);
  }

  .share-dialog__body {
    flex: 1;
    overflow-y: auto;
    padding: var(--jb-space-4) var(--jb-space-5);
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-4);
  }

  .share-dialog__modes {
    margin: 0;
    padding: 0;
    border: none;
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
  }

  .share-dialog__modes legend {
    margin-bottom: var(--jb-space-2);
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-text-muted);
  }

  .share-dialog__mode {
    display: inline-flex;
    cursor: pointer;
  }

  .share-dialog__mode input {
    position: absolute;
    opacity: 0;
    pointer-events: none;
  }

  .share-dialog__mode span {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    padding: 0.4rem 0.75rem;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-surface);
    font-size: 0.875rem;
  }

  .share-dialog__mode input:checked + span {
    border-color: var(--jb-accent);
    background: color-mix(in srgb, var(--jb-accent) 12%, transparent);
  }

  .share-dialog__mode--disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }

  .share-dialog__hint {
    flex-basis: 100%;
    margin: 0;
    font-size: 0.75rem;
    color: var(--jb-text-muted);
  }

  .share-dialog__field {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
    font-size: 0.875rem;
  }

  .share-dialog__field > span {
    font-weight: 600;
    color: var(--jb-text-muted);
    font-size: 0.8125rem;
  }

  .share-dialog__username-row {
    display: flex;
    gap: var(--jb-space-2);
  }

  .share-dialog__chips {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
  }

  .share-dialog__chips li {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-1);
    padding: 0.25rem 0.5rem;
    border-radius: var(--jb-radius-md);
    background: var(--jb-surface);
    border: 1px solid var(--jb-border);
    font-size: 0.8125rem;
  }

  .share-dialog__chips button {
    display: grid;
    place-items: center;
    border: none;
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
    padding: 0;
  }

  :global(.share-dialog__select) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-2);
    width: 100%;
    padding: var(--jb-space-3);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-surface);
    color: var(--jb-text);
    text-align: left;
    cursor: pointer;
  }

  :global(.share-dialog__select-content) {
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

  :global(.share-dialog__select-item) {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-2);
    width: 100%;
    padding: 0.55rem 0.75rem;
    border-radius: var(--jb-radius-sm);
    color: var(--jb-text);
    font-size: 0.875rem;
    cursor: pointer;
    user-select: none;
  }

  :global(.share-dialog__select-item[data-highlighted]) {
    background: var(--jb-surface-hover);
  }

  :global(.share-dialog__select-item[data-disabled]) {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .share-dialog__actions {
    display: flex;
  }

  .share-dialog__created {
    padding: var(--jb-space-3);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-surface);
  }

  .share-dialog__created-label {
    margin: 0 0 var(--jb-space-2);
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-text-muted);
  }

  .share-dialog__url-row {
    display: flex;
    gap: var(--jb-space-2);
    align-items: center;
  }

  .share-dialog__url {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 0.75rem;
    font-family: var(--jb-font-mono, ui-monospace, monospace);
  }

  .share-dialog__existing h3 {
    margin: 0 0 var(--jb-space-2);
    font-size: 0.875rem;
  }

  .share-dialog__state,
  .share-dialog__empty {
    margin: 0;
    padding: var(--jb-space-3);
    color: var(--jb-text-muted);
    font-size: 0.875rem;
    text-align: center;
  }

  .share-dialog__list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
  }

  .share-dialog__list li {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-2);
    padding: var(--jb-space-3);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-surface);
  }

  .share-dialog__list-main {
    display: flex;
    flex-direction: column;
    gap: var(--jb-space-1);
    min-width: 0;
  }

  .share-dialog__list-mode {
    font-size: 0.75rem;
    text-transform: capitalize;
    color: var(--jb-text-muted);
  }

  .share-dialog__list-actions {
    display: flex;
    gap: var(--jb-space-1);
    justify-content: flex-end;
  }

  .share-dialog__footer {
    display: flex;
    justify-content: flex-end;
    gap: var(--jb-space-2);
    padding: var(--jb-space-3) var(--jb-space-5) var(--jb-space-4);
    border-top: 1px solid var(--jb-border);
  }
</style>
