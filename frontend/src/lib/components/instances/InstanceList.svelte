<script lang="ts">
  import Button from "$lib/components/ui/Button.svelte";
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import InstanceForm from "./InstanceForm.svelte";
  import { instances } from "$lib/features/instances/store.svelte";
  import { pingInstance, type InstancePing } from "$lib/features/instances/api";
  import { music } from "$lib/config/music.svelte";
  import { resetSubsonicDetailCaches } from "$lib/subsonic/detail-cache";
  import { toast } from "$lib/ui/toast.svelte";
  import { confirmDialog } from "$lib/ui/confirm.svelte";
  import type {
    InstanceInput,
    SubsonicInstance,
  } from "$lib/features/instances/types";

  interface Props {
    class?: string;
  }

  let { class: className = "" }: Props = $props();

  let editingId = $state<string | null>(null);
  let editInitial = $state<Partial<InstanceInput>>({});
  let testing = $state(false);
  let saving = $state(false);
  let pings = $state<Record<string, InstancePing | "loading">>({});

  async function refreshPing(id: string) {
    pings = { ...pings, [id]: "loading" };
    try {
      const result = await pingInstance(id);
      pings = { ...pings, [id]: result };
    } catch {
      pings = {
        ...pings,
        [id]: { online: false, latencyMs: 0, error: "unreachable" },
      };
    }
  }

  $effect(() => {
    for (const item of instances.items) {
      if (!(item.id in pings)) void refreshPing(item.id);
    }
  });

  function startEdit(item: SubsonicInstance) {
    editingId = item.id;
    editInitial = {
      name: item.name,
      serverUrl: item.serverUrl,
      username: item.username,
    };
  }

  async function activate(item: SubsonicInstance) {
    if (item.id === instances.activeId) return;
    try {
      await instances.activate(item.id);
      resetSubsonicDetailCaches();
      await music.connect({ force: true });
      toast.success(`Switched to ${item.name}`);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to switch server",
      );
    }
  }

  async function remove(item: SubsonicInstance) {
    const ok = await confirmDialog.confirm({
      title: "Remove server",
      message: `Remove ${item.name}? You can add it again later from settings.`,
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok) return;
    try {
      await instances.remove(item.id);
      if (instances.activeId) {
        await music.connect({ force: true });
      }
      toast.success("Server removed");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to remove instance",
      );
    }
  }

  async function handleTest(input: InstanceInput) {
    testing = true;
    try {
      const serverName = await instances.test(input);
      toast.success(`Connected to ${serverName}`);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Connection test failed",
      );
    } finally {
      testing = false;
    }
  }

  async function handleUpdate(id: string, input: InstanceInput) {
    saving = true;
    try {
      await instances.update(id, input);
      if (instances.activeId === id) {
        await music.connect({ force: true });
      }
      toast.success("Server updated");
      editingId = null;
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to update instance",
      );
    } finally {
      saving = false;
    }
  }

  async function refreshLibrary() {
    if (music.libraryRefreshing) return;
    try {
      await music.refreshLibrary();
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to refresh library",
      );
    }
  }
</script>

<div class="instance-list {className}">
  {#each instances.items as item (item.id)}
    <article
      class="instance-list__item"
      class:instance-list__item--active={item.id === instances.activeId}
    >
      {#if editingId === item.id}
        <div class="instance-list__edit">
          <InstanceForm
            {testing}
            {saving}
            submitLabel="Save changes"
            initial={editInitial}
            ontest={handleTest}
            onsubmit={(input) => void handleUpdate(item.id, input)}
          />
          <Button size="sm" variant="ghost" onclick={() => (editingId = null)}>
            Cancel
          </Button>
        </div>
      {:else}
        {@const ping = pings[item.id]}
        <div class="instance-list__copy">
          <h3 class="instance-list__name">{item.name}</h3>
          <p class="instance-list__meta">{item.serverName || item.serverUrl}</p>
          <p class="instance-list__user">{item.username}</p>
          {#if ping !== "loading" && ping !== undefined && ping.online && ping.songCount != null}
            <p class="instance-list__tracks">
              {ping.songCount.toLocaleString()} tracks
            </p>
          {/if}
          <button
            type="button"
            class="instance-list__ping"
            onclick={() => void refreshPing(item.id)}
            title="Refresh ping"
          >
            {#if ping === "loading" || ping === undefined}
              <span class="ping-dot ping-dot--idle"></span>
              <span>Pinging...</span>
            {:else if ping.online}
              <span class="ping-dot ping-dot--online"></span>
              <span>{ping.latencyMs} ms</span>
            {:else}
              <span class="ping-dot ping-dot--offline"></span>
              <span>Offline</span>
            {/if}
          </button>
        </div>
        <div class="instance-list__actions">
          {#if item.id !== instances.activeId}
            <Button
              size="sm"
              variant="ghost"
              onclick={() => void activate(item)}
            >
              Switch
            </Button>
          {:else}
            <span class="instance-list__badge">Active</span>
          {/if}
          {#if item.id === instances.activeId}
            <Button
              size="sm"
              variant="ghost"
              disabled={music.libraryRefreshing}
              onclick={() => void refreshLibrary()}
              aria-label="Refresh library"
              title="Refresh library"
            >
              <MdiIcon name="refresh" size={16} />
            </Button>
          {/if}
          <Button
            size="sm"
            variant="ghost"
            onclick={() => startEdit(item)}
            aria-label="Edit instance"
          >
            <MdiIcon name="pencil" size={16} />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onclick={() => void remove(item)}
            aria-label="Remove instance"
          >
            <MdiIcon name="trash2" size={16} />
          </Button>
        </div>
      {/if}
    </article>
  {/each}
</div>

<style>
  .instance-list {
    display: grid;
    gap: 0;
  }

  .instance-list__item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--jb-space-4);
    padding: var(--jb-space-4) 0;
    border-bottom: 1px solid var(--jb-border);
    background: transparent;
  }

  .instance-list__item:last-child {
    border-bottom: none;
    padding-bottom: 0;
  }

  .instance-list__item:first-child {
    padding-top: 0;
  }

  .instance-list__item--active {
    background: transparent;
  }

  .instance-list__item--active .instance-list__name {
    color: var(--jb-accent);
  }

  .instance-list__edit {
    flex: 1;
    display: grid;
    gap: var(--jb-space-3);
  }

  .instance-list__name {
    margin: 0;
    font-size: 1rem;
  }

  .instance-list__meta,
  .instance-list__user,
  .instance-list__tracks {
    margin: 0.125rem 0 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .instance-list__tracks {
    font-variant-numeric: tabular-nums;
  }

  .instance-list__actions {
    display: flex;
    align-items: center;
    gap: var(--jb-space-2);
    flex-shrink: 0;
  }

  .instance-list__badge {
    font-size: 0.75rem;
    font-weight: 700;
    color: var(--jb-accent);
  }

  .instance-list__ping {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    margin-top: var(--jb-space-2);
    padding: 0;
    border: none;
    background: transparent;
    color: var(--jb-text-muted);
    font-size: 0.75rem;
    font-variant-numeric: tabular-nums;
    cursor: pointer;
  }

  .ping-dot {
    width: 0.5rem;
    height: 0.5rem;
    border-radius: var(--jb-radius-full);
    background: var(--jb-text-subtle);
  }

  .ping-dot--online {
    background: #22c55e;
  }

  .ping-dot--offline {
    background: var(--jb-danger);
  }

  .ping-dot--idle {
    background: var(--jb-text-subtle);
    animation: ping-pulse 1s ease-in-out infinite;
  }

  @keyframes ping-pulse {
    0%,
    100% {
      opacity: 0.4;
    }
    50% {
      opacity: 1;
    }
  }

  @media (max-width: 640px) {
    .instance-list__item {
      flex-direction: column;
      align-items: stretch;
    }

    .instance-list__actions {
      justify-content: flex-end;
    }
  }
</style>
