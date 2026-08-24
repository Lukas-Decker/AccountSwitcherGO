<script lang="ts">
  /* The games view: where its cover art comes from. */
  import { t } from "../../../stores/i18n";
  import {
    gameArtArchiveKey,
    igdbClientId,
    igdbClientSecret,
    saveGameArtArchiveKey,
    saveIgdbCredentials,
  } from "../../../lib/appSettingsModel";
  import { openExternalUrl } from "../../../lib/openExternalUrl";
  import { settingsIcons } from "./settingsIcons";
  import SettingsCard from "./SettingsCard.svelte";
  import SettingsRow from "./SettingsRow.svelte";

  const SIGNUP_URL = "https://www.steamgriddb.com/profile/preferences/api";
  const IGDB_SIGNUP_URL = "https://dev.twitch.tv/console/apps/create";
</script>

<SettingsCard
  title={$t("Settings_Header_Games")}
  icon={settingsIcons.games}
  keywords="games artwork art cover grid banner logo icon steamgriddb api key"
>
  <SettingsRow
    label={$t("Settings_GameArtArchive")}
    hint={$t("Settings_GameArtArchive_Note")}
    controlId="settings-art-archive"
    keywords="artwork art cover grid banner logo steamgriddb api key token"
    stacked
  >
    <input
      id="settings-art-archive"
      class="settings-input"
      type="text"
      spellcheck="false"
      autocomplete="off"
      placeholder={$t("Settings_GameArtArchive_Placeholder")}
      value={$gameArtArchiveKey}
      on:change={(e) => void saveGameArtArchiveKey(e.currentTarget.value)}
    />
    <button
      type="button"
      class="settings-link settings-link--inline"
      on:click={() => void openExternalUrl(SIGNUP_URL, { allowAnyHttps: true })}
    >
      {$t("Settings_GameArtArchive_GetKey")}
    </button>
  </SettingsRow>

  <SettingsRow
    label={$t("Settings_IGDBCredentials")}
    hint={$t("Settings_IGDBCredentials_Note")}
    controlId="settings-igdb-id"
    keywords="igdb twitch client id secret artwork cover catalogue"
    stacked
  >
    <input
      id="settings-igdb-id"
      class="settings-input"
      type="text"
      spellcheck="false"
      autocomplete="off"
      placeholder={$t("Settings_IGDBClientId_Placeholder")}
      value={$igdbClientId}
      on:change={(e) => void saveIgdbCredentials(e.currentTarget.value, $igdbClientSecret)}
    />
    <input
      id="settings-igdb-secret"
      class="settings-input settings-input--stacked"
      type="password"
      spellcheck="false"
      autocomplete="off"
      placeholder={$t("Settings_IGDBClientSecret_Placeholder")}
      value={$igdbClientSecret}
      on:change={(e) => void saveIgdbCredentials($igdbClientId, e.currentTarget.value)}
    />
    <button
      type="button"
      class="settings-link settings-link--inline"
      on:click={() => void openExternalUrl(IGDB_SIGNUP_URL, { allowAnyHttps: true })}
    >
      {$t("Settings_IGDBCredentials_GetKey")}
    </button>
  </SettingsRow>
</SettingsCard>

<style lang="scss">
  /* The link sits under the field it belongs to rather than beside it, so the
     row reads as one instruction: paste a key, and here is where to get one. */
  .settings-link--inline {
    margin-top: 0.375rem;
    align-self: flex-start;
  }

  /* The secret sits under the id it belongs to, so the pair reads as one
     credential rather than two unrelated fields. */
  .settings-input--stacked {
    margin-top: 0.375rem;
  }
</style>
