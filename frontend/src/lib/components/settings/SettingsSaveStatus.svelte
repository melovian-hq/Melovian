<script lang="ts">
  import MdiIcon from "$lib/components/ui/MdiIcon.svelte";
  import type { SaveStatus } from "$lib/settings/save-status.svelte";

  interface Props {
    status: SaveStatus;
  }

  let { status }: Props = $props();
</script>

{#if status.state !== "idle"}
  <p
    class="settings-save-status settings-save-status--{status.state}"
    role={status.state === "error" ? "alert" : "status"}
  >
    {#if status.state === "saving"}
      Saving...
    {:else if status.state === "saved"}
      <MdiIcon name="check" size={13} />
      Saved
    {:else}
      {status.error}
    {/if}
  </p>
{/if}

<style>
  .settings-save-status {
    display: inline-flex;
    align-items: center;
    gap: var(--jb-space-1);
    margin: 0;
    font-size: 0.8125rem;
    line-height: 1.45;
    color: var(--jb-text-subtle);
    white-space: nowrap;
  }

  .settings-save-status--error {
    color: var(--jb-danger);
    white-space: normal;
    text-align: right;
  }
</style>
