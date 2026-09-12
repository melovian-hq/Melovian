<!-- SPDX-FileCopyrightText: 2026 Quad4 Software -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import type { PlaylistViewMode } from "$lib/music/playlist-display";
  import { Tabs } from "bits-ui";

  const VIEW_OPTIONS: {
    id: PlaylistViewMode;
    label: string;
    icon: "viewList" | "viewGrid" | "viewCard";
  }[] = [
    { id: "list", label: "List", icon: "viewList" },
    { id: "grid", label: "Grid", icon: "viewGrid" },
    { id: "card", label: "Cards", icon: "viewCard" },
  ];

  interface Props {
    value?: PlaylistViewMode;
  }

  let { value = $bindable("grid") }: Props = $props();
</script>

<Tabs.Root
  style="display: contents"
  bind:value={
    () => value,
    (next) => {
      value = next as PlaylistViewMode;
    }
  }
>
  <Tabs.List class="playlists-view" aria-label="Playlist layout">
    {#each VIEW_OPTIONS as option (option.id)}
      <Tabs.Trigger
        value={option.id}
        class="playlists-view__btn"
        aria-label={option.label}
        title={option.label}
      >
        <MdiIcon name={option.icon} size={17} />
      </Tabs.Trigger>
    {/each}
  </Tabs.List>
</Tabs.Root>

<style>
  :global(.playlists-view) {
    display: inline-flex;
    gap: 0.15rem;
    padding: 0.15rem;
    border-radius: var(--jb-radius-full);
    border: 1px solid var(--jb-border);
    background: var(--jb-bg-muted);
  }

  :global(.playlists-view__btn) {
    display: inline-grid;
    place-items: center;
    width: 2rem;
    height: 2rem;
    border: none;
    border-radius: var(--jb-radius-full);
    background: transparent;
    color: var(--jb-text-muted);
    cursor: pointer;
    transition:
      background var(--jb-transition),
      color var(--jb-transition),
      box-shadow var(--jb-transition);
  }

  :global(.playlists-view__btn:hover) {
    color: var(--jb-text);
  }

  :global(.playlists-view__btn[data-state="active"]) {
    background: var(--jb-surface);
    color: var(--jb-music-accent);
    box-shadow: var(--jb-shadow-sm);
  }
</style>
