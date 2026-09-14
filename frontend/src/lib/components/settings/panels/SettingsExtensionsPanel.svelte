<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import ExtensionsSettings from "$lib/components/settings/ExtensionsSettings.svelte";
</script>

<SettingsCard
  title="Extensions"
  description="Customize the player with declarative rules or optional scripts."
>
  <ExtensionsSettings />
</SettingsCard>
