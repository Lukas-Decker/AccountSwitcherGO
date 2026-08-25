<script lang="ts">
  /* Version, the update check, and the channel it checks. */
  import { writable } from "svelte/store";
  import { t } from "../../../stores/i18n";
  import { formatAppVersion } from "../../../lib/checkForUpdates";
  import { onCheckForUpdates } from "../../../lib/settingsOperations";
  import { appVersion } from "../../../lib/appSettingsModel";
  import { settingsIcons } from "./settingsIcons";
  import SettingsCard from "./SettingsCard.svelte";
  import SettingsRow from "./SettingsRow.svelte";

  const checking = writable(false);
</script>

<SettingsCard
  title={$t("Settings_Header_Updates")}
  icon={settingsIcons.updates}
  keywords="version update release"
>
  <SettingsRow label={$t("Settings_Version")} keywords="about build">
    <span class="settings-value">{formatAppVersion($appVersion || "0.0.0")}</span>
    <button
      type="button"
      class="settings-btn"
      disabled={$checking}
      on:click={() => void onCheckForUpdates(checking)}
    >
      {$checking ? $t("Button_Loading") : $t("Button_CheckForUpdates")}
    </button>
  </SettingsRow>
</SettingsCard>
