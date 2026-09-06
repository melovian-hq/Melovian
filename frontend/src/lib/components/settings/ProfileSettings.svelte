<script lang="ts">
  import Button from "$lib/components/ui/Button.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import UserAvatar from "$lib/components/ui/UserAvatar.svelte";
  import { music } from "$lib/config/music.svelte";
  import { APP_NAME } from "$lib/brand";
  import { auth } from "$lib/features/auth/store.svelte";
  import { instances } from "$lib/features/instances/store.svelte";
  import {
    ACCEPTED_AVATAR_EXTENSIONS,
    defaultAvatarSeed,
  } from "$lib/profile/profile-settings";
  import { profile } from "$lib/profile/profile.svelte";
  import { toast } from "$lib/ui/toast.svelte";
  import "$lib/settings/settings-page.css";

  let fileInput = $state<HTMLInputElement | undefined>();
  let uploading = $state(false);
  let usernameDraft = $state("");
  let savingUsername = $state(false);

  const canEditUsername = $derived(
    auth.enabled && auth.authenticated && Boolean(auth.user),
  );

  const seed = $derived(defaultAvatarSeed(instances.active, music.serverName));
  const displayName = $derived(
    auth.user?.username ??
      instances.active?.username ??
      instances.active?.serverName ??
      music.serverName ??
      APP_NAME,
  );

  $effect(() => {
    if (auth.user?.username) {
      usernameDraft = auth.user.username;
    }
  });

  async function handleFileChange(event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    input.value = "";
    if (!file) return;

    uploading = true;
    try {
      await profile.setCustomAvatarFromFile(file);
      toast.success("Avatar updated");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to upload avatar",
      );
    } finally {
      uploading = false;
    }
  }

  function removeCustomAvatar() {
    profile.clearCustomAvatar();
    toast.success("Using generated avatar");
  }

  async function saveUsername() {
    const next = usernameDraft.trim();
    if (!next || !canEditUsername) return;
    if (next === auth.user?.username) {
      toast.info("Username unchanged");
      return;
    }
    savingUsername = true;
    try {
      await auth.updateUsername(next);
      toast.success("Username updated");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Could not change username",
      );
    } finally {
      savingUsername = false;
    }
  }
</script>

<div class="profile-settings">
  <div class="profile-settings__avatar-row">
    <UserAvatar {seed} size={96} />

    <div class="profile-settings__copy">
      <p class="profile-settings__name">{displayName}</p>
      <p class="profile-settings__hint">
        {#if profile.customAvatarUrl}
          Using a custom avatar on this device.
        {:else}
          Double-click the generated face for a new look.
        {/if}
      </p>

      <div class="profile-settings__actions">
        <input
          bind:this={fileInput}
          type="file"
          accept="image/jpeg,image/png,image/webp,{ACCEPTED_AVATAR_EXTENSIONS}"
          class="profile-settings__file-input"
          onchange={(event) => void handleFileChange(event)}
        />
        <Button disabled={uploading} onclick={() => fileInput?.click()}>
          {uploading ? "Uploading…" : "Upload avatar"}
        </Button>
        {#if profile.customAvatarUrl}
          <Button variant="ghost" onclick={removeCustomAvatar}>
            Remove custom avatar
          </Button>
        {/if}
      </div>
      <p class="profile-settings__formats">JPG, PNG, or WebP up to 512 KB.</p>
    </div>
  </div>

  {#if canEditUsername}
    <Field
      label="Username"
      hint="Letters, numbers, _, -, and . (2-64 characters)."
    >
      <div class="profile-settings__username-row">
        <input
          class="settings-input"
          bind:value={usernameDraft}
          autocomplete="username"
          maxlength={64}
          disabled={savingUsername}
        />
        <Button
          disabled={savingUsername || !usernameDraft.trim()}
          onclick={() => void saveUsername()}
        >
          {savingUsername ? "Saving…" : "Save"}
        </Button>
      </div>
    </Field>
  {/if}
</div>

<style>
  .profile-settings {
    display: grid;
    gap: var(--jb-space-4);
  }

  .profile-settings__avatar-row {
    display: flex;
    align-items: flex-start;
    gap: var(--jb-space-5);
  }

  .profile-settings__copy {
    display: grid;
    gap: var(--jb-space-3);
    min-width: 0;
  }

  .profile-settings__name {
    margin: 0;
    font-size: 1.125rem;
    font-weight: 700;
    color: var(--jb-text);
  }

  .profile-settings__hint,
  .profile-settings__formats {
    margin: 0;
    font-size: 0.875rem;
    line-height: 1.55;
    color: var(--jb-text-muted);
  }

  .profile-settings__actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
  }

  .profile-settings__file-input {
    display: none;
  }

  .profile-settings__username-row {
    display: flex;
    flex-wrap: wrap;
    gap: var(--jb-space-2);
    align-items: center;
  }

  .profile-settings__username-row .settings-input {
    flex: 1;
    min-width: 12rem;
  }

  @media (max-width: 640px) {
    .profile-settings__avatar-row {
      flex-direction: column;
    }
  }
</style>
