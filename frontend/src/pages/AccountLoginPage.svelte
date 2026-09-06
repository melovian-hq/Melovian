<script lang="ts">
  import PageHeader from "$lib/components/ui/PageHeader.svelte";
  import { APP_NAME } from "$lib/brand";
  import AppLogo from "$lib/components/ui/AppLogo.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import { auth } from "$lib/features/auth/store.svelte";
  import { instances } from "$lib/features/instances/store.svelte";
  import { localLibraries } from "$lib/features/local-libraries/store.svelte";
  import { sources } from "$lib/features/sources/store.svelte";
  import { loadExtensions } from "$lib/extensions/registry";
  import { router } from "$lib/router/router.svelte";
  import { toast } from "$lib/ui/toast.svelte";

  let username = $state("");
  let password = $state("");
  let confirmPassword = $state("");
  let submitting = $state(false);

  const isSetup = $derived(auth.setupRequired);
  const showLocalForm = $derived(auth.showLocalAccountForm);
  const showOIDC = $derived(auth.oidcEnabled);

  function startOIDCLogin() {
    void import("$lib/config/remote-server").then(({ resolveApiUrl }) => {
      window.location.href = resolveApiUrl(auth.oidcLoginUrl);
    });
  }

  async function submit(event: Event) {
    event.preventDefault();
    if (!username.trim() || !password) return;
    if (isSetup && password !== confirmPassword) {
      toast.error("Passwords do not match");
      return;
    }
    submitting = true;
    try {
      if (isSetup) {
        await auth.setup(username.trim(), password);
        toast.success("Account created");
      } else {
        await auth.login(username.trim(), password);
        toast.success("Signed in");
      }
      await Promise.all([instances.init(), localLibraries.init()]);
      await sources.refreshStatus();
      await loadExtensions();
      router.navigate(sources.needsSetup ? "/setup" : "/music", true);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Sign in failed");
    } finally {
      submitting = false;
    }
  }
</script>

<div class="account-login">
  <div class="account-login__panel">
    <div class="account-login__brand">
      <AppLogo size={56} />
    </div>
    <PageHeader
      title={isSetup ? "Create your account" : `Sign in to ${APP_NAME}`}
      subtitle={showOIDC && !showLocalForm
        ? "Sign in with your identity provider to access your saved servers and listening data."
        : isSetup
          ? "Set up the first account for this server. Each user gets their own saved servers, playlists, and stats."
          : "Sign in to access your saved servers and listening data."}
    />

    {#if auth.loading}
      <div class="account-login__loading"><Spinner /></div>
    {:else}
      {#if showOIDC}
        <div class="account-login__oidc">
          <Button type="button" onclick={startOIDCLogin} disabled={submitting}>
            Sign in with SSO
          </Button>
        </div>
      {/if}

      {#if showLocalForm}
        <form class="account-login__form" onsubmit={submit}>
          <label>
            <span>Username</span>
            <input bind:value={username} autocomplete="username" required />
          </label>
          <label>
            <span>Password</span>
            <input
              bind:value={password}
              type="password"
              autocomplete={isSetup ? "new-password" : "current-password"}
              minlength="8"
              required
            />
          </label>
          {#if isSetup}
            <label>
              <span>Confirm password</span>
              <input
                bind:value={confirmPassword}
                type="password"
                autocomplete="new-password"
                minlength="8"
                required
              />
            </label>
          {/if}
          <Button type="submit" disabled={submitting}>
            {isSetup ? "Create account" : "Sign in"}
          </Button>
        </form>
      {:else if !showOIDC}
        <p class="account-login__hint">
          Sign in is not configured for this server.
        </p>
      {/if}
    {/if}
  </div>
</div>

<style>
  .account-login {
    min-height: 100vh;
    display: grid;
    place-content: center;
    padding: var(--jb-space-8);
    background:
      radial-gradient(
        circle at top,
        color-mix(in srgb, var(--jb-accent) 12%, transparent),
        transparent 42%
      ),
      var(--jb-bg);
  }

  .account-login__panel {
    width: min(100%, 28rem);
    padding: var(--jb-space-8);
    border-radius: var(--jb-radius-xl);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
    box-shadow: var(--jb-shadow-lg);
  }

  .account-login__brand {
    display: flex;
    justify-content: center;
    margin-bottom: var(--jb-space-5);
  }

  .account-login__loading {
    display: grid;
    place-content: center;
    min-height: 8rem;
  }

  .account-login__oidc {
    display: grid;
    gap: var(--jb-space-4);
    margin-bottom: var(--jb-space-4);
  }

  .account-login__hint {
    margin: 0;
    color: var(--jb-text-muted);
  }

  .account-login__form {
    display: grid;
    gap: var(--jb-space-4);
  }

  .account-login__form label {
    display: grid;
    gap: var(--jb-space-2);
    font-weight: 600;
  }

  .account-login__form input {
    padding: 0.625rem 0.875rem;
    border-radius: var(--jb-radius-md);
    border: 1px solid var(--jb-border);
    background: var(--jb-bg);
    color: var(--jb-text);
    font: inherit;
  }

  @media (max-width: 480px) {
    .account-login {
      padding: var(--jb-space-4);
      align-content: start;
      padding-top: max(var(--jb-space-8), env(safe-area-inset-top));
    }

    .account-login__panel {
      padding: var(--jb-space-5);
    }
  }
</style>
