<script lang="ts">
  import ContextMenu from "$lib/components/ui/ContextMenu.svelte";
  import PlaylistPicker from "./PlaylistPicker.svelte";
  import { router } from "$lib/router/router.svelte";
  import { homeHidden } from "$lib/music/home-hidden.svelte";
  import {
    loadAlbumSongs,
    playAlbumNow,
    queueAlbum,
    saveAlbumFilesToDevice,
  } from "$lib/music/context-actions";
  import type { ContextMenuEntry } from "$lib/components/ui/context-menu";
  import type { SubsonicAlbum, SubsonicSong } from "$lib/subsonic";
  import { toast } from "$lib/ui/toast.svelte";

  interface Props {
    album: SubsonicAlbum;
    x: number;
    y: number;
    onclose: () => void;
    hideable?: boolean;
  }

  let { album, x, y, onclose, hideable = false }: Props = $props();

  let playlistOpen = $state(false);
  let playlistTracks = $state.raw<SubsonicSong[]>([]);

  const items = $derived.by((): ContextMenuEntry[] => {
    const entries: ContextMenuEntry[] = [
      {
        id: "play",
        label: "Play now",
        icon: "play",
        onclick: () => void playAlbumNow(album.id),
      },
      {
        id: "queue",
        label: "Add to queue",
        icon: "queueAdd",
        onclick: () => void queueAlbum(album.id),
      },
      {
        id: "playlist",
        label: "Add to playlist",
        icon: "listMusic",
        keepOpen: true,
        onclick: () => void openPlaylistPicker(),
      },
      {
        id: "download",
        label: "Download",
        icon: "download",
        onclick: () => void saveAlbumFilesToDevice(album.id),
      },
    ];
    if (album.artistId) {
      entries.push({
        id: "artist",
        label: "Go to artist",
        icon: "accountMusic",
        onclick: () => router.navigate(`/music/artist/${album.artistId}`),
      });
    }
    if (hideable) {
      entries.push({ id: "hide-sep", separator: true });
      entries.push({
        id: "hide",
        label: "Hide",
        icon: "eyeOff",
        onclick: () => {
          homeHidden.hide("album", album.id);
          toast.info(`Hidden ${album.name} from home`);
        },
      });
    }
    return entries;
  });

  async function openPlaylistPicker() {
    const songs = await loadAlbumSongs(album.id);
    if (songs.length === 0) {
      onclose();
      return;
    }
    playlistTracks = songs;
    playlistOpen = true;
  }
</script>

{#if playlistOpen}
  <PlaylistPicker
    open
    tracks={playlistTracks}
    onclose={() => {
      playlistOpen = false;
      onclose();
    }}
  />
{:else}
  <ContextMenu {x} {y} {onclose} label="Album actions" {items} />
{/if}
