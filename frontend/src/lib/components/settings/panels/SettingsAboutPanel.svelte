<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import { APP_NAME } from "$lib/brand";
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import SettingsToggleRow from "$lib/components/settings/SettingsToggleRow.svelte";
  import { API_VERSION, CLIENT_VERSION, getCompatState } from "$lib/compat";
  import { isWailsDesktop } from "$lib/config/runtime";
  import { fetchWithRetry, apiHeaders } from "$lib/core/http/client";
  import Button from "$lib/components/ui/Button.svelte";
  import Spinner from "$lib/components/ui/Spinner.svelte";
  import { toast } from "$lib/ui/toast.svelte";
  import "$lib/settings/settings-page.css";

  type UpdateStatus = {
    currentVersion?: string;
    latestVersion?: string;
    releaseUrl?: string;
    upToDate?: boolean;
    checking?: boolean;
    applying?: boolean;
    stage?: string;
    stageMessage?: string;
    written?: number;
    total?: number;
    canApply?: boolean;
    inContainer?: boolean;
    autoUpdate?: boolean;
    channel?: string;
    signed?: boolean;
    checkError?: string;
    applyError?: string;
    error?: string;
    manual?: boolean;
    appliedVersion?: string;
    desktop?: {
      state?: string;
      manual?: boolean;
      releaseUrl?: string;
      latestVersion?: string;
    };
  };

  let buildDate = $state("");
  let dataDir = $state("");
  let loading = $state(true);
  let upd = $state<UpdateStatus | null>(null);
  let updBusy = $state(false);
  let updTimer: ReturnType<typeof setInterval> | null = null;

  const compat = $derived(getCompatState());
  const capabilityCount = $derived(compat.supported.length);
  const capabilityPreview = $derived(
    compat.supported.slice(0, 8).join(", ") +
      (compat.supported.length > 8 ? "..." : ""),
  );

  function formatLocaleDate(value: string): string {
    const trimmed = value.trim();
    if (!trimmed) return "";
    const parsed = Date.parse(trimmed);
    if (Number.isNaN(parsed)) return trimmed;
    return new Date(parsed).toLocaleDateString();
  }

  const buildDateLabel = $derived(formatLocaleDate(buildDate));

  onMount(() => {
    loading = true;
    fetchWithRetry("/api/config", { headers: apiHeaders() })
      .then((r) => (r.ok ? r.json() : null))
      .then((cfg) => {
        buildDate =
          typeof cfg?.buildDate === "string" ? cfg.buildDate.trim() : "";
        dataDir = typeof cfg?.dataDir === "string" ? cfg.dataDir : "";
      })
      .finally(() => {
        loading = false;
      });
    void refreshUpdateStatus();
    updTimer = setInterval(() => void pollUpdateStatus(), 2000);
  });

  onDestroy(() => {
    if (updTimer) clearInterval(updTimer);
  });

  async function refreshUpdateStatus(): Promise<void> {
    try {
      const r = await fetchWithRetry("/api/update/status", {
        headers: apiHeaders(),
      });
      if (r.ok) upd = (await r.json()) as UpdateStatus;
    } catch {
      // The endpoint is absent on older servers; leave the card quiet.
    }
  }

  async function pollUpdateStatus(): Promise<void> {
    if (!upd?.checking && !upd?.applying) return;
    await refreshUpdateStatus();
  }

  async function checkForUpdates(): Promise<void> {
    updBusy = true;
    try {
      const r = await fetchWithRetry("/api/update/check", {
        method: "POST",
        headers: apiHeaders(),
      });
      if (r.ok || r.status === 202) {
        upd = (await r.json()) as UpdateStatus;
        if (upd.upToDate) toast.success("You are up to date");
      } else {
        toast.error("Update check failed");
      }
    } catch {
      toast.error("Update check failed");
    } finally {
      updBusy = false;
    }
  }

  async function applyUpdate(): Promise<void> {
    updBusy = true;
    try {
      const r = await fetchWithRetry("/api/update/apply", {
        method: "POST",
        headers: apiHeaders(),
        body: "{}",
      });
      if (r.ok || r.status === 202) {
        const data = (await r.json()) as UpdateStatus;
        upd = { ...upd, ...data, applying: data.applying ?? true };
        toast.success("Update started");
      } else {
        const data = await r.json().catch(() => null);
        toast.error(data?.error ?? "Update failed to start");
      }
    } catch {
      toast.error("Update failed to start");
    } finally {
      updBusy = false;
    }
  }

  async function restartToApply(): Promise<void> {
    try {
      const r = await fetchWithRetry("/api/update/restart", {
        method: "POST",
        headers: apiHeaders(),
      });
      if (!r.ok) toast.error("Restart failed; restart the app manually");
    } catch {
      toast.error("Restart failed; restart the app manually");
    }
  }

  async function setAutoUpdate(checked: boolean): Promise<void> {
    const channel = upd?.channel === "prerelease" ? "prerelease" : "stable";
    try {
      const r = await fetchWithRetry("/api/update/settings", {
        method: "PUT",
        headers: apiHeaders(),
        body: JSON.stringify({ autoUpdate: checked, channel }),
      });
      if (r.ok) {
        upd = { ...upd, autoUpdate: checked } as UpdateStatus;
        toast.success(
          checked ? "Automatic updates enabled" : "Automatic updates disabled",
        );
      } else {
        toast.error("Could not save update settings");
      }
    } catch {
      toast.error("Could not save update settings");
    }
  }

  async function setChannel(prerelease: boolean): Promise<void> {
    const channel = prerelease ? "prerelease" : "stable";
    try {
      const r = await fetchWithRetry("/api/update/settings", {
        method: "PUT",
        headers: apiHeaders(),
        body: JSON.stringify({
          autoUpdate: upd?.autoUpdate === true,
          channel,
        }),
      });
      if (r.ok) upd = { ...upd, channel } as UpdateStatus;
    } catch {
      toast.error("Could not save update settings");
    }
  }

  const updProgress = $derived(
    upd && (upd.total ?? 0) > 0
      ? Math.min(100, Math.round(((upd.written ?? 0) / (upd.total ?? 1)) * 100))
      : null,
  );
  const updReady = $derived(
    upd?.stage === "done" || upd?.desktop?.state === "ready",
  );
</script>

<SettingsCard
  title="About"
  description={`Client and server identity for this ${APP_NAME} install.`}
>
  {#if loading}
    <div class="about-panel__loading">
      <Spinner />
    </div>
  {:else}
    <dl class="about-panel">
      <div class="about-panel__row">
        <dt>App</dt>
        <dd>{APP_NAME}</dd>
      </div>
      <div class="about-panel__row">
        <dt>Client version</dt>
        <dd>{CLIENT_VERSION}</dd>
      </div>
      <div class="about-panel__row">
        <dt>Server version</dt>
        <dd>{compat.serverVersion || "Unknown"}</dd>
      </div>
      {#if buildDateLabel}
        <div class="about-panel__row">
          <dt>Build date</dt>
          <dd>{buildDateLabel}</dd>
        </div>
      {/if}
      <div class="about-panel__row">
        <dt>API version</dt>
        <dd>{compat.apiVersion || API_VERSION}</dd>
      </div>
      <div class="about-panel__row">
        <dt>Capabilities</dt>
        <dd>
          {capabilityCount} supported
          {#if capabilityCount > 0}
            <span class="about-panel__caps">({capabilityPreview})</span>
          {/if}
        </dd>
      </div>
      {#if dataDir}
        <div class="about-panel__row">
          <dt>Data directory</dt>
          <dd><code class="settings-page__path">{dataDir}</code></dd>
        </div>
      {/if}
    </dl>
  {/if}
</SettingsCard>

<SettingsCard
  title="Updates"
  description={`${APP_NAME} checks the signed release feed for new versions.`}
>
  {#if upd}
    <div class="about-panel__row">
      <dt>Update status</dt>
      <dd>
        {#if upd.checking || updBusy}
          Checking for updates...
        {:else if upd.applying}
          {upd.stageMessage || "Installing update..."}
        {:else if upd.applyError || upd.checkError || upd.error}
          <span class="update-panel__error"
            >{upd.applyError || upd.checkError || upd.error}</span
          >
        {:else if updReady}
          Update to v{upd.latestVersion ?? upd.appliedVersion} installed. Restart
          to finish.
        {:else if upd.latestVersion && !upd.upToDate}
          v{upd.latestVersion} is available
          {#if upd.releaseUrl}
            <a href={upd.releaseUrl} target="_blank" rel="noreferrer"
              >release notes</a
            >
          {/if}
        {:else if upd.upToDate}
          Up to date
        {:else}
          Not checked yet
        {/if}
      </dd>
    </div>

    {#if upd.applying && updProgress !== null}
      <div
        class="update-panel__progress"
        role="progressbar"
        aria-valuenow={updProgress}
      >
        <div class="update-panel__bar" style:width="{updProgress}%"></div>
      </div>
    {/if}

    <div class="update-panel__actions">
      <Button
        variant="ghost"
        disabled={updBusy || upd.checking || upd.applying}
        onclick={() => void checkForUpdates()}
      >
        Check for updates
      </Button>
      {#if upd.latestVersion && !upd.upToDate && !updReady}
        {#if upd.canApply || upd.desktop}
          <Button
            disabled={updBusy || upd.applying}
            onclick={() => void applyUpdate()}
          >
            Download and install
          </Button>
        {:else if upd.releaseUrl}
          <Button onclick={() => window.open(upd?.releaseUrl, "_blank")}>
            Download release
          </Button>
        {/if}
      {:else if updReady && upd.desktop}
        <Button onclick={() => void restartToApply()}>Restart to apply</Button>
      {/if}
    </div>

    {#if upd.inContainer}
      <p class="update-panel__hint">
        This install runs in a container. Update by pulling a new image.
      </p>
    {:else if !upd.canApply && !upd.desktop}
      <p class="update-panel__hint">
        The server binary is not writable from this process. Run
        <code class="settings-page__path">melovian-server --update</code> or
        install <code class="settings-page__path">melovian-updater</code>.
      </p>
    {/if}
    {#if !upd.signed}
      <p class="update-panel__hint">
        This build has no release signing key compiled in; updates are verified
        by checksum only.
      </p>
    {/if}

    {#if isWailsDesktop()}
      <SettingsToggleRow
        label="Automatically download and install updates"
        checked={upd.autoUpdate === true}
        onchange={(checked) => void setAutoUpdate(checked)}
      />
    {/if}
    <SettingsToggleRow
      label="Include pre-release versions"
      checked={upd.channel === "prerelease"}
      onchange={(checked) => void setChannel(checked)}
    />
  {:else}
    <p class="update-panel__hint">
      Update status is not available from this server.
    </p>
  {/if}
</SettingsCard>

<style>
  .about-panel__loading {
    display: grid;
    place-content: center;
    min-height: 6rem;
  }

  .about-panel {
    margin: 0;
    display: grid;
    gap: var(--jb-space-3);
  }

  .about-panel__row {
    display: grid;
    gap: 0.25rem;
  }

  .about-panel__row dt {
    margin: 0;
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--jb-text-muted);
  }

  .about-panel__row dd {
    margin: 0;
    font-size: 0.9375rem;
    color: var(--jb-text);
    overflow-wrap: anywhere;
  }

  .about-panel__caps {
    display: block;
    margin-top: 0.25rem;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
    line-height: 1.45;
  }

  .update-panel__actions {
    display: flex;
    gap: var(--jb-space-2);
    margin-top: var(--jb-space-3);
  }

  .update-panel__progress {
    height: 0.375rem;
    margin-top: var(--jb-space-2);
    border-radius: 999px;
    background: var(--jb-surface-2, rgba(255, 255, 255, 0.08));
    overflow: hidden;
  }

  .update-panel__bar {
    height: 100%;
    background: var(--jb-accent, #4f8cff);
    transition: width 0.25s ease;
  }

  .update-panel__error {
    color: var(--jb-danger, #e5534b);
  }

  .update-panel__hint {
    margin: var(--jb-space-2) 0 0;
    font-size: 0.8125rem;
    color: var(--jb-text-muted);
    line-height: 1.45;
  }
</style>
