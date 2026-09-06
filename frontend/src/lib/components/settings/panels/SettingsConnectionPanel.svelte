<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import { APP_NAME } from "$lib/brand";
  import SettingsToggleRow from "$lib/components/settings/SettingsToggleRow.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import { connection } from "$lib/music/connection.svelte";
  import { toast } from "$lib/ui/toast.svelte";
  import "$lib/settings/settings-page.css";

  let settings = $derived(connection.settings);

  function resetSettings() {
    void connection.reloadSettings().then(() => {
      settings = { ...connection.settings };
      toast.success("Connection settings reset");
    });
  }

  function patchConnectionSettings(
    patch: Partial<typeof settings>,
    message = "Connection settings saved",
  ) {
    settings = { ...settings, ...patch };
    void connection.updateSettings(settings).then(() => {
      toast.success(message);
    });
  }
</script>

<SettingsCard
  title="Connection and recovery"
  description={`${APP_NAME} remembers disconnects and adapts retry timing from past events.`}
>
  <SettingsToggleRow
    label="Auto-reconnect when server is unreachable"
    checked={settings.autoReconnect}
    onchange={(checked) => patchConnectionSettings({ autoReconnect: checked })}
  />

  <SettingsToggleRow
    label="Retry immediately when network comes back"
    checked={settings.retryOnOnline}
    onchange={(checked) => patchConnectionSettings({ retryOnOnline: checked })}
  />

  <SettingsToggleRow
    label="Remember disconnect history for smarter retries"
    checked={settings.rememberHistory}
    onchange={(checked) =>
      patchConnectionSettings({ rememberHistory: checked })}
  />

  {#if connection.history.length > 0}
    <p class="settings-page__meta">
      Last disconnect:
      {connection.lastDisconnectedAt
        ? new Date(connection.lastDisconnectedAt).toLocaleString()
        : "none"}
      · Last connect:
      {connection.lastConnectedAt
        ? new Date(connection.lastConnectedAt).toLocaleString()
        : "none"}
      · {connection.history.length} events remembered
    </p>
  {/if}

  {#snippet footer()}
    <Button variant="ghost" onclick={resetSettings}>Reset defaults</Button>
    <Button variant="ghost" onclick={() => connection.forceReconnect()}>
      Reconnect now
    </Button>
  {/snippet}
</SettingsCard>
