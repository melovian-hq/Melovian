<script lang="ts">
  import type { SubsonicSong } from "$lib/subsonic";

  interface Props {
    track: SubsonicSong;
    embedded?: boolean;
  }

  let { track, embedded = false }: Props = $props();
</script>

<div class="video-watch-stub" data-track-id={track.id} data-embedded={embedded}>
  Video panel for {track.title}
</div>
