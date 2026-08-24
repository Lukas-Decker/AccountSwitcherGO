<script lang="ts">
  /* Interface language, plus the ways to help with or credit the translations. */
  import { onMount } from "svelte";
  import { get } from "svelte/store";
  import * as PlatformService from "../../../../bindings/account-switcher/internal/platform/platformservice.js";
  import type { CrowdinTranslatorsList } from "../../../lib/crowdinTranslators";
  import { openExternalUrl } from "../../../lib/openExternalUrl";
  import { availableLocales, locale, setUserLanguage, t } from "../../../stores/i18n";
  import { offlineMode } from "../../../stores/offlineMode";
  import { openAlertNoButton } from "../../../stores/modal";
  import CrowdinTranslatorsModalBody from "../../modals/CrowdinTranslatorsModalBody.svelte";
  import { settingsIcons } from "./settingsIcons";
  import SettingsCard from "./SettingsCard.svelte";
  import SettingsRow from "./SettingsRow.svelte";
  import SettingsSelect from "./SettingsSelect.svelte";

  /**
   * Which translation links this build can offer. The project this app was
   * forked from ran its own Crowdin; this build points at nothing until one is
   * configured, so the links are hidden rather than opening a blank page.
   */
  let translationLinks: { projectUrl: string; creditsAvailable: boolean } = {
    projectUrl: "",
    creditsAvailable: false,
  };

  onMount(async () => {
    try {
      translationLinks = await PlatformService.GetTranslationLinks();
    } catch {
      // Leaving both off is the safe reading: no links beats broken links.
    }
  });

  function nameFor(code: string): string {
    const names = new Intl.DisplayNames([$locale.replace(/_/g, "-")], { type: "language" });
    return names.of(code.replace(/_/g, "-")) ?? code;
  }

  $: languageOptions = availableLocales.map((code) => ({
    value: code,
    label: `${code} - ${nameFor(code)}`,
  }));

  async function openCredits(): Promise<void> {
    let list: CrowdinTranslatorsList = { proofReaders: [], translators: [] };
    let loadError: string | undefined;

    if (get(offlineMode)) {
      loadError = "OFFLINE MODE";
    } else {
      try {
        list = await PlatformService.GetCrowdinTranslators();
      } catch {
        loadError = "Failed to load Crowdin supporters!";
      }
    }

    void openAlertNoButton({
      title: get(t)("Modal_Crowdin_Header"),
      bodyComponent: CrowdinTranslatorsModalBody,
      bodyProps: { list, loadError },
    });
  }
</script>

<SettingsCard
  title={$t("Settings_Header_Language")}
  icon={settingsIcons.language}
  keywords="locale translation"
>
  <SettingsRow label={$t("Header_ChooseLanguage")} controlId="settings-language">
    <SettingsSelect
      id="settings-language"
      label={$t("Header_ChooseLanguage")}
      options={languageOptions}
      value={$locale}
      fallbackLabel={nameFor($locale)}
      on:select={(e) => void setUserLanguage(e.detail)}
    />
  </SettingsRow>

  {#if translationLinks.projectUrl || translationLinks.creditsAvailable}
    <SettingsRow label={$t("Settings_Translations")} keywords="crowdin translate contribute credits">
      {#if translationLinks.projectUrl}
        <button
          type="button"
          class="settings-link"
          on:click={() => void openExternalUrl(translationLinks.projectUrl)}
        >
          {$t("Settings_HelpTranslate")}
        </button>
      {/if}
      {#if translationLinks.creditsAvailable}
        <button type="button" class="settings-link" on:click={() => void openCredits()}>
          {$t("Settings_ViewTranslators")}
        </button>
      {/if}
    </SettingsRow>
  {/if}
</SettingsCard>
