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

  const lastfmStatus = createSaveStatus();

  let settings = $state<musicApi.LastFMSettings | null>(null);
  let apiKey = $state("");
  let apiSecret = $state("");
  let sessionKey = $state("");
  let endpoint = $state("");
  let testing = $state(false);

  const enabled = $derived(extensionFeatures.isEnabled("lastfm"));
  const credentialsComplete = $derived(
    apiKey.trim() !== "" && apiSecret.trim() !== "" && sessionKey.trim() !== "",
  );
  const canTest = $derived(
    settings !== null && (credentialsComplete || settings.hasToken),
  );

  onMount(() => {
    void load();
  });

  async function load() {
    const next = await musicApi.getLastFMSettings().catch(() => null);
    if (!next) {
      settings = {
        enabled: false,
        hasToken: false,
        apiKey: "",
        apiSecret: "",
        sessionKey: "",
        endpoint: "",
      };
      return;
    }
    settings = next;
    apiKey = next.apiKey;
    apiSecret = "";
    sessionKey = "";
    endpoint = next.endpoint;
  }

  async function test() {
    if (!canTest) return;
    testing = true;
    try {
      const result = await musicApi.testLastFMToken(
        apiKey.trim(),
        apiSecret.trim(),
        sessionKey.trim(),
        endpoint.trim() || "https://ws.audioscrobbler.com/2.0/",
      );
      if (!result) {
        toast.error("Could not validate credentials");
        return;
      }
      if (result.ok) {
        toast.success(`Credentials valid for ${result.userName ?? "Last.fm"}`);
      } else {
        toast.error("Credentials are not valid");
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Credential test failed",
      );
    } finally {
      testing = false;
    }
  }

  // The server stores the payload verbatim, so a save only runs while all
  // three credential fields are filled. Empty fields would wipe the stored
  // credentials.
  async function save() {
    if (!credentialsComplete) return;
    lastfmStatus.begin();
    try {
      const next = await musicApi.saveLastFMSettings({
        apiKey: apiKey.trim(),
        apiSecret: apiSecret.trim(),
        sessionKey: sessionKey.trim(),
        endpoint: endpoint.trim() || "https://ws.audioscrobbler.com/2.0/",
      });
      if (next) {
        settings = next;
        endpoint = next.endpoint;
        lastfmStatus.saved();
      } else {
        lastfmStatus.failed("Could not save Last.fm settings");
        toast.error("Could not save Last.fm settings");
      }
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Could not save Last.fm settings";
      lastfmStatus.failed(message);
      toast.error(message);
    }
  }

  const debouncedSave = useDebounce(() => save(), 500);

  function queueSave() {
    void debouncedSave().catch(() => {});
  }
</script>

<SettingsCard title="Last.fm" description="Scrobble listens to Last.fm.">
  {#snippet status()}
    <SettingsSaveStatus status={lastfmStatus} />
  {/snippet}
  {#if !enabled}
    <p class="lastfm-settings__note">
      Enable the Last.fm extension in the Extensions tab to start scrobbling.
    </p>
  {/if}
  <label class="lastfm-settings__label" for="lastfm-endpoint"> Endpoint </label>
  <Input
    id="lastfm-endpoint"
    type="url"
    placeholder="https://ws.audioscrobbler.com/2.0/"
    bind:value={endpoint}
    oninput={queueSave}
    disabled={!settings}
  />
  <label class="lastfm-settings__label" for="lastfm-api-key">API key</label>
  <Input
    id="lastfm-api-key"
    type="text"
    placeholder="Your Last.fm API key"
    bind:value={apiKey}
    oninput={queueSave}
    disabled={!settings}
  />
  <label class="lastfm-settings__label" for="lastfm-api-secret">
    API secret
  </label>
  <Input
    id="lastfm-api-secret"
    type="password"
    placeholder="Your Last.fm API secret"
    bind:value={apiSecret}
    oninput={queueSave}
    disabled={!settings}
  />
  <label class="lastfm-settings__label" for="lastfm-session-key">
    Session key
  </label>
  <Input
    id="lastfm-session-key"
    type="password"
    placeholder="Your Last.fm session key"
    bind:value={sessionKey}
    oninput={queueSave}
    disabled={!settings}
  />
  {#if settings?.hasToken && !credentialsComplete}
    <p class="lastfm-settings__status">
      Credentials are saved. Enter the API secret and session key to update
      them.
    </p>
  {/if}
  {#snippet footer()}
    <Button
      size="sm"
      disabled={!canTest || testing}
      onclick={() => void test()}
    >
      {testing ? "Testing..." : "Test credentials"}
    </Button>
  {/snippet}
</SettingsCard>

<style>
  .lastfm-settings__note {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }

  .lastfm-settings__label {
    font-size: 0.8125rem;
    font-weight: 600;
  }

  .lastfm-settings__status {
    margin: 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
  }
</style>
