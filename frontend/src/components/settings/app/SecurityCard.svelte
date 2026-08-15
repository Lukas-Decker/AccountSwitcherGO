<script lang="ts">
  /*
    App password, encryption of the saved account data, and the two states that
    need the user to step in: a quarantine, and a restore that was cut short.
  */
  import { onMount } from "svelte";
  import { get, writable } from "svelte/store";
  import { t } from "../../../stores/i18n";
  import { pushToast } from "../../../stores/toast";
  import { formatToastWithError } from "../../../lib/formatWailsError";
  import { openConfirm, openPasswordSetupModal, openPrompt } from "../../../stores/modal";
  import {
    deleteQuarantine,
    disableSavedAccountEncryption,
    enableSavedAccountEncryption,
    isWeakPassword,
    listQuarantines,
    loadSecurityStatus,
    removeAppPassword,
    repairInterruptedRestore,
    retryQuarantineImport,
    securityStatus,
    setAppPassword,
    type SecurityQuarantineInfo,
  } from "../../../stores/security";
  import { settingsIcons } from "./settingsIcons";
  import SettingsCard from "./SettingsCard.svelte";
  import SettingsRow from "./SettingsRow.svelte";

  const busy = writable(false);
  let quarantines: SecurityQuarantineInfo[] = [];

  async function refresh(): Promise<void> {
    try {
      const status = await loadSecurityStatus();
      quarantines = status.quarantineCount > 0 ? await listQuarantines() : [];
    } catch {
      quarantines = [];
    }
  }

  function askPassword(title: string, body: string): Promise<string | null> {
    return openPrompt({
      title,
      body,
      inputType: "password",
      positiveLabel: $t("Ok"),
      negativeLabel: $t("Button_Cancel"),
    });
  }

  async function run(action: () => Promise<void>, failureKey = "Security_PasswordActionFailed"): Promise<void> {
    busy.set(true);
    try {
      await action();
      await refresh();
    } catch (e) {
      await refresh();
      pushToast({ type: "error", message: formatToastWithError($t(failureKey), e), duration: 8000 });
    } finally {
      busy.set(false);
    }
  }

  async function onSetPassword(): Promise<void> {
    if (get(busy)) return;
    const result = await openPasswordSetupModal({
      title: $t("Security_SetAppPassword"),
      positiveLabel: $t("Security_SetAppPassword"),
      negativeLabel: $t("Button_Cancel"),
    });
    if (!result) return;
    await run(async () => {
      await setAppPassword(result.password);
      pushToast({ type: "success", message: $t("Security_AppPasswordSet"), duration: 4000 });
    }, "Toast_SaveFailed");
  }

  async function onRemovePassword(): Promise<void> {
    if (get(busy)) return;
    const password = await askPassword(
      $t("Security_RemoveAppPassword"),
      $t(
        $securityStatus.savedAccountDataEncrypted
          ? "Security_RemovePasswordEncryptedBody"
          : "Security_CurrentPasswordBody",
      ),
    );
    if (password === null) return;
    await run(async () => {
      await removeAppPassword(password);
      pushToast({ type: "success", message: $t("Security_AppPasswordRemoved"), duration: 5000 });
    });
  }

  async function onToggleEncryption(): Promise<void> {
    if (get(busy) || $securityStatus.operationBusy) return;
    const next = !$securityStatus.savedAccountDataEncrypted;
    const password = await askPassword(
      $t(next ? "Security_EnableEncryption" : "Security_DisableEncryption"),
      $t(next ? "Security_EnableEncryptionBody" : "Security_DisableEncryptionBody"),
    );
    if (password === null) {
      await refresh();
      return;
    }
    if (next && isWeakPassword(password)) {
      const proceed = await openConfirm({
        title: $t("Security_WeakPasswordTitle"),
        body: $t("Security_WeakPasswordBody"),
        positiveLabel: $t("Security_WeakPasswordContinue"),
        negativeLabel: $t("Button_Cancel"),
        style: "okcancel",
      });
      if (!proceed) {
        await refresh();
        return;
      }
    }
    await run(async () => {
      if (next) {
        await enableSavedAccountEncryption(password);
      } else {
        await disableSavedAccountEncryption(password);
      }
      pushToast({
        type: "success",
        message: $t(next ? "Security_EncryptionEnabled" : "Security_EncryptionDisabled"),
        duration: 5000,
      });
    });
  }

  async function onRetryQuarantine(id: string): Promise<void> {
    const password = await askPassword($t("Security_QuarantineRetry"), $t("Security_QuarantineRetryBody"));
    if (password === null) return;
    await run(async () => {
      await retryQuarantineImport(id, password);
      pushToast({ type: "success", message: $t("Security_QuarantineRetryDone"), duration: 5000 });
    });
  }

  async function onDeleteQuarantine(id: string): Promise<void> {
    const confirmed = await openConfirm({
      title: $t("Security_QuarantineDelete"),
      body: $t("Security_QuarantineDeleteBody"),
      positiveLabel: $t("Security_QuarantineDelete"),
      negativeLabel: $t("Button_Cancel"),
      style: "okcancel",
    });
    if (!confirmed) return;
    await run(() => deleteQuarantine(id));
  }

  async function onRepairRestore(): Promise<void> {
    const confirmed = await openConfirm({
      title: $t("Security_InterruptedRestore_Title"),
      body: $t("Security_InterruptedRestore_Body"),
      positiveLabel: $t("Security_InterruptedRestore_Repair"),
      negativeLabel: $t("Security_InterruptedRestore_Later"),
      style: "yesno",
    });
    if (!confirmed) return;
    await run(async () => {
      await repairInterruptedRestore();
      pushToast({ type: "success", message: $t("Security_InterruptedRestore_Repaired"), duration: 5000 });
    }, "Security_InterruptedRestore_RepairFailed");
  }

  onMount(() => {
    void refresh();
  });
</script>

<SettingsCard
  title={$t("Settings_Header_Security")}
  icon={settingsIcons.security}
  keywords="password encryption lock"
>
  <SettingsRow label={$t("Settings_AppPassword")} keywords="password lock unlock">
    {#if $securityStatus.appPasswordSet}
      <button
        type="button"
        class="settings-btn settings-btn--danger"
        disabled={$busy}
        on:click={() => void onRemovePassword()}
      >
        {$t("Security_RemoveAppPassword")}
      </button>
    {:else}
      <button type="button" class="settings-btn" disabled={$busy} on:click={() => void onSetPassword()}>
        {$t("Security_SetAppPassword")}
      </button>
    {/if}
  </SettingsRow>

  {#if $securityStatus.appPasswordSet}
    <SettingsRow
      label={$t("Security_EncryptSavedAccountData")}
      keywords="encrypt cache saved data"
      disabled={$securityStatus.operationBusy}
    >
      <button
        type="button"
        class="settings-btn"
        class:settings-btn--active={$securityStatus.savedAccountDataEncrypted}
        aria-pressed={$securityStatus.savedAccountDataEncrypted}
        disabled={$busy || $securityStatus.operationBusy}
        on:click={() => void onToggleEncryption()}
      >
        {$t($securityStatus.savedAccountDataEncrypted ? "Security_DisableEncryption" : "Security_EnableEncryption")}
      </button>
    </SettingsRow>
  {/if}

  {#if $securityStatus.interruptedRestorePending}
    <SettingsRow
      label={$t("Security_InterruptedRestore_Title")}
      hint={$t("Security_InterruptedRestorePending")}
      stacked
      keywords="repair restore interrupted"
    >
      <button
        type="button"
        class="settings-btn"
        disabled={$busy}
        on:click={() => void onRepairRestore()}
      >
        {$t("Security_InterruptedRestore_Repair")}
      </button>
    </SettingsRow>
  {/if}

  {#each quarantines as quarantine (quarantine.id)}
    <SettingsRow
      label={$t("Security_QuarantineStatus", { count: quarantines.length })}
      hint={quarantine.accounts.join(", ")}
      stacked
      keywords="quarantine encrypted import"
    >
      <button
        type="button"
        class="settings-btn"
        disabled={$busy}
        on:click={() => void onRetryQuarantine(quarantine.id)}
      >
        {$t("Security_QuarantineRetry")}
      </button>
      <button
        type="button"
        class="settings-btn settings-btn--danger"
        disabled={$busy}
        on:click={() => void onDeleteQuarantine(quarantine.id)}
      >
        {$t("Security_QuarantineDelete")}
      </button>
    </SettingsRow>
  {/each}
</SettingsCard>
