<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import { APP_NAME } from "$lib/brand";
  import SettingsToggleRow from "$lib/components/settings/SettingsToggleRow.svelte";
  import Field from "$lib/components/ui/Field.svelte";
  import Button from "$lib/components/ui/Button.svelte";
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
  import "$lib/settings/settings-page.css";

  let graphicsEnv = $state<GraphicsEnvironment | null>(null);
  let graphicsSettings = $state<GraphicsSettings>(mergeGraphicsSettings(null));
  let graphicsSaving = $state(false);

  $effect(() => {
    if (!nativeDesktopAvailable()) return;
    void loadGraphicsEnvironment().then((env) => {
      graphicsEnv = env;
      if (env) {
        graphicsSettings = mergeGraphicsSettings(env.settings);
      }
    });
  });

  async function handleSaveGraphicsSettings() {
    graphicsSaving = true;
    try {
      await saveGraphicsSettings(graphicsSettings);
      toast.success(`Graphics settings saved. Restart ${APP_NAME} to apply.`);
      graphicsEnv = await loadGraphicsEnvironment();
      if (graphicsEnv) {
        graphicsSettings = mergeGraphicsSettings(graphicsEnv.settings);
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to save graphics settings",
      );
    } finally {
      graphicsSaving = false;
    }
  }
</script>

{#if nativeDesktopAvailable() && graphicsEnv?.supported}
  <SettingsCard
    title="Graphics stability"
    description="Linux WebKitGTK workarounds for GPU and display-driver crashes."
  >
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
      }}
    />
    {#if graphicsEnv.wayland || graphicsEnv.nvidia}
      <Field
        label="NVIDIA explicit sync workaround"
        hint="Sets __NV_DISABLE_EXPLICIT_SYNC for Wayland + NVIDIA stability issues."
      >
        <select
          class="settings-input"
          value={graphicsSettings.nvDisableExplicitSync}
          onchange={(e) => {
            graphicsSettings = {
              ...graphicsSettings,
              nvDisableExplicitSync: (e.currentTarget as HTMLSelectElement)
                .value as NvExplicitSyncMode,
            };
          }}
        >
          <option value="auto">{nvExplicitSyncLabel("auto")}</option>
          <option value="on">{nvExplicitSyncLabel("on")}</option>
          <option value="off">{nvExplicitSyncLabel("off")}</option>
        </select>
      </Field>
    {/if}
    <div class="settings-page__actions">
      <Button
        onclick={() => void handleSaveGraphicsSettings()}
        disabled={graphicsSaving}
      >
        {graphicsSaving ? "Saving…" : "Save graphics settings"}
      </Button>
    </div>
  </SettingsCard>
{/if}
