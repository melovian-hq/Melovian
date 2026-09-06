<script lang="ts">
  import ContextMenu from "$lib/components/ui/ContextMenu.svelte";
  import { router } from "$lib/router/router.svelte";
  import { homeHidden } from "$lib/music/home-hidden.svelte";
  import { playArtistNow, queueArtist } from "$lib/music/context-actions";
  import type { ContextMenuEntry } from "$lib/components/ui/context-menu";
  import type { SubsonicArtist } from "$lib/subsonic";
  import { toast } from "$lib/ui/toast.svelte";

  interface Props {
    artist: SubsonicArtist;
    x: number;
    y: number;
    onclose: () => void;
    hideable?: boolean;
  }

  let { artist, x, y, onclose, hideable = false }: Props = $props();

  const items = $derived.by((): ContextMenuEntry[] => {
    const entries: ContextMenuEntry[] = [
      {
        id: "play",
        label: "Play now",
        icon: "play",
        onclick: () => void playArtistNow(artist.id),
      },
      {
        id: "shuffle",
        label: "Shuffle",
        icon: "shuffle",
        onclick: () => void playArtistNow(artist.id, true),
      },
      {
        id: "queue",
        label: "Add to queue",
        icon: "queueAdd",
        onclick: () => void queueArtist(artist.id),
      },
      {
        id: "open",
        label: "Go to artist",
        icon: "accountMusic",
        onclick: () => router.navigate(`/music/artist/${artist.id}`),
      },
    ];
    if (hideable) {
      entries.push({ id: "hide-sep", separator: true });
      entries.push({
        id: "hide",
        label: "Hide",
        icon: "eyeOff",
        onclick: () => {
          homeHidden.hide("artist", artist.id);
          toast.info(`Hidden ${artist.name} from home`);
        },
      });
    }
    return entries;
  });
</script>

<ContextMenu {x} {y} {onclose} label="Artist actions" {items} />
