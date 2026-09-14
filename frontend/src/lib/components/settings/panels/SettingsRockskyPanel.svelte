<script lang="ts">
  import { onMount } from "svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import Input from "$lib/components/ui/Input.svelte";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import SettingsSaveStatus from "$lib/components/settings/SettingsSaveStatus.svelte";
  import { useDebounce } from "runed";
  import * as musicApi from "$lib/music/api";
  import { extensionFeatures } from "$lib/extensions/features.svelte";
  import { toast } from "$lib/ui/toast.svelte";
  import { createSaveStatus } from "$lib/settings/save-status.svelte";

  const rockskyStatus = createSaveStatus();

  let settings = $state<musicApi.RockskySettings | null>(null);
  let token = $state("");
  let testing = $state(false);

  const enabled = $derived(extensionFeatures.isEnabled("rocksky"));

  onMount(() => {
    void load();
  });

  async function load() {
    const next = await musicApi.getRockskySettings().catch(() => null);
    if (!next) {
      settings = { enabled: false, hasToken: false, token: "" };
      return;
    }
    settings = next;
    token = "";
  }

  async function test() {
    const value = token.trim();
    if (!value && !settings?.hasToken) return;
    testing = true;
    try {
      const result = await musicApi.testRockskyToken(value);
      if (!result) {
        toast.error("Could not validate token");
        return;
      }
      if (result.ok) {
        toast.success(`Token valid for ${result.userName ?? "Rocksky"}`);
      } else {
        toast.error("Token is not valid");
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Token test failed");
    } finally {
      testing = false;
    }
  }

  async function save() {
    const value = token.trim();
    if (!value) return;
    rockskyStatus.begin();
    try {
      const next = await musicApi.saveRockskySettings({ token: value });
      if (next) {
        settings = next;
        rockskyStatus.saved();
      } else {
        rockskyStatus.failed("Could not save Rocksky token");
        toast.error("Could not save Rocksky token");
      }
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Could not save Rocksky token";
      rockskyStatus.failed(message);
      toast.error(message);
    }
  }

  const debouncedSave = useDebounce(() => save(), 500);

  function queueSave() {
    void debouncedSave().catch(() => {});
  }
</script>

<SettingsCard title="Rocksky" description="Scrobble listens to Rocksky.">
  {#snippet status()}
    <SettingsSaveStatus status={rockskyStatus} />
  {/snippet}
  {#if !enabled}
    <p class="rocksky-settings__note">
      Enable the Rocksky extension in the Extensions tab to start scrobbling.
    </p>
  {/if}
  <label class="rocksky-settings__label" for="rocksky-token"> API token </label>
  <Input
    id="rocksky-token"
    type="password"
    placeholder="Paste your Rocksky API token"
    bind:value={token}
    oninput={queueSave}
    disabled={!settings}
  />
  {#if settings?.hasToken}
    <p class="rocksky-settings__status">
      A token is saved. Typing a new one replaces it.
    </p>
  {/if}
  {#snippet footer()}
    <Button
      size="sm"
      disabled={(!token.trim() && !settings?.hasToken) || testing || !settings}
      onclick={() => void test()}
    >
      {testing ? "Testing..." : "Test token"}
    </Button>
  {/snippet}
</SettingsCard>

<style>
  .rocksky-settings__note {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .rocksky-settings__label {
    font-size: 0.8125rem;
    font-weight: 600;
  }

  .rocksky-settings__status {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }
</style>
