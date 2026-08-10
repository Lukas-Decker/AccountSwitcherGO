<script lang="ts">
  import { get } from "svelte/store";
  import { route } from "../stores/nav";
  import { t } from "../stores/i18n";
  import { pushToast } from "../stores/toast";
  import { formatToastWithError } from "../lib/formatWailsError";
  import * as PlatformService from "../../bindings/account-switcher/internal/platform/platformservice.js";
  import { appBgInfo, platformBgInfo, userOverriddenAppBg, setUserOverride } from "../stores/backgroundImage";
  import { currentThemeBgUrl, currentThemeHueRotate, setThemeHueRotate } from "../lib/themes";
  import ThemePickerControls from "./ThemePickerControls.svelte";
  import BackgroundSettings from "./BackgroundSettings.svelte";

  $: showResetToThemeBg = !!$currentThemeBgUrl && ($appBgInfo.hasImage || $userOverriddenAppBg);

  // The slider drives the palette live while dragging; only the value that is
  // settled on is written to disk.
  let hueDraft = 0;
  let hueDragging = false;
  $: if (!hueDragging) {
    hueDraft = $currentThemeHueRotate;
  }

  function onHueInput(event: Event): void {
    hueDragging = true;
    hueDraft = Number((event.currentTarget as HTMLInputElement).value);
    void setThemeHueRotate(hueDraft);
  }

  function onHueCommit(): void {
    hueDragging = false;
    void setThemeHueRotate(hueDraft);
  }

  function resetHue(): void {
    hueDragging = false;
    hueDraft = 0;
    void setThemeHueRotate(0);
  }

  async function resetToThemeBg(): Promise<void> {
    try {
      if ($appBgInfo.hasImage) {
        await PlatformService.ClearAppBackground();
      }
      await setUserOverride(false);
      const info = await PlatformService.GetAppBackground();
      appBgInfo.set(info);
    } catch (e) {
      pushToast({
        type: "error",
        message: formatToastWithError(get(t)("Toast_SaveFailed"), e),
        duration: 8000,
      });
    }
  }
</script>

<h2 class="SettingsHeader">{$t("Settings_Header_Theme")}</h2>
<div>
  <ThemePickerControls>
    <button
      slot="after-controls"
      type="button"
      class="btnicontext"
      on:click={() => route.set({ page: "preview-css" })}
    >
      {$t("PreviewCss")}
    </button>
  </ThemePickerControls>
</div>

<div class="rowSetting hue-row">
  <label class="hue-label" for="settings-theme-hue">{$t("Settings_ThemeHue")}</label>
  <input
    id="settings-theme-hue"
    class="hue-slider"
    type="range"
    min="0"
    max="359"
    step="1"
    bind:value={hueDraft}
    on:input={onHueInput}
    on:change={onHueCommit}
    on:pointerup={onHueCommit}
    aria-describedby="settings-theme-hue-value"
  />
  <span id="settings-theme-hue-value" class="hue-value">{hueDraft}&deg;</span>
  <button type="button" class="btnicontext" disabled={hueDraft === 0} on:click={resetHue}>
    {$t("Settings_ThemeHueReset")}
  </button>
</div>
<p class="hue-hint">{$t("Settings_ThemeHue_Hint")}</p>

{#if $appBgInfo.hasImage || showResetToThemeBg}
  <div class="bg-settings-row">
    {#if showResetToThemeBg}
      <button type="button" class="btnicontext" on:click={() => void resetToThemeBg()}>
        {$t("Settings_ResetToThemeBackground")}
      </button>
    {/if}
    {#if $appBgInfo.hasImage}
      <BackgroundSettings target="app" />
    {/if}
  </div>
{/if}

{#if $platformBgInfo.hasImage}
  <div class="bg-settings-row">
    <BackgroundSettings target="platform" />
  </div>
{/if}

<style lang="scss">
  button {
    position: relative;
    height: 38px;
  }

  .hue-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
  }

  .hue-label {
    flex: 0 0 auto;
  }

  .hue-slider {
    flex: 1 1 12rem;
    min-width: 8rem;
    height: 1.25rem;
    // The track is the rotation itself, so the control shows what it does.
    background: linear-gradient(
      to right,
      hsl(0, 80%, 60%),
      hsl(60, 80%, 60%),
      hsl(120, 80%, 60%),
      hsl(180, 80%, 60%),
      hsl(240, 80%, 60%),
      hsl(300, 80%, 60%),
      hsl(360, 80%, 60%)
    );
    border-radius: 0.625rem;
    appearance: none;
    cursor: pointer;

    &::-webkit-slider-thumb {
      appearance: none;
      width: 1rem;
      height: 1rem;
      border-radius: 50%;
      background: var(--whiteSecondary, #fff);
      border: 2px solid var(--black, #21222c);
      cursor: grab;
    }

    &:active::-webkit-slider-thumb {
      cursor: grabbing;
    }
  }

  .hue-value {
    flex: 0 0 3.5rem;
    text-align: right;
    font-variant-numeric: tabular-nums;
  }

  .hue-hint {
    margin: 0.15rem 0 0.5rem;
    font-size: 0.85rem;
    opacity: 0.75;
  }
</style>
