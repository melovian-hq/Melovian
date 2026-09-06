<script lang="ts">
  import ListenHistoryRow from "$lib/components/music/ListenHistoryRow.svelte";
  import { groupListenEventsByDay } from "$lib/music/listen-history-groups";
  import type { ListenEvent } from "$lib/subsonic/types";

  interface Props {
    events: ListenEvent[];
    onplay: (index: number) => void;
    onNearEnd?: () => void;
    class?: string;
  }

  let { events, onplay, onNearEnd, class: className = "" }: Props = $props();

  const groups = $derived.by(() => {
    let offset = 0;
    return groupListenEventsByDay(events).map((group) => {
      const startIndex = offset;
      offset += group.events.length;
      return { ...group, startIndex };
    });
  });

  function nearEnd(node: HTMLElement) {
    if (!onNearEnd) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          onNearEnd();
        }
      },
      { rootMargin: "800px 0px" },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }
</script>

<div class="history-list {className}">
  {#each groups as group (group.key)}
    <section
      class="history-list__group"
      aria-labelledby="history-day-{group.key}"
    >
      <h2 class="history-list__day" id="history-day-{group.key}">
        {group.label}
      </h2>
      {#each group.events as event, index (event.id)}
        <ListenHistoryRow
          {event}
          onplay={() => onplay(group.startIndex + index)}
        />
      {/each}
    </section>
  {/each}
  {#if onNearEnd}
    <div class="history-list__sentinel" {@attach nearEnd}></div>
  {/if}
</div>

<style>
  .history-list {
    display: flex;
    flex-direction: column;
    content-visibility: auto;
  }

  .history-list__group {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .history-list__day {
    margin: 0;
    padding: var(--jb-space-4) 0 var(--jb-space-2);
    font-size: 0.875rem;
    font-weight: 700;
    color: var(--jb-text);
  }

  .history-list__sentinel {
    height: 1px;
  }
</style>
