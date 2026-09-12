<!-- SPDX-FileCopyrightText: 2026 Quad4 Software -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import SourceIcon from "$lib/components/ui/SourceIcon.svelte";
  import type { PlaylistKind } from "$lib/music/playlist-display";
  import { Tabs } from "bits-ui";

  interface Props {
    value?: PlaylistKind;
  }

  let { value = $bindable("server") }: Props = $props();
</script>

<Tabs.Root
  style="display: contents"
  bind:value={
    () => value,
    (next) => {
      value = next as PlaylistKind;
    }
  }
>
  <Tabs.List class="playlists-kind" aria-label="Playlist source">
    <Tabs.Trigger value="server" class="playlists-kind__btn">
      <SourceIcon kind="server" size={16} />
      Server
    </Tabs.Trigger>
    <Tabs.Trigger value="local" class="playlists-kind__btn">
      <MdiIcon name="folderOpen" size={16} />
      Local
    </Tabs.Trigger>
  </Tabs.List>
</Tabs.Root>

<style>
  :global(.playlists-kind) {
    display: inline-flex;
    gap: var(--jb-space-1);
    padding: 0.2rem;
    margin-bottom: var(--jb-space-6);
    border-radius: var(--jb-radius-full);
    border: 1px solid var(--jb-border);
    background: var(--jb-surface);
  }

  :global(.playlists-kind__btn) {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    border: none;
    background: transparent;
    color: var(--jb-text-muted);
    font-size: 0.875rem;
    font-weight: 650;
    padding: 0.45rem 0.9rem;
    border-radius: var(--jb-radius-full);
    cursor: pointer;
  }

  :global(.playlists-kind__btn[data-state="active"]) {
    background: var(--jb-bg-muted);
    color: var(--jb-text);
    box-shadow: var(--jb-shadow-sm);
  }
</style>
