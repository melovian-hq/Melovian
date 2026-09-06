<script lang="ts">
  import { APP_NAME } from "$lib/brand";

  interface Props {
    size?: number;
    class?: string;
    alt?: string;
  }

  let { size = 32, class: className = "", alt = APP_NAME }: Props = $props();
</script>

<img
  src="/logo.png"
  {alt}
  class="app-logo {className}"
  style:width="{size}px"
  style:height="{size}px"
  width={size}
  height={size}
/>

<style>
  .app-logo {
    display: block;
    flex-shrink: 0;
    border-radius: var(--jb-radius-md);
    object-fit: cover;
  }
</style>
