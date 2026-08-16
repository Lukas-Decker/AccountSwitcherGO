<script lang="ts">
  /*
    Theme and accent pickers for the CSS preview page, on the same
    SettingsSelect component the App settings Theme card uses: two dropdowns
    doing the same job should look like the same kind of control.
  */
  import { tick } from "svelte";
  import { t } from "../stores/i18n";
  import "../styles/SettingsGrid.scss";
  import SettingsSelect from "./settings/app/SettingsSelect.svelte";
  import {
    currentThemeAccentKey,
    currentThemeCustomAccentColor,
    currentThemeId,
    currentWindowsThemeAccentColor,
    listThemes,
    resolveThemeAccent,
    setUserTheme,
    setUserThemeAccentCustom,
    setUserThemeAccentPreset,
    supportsWindowsThemeAccent,
    WINDOWS_THEME_ACCENT_KEY,
  } from "../lib/themes";

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

  async function pickAccent(id: string): Promise<void> {
    if (id !== CUSTOM_ACCENT_KEY) {
      await setUserThemeAccentPreset(id);
      return;
    }
    await setUserThemeAccentCustom(currentAccent?.color || customPreviewColor);
    // Straight into the colour picker: choosing "Custom" is not the goal.
    await tick();
    if (typeof customAccentInput?.showPicker === "function") {
      customAccentInput.showPicker();
      return;
    }
    customAccentInput?.focus();
  }

  async function updateCustomAccent(event: Event): Promise<void> {
    const input = event.currentTarget as HTMLInputElement | null;
    if (!input) {
      return;
    }
    await setUserThemeAccentCustom(input.value);
  }
</script>

<div class="theme-control-group">
  <div class="rowDropdown">
    <span>{$t("Settings_CurrentStyle")}</span>
    <SettingsSelect
      id="preview-theme"
      label={$t("Settings_CurrentStyle")}
      options={themeOptions}
      value={$currentThemeId}
      fallbackLabel={currentTheme?.label ?? ""}
      on:select={(e) => void setUserTheme(e.detail)}
    />
  </div>

  {#if currentTheme && currentAccent}
    <div class="rowDropdown">
      <span>{$t("Settings_AccentColor")}</span>
      <SettingsSelect
        id="preview-accent"
        label={$t("Settings_AccentColor")}
        options={accentOptions}
        value={accentValue}
        triggerColor={currentAccent.color}
        fallbackLabel={currentAccent.label}
        on:select={(e) => void pickAccent(e.detail)}
      />
    </div>
  {/if}

  {#if currentAccent?.isCustom}
    <div class="accent-custom-picker-row">
      <span>{$t("Settings_AccentColor_CustomPicker")}</span>
      <input
        bind:this={customAccentInput}
        type="color"
        class="accent-custom-picker"
        value={currentAccent.color}
        on:input={(event) => void updateCustomAccent(event)}
        aria-label={$t("Settings_AccentColor_CustomPicker")}
      />
    </div>
  {/if}

  <slot name="after-controls"></slot>
</div>
