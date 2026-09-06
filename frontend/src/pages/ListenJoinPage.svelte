<script lang="ts">
  import AppShell from "$lib/components/layout/AppShell.svelte";
  import { APP_NAME } from "$lib/brand";
  import Button from "$lib/components/ui/Button.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import Link from "$lib/router/Link.svelte";
  import { auth } from "$lib/features/auth/store.svelte";
  import { deviceSync } from "$lib/music/device-sync.svelte";
  import { eventSocket } from "$lib/core/events/ws.svelte";
  import { router } from "$lib/router/router.svelte";

  interface Props {
    token: string;
  }

  let { token }: Props = $props();

  let joining = $state(false);
  let error = $state<string | null>(null);
  let joined = $state(false);

  async function join() {
    if (!token.trim()) {
      error = "Missing invite token";
      return;
    }
    joining = true;
    error = null;
    try {
      if (!eventSocket.connected) {
        eventSocket.connect();
        await new Promise((r) => setTimeout(r, 400));
      }
      deviceSync.start();
      const ok = await deviceSync.joinByInviteToken(token.trim());
      if (!ok) {
        error = deviceSync.lastError ?? "Could not join party";
        return;
      }
      joined = true;
      deviceSync.panelOpen = true;
      router.navigate("/music/now-playing");
    } catch (err) {
      error = err instanceof Error ? err.message : "Could not join party";
    } finally {
      joining = false;
    }
  }

  $effect(() => {
    void token;
    if (!auth.enabled || !auth.authenticated) return;
    if (joining || joined) return;
    void join();
  });
</script>

<AppShell>
  <div class="listen-join">
    {#if !auth.enabled || !auth.authenticated}
      <EmptyState
        title="Sign in to join"
        message={`Listen together parties need a ${APP_NAME} account.`}
      >
        {#snippet actions()}
          <Link href="/account/login" class="listen-join__link">Sign in</Link>
        {/snippet}
      </EmptyState>
    {:else if joining && !error}
      <div class="listen-join__loading">
        <Spinner />
        <p>Joining party…</p>
      </div>
    {:else if error}
      <EmptyState title="Could not join" message={error}>
        {#snippet actions()}
          <Button onclick={() => void join()} disabled={joining}>
            Try again
          </Button>
          <Link href="/music" class="listen-join__link">Home</Link>
        {/snippet}
      </EmptyState>
    {:else}
      <div class="listen-join__loading">
        <Spinner />
        <p>Opening player…</p>
      </div>
    {/if}
  </div>
</AppShell>

<style>
  .listen-join {
    max-width: var(--jb-content-narrow);
    min-height: 16rem;
    display: grid;
    place-content: center;
  }

  .listen-join__loading {
    display: grid;
    justify-items: center;
    gap: var(--jb-space-3);
    color: var(--jb-text-muted);
  }

  :global(.listen-join__link) {
    color: var(--jb-accent);
    font-weight: 600;
  }
</style>
