<script lang="ts">
  import ContextMenu from "$lib/components/ui/ContextMenu.svelte";
  import PlaylistPicker from "./PlaylistPicker.svelte";
  import { music, type PersonalMix } from "$lib/config/music.svelte";
  import { homeHidden } from "$lib/music/home-hidden.svelte";
  import type { ContextMenuEntry } from "$lib/components/ui/context-menu";
  import type { SubsonicSong } from "$lib/subsonic";
  import { toast } from "$lib/ui/toast.svelte";

  interface Props {
    mix: PersonalMix;
    x: number;
    y: number;
    onclose: () => void;
    hideable?: boolean;
  }

  let { mix, x, y, onclose, hideable = false }: Props = $props();

  let playlistOpen = $state(false);
  let playlistTracks = $state.raw<SubsonicSong[]>([]);

  const items = $derived.by((): ContextMenuEntry[] => {
    const entries: ContextMenuEntry[] = [
      {
        id: "play",
        label: "Play now",
        icon: "play",
        onclick: () => void music.playMix(mix),
      },
      {
        id: "next",
        label: "Play next",
        icon: "playNext",
        onclick: () => void playMixNext(),
      },
      {
        id: "queue",
        label: "Add to queue",
        icon: "queueAdd",
        onclick: () => void addMixToQueue(),
      },
      {
        id: "refresh",
        label: "Refresh mix",
        icon: "refresh",
        disabled: music.mixesRegenerating,
        onclick: () => void refreshMix(),
      },
      {
        id: "save",
        label: "Save to playlists",
        icon: "listMusic",
        keepOpen: true,
        onclick: () => void openSave(),
      },
    ];
    if (hideable) {
      entries.push({ id: "hide-sep", separator: true });
      entries.push({
        id: "hide",
        label: "Hide",
        icon: "eyeOff",
        onclick: () => {
          homeHidden.hide("mix", mix.id);
          toast.info(`Hidden ${mix.title} from home`);
        },
      });
    }
    return entries;
  });

  async function addMixToQueue() {
    const hydrated = await music.ensureMix(mix.id);
    if (!hydrated?.tracks.length) return;
    music.addTracksToQueue(hydrated.tracks);
  }

  async function playMixNext() {
    const hydrated = await music.ensureMix(mix.id);
    if (!hydrated?.tracks.length) return;
    music.playTracksNext(hydrated.tracks);
  }

  async function refreshMix() {
    try {
      await music.regenerateMix(mix.id);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to refresh mix");
    }
  }

  async function openSave() {
    const hydrated = await music.ensureMix(mix.id);
    if (!hydrated || hydrated.tracks.length === 0) {
      toast.warning("No tracks to save yet");
      onclose();
      return;
    }
    playlistTracks = hydrated.tracks;
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
  <ContextMenu {x} {y} {onclose} label="Mix actions" {items} />
{/if}
