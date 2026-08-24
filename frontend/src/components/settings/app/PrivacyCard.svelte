<script lang="ts">
  /* What leaves the machine, and what a viewer or a screenshot can see. */
  import { t } from "../../../stores/i18n";
  import { streamerState } from "../../../stores/streamerMode";
  import {
    autoStreamerMode,
    hideFromScreenshots,
    offlineModeToggle,
    streamerMode,
  } from "../../../lib/appSettingsModel";
  import { settingsIcons } from "./settingsIcons";
  import SettingsCard from "./SettingsCard.svelte";
  import SettingsRow from "./SettingsRow.svelte";
  import SettingsSwitch from "./SettingsSwitch.svelte";
</script>

<SettingsCard
  title={$t("Settings_Header_Privacy")}
  icon={settingsIcons.privacy}
  keywords="privacy streaming offline capture"
>
  <SettingsRow
    label={$t("Settings_OfflineMode")}
    controlId="settings-offline"
    keywords="network internet"
  >
    <SettingsSwitch id="settings-offline" toggle={offlineModeToggle} label={$t("Settings_OfflineMode")} />
  </SettingsRow>

  <SettingsRow
    label={$t("Settings_StreamerMode")}
    hint={$t("Settings_StreamerMode_Tooltip")}
    controlId="settings-streamer"
    keywords="obs twitch stream hide names"
  >
    <SettingsSwitch id="settings-streamer" toggle={streamerMode} label={$t("Settings_StreamerMode")} />
  </SettingsRow>

  <SettingsRow
    label={$t("Settings_AutoStreamerMode")}
    hint={$t("Settings_AutoStreamerMode_Tooltip")}
    controlId="settings-auto-streamer"
    keywords="obs xsplit streamlabs automatic"
  >
    <SettingsSwitch
      id="settings-auto-streamer"
      toggle={autoStreamerMode}
      label={$t("Settings_AutoStreamerMode")}
    />
    <svelte:fragment slot="under">
      {#if $streamerState.autoEnabled && $streamerState.autoActive}
        <p class="setting-row__hint">
          {$t("Settings_AutoStreamerMode_Active", { app: $streamerState.detectedExe })}
        </p>
      {/if}
    </svelte:fragment>
  </SettingsRow>

  <SettingsRow
    label={$t("Settings_HideFromScreenshots")}
    hint={$t("Settings_HideFromScreenshots_Tooltip")}
    controlId="settings-hide-screenshots"
    keywords="screen capture recording share"
  >
    <SettingsSwitch
      id="settings-hide-screenshots"
      toggle={hideFromScreenshots}
      label={$t("Settings_HideFromScreenshots")}
    />
  </SettingsRow>
</SettingsCard>
