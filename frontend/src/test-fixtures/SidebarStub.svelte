<script lang="ts">
  let { collapsed = false }: { collapsed?: boolean } = $props();
</script>

<nav class="sidebar-stub" data-collapsed={collapsed}></nav>
