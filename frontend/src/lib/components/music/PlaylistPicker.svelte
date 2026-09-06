<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import SourceIcon from "$lib/components/ui/SourceIcon.svelte";
  import { music } from "$lib/config/music.svelte";
  import { sources } from "$lib/features/sources/store.svelte";
  import type { SubsonicSong } from "$lib/subsonic";

  interface Props {
    open?: boolean;
    tracks?: SubsonicSong[];
    onclose?: () => void;
  }

  let { open = false, tracks = [], onclose }: Props = $props();

  let newName = $state("");
  let newServerName = $state("");
  let creating = $state(false);
  let creatingServer = $state(false);

  const showServer = $derived(sources.hasSubsonicActive);
  const showLocal = $derived(
    sources.hasLocalActive || music.playlists.length > 0,
  );

  async function addToExisting(playlistId: string) {
    if (tracks.length === 0) return;
    await music.addTracksToPlaylist(playlistId, tracks);
    onclose?.();
  }

  async function addToServerPlaylist(playlistId: string) {
    if (tracks.length === 0) return;
    await music.addTracksToServerPlaylist(playlistId, tracks);
    onclose?.();
  }

  async function createAndAdd() {
    const name = newName.trim();
    if (!name || tracks.length === 0) return;
    creating = true;
    try {
      const pl = await music.createPlaylist(name);
      await music.addTracksToPlaylist(pl.id, tracks);
      newName = "";
      onclose?.();
    } finally {
      creating = false;
    }
  }

  async function createServerAndAdd() {
    const name = newServerName.trim();
    if (!name || tracks.length === 0) return;
    creatingServer = true;
    try {
      const pl = await music.createServerPlaylist(
        name,
        tracks.map((track) => track.id),
      );
      newServerName = "";
      onclose?.();
      void pl;
    } finally {
      creatingServer = false;
    }
  }
</script>

{#if open}
  <div
    class="picker-backdrop"
    onclick={() => onclose?.()}
    role="presentation"
  ></div>
  <div class="picker" role="dialog" aria-label="Add to playlist">
    <header class="picker__header">
      <h3>Add to playlist</h3>
      <p>{tracks.length} track{tracks.length === 1 ? "" : "s"} selected</p>
    </header>

    {#if showServer}
      <section class="picker__section">
        <h4 class="picker__section-title">
          <SourceIcon kind="server" size={16} />
          Server playlists
        </h4>
        <form
          class="picker__create"
          onsubmit={(e) => {
            e.preventDefault();
            createServerAndAdd();
          }}
        >
          <input
            bind:value={newServerName}
            placeholder="New server playlist"
            autocomplete="off"
          />
          <button
            type="submit"
            disabled={creatingServer || !newServerName.trim()}
          >
            <MdiIcon name="plus" size={16} />
            Create
          </button>
        </form>
        <ul class="picker__list">
          {#each music.serverPlaylists as pl (pl.id)}
            <li>
              <button type="button" onclick={() => addToServerPlaylist(pl.id)}>
                <SourceIcon kind="server" size={18} />
                <span>{pl.name}</span>
                <span class="picker__count">{pl.songCount ?? 0}</span>
              </button>
            </li>
          {:else}
            <li class="picker__empty">No server playlists yet.</li>
          {/each}
        </ul>
      </section>
    {/if}

    {#if showLocal}
      <section class="picker__section">
        <h4 class="picker__section-title">
          <MdiIcon name="folderOpen" size={16} />
          Local playlists
        </h4>
        <form
          class="picker__create"
          onsubmit={(e) => {
            e.preventDefault();
            createAndAdd();
          }}
        >
          <input
            bind:value={newName}
            placeholder="New local playlist"
            autocomplete="off"
          />
          <button type="submit" disabled={creating || !newName.trim()}>
            <MdiIcon name="plus" size={16} />
            Create
          </button>
        </form>
        <ul class="picker__list">
          {#each music.playlists as pl (pl.id)}
            <li>
              <button type="button" onclick={() => addToExisting(pl.id)}>
                <MdiIcon name="folderOpen" size={18} />
                <span>{pl.name}</span>
                <span class="picker__count">{pl.trackCount}</span>
              </button>
            </li>
          {:else}
            <li class="picker__empty">No local playlists yet.</li>
          {/each}
        </ul>
      </section>
    {/if}
  </div>
{/if}

<style>
  .picker-backdrop {
    position: fixed;
    inset: 0;
    background: rgb(0 0 0 / 0.45);
    z-index: 70;
  }

  .picker {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    z-index: 71;
    width: min(24rem, calc(100vw - 2rem));
    max-height: min(32rem, 80vh);
    overflow: auto;
    padding: var(--jb-space-5);
    border-radius: var(--jb-radius-xl);
    background: var(--jb-bg-elevated);
    border: 1px solid var(--jb-border);
    box-shadow: var(--jb-shadow-lg);
  }

  .picker__header h3 {
    margin: 0 0 0.25rem;
    font-size: 1.125rem;
  }

  .picker__header p {
    margin: 0 0 var(--jb-space-4);
    font-size: 0.875rem;
    color: var(--jb-text-muted);
  }

  .picker__section + .picker__section {
    margin-top: var(--jb-space-5);
    padding-top: var(--jb-space-5);
    border-top: 1px solid var(--jb-border);
  }

  .picker__section-title {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-2);
    margin: 0 0 var(--jb-space-3);
    font-size: 0.875rem;
    font-weight: 700;
    color: var(--jb-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .picker__create {
    display: flex;
    gap: var(--jb-space-2);
    margin-bottom: var(--jb-space-4);
  }

  .picker__create input {
    flex: 1;
    padding: 0.5rem 0.75rem;
    border-radius: var(--jb-radius-md);
    border: 1px solid var(--jb-border);
    background: var(--jb-bg);
    color: var(--jb-text);
    font: inherit;
  }

  .picker__create button {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-1);
    padding: 0.5rem 0.875rem;
    border: none;
    border-radius: var(--jb-radius-md);
    background: var(--jb-music-accent);
    color: white;
    font-weight: 600;
    cursor: pointer;
  }

  .picker__create button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .picker__list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: var(--jb-space-1);
  }

  .picker__list button {
    display: flex;
    align-items: center;
    gap: var(--jb-space-3);
    width: 100%;
    padding: var(--jb-space-3);
    border: none;
    border-radius: var(--jb-radius-md);
    background: transparent;
    color: inherit;
    text-align: left;
    cursor: pointer;
    font: inherit;
  }

  .picker__list button:hover {
    background: var(--jb-surface-hover);
  }

  .picker__count {
    margin-left: auto;
    font-size: 0.8125rem;
    color: var(--jb-text-subtle);
  }

  .picker__empty {
    padding: var(--jb-space-4);
    text-align: center;
    color: var(--jb-text-muted);
    font-size: 0.875rem;
  }
</style>
