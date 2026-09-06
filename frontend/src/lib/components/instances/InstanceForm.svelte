<script lang="ts">
  import Button from "$lib/components/ui/Button.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import Input from "$lib/components/ui/Input.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import {
    displayNameFromServerUrl,
    normalizeServerUrl,
  } from "$lib/features/instances/normalize-url";
  import type { InstanceInput } from "$lib/features/instances/types";

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

  const actionCount = $derived((ontest ? 1 : 0) + (oncancel ? 1 : 0) + 1);

  $effect(() => {
    name = initial.name ?? "";
    serverUrl = initial.serverUrl ?? "";
    username = initial.username ?? "";
    password = initial.password ?? "";
    nameTouched = Boolean(initial.name?.trim());
  });

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
  class="instance-form"
  class:instance-form--sticky={stickyActions}
  class:instance-form--stack-actions={actionCount >= 3}
  onsubmit={(event) => {
    event.preventDefault();
    void onsubmit?.(payload());
  }}
>
  <div class="instance-form__fields">
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
        autocomplete="username"
        enterkeyhint="next"
        autocapitalize="off"
        spellcheck={false}
        required
      />
    </Field>

    <Field label="Password">
      <div class="instance-form__password">
        <Input
          bind:value={password}
          type={showPassword ? "text" : "password"}
          autocomplete="current-password"
          enterkeyhint="go"
          required={!initial.serverUrl}
          class="instance-form__password-input"
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
