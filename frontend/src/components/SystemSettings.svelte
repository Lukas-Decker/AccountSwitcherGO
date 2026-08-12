<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import { get, writable } from "svelte/store";
  import { t } from "../stores/i18n";
  import { pushToast } from "../stores/toast";
  import { formatToastWithError } from "../lib/formatWailsError";
  import { tooltip } from "../lib/actions/tooltip";
  import * as PlatformService from "../../bindings/account-switcher/internal/platform/platformservice.js";
  import { offlineMode, setUserOfflineMode } from "../stores/offlineMode";
  import { setAutoStreamerMode, setStreamerMode, streamerState } from "../stores/streamerMode";
  import { openConfirm, openFeedbackModal, openPasswordSetupModal, openPrompt } from "../stores/modal";
  import { animationsEnabled, loadAnimationsEnabled, setAnimationsEnabled } from "../stores/animationSettings";
  import {
    deleteQuarantine,
    disableSavedAccountEncryption,
    enableSavedAccountEncryption,
    isWeakPassword,
    listQuarantines,
    loadSecurityStatus,
    repairInterruptedRestore,
    removeAppPassword,
    retryQuarantineImport,
    securityStatus,
    setAppPassword,
    type SecurityQuarantineInfo,
  } from "../stores/security";
  import {
    commandPaletteHotkey,
    formatCommandPaletteHotkeyEvent,
    loadCommandPaletteHotkey,
    normalizeCommandPaletteHotkey,
    setCommandPaletteHotkey,
  } from "../stores/commandPalette";
  import {
    applyControllerSupportEnabled,
    loadControllerSupportEnabled,
    setControllerSupportEnabled,
  } from "../stores/controllerSupport";
  import { formatAppVersion } from "../lib/checkForUpdates";
  import { createToggle } from "../lib/useToggleSetting";
  import { openMoveUserDataModal, openUserDataFolder, onCheckForUpdates } from "../lib/settingsOperations";

  let isWindows = false;
  let currentVersion = "";
  let userDataPath = "";

  const userDataMoveLoading = writable(false);
  const updateCheckLoading = writable(false);
  const offlineLoading = writable(false);
  const securityLoading = writable(false);
  let quarantines: SecurityQuarantineInfo[] = [];
  let commandPaletteHotkeyCaptureActive = false;

  function stopCommandPaletteHotkeyCapture(): void {
    if (!commandPaletteHotkeyCaptureActive) return;
    commandPaletteHotkeyCaptureActive = false;
    window.removeEventListener("keydown", onCommandPaletteHotkeyCaptureKeydown, true);
  }

  function startCommandPaletteHotkeyCapture(): void {
    if (commandPaletteHotkeyCaptureActive) return;
    commandPaletteHotkeyCaptureActive = true;
    window.addEventListener("keydown", onCommandPaletteHotkeyCaptureKeydown, true);
  }

  function toggleCommandPaletteHotkeyCapture(): void {
    if (commandPaletteHotkeyCaptureActive) {
      stopCommandPaletteHotkeyCapture();
      return;
    }
    startCommandPaletteHotkeyCapture();
  }

  function onCommandPaletteHotkeyCaptureKeydown(e: KeyboardEvent): void {
    if (!commandPaletteHotkeyCaptureActive) return;
    e.preventDefault();
    e.stopPropagation();
    if (e.key === "Escape") {
      stopCommandPaletteHotkeyCapture();
      return;
    }
    const next = formatCommandPaletteHotkeyEvent(e);
    if (!next) return;
    stopCommandPaletteHotkeyCapture();
    void setCommandPaletteHotkey(next);
  }

  const exitToTray = createToggle(
    () => PlatformService.GetExitToTray(),
    (v) => PlatformService.SetExitToTray(v),
    get(t)("Settings_ExitToTray"),
  );

  const minimizeOnSwitch = createToggle(
    () => PlatformService.GetMinimizeOnSwitch(),
    (v) => PlatformService.SetMinimizeOnSwitch(v),
    get(t)("Settings_MinimizeOnSwitch"),
  );

  const startTrayWithWindows = createToggle(
    () => PlatformService.GetStartTrayWithWindows(),
    (v) => PlatformService.SetStartTrayWithWindows(v),
    get(t)("Settings_Tray_StartWindows"),
  );

  const startProgramCentered = createToggle(
    () => PlatformService.GetStartProgramCentered(),
    (v) => PlatformService.SetStartProgramCentered(v),
    get(t)("Settings_StartCentered"),
  );

  // Named ...Toggle so the local settings control does not shadow the streamerMode
  // store the rest of the app reads.
  const streamerModeToggle = createToggle(
    () => PlatformService.GetStreamerMode(),
    (v) => setStreamerMode(v),
    get(t)("Settings_StreamerMode"),
  );

  const autoStreamerModeToggle = createToggle(
    () => PlatformService.GetAutoStreamerMode(),
    (v) => setAutoStreamerMode(v),
    get(t)("Settings_AutoStreamerMode"),
  );

  const protocol = createToggle(
    () => PlatformService.GetProtocolEnabled(),
    (v) => PlatformService.SetProtocolEnabled(v),
    get(t)("Settings_Protocol"),
  );
  protocol.toggle = async () => {
    if (get(protocol.loading)) return;
    const next = !get(protocol.value);
    protocol.loading.set(true);
    try {
      await PlatformService.SetProtocolEnabled(next);
      protocol.value.set(next);
      pushToast({ type: "success", message: next ? $t("Toast_ProtocolEnabled") : $t("Toast_ProtocolDisabled"), duration: 6000 });
    } catch (e) {
      pushToast({ type: "error", message: formatToastWithError($t("Toast_SaveFailed"), e), duration: 8000 });
    } finally {
      protocol.loading.set(false);
    }
  };

  const animations = createToggle(
    async () => { await loadAnimationsEnabled(); return get(animationsEnabled); },
    (v) => setAnimationsEnabled(v),
    get(t)("Settings_AnimationsEnabled"),
  );
  animations.toggle = async () => {
    if (get(animations.loading)) return;
    const next = !get(animations.value);
    animations.value.set(next);
    animations.loading.set(true);
    try {
      await setAnimationsEnabled(next);
      pushToast({ type: "success", message: get(t)("Toast_SavedItem", { item: get(t)("Settings_AnimationsEnabled") }), duration: 3000 });
    } catch (e) {
      animations.value.set(get(animationsEnabled));
      pushToast({ type: "error", message: formatToastWithError($t("Toast_SaveFailed"), e), duration: 8000 });
    } finally {
      animations.loading.set(false);
    }
  };

  const controllerSupport = createToggle(
    () => loadControllerSupportEnabled(),
    (v) => setControllerSupportEnabled(v),
    get(t)("Settings_ControllerSupport"),
  );

  const desktopHomeShortcut = createToggle(
    () => PlatformService.GetDesktopHomeShortcutExists(),
    (v) => PlatformService.SetDesktopHomeShortcut(v),
    get(t)("Settings_DesktopShortcut"),
  );
  async function refreshDesktopShortcutState(): Promise<void> {
    try {
      desktopHomeShortcut.value.set(await PlatformService.GetDesktopHomeShortcutExists());
    } catch {
      desktopHomeShortcut.value.set(false);
    }
  }
  desktopHomeShortcut.toggle = async () => {
    if (get(desktopHomeShortcut.loading)) return;
    const next = !get(desktopHomeShortcut.value);
    desktopHomeShortcut.loading.set(true);
    try {
      await PlatformService.SetDesktopHomeShortcut(next);
      await refreshDesktopShortcutState();
      pushToast({ type: "success", message: get(t)("Toast_SavedItem", { item: get(t)("Settings_DesktopShortcut") }), duration: 4000 });
    } catch (e) {
      pushToast({ type: "error", message: formatToastWithError($t("Toast_SaveFailed"), e), duration: 8000 });
    } finally {
      desktopHomeShortcut.loading.set(false);
    }
  };

  const discordRpc = createToggle(
    () => PlatformService.GetDiscordRpc(),
    (v) => PlatformService.SetDiscordRpc(v),
    get(t)("Settings_DiscordRpc"),
    () => !get(offlineMode),
  );
  discordRpc.toggle = async () => {
    if (get(discordRpc.loading) || get(offlineMode)) return;
    const next = !get(discordRpc.value);
    discordRpc.loading.set(true);
    try {
      await PlatformService.SetDiscordRpc(next);
      discordRpc.value.set(next);
      pushToast({ type: "success", message: get(t)("Toast_SavedItem", { item: get(t)("Settings_DiscordRpc") }), duration: 4000 });
    } catch (e) {
      pushToast({ type: "error", message: formatToastWithError($t("Toast_SaveFailed"), e), duration: 8000 });
    } finally {
      discordRpc.loading.set(false);
    }
  };

  const prereleaseUpdates = createToggle(
    () => PlatformService.GetPrereleaseUpdates(),
    (v) => PlatformService.SetPrereleaseUpdates(v),
    get(t)("Settings_PrereleaseUpdates"),
  );

  type SettingsSnapshot = Awaited<ReturnType<typeof PlatformService.ReadSettings>>;

  // Discord names the presence after the application the id belongs to, so this
  // is the only way to control what shows up in Discord.
  let discordAppId = "";

  async function saveDiscordAppId(value: string): Promise<void> {
    const next = value.trim();
    discordAppId = next;
    try {
      await PlatformService.SetDiscordAppID(next);
    } catch (e) {
      pushToast({ type: "error", message: formatToastWithError(get(t)("Toast_SaveFailed"), e), duration: 8000 });
    }
  }

  function applySettingsSnapshot(settings: SettingsSnapshot): void {
    offlineMode.set(settings.offlineMode);
    discordAppId = settings.discordAppId ?? "";
    protocol.value.set(settings.protocolEnabled);
    exitToTray.value.set(settings.exitToTray);
    prereleaseUpdates.value.set(settings.prereleaseUpdates);
    discordRpc.value.set(settings.discordRpc);
    minimizeOnSwitch.value.set(settings.minimizeOnSwitch);
    startTrayWithWindows.value.set(settings.startTrayWithWindows);
    startProgramCentered.value.set(settings.startProgramCentered);
    streamerModeToggle.value.set(settings.streamerMode);
    autoStreamerModeToggle.value.set(settings.autoStreamerMode);
    animationsEnabled.set(settings.animationsEnabled);
    animations.value.set(settings.animationsEnabled);
    controllerSupport.value.set(applyControllerSupportEnabled(settings.controllerSupportEnabled));
    commandPaletteHotkey.set(normalizeCommandPaletteHotkey(settings.commandPaletteHotkey));
    currentVersion = settings.appVersion || "";
  }

  async function initSettingsIndividually(): Promise<void> {
    void protocol.init();
    void exitToTray.init();
    void prereleaseUpdates.init();
    void discordRpc.init();
    void PlatformService.GetDiscordAppID()
      .then((v) => { discordAppId = v || ""; })
      .catch(() => {});
    void minimizeOnSwitch.init();
    void startProgramCentered.init();
    void streamerModeToggle.init();
    void autoStreamerModeToggle.init();
    void animations.init();
    void controllerSupport.init();
    void loadCommandPaletteHotkey();
    void PlatformService.GetAppVersion()
      .then((v) => { currentVersion = v || ""; })
      .catch(() => { currentVersion = ""; });
    if (isWindows) {
      void startTrayWithWindows.init();
    }
  }

  async function hydrateSettings(): Promise<void> {
    try {
      applySettingsSnapshot(await PlatformService.ReadSettings());
    } catch {
      await initSettingsIndividually();
    }
  }

  async function toggleOfflineMode(): Promise<void> {
    if (get(offlineLoading)) return;
    const next = !get(offlineMode);
    offlineLoading.set(true);
    try {
      await setUserOfflineMode(next);
      if (next) {
        discordRpc.value.set(false);
      }
      pushToast({ type: "success", message: next ? $t("Toast_OfflineModeEnabled") : $t("Toast_OfflineModeDisabled"), duration: 6000 });
    } catch (e) {
      pushToast({ type: "error", message: formatToastWithError($t("Toast_SaveFailed"), e), duration: 8000 });
    } finally {
      offlineLoading.set(false);
    }
  }

  async function refreshSecurity(): Promise<void> {
    try {
      const status = await loadSecurityStatus();
      quarantines = status.quarantineCount > 0 ? await listQuarantines() : [];
    } catch {
      quarantines = [];
    }
  }

  async function promptSecurityPassword(title: string, body: string): Promise<string | null> {
    return openPrompt({
      title,
      body,
      inputType: "password",
      positiveLabel: $t("Ok"),
      negativeLabel: $t("Button_Cancel"),
    });
  }

  async function onSetAppPassword(): Promise<void> {
    if (get(securityLoading)) return;
    const result = await openPasswordSetupModal({
      title: $t("Security_SetAppPassword"),
      positiveLabel: $t("Security_SetAppPassword"),
      negativeLabel: $t("Button_Cancel"),
    });
    if (!result) return;
    securityLoading.set(true);
    try {
      await setAppPassword(result.password);
      await refreshSecurity();
      pushToast({ type: "success", message: $t("Security_AppPasswordSet"), duration: 4000 });
    } catch (e) {
      pushToast({ type: "error", message: formatToastWithError($t("Toast_SaveFailed"), e), duration: 8000 });
    } finally {
      securityLoading.set(false);
    }
  }

  async function onRemoveAppPassword(): Promise<void> {
    if (get(securityLoading)) return;
    const password = await promptSecurityPassword(
      $t("Security_RemoveAppPassword"),
      $t($securityStatus.savedAccountDataEncrypted ? "Security_RemovePasswordEncryptedBody" : "Security_CurrentPasswordBody"),
    );
    if (password === null) return;
    securityLoading.set(true);
    try {
      await removeAppPassword(password);
      await refreshSecurity();
      pushToast({ type: "success", message: $t("Security_AppPasswordRemoved"), duration: 5000 });
    } catch (e) {
      pushToast({ type: "error", message: formatToastWithError($t("Security_PasswordActionFailed"), e), duration: 8000 });
    } finally {
      securityLoading.set(false);
    }
  }

  async function onToggleSavedDataEncryption(next: boolean): Promise<void> {
    if (get(securityLoading)) return;
    const password = await promptSecurityPassword(
      next ? $t("Security_EnableEncryption") : $t("Security_DisableEncryption"),
      next ? $t("Security_EnableEncryptionBody") : $t("Security_DisableEncryptionBody"),
    );
    if (password === null) {
      await refreshSecurity();
      return;
    }
    if (next && isWeakPassword(password)) {
      const ok = await openConfirm({
        title: $t("Security_WeakPasswordTitle"),
        body: $t("Security_WeakPasswordBody"),
        positiveLabel: $t("Security_WeakPasswordContinue"),
        negativeLabel: $t("Button_Cancel"),
        style: "okcancel",
      });
      if (!ok) {
        await refreshSecurity();
        return;
      }
    }
    securityLoading.set(true);
    try {
      if (next) {
        await enableSavedAccountEncryption(password);
      } else {
        await disableSavedAccountEncryption(password);
      }
      await refreshSecurity();
      pushToast({ type: "success", message: next ? $t("Security_EncryptionEnabled") : $t("Security_EncryptionDisabled"), duration: 5000 });
    } catch (e) {
      await refreshSecurity();
      pushToast({ type: "error", message: formatToastWithError($t("Security_PasswordActionFailed"), e), duration: 8000 });
    } finally {
      securityLoading.set(false);
    }
  }

  function onSavedDataEncryptionClick(e: MouseEvent): void {
    e.preventDefault();
    if (get(securityLoading) || $securityStatus.operationBusy) return;
    void onToggleSavedDataEncryption(!$securityStatus.savedAccountDataEncrypted);
  }

  async function onRetryQuarantine(id: string): Promise<void> {
    const password = await promptSecurityPassword($t("Security_QuarantineRetry"), $t("Security_QuarantineRetryBody"));
    if (password === null) return;
    securityLoading.set(true);
    try {
      await retryQuarantineImport(id, password);
      await refreshSecurity();
      pushToast({ type: "success", message: $t("Security_QuarantineRetryDone"), duration: 5000 });
    } catch (e) {
      pushToast({ type: "error", message: formatToastWithError($t("Security_PasswordActionFailed"), e), duration: 8000 });
    } finally {
      securityLoading.set(false);
    }
  }

  async function onDeleteQuarantine(id: string): Promise<void> {
    const ok = await openConfirm({
      title: $t("Security_QuarantineDelete"),
      body: $t("Security_QuarantineDeleteBody"),
      positiveLabel: $t("Security_QuarantineDelete"),
      negativeLabel: $t("Button_Cancel"),
      style: "okcancel",
    });
    if (!ok) return;
    securityLoading.set(true);
    try {
      await deleteQuarantine(id);
      await refreshSecurity();
    } finally {
      securityLoading.set(false);
    }
  }

  async function onRepairInterruptedRestore(): Promise<void> {
    const ok = await openConfirm({
      title: $t("Security_InterruptedRestore_Title"),
      body: $t("Security_InterruptedRestore_Body"),
      positiveLabel: $t("Security_InterruptedRestore_Repair"),
      negativeLabel: $t("Security_InterruptedRestore_Later"),
      style: "yesno",
    });
    if (!ok) return;
    securityLoading.set(true);
    try {
      await repairInterruptedRestore();
      await refreshSecurity();
      pushToast({ type: "success", message: $t("Security_InterruptedRestore_Repaired"), duration: 5000 });
    } catch (e) {
      pushToast({ type: "error", message: formatToastWithError($t("Security_InterruptedRestore_RepairFailed"), e), duration: 8000 });
    } finally {
      securityLoading.set(false);
    }
  }

  onMount(() => {
    isWindows = /windows/i.test(navigator.userAgent) || /win32/i.test(navigator.userAgent);
    void hydrateSettings();
    void PlatformService.GetUserDataLocation()
      .then((v) => { userDataPath = v || ""; })
      .catch(() => { userDataPath = ""; });
    void refreshSecurity();
    if (isWindows) {
      void desktopHomeShortcut.init();
    }
  });

  onDestroy(() => {
    stopCommandPaletteHotkeyCapture();
  });
</script>

<h2 class="SettingsHeader">{$t("Settings_Header_System")}</h2>

<div class="multilineSetting">
  <span>{$t("Settings_CurrentDataLocation", { path: userDataPath || "…" })}</span>
  <span>
    <button
      type="button"
      class="fancyLink"
      disabled={$userDataMoveLoading}
      on:click={() => void openMoveUserDataModal(userDataMoveLoading, userDataPath)}
    >{$t("Settings_SetDataLocation")}</button>
    <button
      type="button"
      class="fancyLink"
      on:click={() => void openUserDataFolder()}
    >{$t("Settings_OpenUserDataFolder")}</button>
    </span>
</div>

<div class="security-settings">
  <div class="rowDropdown security-password-row">
    <span>{$t("Settings_Header_Security")}</span>
    {#if $securityStatus.appPasswordSet}
      <span class="security-password-controls">
        <button
          type="button"
          class="btnicontext"
          disabled={$securityLoading}
          on:click={() => void onRemoveAppPassword()}
        >
          {$t("Security_RemoveAppPassword")}
        </button>
        <span class="security-encryption-inline">
          <span class="form-check">
            <input
              id="security-encrypt-cache"
              type="checkbox"
              checked={$securityStatus.savedAccountDataEncrypted}
              disabled={$securityLoading || $securityStatus.operationBusy}
              on:click={onSavedDataEncryptionClick}
            />
            <label class="form-check-label" for="security-encrypt-cache"></label>
          </span>
          <label for="security-encrypt-cache">{$t("Security_EncryptSavedAccountData")}</label>
        </span>
      </span>
    {:else}
      <button
        type="button"
        class="btnicontext"
        disabled={$securityLoading}
        on:click={() => void onSetAppPassword()}
      >
        {$t("Security_SetAppPassword")}
      </button>
    {/if}
  </div>

  {#if $securityStatus.interruptedRestorePending}
    <div class="multilineSetting security-warning">
      <span>{$t("Security_InterruptedRestorePending")}</span>
      <button type="button" class="btnicontext" disabled={$securityLoading} on:click={() => void onRepairInterruptedRestore()}>
        {$t("Security_InterruptedRestore_Repair")}
      </button>
    </div>
  {/if}

  {#if quarantines.length > 0}
    <div class="multilineSetting security-warning">
      <span>{$t("Security_QuarantineStatus", { count: quarantines.length })}</span>
      {#each quarantines as q}
        <span class="security-quarantine-row">
          <span>{q.accounts.join(", ")}</span>
          <button type="button" class="btnicontext" disabled={$securityLoading} on:click={() => void onRetryQuarantine(q.id)}>
            {$t("Security_QuarantineRetry")}
          </button>
          <button type="button" class="btnicontext" disabled={$securityLoading} on:click={() => void onDeleteQuarantine(q.id)}>
            {$t("Security_QuarantineDelete")}
          </button>
        </span>
      {/each}
    </div>
  {/if}
</div>

<div class="rowSetting">
  <div class="form-check">
    <input id="gs-offline" type="checkbox" checked={$offlineMode} disabled={$offlineLoading} on:change={() => void toggleOfflineMode()} />
    <label class="form-check-label" for="gs-offline"></label>
  </div>
  <label for="gs-offline" use:tooltip={$t("Settings_OfflineMode")}>{$t("Settings_OfflineMode")}</label>
</div>

<div class="rowSetting">
  <div class="form-check">
    <input id="gs-streamer-mode" type="checkbox" checked={$streamerModeToggle.value} disabled={$streamerModeToggle.loading} on:change={() => void streamerModeToggle.toggle()} />
    <label class="form-check-label" for="gs-streamer-mode"></label>
  </div>
  <label for="gs-streamer-mode" use:tooltip={$t("Settings_StreamerMode_Tooltip")}>{$t("Settings_StreamerMode")}</label>
</div>

<div class="rowSetting">
  <div class="form-check">
    <input id="gs-auto-streamer-mode" type="checkbox" checked={$autoStreamerModeToggle.value} disabled={$autoStreamerModeToggle.loading} on:change={() => void autoStreamerModeToggle.toggle()} />
    <label class="form-check-label" for="gs-auto-streamer-mode"></label>
  </div>
  <label for="gs-auto-streamer-mode" use:tooltip={$t("Settings_AutoStreamerMode_Tooltip")}>{$t("Settings_AutoStreamerMode")}</label>
</div>

{#if $streamerState.autoEnabled && $streamerState.autoActive}
  <p class="streamer-active-note">{$t("Settings_AutoStreamerMode_Active", { app: $streamerState.detectedExe })}</p>
{/if}

<div class="rowSetting">
  <div class="form-check">
    <input id="gs-min-switch" type="checkbox" checked={$minimizeOnSwitch.value} disabled={$minimizeOnSwitch.loading} on:change={() => void minimizeOnSwitch.toggle()} />
    <label class="form-check-label" for="gs-min-switch"></label>
  </div>
  <label for="gs-min-switch" use:tooltip={$t("Settings_MinimizeOnSwitch")}>{$t("Settings_MinimizeOnSwitch")}</label>
</div>

{#if isWindows}
  <div class="rowSetting">
    <div class="form-check">
      <input id="gs-start-tray-win" type="checkbox" checked={$startTrayWithWindows.value} disabled={$startTrayWithWindows.loading} on:change={() => void startTrayWithWindows.toggle()} />
      <label class="form-check-label" for="gs-start-tray-win"></label>
    </div>
    <label for="gs-start-tray-win" use:tooltip={$t("Settings_Tray_StartWindows")}>{$t("Settings_Tray_StartWindows")}</label>

    <div class="form-check">
      <input id="gs-exit-tray" type="checkbox" checked={$exitToTray.value} disabled={$exitToTray.loading} on:change={() => void exitToTray.toggle()} />
      <label class="form-check-label" for="gs-exit-tray"></label>
    </div>
    <label for="gs-exit-tray" use:tooltip={$t("Settings_ExitToTray")}>{$t("Settings_ExitToTray")}</label>
  </div>

  <div class="rowSetting">
    <div class="form-check">
      <input id="gs-protocol" type="checkbox" checked={$protocol.value} disabled={$protocol.loading} on:change={() => void protocol.toggle()} />
      <label class="form-check-label" for="gs-protocol"></label>
    </div>
    <label for="gs-protocol" use:tooltip={$t("Settings_Protocol")}>{$t("Settings_Protocol")}</label>
  </div>
{/if}

<div class="rowSetting">
  <div class="form-check">
    <input id="gs-start-centered" type="checkbox" checked={$startProgramCentered.value} disabled={$startProgramCentered.loading} on:change={() => void startProgramCentered.toggle()} />
    <label class="form-check-label" for="gs-start-centered"></label>
  </div>
  <label for="gs-start-centered" use:tooltip={$t("Settings_StartCentered")}>{$t("Settings_StartCentered")}</label>
</div>

<div class="rowSetting">
  <div class="form-check">
    <input type="checkbox" id="settings-animations" checked={$animations.value} disabled={$animations.loading} on:change={() => void animations.toggle()} />
    <label class="form-check-label" for="settings-animations"></label>
  </div>
  <label for="settings-animations">{$t("Settings_AnimationsEnabled")}</label>
</div>

<div class="rowSetting">
  <div class="form-check">
    <input
      type="checkbox"
      id="settings-controller-support"
      checked={$controllerSupport.value}
      disabled={$controllerSupport.loading}
      on:change={() => void controllerSupport.toggle()}
    />
    <label class="form-check-label" for="settings-controller-support"></label>
  </div>
  <label for="settings-controller-support">{$t("Settings_ControllerSupport")}</label>
</div>

{#if isWindows}
  <div class="rowSetting">
    <div class="form-check">
      <input id="gs-desktop-home" type="checkbox" checked={$desktopHomeShortcut.value} disabled={$desktopHomeShortcut.loading} on:change={() => void desktopHomeShortcut.toggle()} />
      <label class="form-check-label" for="gs-desktop-home"></label>
    </div>
    <label for="gs-desktop-home">{$t("Settings_DesktopShortcut")}</label>
  </div>
{/if}

<div class="rowDropdown hotkey-row">
  <span>{$t("Settings_CommandPaletteHotkey")}</span>
  <button
    type="button"
    class="btnicontext hotkey-button"
    class:capturing={commandPaletteHotkeyCaptureActive}
    aria-pressed={commandPaletteHotkeyCaptureActive}
    on:click={toggleCommandPaletteHotkeyCapture}
  >
    {commandPaletteHotkeyCaptureActive ? $t("Settings_CommandPaletteHotkey_Prompt") : $commandPaletteHotkey}
  </button>
</div>

<div class="rowDropdown version-row">
  <span>{formatAppVersion(currentVersion || "0.0.0")}</span>
  <button
    type="button"
    class="btnicontext"
    disabled={$updateCheckLoading}
    on:click={() => void onCheckForUpdates(updateCheckLoading)}
  >
    {$t("Button_CheckForUpdates")}
  </button>
  <button type="button" class="btnicontext" on:click={() => void openFeedbackModal({ mode: "suggestion" })}>
    {$t("Settings_SuggestFeature")}
  </button>
  <div>
    <div class="form-check">
      <input
        id="settings-prerelease-updates"
        type="checkbox"
        checked={$prereleaseUpdates.value}
        disabled={$prereleaseUpdates.loading}
        on:change={() => void prereleaseUpdates.toggle()}
      />
      <label class="form-check-label" for="settings-prerelease-updates"></label>
    </div>
    <label for="settings-prerelease-updates">{$t("Settings_PrereleaseUpdates")}</label>
  </div>
</div>

<h2 class="SettingsHeader">{$t("Settings_DiscordRpc")}</h2>

<div class="rowSetting">
  <div class="form-check">
    <input id="gs-discord-rpc" type="checkbox" checked={$discordRpc.value} disabled={$discordRpc.loading || $offlineMode} on:change={() => void discordRpc.toggle()} />
    <label class="form-check-label" for="gs-discord-rpc"></label>
  </div>
  <label for="gs-discord-rpc">{$t("Settings_DiscordRpc")}</label>
</div>

<div class="rowSetting discord-appid-row">
  <label for="gs-discord-app-id">{$t("Settings_DiscordAppId")}</label>
  <input
    id="gs-discord-app-id"
    class="modal-input discord-appid-input"
    type="text"
    inputmode="numeric"
    spellcheck="false"
    autocomplete="off"
    placeholder="000000000000000000"
    value={discordAppId}
    disabled={$offlineMode}
    on:change={(e) => void saveDiscordAppId(e.currentTarget.value)}
  />
</div>
<p class="discord-appid-hint">{$t("Settings_DiscordAppId_Hint")}</p>

<style lang="scss">
  .discord-appid-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
  }

  .discord-appid-input {
    flex: 1 1 14rem;
    min-width: 10rem;
    max-width: 22rem;
    padding: 0.35rem 0.5rem;
    font-variant-numeric: tabular-nums;
  }

  .discord-appid-hint {
    margin: 0.15rem 0 0.75rem;
    font-size: 0.85rem;
    opacity: 0.75;
  }

  .streamer-active-note {
    margin: 0.15rem 0 0.75rem;
    font-size: 0.85rem;
    opacity: 0.75;
  }

  button:not(.fancyLink) {
    position: relative;
    height: 38px;
  }

  .version-row {
    margin-top: 0.25rem;
    flex-wrap: wrap;
  }

  .hotkey-row {
    gap: 0.75rem;
  }

  .security-settings {
    display: grid;
    gap: 0.25rem;
    margin-bottom: 0.35rem;
  }

  .security-password-row {
    align-items: center;
  }

  .security-password-controls,
  .security-encryption-inline {
    display: inline-flex;
    align-items: center;
    gap: 0.65rem;
    flex-wrap: wrap;
    justify-content: flex-end;
  }

  .security-encryption-inline {
    gap: 0.4rem;
  }

  .security-warning {
    color: var(--whiteSecondary);
  }

  .security-quarantine-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .hotkey-button {
    min-width: 7rem;
  }

  .hotkey-button.capturing {
    background: var(--accent);
    color: var(--text-on-bright-bg);
  }
</style>
