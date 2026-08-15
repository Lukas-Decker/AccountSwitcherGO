<script lang="ts">
  /* Where the user data lives, and the two ways to act on that folder. */
  import { writable } from "svelte/store";
  import { t } from "../../../stores/i18n";
  import { userDataPath } from "../../../lib/appSettingsModel";
  import { openMoveUserDataModal, openUserDataFolder } from "../../../lib/settingsOperations";
  import { settingsIcons } from "./settingsIcons";
  import SettingsCard from "./SettingsCard.svelte";
  import SettingsRow from "./SettingsRow.svelte";

  const moving = writable(false);
</script>

<SettingsCard
  title={$t("Settings_Header_Data")}
  icon={settingsIcons.data}
  keywords="storage folder path portable appdata"
>
  <SettingsRow
    label={$t("Settings_UserDataLocation")}
    stacked
    keywords="location move folder portable appdata"
  >
    <svelte:fragment slot="under">
      <span class="settings-path">{$userDataPath || "..."}</span>
    </svelte:fragment>
    <!-- These labels are bracketed in the translations, so they stay links. -->
    <button
      type="button"
      class="settings-link"
      disabled={$moving}
      on:click={() => void openMoveUserDataModal(moving, $userDataPath)}
    >
      {$t("Settings_SetDataLocation")}
    </button>
    <button type="button" class="settings-link" on:click={() => void openUserDataFolder()}>
      {$t("Settings_OpenUserDataFolder")}
    </button>
  </SettingsRow>
</SettingsCard>
