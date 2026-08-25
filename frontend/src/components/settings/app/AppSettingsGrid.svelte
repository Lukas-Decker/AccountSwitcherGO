<script lang="ts">
  /*
    The app settings, as a grid of cards.

    Columns come from auto-fit rather than breakpoints, so the same markup is
    one column in a narrow window and three in a wide one. Cards report whether
    they survived the current search, which is how the "nothing found" line
    knows to appear.
  */
  import { onMount } from "svelte";
  import { writable } from "svelte/store";
  import { t } from "../../../stores/i18n";
  import { hydrateAppSettings } from "../../../lib/appSettingsModel";
  import { masonryGrid } from "../../../lib/actions/masonryGrid";
  import { createMatchRegistry, provideCardRegistry, provideSettingsQuery } from "../../../lib/settingsSearch";
  import "../../../styles/SettingsGrid.scss";
  import AppearanceCard from "./AppearanceCard.svelte";
  import LanguageCard from "./LanguageCard.svelte";
  import WindowCard from "./WindowCard.svelte";
  import ContextMenuCard from "./ContextMenuCard.svelte";
  import GamesCard from "./GamesCard.svelte";
  import PrivacyCard from "./PrivacyCard.svelte";
  import SecurityCard from "./SecurityCard.svelte";
  import InputCard from "./InputCard.svelte";
  import DiscordCard from "./DiscordCard.svelte";
  import DataCard from "./DataCard.svelte";
  import UpdatesCard from "./UpdatesCard.svelte";
  import AdvancedCard from "./AdvancedCard.svelte";

  /** Heading above the grid. The platform settings page passes its own. */
  export let title = "";

  const query = writable("");
  provideSettingsQuery(query);

  const cards = createMatchRegistry();
  provideCardRegistry(cards);

  const anyCardVisible = cards.anyMatched;

  let searchInput: HTMLInputElement | undefined;

  function clearQuery(): void {
    query.set("");
    searchInput?.focus();
  }

  function onSearchKeyDown(e: KeyboardEvent): void {
    if (e.key === "Escape" && $query) {
      // Clear the filter first; a second press leaves the page.
      e.preventDefault();
      e.stopPropagation();
      clearQuery();
    }
  }

  onMount(() => {
    void hydrateAppSettings();
  });
</script>

<div class="settings-page">
  <div class="settings-page__head">
    <h1 class="settings-page__title">{title || $t("Settings_Header_AppWide")}</h1>
    <div class="settings-search">
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" aria-hidden="true">
        <path
          d="M10 3a7 7 0 105.2 11.7l4.5 4.6 1.4-1.4-4.6-4.5A7 7 0 0010 3zm0 2a5 5 0 110 10 5 5 0 010-10z"
        />
      </svg>
      <input
        type="search"
        bind:this={searchInput}
        bind:value={$query}
        placeholder={$t("Settings_Search")}
        aria-label={$t("Settings_Search")}
        spellcheck="false"
        autocomplete="off"
        on:keydown={onSearchKeyDown}
      />
      {#if $query}
        <button
          type="button"
          class="settings-search__clear"
          aria-label={$t("Settings_Search_Clear")}
          on:click={clearQuery}
        >
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" aria-hidden="true">
            <path d="M19 6.4L17.6 5 12 10.6 6.4 5 5 6.4l5.6 5.6L5 17.6 6.4 19l5.6-5.6 5.6 5.6 1.4-1.4-5.6-5.6z" />
          </svg>
        </button>
      {/if}
    </div>
  </div>

  <div class="settings-grid" use:masonryGrid>
    <AppearanceCard />
    <GamesCard />
    <ContextMenuCard />
    <LanguageCard />
    <WindowCard />
    <PrivacyCard />
    <SecurityCard />
    <InputCard />
    <DiscordCard />
    <DataCard />
    <UpdatesCard />
    <AdvancedCard />
  </div>

  {#if !$anyCardVisible}
    <p class="settings-empty">{$t("Settings_Search_NoResults", { query: $query })}</p>
  {/if}
</div>
