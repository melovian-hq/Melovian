<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import { APP_NAME } from "$lib/brand";
  import SettingsToggleRow from "$lib/components/settings/SettingsToggleRow.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import { nativeDesktopAvailable } from "$lib/config/runtime";
  import {
    loadDesktopIntegrationSettings,
    type CloseBehavior,
  } from "$lib/desktop/desktop-integration-settings";
  import {
    setCloseBehavior,
    setTaskbarIntegrationEnabled,
  } from "$lib/desktop/window-close";
  import { setNativeTitleBarEnabled } from "$lib/desktop/window-chrome";
  import "$lib/settings/settings-page.css";

  let desktopIntegration = $state(loadDesktopIntegrationSettings());
</script>

{#if nativeDesktopAvailable()}
  <SettingsCard
    title="Desktop integration"
    description="Media keys and system player integration on supported platforms."
  >
    <p class="settings-page__meta">
      On Linux, {APP_NAME} registers an MPRIS media player for system media keys,
      volume OSD, and desktop environment controls. Windows and macOS use native media
      session hooks where available.
    </p>
    <SettingsToggleRow
      label="Native window title bar"
      description={`Use the system title bar and hide ${APP_NAME}'s custom window controls.`}
      checked={desktopIntegration.nativeTitleBar}
      onchange={(enabled) => {
        desktopIntegration = {
          ...desktopIntegration,
          nativeTitleBar: enabled,
        };
        setNativeTitleBarEnabled(enabled);
      }}
    />
    <SettingsToggleRow
      label="System tray integration"
      description={`Show ${APP_NAME} in the system tray for quick show, hide, and playback control.`}
      checked={desktopIntegration.taskbarEnabled}
      onchange={(enabled) => {
        desktopIntegration = {
          ...desktopIntegration,
          taskbarEnabled: enabled,
        };
        setTaskbarIntegrationEnabled(enabled);
      }}
    />
    <Field
      label="When closing the window"
      hint={desktopIntegration.taskbarEnabled
        ? "Choose what happens when you click the window close button."
        : `With tray integration off, background mode minimizes ${APP_NAME} to the taskbar.`}
    >
      <select
        class="settings-input"
        value={desktopIntegration.closeBehavior}
        onchange={(e) => {
          const closeBehavior = (e.currentTarget as HTMLSelectElement)
            .value as CloseBehavior;
          desktopIntegration = {
            ...desktopIntegration,
            closeBehavior,
          };
          setCloseBehavior(closeBehavior);
        }}
      >
        <option value="ask">Ask every time</option>
        <option value="quit">Quit application</option>
        <option value="background">
          {desktopIntegration.taskbarEnabled
            ? "Keep running in background"
            : "Minimize to taskbar"}
        </option>
      </select>
    </Field>
  </SettingsCard>
{/if}
