<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import * as accountApi from "$lib/features/auth/account";
  import { auth } from "$lib/features/auth/store.svelte";
  import { music } from "$lib/config/music.svelte";
  import { instances } from "$lib/features/instances/store.svelte";
  import { router } from "$lib/router/router.svelte";
  import { toast } from "$lib/ui/toast.svelte";

  let sessions = $state<accountApi.AuthSession[]>([]);
  let loadingSessions = $state(false);
  let currentPassword = $state("");
  let newPassword = $state("");
  let confirmPassword = $state("");
  let deletePassword = $state("");
  let savingPassword = $state(false);
  let deletingAccount = $state(false);
  let confirmDelete = $state(false);

  async function loadSessions() {
    loadingSessions = true;
    try {
      sessions = await accountApi.listSessions();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to load sessions",
      );
    } finally {
      loadingSessions = false;
    }
  }

  $effect(() => {
    if (auth.enabled && auth.authenticated) {
      void loadSessions();
    }
  });

  async function revokeSession(id: string) {
    try {
      await accountApi.revokeSession(id);
      toast.success("Session revoked");
      await loadSessions();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to revoke session",
      );
    }
  }

  async function handleChangePassword() {
    if (newPassword !== confirmPassword) {
      toast.error("New passwords do not match");
      return;
    }
    savingPassword = true;
    try {
      await accountApi.changePassword(currentPassword, newPassword);
      currentPassword = "";
      newPassword = "";
      confirmPassword = "";
      toast.success("Password updated");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to change password",
      );
    } finally {
      savingPassword = false;
    }
  }

  async function handleDeleteAccount() {
    if (!confirmDelete) {
      confirmDelete = true;
      return;
    }
    deletingAccount = true;
    try {
      await accountApi.deleteAccount(deletePassword);
      auth.authenticated = false;
      auth.user = null;
      music.disconnect();
      await instances.init();
      router.navigate("/account/login", true);
      toast.success("Account deleted");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to delete account",
      );
    } finally {
      deletingAccount = false;
    }
  }
</script>

<SettingsCard title="Sessions" description="Active sign-ins for this account.">
  {#if loadingSessions}
    <div class="account-sessions__loading">
      <Spinner />
    </div>
  {:else if sessions.length === 0}
    <p>No active sessions.</p>
  {:else}
    <ul class="account-sessions">
      {#each sessions as session (session.id)}
        <li class="account-sessions__item">
          <div>
            <strong>{session.current ? "This device" : "Session"}</strong>
            <span>Expires {new Date(session.expiresAt).toLocaleString()}</span>
          </div>
          {#if !session.current}
            <Button
              variant="ghost"
              onclick={() => void revokeSession(session.id)}
            >
              Revoke
            </Button>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}
</SettingsCard>

<SettingsCard title="Password" description="Change your account password.">
  <form
    class="account-form"
    onsubmit={(event) => {
      event.preventDefault();
      void handleChangePassword();
    }}
  >
    <Field label="Current password">
      <input
        type="password"
        bind:value={currentPassword}
        autocomplete="current-password"
      />
    </Field>
    <Field label="New password">
      <input
        type="password"
        bind:value={newPassword}
        autocomplete="new-password"
      />
    </Field>
    <Field label="Confirm new password">
      <input
        type="password"
        bind:value={confirmPassword}
        autocomplete="new-password"
      />
    </Field>
    <Button type="submit" disabled={savingPassword}>Update password</Button>
  </form>
</SettingsCard>

<SettingsCard
  title="Delete account"
  description="Permanently remove your account and sessions."
>
  <form
    class="account-form"
    onsubmit={(event) => {
      event.preventDefault();
      void handleDeleteAccount();
    }}
  >
    <Field label="Confirm password">
      <input
        type="password"
        bind:value={deletePassword}
        autocomplete="current-password"
      />
    </Field>
    <Button type="submit" variant="ghost" disabled={deletingAccount}>
      {confirmDelete ? "Confirm delete account" : "Delete account"}
    </Button>
  </form>
</SettingsCard>

<style>
  .account-sessions {
    display: grid;
    gap: var(--jb-space-2);
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .account-sessions__loading {
    display: grid;
    place-content: center;
    min-height: 4rem;
  }

  .account-sessions__item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-3);
    padding: var(--jb-space-3);
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
  }

  .account-sessions__item div {
    display: grid;
    gap: 0.125rem;
  }

  .account-sessions__item span {
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .account-form {
    display: grid;
    gap: var(--jb-space-3);
    max-width: 24rem;
  }

  .account-form input {
    width: 100%;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--jb-border);
    border-radius: var(--jb-radius-md);
    background: var(--jb-surface);
    color: var(--jb-text);
  }
</style>
