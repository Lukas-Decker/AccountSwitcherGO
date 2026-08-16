<script lang="ts">
  import { onMount } from "svelte";
  // The checkbox pair below is drawn by the global form-check rules, which
  // only pages that import this sheet get; a cold deep-link needs it too.
  import "../styles/Settings.scss";
  import AccountCardEditor from "../components/accountcard/AccountCardEditor.svelte";
  import { appBarTitle, previousPage, route } from "../stores/nav";
  import { t } from "../stores/i18n";
  import { pushToast } from "../stores/toast";
  import { formatToastWithError } from "../lib/formatWailsError";
  import { accountCardConfig, setAccountCardConfig } from "../stores/accountCard";
  import {
    GetPlatformSettings,
    SavePlatformSettings,
  } from "../../bindings/account-switcher/internal/platform/platformservice.js";
  import { presetLayout } from "../lib/accountCard/presets";
  import {
    CORE_BLOCK_KINDS,
    validateConfig,
    type AccountCardConfig,
    type CardBlockKind,
  } from "../lib/accountCard/types";

  /** Absent means the card every platform falls back to. */
  export let platformName: string | undefined = undefined;

  /** Steam draws more than the shared set; see its adapter's cardBlocks. */
  const STEAM_BLOCKS: CardBlockKind[] = [
    ...CORE_BLOCK_KINDS,
    "accountLogin",
    "platformId",
    "statusLine",
    "badges",
  ];

  let platformSettings: Record<string, unknown> | null = null;
  let loading = false;
  let loadError = "";

  $: isPlatform = !!platformName;
  $: available = platformName === "Steam" ? STEAM_BLOCKS : CORE_BLOCK_KINDS;

  $: appBarTitle.set(
    isPlatform
      ? $t("CardEditor_TitleForPlatform", { platform: platformName ?? "" })
      : $t("CardEditor_Title"),
  );

  $: route.set(isPlatform ? { page: "card-editor", platformName } : { page: "card-editor" });
  $: previousPage.set(
    isPlatform ? { page: "platform-settings", platformName: platformName! } : { page: "settings" },
  );

  $: overrideEnabled = platformSettings?.AccountCardCustomizationEnabled === true;

  // While a platform is not overriding, the editor shows the global card, so the
  // toggle reads as "this platform, or the one everything else uses".
  $: shownConfig = !isPlatform
    ? $accountCardConfig
    : overrideEnabled
      ? validateConfig(platformSettings?.AccountCard, presetLayout("small"))
      : $accountCardConfig;

  onMount(() => {
    if (platformName) void loadPlatform(platformName);
  });

  async function loadPlatform(key: string): Promise<void> {
    loading = true;
    loadError = "";
    try {
      platformSettings = (await GetPlatformSettings(key)) as unknown as Record<string, unknown>;
    } catch (e) {
      loadError = formatToastWithError($t("Toast_SaveFailed"), e);
    } finally {
      loading = false;
    }
  }

  async function persistPlatform(): Promise<void> {
    if (!platformName || !platformSettings) return;
    try {
      await SavePlatformSettings(platformName, platformSettings as never);
    } catch (e) {
      pushToast({ type: "error", message: formatToastWithError($t("Toast_SaveFailed"), e), duration: 8000 });
    }
  }

  async function onChange(config: AccountCardConfig): Promise<void> {
    if (!isPlatform) {
      await setAccountCardConfig(config);
      return;
    }
    if (!overrideEnabled || !platformSettings) return;
    platformSettings = { ...platformSettings, AccountCard: config };
    await persistPlatform();
  }

  async function toggleOverride(): Promise<void> {
    if (!platformSettings) return;
    const next = !overrideEnabled;
    const patch: Record<string, unknown> = {
      ...platformSettings,
      AccountCardCustomizationEnabled: next,
    };
    // Seed from what is on screen the first time, and never clear what is
    // stored when switching off: the layout is meant to come back intact.
    if (next && !patch.AccountCard) patch.AccountCard = { ...$accountCardConfig };
    platformSettings = patch;
    await persistPlatform();
  }
</script>

<div class="main-content cardeditor-page">
  <!-- The title bar shows this visually; the page itself needs a heading so
       its section legends hang off a real outline. -->
  <h1 class="sr-only">
    {isPlatform
      ? $t("CardEditor_TitleForPlatform", { platform: platformName ?? "" })
      : $t("CardEditor_Title")}
  </h1>
  {#if loadError}
    <p class="cardeditor-page__error">{loadError}</p>
  {/if}

  {#if isPlatform}
    <div class="cardeditor-page__override">
      <!-- The app draws checkboxes as the adjacent label; the input itself
           is transparent, so the pair has to stay together. -->
      <span class="cardeditor-page__toggle">
        <span class="form-check">
          <input
            type="checkbox"
            class="form-check-input"
            id="cardeditor-override"
            checked={overrideEnabled}
            disabled={loading || !platformSettings}
            on:change={() => void toggleOverride()}
          />
          <label class="form-check-label" for="cardeditor-override"></label>
        </span>
        <label for="cardeditor-override">{$t("CardEditor_PlatformOverride")}</label>
      </span>
      <p class="cardeditor-page__hint">{$t("CardEditor_PlatformOverride_Hint")}</p>
    </div>
  {/if}

  {#if !loading}
    <AccountCardEditor
      config={shownConfig}
      {available}
      disabled={isPlatform && !overrideEnabled}
      on:change={(e) => void onChange(e.detail)}
    />
  {/if}
</div>

<style lang="scss">
  .cardeditor-page {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    padding: 0.75rem 0.9375rem 1.5rem;
    overflow-y: auto;
  }

  .cardeditor-page__error {
    margin: 0;
    color: var(--red);
  }

  .cardeditor-page__override {
    padding-bottom: 0.5625rem;
    border-bottom: 1px solid var(--border-bar-bg, var(--button-bg));
  }

  .cardeditor-page__toggle {
    display: inline-flex;
    align-items: center;
    gap: 0.3375rem;
    font-weight: 600;

    label {
      cursor: pointer;
    }
  }

  .cardeditor-page__hint {
    margin: 0.225rem 0 0;
    font-size: 0.615rem;
    opacity: 0.72;
    max-width: 60ch;
  }
</style>
