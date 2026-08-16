<script lang="ts">
  /* Theme, accent, colour rotation, and whatever background image is set. */
  import { get } from "svelte/store";
  import { route } from "../../../stores/nav";
  import { t } from "../../../stores/i18n";
  import { pushToast } from "../../../stores/toast";
  import { formatToastWithError } from "../../../lib/formatWailsError";
  import * as PlatformService from "../../../../bindings/account-switcher/internal/platform/platformservice.js";
  import {
    appBgInfo,
    platformBgInfo,
    setUserOverride,
    userOverriddenAppBg,
  } from "../../../stores/backgroundImage";
  import {
    currentThemeAccentKey,
    currentThemeBgUrl,
    currentThemeCustomAccentColor,
    currentThemeHueRotate,
    currentThemeId,
    currentWindowsThemeAccentColor,
    listThemes,
    resolveThemeAccent,
    setThemeHueRotate,
    setUserTheme,
    setUserThemeAccentCustom,
    setUserThemeAccentPreset,
    supportsWindowsThemeAccent,
    WINDOWS_THEME_ACCENT_KEY,
  } from "../../../lib/themes";
  import { animations, roundedCornersToggle } from "../../../lib/appSettingsModel";
  import { settingsIcons } from "./settingsIcons";
  import SettingsCard from "./SettingsCard.svelte";
  import SettingsRow from "./SettingsRow.svelte";
  import SettingsSelect from "./SettingsSelect.svelte";
  import SettingsSwitch from "./SettingsSwitch.svelte";
  import BackgroundControls from "./BackgroundControls.svelte";
  import { uiScaleEffective, uiScaleSetting, setUiScale } from "../../../stores/uiScale";
  import { applyUiScale, UI_SCALE_AUTO, UI_SCALE_MAX, UI_SCALE_MIN } from "../../../lib/uiScale";

  const CUSTOM_ACCENT_KEY = "__custom__";

  const themes = listThemes();
  const showWindowsAccent = supportsWindowsThemeAccent();

  let customAccentInput: HTMLInputElement | null = null;

  $: themeOptions = themes.map((theme) => ({ value: theme.id, label: theme.label }));
  $: currentTheme = themes.find((theme) => theme.id === $currentThemeId) ?? themes[0];
  $: currentAccent = currentTheme
    ? resolveThemeAccent(currentTheme.id, $currentThemeAccentKey, $currentThemeCustomAccentColor)
    : null;
  $: customPreviewColor = $currentThemeCustomAccentColor || currentAccent?.color || "#ffffff";

  $: accentOptions = [
    { value: CUSTOM_ACCENT_KEY, label: $t("Settings_AccentColor_Custom"), color: customPreviewColor },
    ...(showWindowsAccent
      ? [
          {
            value: WINDOWS_THEME_ACCENT_KEY,
            label: $t("Settings_WindowsAccent"),
            color: $currentWindowsThemeAccentColor || "#0078d4",
          },
        ]
      : []),
    ...(currentTheme?.accents ?? []).map((accent) => ({
      value: accent.id,
      label: accent.label,
      color: accent.color,
    })),
  ];

  $: accentValue = currentAccent?.isCustom
    ? CUSTOM_ACCENT_KEY
    : ($currentThemeAccentKey || currentTheme?.defaultAccentKey || "");

  // Dragging drives the palette live; only the settled value reaches the disk.
  let hueDraft = 0;
  let hueDragging = false;
  $: if (!hueDragging) {
    hueDraft = $currentThemeHueRotate;
  }

  $: showResetToThemeBg = !!$currentThemeBgUrl && ($appBgInfo.hasImage || $userOverriddenAppBg);

  async function pickAccent(id: string): Promise<void> {
    if (id !== CUSTOM_ACCENT_KEY) {
      await setUserThemeAccentPreset(id);
      return;
    }
    await setUserThemeAccentCustom(currentAccent?.color || customPreviewColor);
    // Straight into the colour picker: choosing "Custom" is not the goal.
    if (typeof customAccentInput?.showPicker === "function") {
      customAccentInput.showPicker();
      return;
    }
    customAccentInput?.focus();
  }

  // Shown as a percentage. Dragging applies live so the effect is visible while
  // choosing it; only the committed value is written to settings.
  let uiScaleDraft = 100;
  let uiScaleDragging = false;
  $: if (!uiScaleDragging) {
    uiScaleDraft = Math.round($uiScaleEffective * 100);
  }

  function onUiScaleInput(e: Event): void {
    uiScaleDragging = true;
    uiScaleDraft = Number((e.currentTarget as HTMLInputElement).value);
    applyUiScale(uiScaleDraft / 100);
  }

  function onUiScaleCommit(): void {
    if (!uiScaleDragging) return;
    uiScaleDragging = false;
    void setUiScale(uiScaleDraft / 100);
  }

  function resetUiScale(): void {
    uiScaleDragging = false;
    void setUiScale(UI_SCALE_AUTO);
  }

  function onHueInput(e: Event): void {
    hueDragging = true;
    hueDraft = Number((e.currentTarget as HTMLInputElement).value);
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
      appBgInfo.set(await PlatformService.GetAppBackground());
    } catch (e) {
      pushToast({
        type: "error",
        message: formatToastWithError(get(t)("Toast_SaveFailed"), e),
        duration: 8000,
      });
    }
  }
</script>

<SettingsCard
  title={$t("Settings_Header_Theme")}
  icon={settingsIcons.appearance}
  keywords="appearance colour color style look"
>
  <SettingsRow label={$t("Settings_CurrentStyle")} controlId="settings-theme">
    <SettingsSelect
      id="settings-theme"
      label={$t("Settings_CurrentStyle")}
      options={themeOptions}
      value={$currentThemeId}
      fallbackLabel={currentTheme?.label ?? ""}
      on:select={(e) => void setUserTheme(e.detail)}
    />
  </SettingsRow>

  {#if currentTheme && currentAccent}
    <SettingsRow label={$t("Settings_AccentColor")} controlId="settings-accent">
      <SettingsSelect
        id="settings-accent"
        label={$t("Settings_AccentColor")}
        options={accentOptions}
        value={accentValue}
        triggerColor={currentAccent.color}
        fallbackLabel={currentAccent.label}
        on:select={(e) => void pickAccent(e.detail)}
      />
    </SettingsRow>
  {/if}

  {#if currentAccent?.isCustom}
    <SettingsRow label={$t("Settings_AccentColor_CustomPicker")} controlId="settings-accent-custom">
      <input
        id="settings-accent-custom"
        bind:this={customAccentInput}
        class="settings-color"
        type="color"
        value={currentAccent.color}
        aria-label={$t("Settings_AccentColor_CustomPicker")}
        on:input={(e) => void setUserThemeAccentCustom(e.currentTarget.value)}
      />
    </SettingsRow>
  {/if}

  <SettingsRow
    label={$t("Settings_ThemeHue")}
    hint={$t("Settings_ThemeHue_Hint")}
    controlId="settings-theme-hue"
    stacked
    keywords="hue rotation colour wheel"
  >
    <input
      id="settings-theme-hue"
      class="settings-slider settings-slider--hue"
      type="range"
      min="0"
      max="359"
      step="1"
      bind:value={hueDraft}
      on:input={onHueInput}
      on:change={onHueCommit}
      on:pointerup={onHueCommit}
    />
    <span class="settings-value">{hueDraft}&deg;</span>
    <button
      type="button"
      class="settings-btn settings-btn--ghost"
      disabled={hueDraft === 0}
      on:click={resetHue}
    >
      {$t("Settings_ThemeHueReset")}
    </button>
  </SettingsRow>

  <SettingsRow
    label={$t("Settings_UiScale")}
    hint={$t("Settings_UiScale_Hint")}
    controlId="settings-ui-scale"
    stacked
    keywords="ui scale zoom dpi size resolution readability"
  >
    <input
      id="settings-ui-scale"
      class="settings-slider"
      type="range"
      min={UI_SCALE_MIN * 100}
      max={UI_SCALE_MAX * 100}
      step="5"
      value={uiScaleDraft}
      on:input={onUiScaleInput}
      on:change={onUiScaleCommit}
      on:pointerup={onUiScaleCommit}
    />
    <span class="settings-value">{uiScaleDraft}%</span>
    <button
      type="button"
      class="settings-btn settings-btn--ghost"
      disabled={$uiScaleSetting === 0}
      on:click={resetUiScale}
    >
      {$t("Settings_UiScale_Auto")}
    </button>
  </SettingsRow>

  <SettingsRow label={$t("Settings_RoundedCorners")} controlId="settings-rounded">
    <SettingsSwitch
      id="settings-rounded"
      toggle={roundedCornersToggle}
      label={$t("Settings_RoundedCorners")}
    />
  </SettingsRow>

  <SettingsRow
    label={$t("Settings_AnimationsEnabled")}
    controlId="settings-animations"
    keywords="motion transitions"
  >
    <SettingsSwitch
      id="settings-animations"
      toggle={animations}
      label={$t("Settings_AnimationsEnabled")}
    />
  </SettingsRow>

  {#if $appBgInfo.hasImage}
    <BackgroundControls
      target="app"
      onResetToTheme={showResetToThemeBg ? () => void resetToThemeBg() : null}
    />
  {:else if showResetToThemeBg}
    <SettingsRow label={$t("Settings_BackgroundImage")} keywords="background image wallpaper">
      <button type="button" class="settings-btn settings-btn--ghost" on:click={() => void resetToThemeBg()}>
        {$t("Settings_ResetToThemeBackground")}
      </button>
    </SettingsRow>
  {/if}

  {#if $platformBgInfo.hasImage}
    <BackgroundControls target="platform" />
  {/if}

  <SettingsRow label={$t("Settings_PreviewCssHeader")} keywords="css theme preview developer">
    <button
      type="button"
      class="settings-btn settings-btn--ghost"
      on:click={() => route.set({ page: "preview-css" })}
    >
      {$t("PreviewCss")}
    </button>
  </SettingsRow>
</SettingsCard>
