<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import { APP_NAME } from "$lib/brand";
  import ProfileSettings from "$lib/components/settings/ProfileSettings.svelte";
  import "$lib/settings/settings-page.css";
</script>

<SettingsCard
  title="Profile"
  description={`Your avatar appears in the sidebar and across ${APP_NAME} on this device.`}
>
  <ProfileSettings />
</SettingsCard>
