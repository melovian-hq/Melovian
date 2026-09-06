<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import EmptyState from "$lib/components/ui/EmptyState.svelte";
  import { tasks, type BackgroundTask } from "$lib/tasks/tasks.svelte";
  import "$lib/settings/settings-page.css";

  const items = $derived.by(() => {
    void tasks.items;
    return tasks.list();
  });
  const running = $derived(items.filter((item) => item.status === "running"));
  const recent = $derived(items.filter((item) => item.status !== "running"));

  function statusLabel(status: BackgroundTask["status"]): string {
    if (status === "running") return "Running";
    if (status === "error") return "Failed";
    return "Done";
  }

  function formatTime(ms: number): string {
    try {
      return new Date(ms).toLocaleString(undefined, {
        month: "short",
        day: "numeric",
        hour: "numeric",
        minute: "2-digit",
      });
    } catch {
      return "";
    }
  }
</script>

<SettingsCard
  title="Tasks"
  description="Library scans, mix refreshes, and lyrics fetches running in the background."
>
  {#if items.length === 0}
    <EmptyState
      title="No recent tasks"
      message="Scans, mix regeneration, and lyrics fetches will show up here."
      icon="history"
      embedded
    />
  {:else}
    <div class="tasks-panel">
      {#if running.length > 0}
        <section class="tasks-panel__section">
          <h3 class="tasks-panel__heading">Running</h3>
          <ul class="tasks-panel__list">
            {#each running as item (item.id)}
              <li class="tasks-panel__item">
                <div class="tasks-panel__copy">
                  <p class="tasks-panel__title">{item.title}</p>
                  {#if item.detail}
                    <p class="tasks-panel__detail">{item.detail}</p>
                  {/if}
                </div>
                <span class="tasks-panel__badge tasks-panel__badge--running">
                  {statusLabel(item.status)}
                </span>
              </li>
            {/each}
          </ul>
        </section>
      {/if}

      {#if recent.length > 0}
        <section class="tasks-panel__section">
          <h3 class="tasks-panel__heading">Recent</h3>
          <ul class="tasks-panel__list">
            {#each recent as item (item.id)}
              <li class="tasks-panel__item">
                <div class="tasks-panel__copy">
                  <p class="tasks-panel__title">{item.title}</p>
                  {#if item.detail}
                    <p class="tasks-panel__detail">{item.detail}</p>
                  {/if}
                  <p class="tasks-panel__time">{formatTime(item.updatedAt)}</p>
                </div>
                <span
                  class="tasks-panel__badge"
                  class:tasks-panel__badge--error={item.status === "error"}
                  class:tasks-panel__badge--done={item.status === "done"}
                >
                  {statusLabel(item.status)}
                </span>
              </li>
            {/each}
          </ul>
        </section>
      {/if}
    </div>
  {/if}
</SettingsCard>

<style>
  .tasks-panel {
    display: grid;
    gap: var(--jb-space-5);
  }

  .tasks-panel__section {
    display: grid;
    gap: var(--jb-space-2);
  }

  .tasks-panel__heading {
    margin: 0;
    font-size: 0.75rem;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--jb-text-subtle);
  }

  .tasks-panel__list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: var(--jb-space-2);
  }

  .tasks-panel__item {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--jb-space-3);
    padding: var(--jb-space-3) 0;
    border-bottom: 1px solid
      color-mix(in srgb, var(--jb-border, currentColor) 40%, transparent);
  }

  .tasks-panel__item:last-child {
    border-bottom: none;
    padding-bottom: 0;
  }

  .tasks-panel__copy {
    min-width: 0;
    display: grid;
    gap: 0.2rem;
  }

  .tasks-panel__title {
    margin: 0;
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--jb-text);
  }

  .tasks-panel__detail,
  .tasks-panel__time {
    margin: 0;
    font-size: 0.8125rem;
    line-height: 1.45;
    color: var(--jb-text-muted);
  }

  .tasks-panel__badge {
    flex-shrink: 0;
    font-size: 0.75rem;
    font-weight: 600;
    padding: 0.2rem 0.5rem;
    border-radius: var(--jb-radius-sm);
    background: var(--jb-bg-muted);
    color: var(--jb-text-muted);
  }

  .tasks-panel__badge--running {
    background: color-mix(in srgb, var(--jb-accent) 18%, transparent);
    color: var(--jb-accent);
  }

  .tasks-panel__badge--done {
    color: var(--jb-text);
  }

  .tasks-panel__badge--error {
    background: color-mix(in srgb, var(--jb-danger, #c44) 16%, transparent);
    color: var(--jb-danger, #c44);
  }
</style>
