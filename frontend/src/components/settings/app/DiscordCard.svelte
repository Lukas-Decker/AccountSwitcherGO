<script lang="ts">
  /* Rich presence, and which Discord application the presence is published as. */
  import { t } from "../../../stores/i18n";
  import { offlineMode } from "../../../stores/offlineMode";
  import { discordAppId, discordRpc, saveDiscordAppId } from "../../../lib/appSettingsModel";
  import { settingsIcons } from "./settingsIcons";
  import SettingsCard from "./SettingsCard.svelte";
  import SettingsRow from "./SettingsRow.svelte";
  import SettingsSwitch from "./SettingsSwitch.svelte";
</script>

<SettingsCard
  title={$t("Settings_Header_Discord")}
  icon={settingsIcons.discord}
  keywords="discord presence rpc status"
>
  <SettingsRow
    label={$t("Settings_DiscordRpc")}
    controlId="settings-discord-rpc"
    disabled={$offlineMode}
    hint={$offlineMode ? $t("Settings_OfflineMode") : ""}
    keywords="rich presence"
  >
    <SettingsSwitch
      id="settings-discord-rpc"
      toggle={discordRpc}
      label={$t("Settings_DiscordRpc")}
      disabled={$offlineMode}
    />
  </SettingsRow>

  <SettingsRow
    label={$t("Settings_DiscordAppId")}
    hint={$t("Settings_DiscordAppId_Hint")}
    controlId="settings-discord-app-id"
    stacked
    disabled={$offlineMode}
    keywords="application id developer portal"
  >
    <input
      id="settings-discord-app-id"
      class="settings-input"
      type="text"
      inputmode="numeric"
      spellcheck="false"
      autocomplete="off"
      placeholder="000000000000000000"
      value={$discordAppId}
      disabled={$offlineMode}
      on:change={(e) => void saveDiscordAppId(e.currentTarget.value)}
    />
  </SettingsRow>
</SettingsCard>
