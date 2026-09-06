<script lang="ts">
  import SettingsCard from "$lib/components/settings/SettingsCard.svelte";
  import { APP_NAME } from "$lib/brand";
  import SettingsToggleRow from "$lib/components/settings/SettingsToggleRow.svelte";
  import Button from "$lib/components/ui/Button.svelte";
  import { instances } from "$lib/features/instances/store.svelte";
  import { localLibraries } from "$lib/features/local-libraries/store.svelte";
  import {
    BACKUP_SECTIONS,
    BACKUP_SECTION_IDS,
    createBackup,
    exportBackupFile,
    importBackup,
    parseBackupFile,
    type BackupSectionId,
  } from "$lib/backup/backup";
  import { toast } from "$lib/ui/toast.svelte";
  import "$lib/settings/settings-page.css";

  let backupSections = $state<Record<BackupSectionId, boolean>>({
    devicePreferences: true,
    serverSettings: true,
    playlists: true,
    instances: true,
    localLibraries: true,
  });
  let backupExporting = $state(false);
  let backupImporting = $state(false);
  let backupFileInput: HTMLInputElement | undefined = $state();

  function selectedBackupSections(): BackupSectionId[] {
    return BACKUP_SECTION_IDS.filter((id) => backupSections[id]);
  }

  function toggleAllBackupSections(enabled: boolean) {
    backupSections = Object.fromEntries(
      BACKUP_SECTION_IDS.map((id) => [id, enabled]),
    ) as Record<BackupSectionId, boolean>;
  }

  async function handleExportBackup() {
    const sections = selectedBackupSections();
    if (sections.length === 0) {
      toast.error("Select at least one backup section");
      return;
    }
    backupExporting = true;
    try {
      const backup = await createBackup(sections);
      exportBackupFile(backup);
      toast.success("Backup exported");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Backup export failed");
    } finally {
      backupExporting = false;
    }
  }

  async function handleImportBackupFile(event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    input.value = "";
    if (!file) return;

    const sections = selectedBackupSections();
    if (sections.length === 0) {
      toast.error("Select at least one section to import");
      return;
    }

    backupImporting = true;
    try {
      const raw = await file.text();
      const backup = parseBackupFile(raw);
      const result = await importBackup(backup, sections);
      for (const warning of result.warnings) {
        toast.info(warning);
      }
      if (result.imported.length === 0) {
        toast.error("Nothing was imported from that backup");
        return;
      }
      toast.success(
        `Imported ${result.imported.length} section${result.imported.length === 1 ? "" : "s"}. Reload the app to apply device preferences.`,
      );
      await instances.init();
      await localLibraries.init();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Backup import failed");
    } finally {
      backupImporting = false;
    }
  }
</script>

<SettingsCard
  title="Backup and restore"
  description={`Export or import ${APP_NAME} settings, playlists, servers, and local libraries.`}
>
  <p class="settings-page__meta">
    Backups are saved as JSON. Subsonic server passwords are not included for
    security. Add passwords to the backup file or re-enter servers manually
    after import.
  </p>
  <div class="settings-page__backup-actions">
    <Button variant="ghost" onclick={() => toggleAllBackupSections(true)}>
      Select all
    </Button>
    <Button variant="ghost" onclick={() => toggleAllBackupSections(false)}>
      Clear all
    </Button>
  </div>
  <div class="settings-page__backup-sections">
    {#each BACKUP_SECTIONS as section (section.id)}
      <SettingsToggleRow
        label={section.label}
        description={section.description}
        checked={backupSections[section.id]}
        onchange={(enabled) => {
          backupSections = {
            ...backupSections,
            [section.id]: enabled,
          };
        }}
      />
    {/each}
  </div>
  <div class="settings-page__backup-buttons">
    <Button
      onclick={() => void handleExportBackup()}
      disabled={backupExporting || backupImporting}
    >
      {backupExporting ? "Exporting…" : "Export backup"}
    </Button>
    <input
      bind:this={backupFileInput}
      type="file"
      accept="application/json,.json"
      class="settings-page__backup-input"
      onchange={(event) => void handleImportBackupFile(event)}
    />
    <Button
      variant="ghost"
      disabled={backupExporting || backupImporting}
      onclick={() => backupFileInput?.click()}
    >
      {backupImporting ? "Importing…" : "Import backup"}
    </Button>
  </div>
</SettingsCard>
