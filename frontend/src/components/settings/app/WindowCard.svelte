<script lang="ts">
  /* How the window and the tray icon behave. Several of these are Windows only. */
  import { t } from "../../../stores/i18n";
  import {
    desktopShortcut,
    exitToTray,
    isWindowsHost,
    minimizeOnSwitch,
    protocolEnabled,
    startProgramCentered,
    startTrayWithWindows,
  } from "../../../lib/appSettingsModel";
  import { settingsIcons } from "./settingsIcons";
  import SettingsCard from "./SettingsCard.svelte";
  import SettingsRow from "./SettingsRow.svelte";
  import SettingsSwitch from "./SettingsSwitch.svelte";

  const windowsHost = isWindowsHost();
</script>

<SettingsCard
  title={$t("Settings_Header_Window")}
  icon={settingsIcons.window}
  keywords="window tray startup shortcut"
>
  {#if windowsHost}
    <SettingsRow
      label={$t("Settings_Tray_StartWindows")}
      controlId="settings-start-tray"
      keywords="autostart boot"
    >
      <SettingsSwitch
        id="settings-start-tray"
        toggle={startTrayWithWindows}
        label={$t("Settings_Tray_StartWindows")}
      />
    </SettingsRow>

    <SettingsRow label={$t("Settings_ExitToTray")} controlId="settings-exit-tray" keywords="close minimise">
      <SettingsSwitch id="settings-exit-tray" toggle={exitToTray} label={$t("Settings_ExitToTray")} />
    </SettingsRow>
  {/if}

  <SettingsRow
    label={$t("Settings_MinimizeOnSwitch")}
    controlId="settings-minimize-switch"
    keywords="minimise hide"
  >
    <SettingsSwitch
      id="settings-minimize-switch"
      toggle={minimizeOnSwitch}
      label={$t("Settings_MinimizeOnSwitch")}
    />
  </SettingsRow>

  <SettingsRow
    label={$t("Settings_StartCentered")}
    controlId="settings-start-centered"
    keywords="position launch"
  >
    <SettingsSwitch
      id="settings-start-centered"
      toggle={startProgramCentered}
      label={$t("Settings_StartCentered")}
    />
  </SettingsRow>

  {#if windowsHost}
    <SettingsRow
      label={$t("Settings_DesktopShortcut")}
      controlId="settings-desktop-shortcut"
      keywords="icon desktop"
    >
      <SettingsSwitch
        id="settings-desktop-shortcut"
        toggle={desktopShortcut}
        label={$t("Settings_DesktopShortcut")}
      />
    </SettingsRow>

    <SettingsRow
      label={$t("Settings_Protocol")}
      controlId="settings-protocol"
      keywords="accswitcher url handler link"
    >
      <SettingsSwitch id="settings-protocol" toggle={protocolEnabled} label={$t("Settings_Protocol")} />
    </SettingsRow>
  {/if}
</SettingsCard>
