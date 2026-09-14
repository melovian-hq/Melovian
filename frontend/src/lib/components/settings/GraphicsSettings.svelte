<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import SettingsSaveStatus from "$lib/components/settings/SettingsSaveStatus.svelte";
  import SettingsToggleRow from "$lib/components/settings/SettingsToggleRow.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import Select from "$lib/components/ui/Select.svelte";
  import { nativeDesktopAvailable } from "$lib/config/runtime";
  import {
    appliedEnvSummary,
    loadGraphicsEnvironment,
    mergeGraphicsSettings,
    nvExplicitSyncLabel,
    saveGraphicsSettings,
    type GraphicsEnvironment,
    type GraphicsSettings,
    type NvExplicitSyncMode,
  } from "$lib/desktop/graphics-settings";
  import { toast } from "$lib/ui/toast.svelte";
  import { createSaveStatus } from "$lib/settings/save-status.svelte";
  import "$lib/settings/settings-page.css";

  const graphicsStatus = createSaveStatus();

  let graphicsEnv = $state<GraphicsEnvironment | null>(null);
  let graphicsSettings = $state<GraphicsSettings>(mergeGraphicsSettings(null));

  const nvSyncOptions = (["auto", "on", "off"] as NvExplicitSyncMode[]).map(
    (mode) => ({ value: mode, label: nvExplicitSyncLabel(mode) }),
  );

  $effect(() => {
    if (!nativeDesktopAvailable()) return;
    void loadGraphicsEnvironment().then((env) => {
      graphicsEnv = env;
      if (env) {
        graphicsSettings = mergeGraphicsSettings(env.settings);
      }
    });
  });

  async function applyGraphicsSettings() {
    graphicsStatus.begin();
    try {
      await saveGraphicsSettings(graphicsSettings);
      graphicsStatus.saved();
      graphicsEnv = await loadGraphicsEnvironment();
      if (graphicsEnv) {
        graphicsSettings = mergeGraphicsSettings(graphicsEnv.settings);
      }
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Could not save graphics settings";
      graphicsStatus.failed(message);
      toast.error(message);
    }
  }
</script>

{#if nativeDesktopAvailable() && graphicsEnv?.supported}
  <SettingsCard
    title="Graphics stability"
    description="Linux WebKitGTK workarounds for GPU and display-driver crashes."
  >
    {#snippet status()}
      <SettingsSaveStatus status={graphicsStatus} />
    {/snippet}
    <p class="settings-page__meta">
      Session: {graphicsEnv.wayland ? "Wayland" : "X11"}
      {#if graphicsEnv.nvidia}
        · NVIDIA driver detected
      {/if}
    </p>
    <p class="settings-page__meta">
      Active this session: {appliedEnvSummary(graphicsEnv.appliedEnv)}
    </p>
    <p class="settings-page__meta">
      {graphicsEnv.restartRequiredNote ?? "Changes apply on the next launch."}
    </p>
    <SettingsToggleRow
      label="Disable DMA-BUF renderer"
      description="Recommended on Linux. Avoids WebKitWebProcess GBM segfaults on many Mesa and NVIDIA setups."
      checked={graphicsSettings.disableDmabufRenderer}
      onchange={(enabled) => {
        graphicsSettings = {
          ...graphicsSettings,
          disableDmabufRenderer: enabled,
        };
        void applyGraphicsSettings();
      }}
    />
    <SettingsToggleRow
      label="Disable accelerated compositing"
      description="Last-resort workaround for blank windows or resize crashes. Uses software compositing."
      checked={graphicsSettings.disableCompositingMode}
      onchange={(enabled) => {
        graphicsSettings = {
          ...graphicsSettings,
          disableCompositingMode: enabled,
        };
        void applyGraphicsSettings();
      }}
    />
    {#if graphicsEnv.wayland || graphicsEnv.nvidia}
      <Field
        label="NVIDIA explicit sync workaround"
        hint="Sets __NV_DISABLE_EXPLICIT_SYNC for Wayland + NVIDIA stability issues."
      >
        <Select
          value={graphicsSettings.nvDisableExplicitSync}
          options={nvSyncOptions}
          onchange={(nvDisableExplicitSync) => {
            graphicsSettings = {
              ...graphicsSettings,
              nvDisableExplicitSync,
            };
            void applyGraphicsSettings();
          }}
        />
      </Field>
    {/if}
  </SettingsCard>
{/if}
