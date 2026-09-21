<script lang="ts">
  import { onMount } from "svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import Input from "$lib/components/ui/Input.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import { detectServers } from "$lib/features/instances/api";
  import {
    displayNameFromServerUrl,
    normalizeServerUrl,
  } from "$lib/features/instances/normalize-url";
  import type {
    DetectedServer,
    InstanceInput,
  } from "$lib/features/instances/types";

  interface Props {
    submitLabel?: string;
    testing?: boolean;
    saving?: boolean;
    initial?: Partial<InstanceInput>;
    stickyActions?: boolean;
    autofocusUrl?: boolean;
    onsubmit?: (input: InstanceInput) => void | Promise<void>;
    ontest?: (input: InstanceInput) => void | Promise<void>;
    oncancel?: () => void;
  }

  let {
    submitLabel = "Save server",
    testing = false,
    saving = false,
    initial = {},
    stickyActions = false,
    autofocusUrl = false,
    onsubmit,
    ontest,
    oncancel,
  }: Props = $props();

  let name = $state("");
  let serverUrl = $state("");
  let username = $state("");
  let password = $state("");
  let showPassword = $state(false);
  let nameTouched = $state(false);
  let formEl = $state<HTMLFormElement | null>(null);
  let detecting = $state(false);
  let detected = $state<DetectedServer[]>([]);

  const actionCount = $derived((ontest ? 1 : 0) + (oncancel ? 1 : 0) + 1);
  const isEdit = $derived(Boolean(initial.serverUrl));

  $effect(() => {
    name = initial.name ?? "";
    serverUrl = initial.serverUrl ?? "";
    username = initial.username ?? "";
    password = initial.password ?? "";
    nameTouched = Boolean(initial.name?.trim());
  });

  onMount(() => {
    if (isEdit) return;
    detecting = true;
    detectServers()
      .then((servers) => {
        detected = servers;
      })
      .catch(() => {
        detected = [];
      })
      .finally(() => {
        detecting = false;
      });
  });

  function pickDetected(server: DetectedServer) {
    serverUrl = server.url;
    if (!nameTouched && !name.trim()) {
      name = displayNameFromServerUrl(server.url);
    }
    formEl?.querySelector<HTMLInputElement>('input[name="username"]')?.focus();
  }

  function payload(): InstanceInput {
    const normalizedUrl = normalizeServerUrl(serverUrl);
    const trimmedName = name.trim();
    return {
      name: trimmedName || displayNameFromServerUrl(normalizedUrl) || "Server",
      serverUrl: normalizedUrl,
      username: username.trim(),
      password,
    };
  }

  function onUrlBlur() {
    const normalized = normalizeServerUrl(serverUrl);
    if (normalized && normalized !== serverUrl.trim()) {
      serverUrl = normalized;
    }
    if (!nameTouched && !name.trim() && normalized) {
      name = displayNameFromServerUrl(normalized);
    }
  }

  function onNameInput() {
    nameTouched = true;
  }
</script>

<form
  bind:this={formEl}
  class="instance-form"
  class:instance-form--sticky={stickyActions}
  class:instance-form--stack-actions={actionCount >= 3}
  onsubmit={(event) => {
    event.preventDefault();
    void onsubmit?.(payload());
  }}
>
  <div class="instance-form__fields">
    {#if !isEdit && (detecting || detected.length > 0)}
      <div class="instance-form__detect">
        <span class="instance-form__detect-title">
          Detected on this network
        </span>
        {#if detecting}
          <span class="instance-form__detect-status">
            <Spinner class="instance-form__detect-spinner" />
            Looking for servers...
          </span>
        {:else}
          <ul class="instance-form__detect-list">
            {#each detected as server (server.url)}
              <li>
                <button
                  type="button"
                  class="instance-form__detect-row"
                  onclick={() => pickDetected(server)}
                >
                  <MdiIcon name="server" size={18} />
                  <span class="instance-form__detect-name">
                    {server.serverName}{server.version
                      ? ` ${server.version}`
                      : ""}
                  </span>
                  <span class="instance-form__detect-url">{server.url}</span>
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      </div>
    {/if}

    <Field
      label="Server URL"
      hint="Navidrome or any Subsonic API base. https:// is added if you leave it off."
    >
      <Input
        bind:value={serverUrl}
        type="url"
        inputmode="url"
        enterkeyhint="next"
        autocapitalize="off"
        spellcheck={false}
        placeholder="music.example.com or http://192.168.1.10:4533"
        autocomplete="url"
        autofocus={autofocusUrl}
        required
        onblur={onUrlBlur}
      />
    </Field>

    <Field label="Display name" hint="Optional. Defaults to the server host.">
      <Input
        bind:value={name}
        placeholder="Home server"
        autocomplete="off"
        enterkeyhint="next"
        oninput={onNameInput}
      />
    </Field>

    <Field label="Username">
      <Input
        bind:value={username}
        name="username"
        autocomplete="username"
        enterkeyhint="next"
        autocapitalize="off"
        spellcheck={false}
        required
      />
    </Field>

    <Field label="Password" group>
      <div class="instance-form__password">
        <Input
          bind:value={password}
          type={showPassword ? "text" : "password"}
          autocomplete="current-password"
          enterkeyhint="go"
          required={!initial.serverUrl}
          class="instance-form__password-input"
          aria-label="Password"
        />
        <button
          type="button"
          class="instance-form__password-toggle"
          onclick={() => (showPassword = !showPassword)}
          aria-label={showPassword ? "Hide password" : "Show password"}
        >
          <MdiIcon name={showPassword ? "eyeOff" : "eye"} size={20} />
        </button>
      </div>
    </Field>
  </div>

  <div class="instance-form__actions">
    {#if ontest}
      <Button
        type="button"
        variant="ghost"
        size="md"
        disabled={testing || saving}
        onclick={() => void ontest?.(payload())}
      >
        {testing ? "Testing..." : "Test connection"}
      </Button>
    {/if}
    {#if oncancel}
      <Button type="button" variant="ghost" size="md" onclick={oncancel}>
        Cancel
      </Button>
    {/if}
    <Button type="submit" size="lg" disabled={saving || testing}>
      {saving ? "Saving..." : submitLabel}
    </Button>
  </div>
</form>

<style>
  .instance-form {
    display: grid;
    gap: var(--jb-space-5);
  }

  .instance-form__fields {
    display: grid;
    gap: var(--jb-space-4);
  }

  .instance-form__detect {
    display: grid;
    gap: var(--jb-space-2);
    padding: var(--jb-space-3);
    border: 1px dashed var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-bg-subtle);
  }

  .instance-form__detect-title {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-text-muted);
  }

  .instance-form__detect-status {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    color: var(--jb-text-subtle);
    font-size: 0.8125rem;
  }

  .instance-form__detect-status :global(.spinner__ring) {
    width: 0.875rem;
    height: 0.875rem;
  }

  .instance-form__detect-list {
    display: grid;
    gap: var(--jb-space-1);
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .instance-form__detect-row {
    display: grid;
    grid-template-columns: auto 1fr;
    align-items: center;
    column-gap: var(--jb-space-2);
    width: 100%;
    padding: var(--jb-space-2) var(--jb-space-3);
    border: none;
    border-radius: var(--jb-radius-md);
    background: transparent;
    color: var(--jb-text);
    font: inherit;
    text-align: left;
    cursor: pointer;
  }

  .instance-form__detect-row:hover {
    background: var(--jb-accent-muted);
    color: var(--jb-accent);
  }

  .instance-form__detect-row:focus-visible {
    background: var(--jb-accent-muted);
    color: var(--jb-accent);
    outline: none;
    box-shadow: var(--jb-focus-ring);
  }

  .instance-form__detect-name {
    min-width: 0;
    overflow: hidden;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .instance-form__detect-url {
    grid-column: 2;
    min-width: 0;
    overflow: hidden;
    color: var(--jb-text-subtle);
    font-size: 0.8125rem;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .instance-form__password {
    position: relative;
    display: grid;
  }

  :global(.instance-form__password-input) {
    padding-right: 2.75rem;
  }

  .instance-form__password-toggle {
    position: absolute;
    top: 50%;
    right: 0.35rem;
    transform: translateY(-50%);
    display: grid;
    place-content: center;
    width: 2.5rem;
    height: 2.5rem;
    border: none;
    border-radius: var(--jb-radius-md);
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
  }

  .instance-form__password-toggle:hover {
    color: var(--jb-text);
  }

  .instance-form__actions {
    display: flex;
    gap: var(--jb-space-3);
    flex-wrap: wrap;
    align-items: center;
  }

  .instance-form__actions :global(.btn--primary) {
    flex: 1 1 10rem;
  }

  .instance-form--sticky .instance-form__fields {
    padding-bottom: 0.5rem;
    scroll-margin-bottom: 5.5rem;
  }

  .instance-form--sticky .instance-form__fields :global(.field:last-child) {
    scroll-margin-bottom: 5.5rem;
  }

  .instance-form--sticky .instance-form__actions {
    position: sticky;
    bottom: 0;
    z-index: 2;
    padding-top: var(--jb-space-3);
    padding-bottom: var(--jb-space-3);
    border-top: 1px solid var(--jb-border);
    background: color-mix(in srgb, var(--jb-surface) 92%, transparent);
    backdrop-filter: blur(8px);
  }

  @media (max-width: 640px) {
    .instance-form__actions :global(.btn--primary) {
      flex: 1 1 auto;
    }

    .instance-form__actions :global(.btn--ghost) {
      flex: 0 1 auto;
    }

    .instance-form--stack-actions .instance-form__actions {
      flex-direction: column-reverse;
    }

    .instance-form--stack-actions .instance-form__actions :global(.btn) {
      width: 100%;
      flex: 0 0 auto;
    }

    .instance-form--stack-actions.instance-form--sticky .instance-form__fields {
      padding-bottom: 0.5rem;
      scroll-margin-bottom: 9rem;
    }

    .instance-form--stack-actions.instance-form--sticky
      .instance-form__fields
      :global(.field:last-child) {
      scroll-margin-bottom: 9rem;
    }
  }
</style>
