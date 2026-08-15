<script lang="ts">
  /*
    Settings that are only useful when something has gone wrong, or when the
    same administrator prompt has been answered one time too many.
  */
  import { t } from "../../../stores/i18n";
  import { debugLogging, isWindowsHost, skipElevatePrompt } from "../../../lib/appSettingsModel";
  import { settingsIcons } from "./settingsIcons";
  import SettingsCard from "./SettingsCard.svelte";
  import SettingsRow from "./SettingsRow.svelte";
  import SettingsSwitch from "./SettingsSwitch.svelte";

  const windowsHost = isWindowsHost();
</script>

<SettingsCard
  title={$t("Settings_Header_Advanced")}
  icon={settingsIcons.advanced}
  keywords="advanced troubleshooting"
>
  <SettingsRow
    label={$t("Settings_DebugLogging")}
    hint={$t("Settings_DebugLogging_Hint")}
    controlId="settings-debug-logging"
    keywords="log diagnostics console troubleshooting"
  >
    <SettingsSwitch
      id="settings-debug-logging"
      toggle={debugLogging}
      label={$t("Settings_DebugLogging")}
    />
  </SettingsRow>

  {#if windowsHost}
    <SettingsRow
      label={$t("Settings_SkipElevatePrompt")}
      hint={$t("Settings_SkipElevatePrompt_Hint")}
      controlId="settings-skip-elevate"
      keywords="administrator elevation uac restart"
    >
      <SettingsSwitch
        id="settings-skip-elevate"
        toggle={skipElevatePrompt}
        label={$t("Settings_SkipElevatePrompt")}
      />
    </SettingsRow>
  {/if}
</SettingsCard>
